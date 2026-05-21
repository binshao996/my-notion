package permission

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bin-ke/my-notion/pkg/db"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Service handles page-level permission checks and caching.
type Service struct {
	DB          *gorm.DB
	RedisClient *redis.Client
}

func NewService(database *gorm.DB) *Service {
	return &Service{DB: database}
}

// SetRedisClient injects a Redis client for permission caching.
func (s *Service) SetRedisClient(rdb *redis.Client) {
	s.RedisClient = rdb
}

// Set creates or updates a page permission. Returns the created/updated permission.
func (s *Service) Set(pageID uint, subjectType string, subjectID uint, role string) (*db.PagePermission, error) {
	var perm db.PagePermission
	err := s.DB.Where("page_id = ? AND subject_type = ? AND subject_id = ?", pageID, subjectType, subjectID).
		Assign(db.PagePermission{Role: role}).
		FirstOrCreate(&perm).Error
	if err != nil {
		return nil, err
	}

	// Invalidate cache for this page after permission change
	s.InvalidateCache(pageID)

	return &perm, nil
}

// Remove deletes a permission by its ID.
func (s *Service) Remove(permissionID uint) error {
	// Load the permission first so we can invalidate the cache
	var perm db.PagePermission
	if err := s.DB.First(&perm, permissionID).Error; err != nil {
		return s.DB.Delete(&db.PagePermission{}, permissionID).Error
	}
	if err := s.DB.Delete(&db.PagePermission{}, permissionID).Error; err != nil {
		return err
	}
	s.InvalidateCache(perm.PageID)
	return nil
}

// ListByPage returns all permissions for a page.
func (s *Service) ListByPage(pageID uint) ([]db.PagePermission, error) {
	var perms []db.PagePermission
	err := s.DB.Where("page_id = ?", pageID).Find(&perms).Error
	return perms, err
}

// CheckAccess walks the parent chain of a page to determine the user's effective role.
// Returns the role string ("editor", "commenter", "viewer") or empty string if no access.
// Workspace owners always get "editor".
// When RedisClient is set, results are cached for 60 seconds under key perm:{userID}:{pageID}.
func (s *Service) CheckAccess(userID uint, pageID uint) string {
	// Check Redis cache first
	if s.RedisClient != nil {
		cacheKey := fmt.Sprintf("perm:%d:%d", userID, pageID)
		ctx := context.Background()
		cached, err := s.RedisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			return cached
		}
		if err != redis.Nil {
			log.Printf("permission: redis get error: %v", err)
		}
	}

	// Walk up the parent chain
	role := s.computeAccess(userID, pageID)

	// Cache the result
	if s.RedisClient != nil {
		cacheKey := fmt.Sprintf("perm:%d:%d", userID, pageID)
		ctx := context.Background()
		if err := s.RedisClient.Set(ctx, cacheKey, role, 60*time.Second).Err(); err != nil {
			log.Printf("permission: redis set error: %v", err)
		}
	}

	return role
}

// computeAccess performs the actual DB lookup for access, without caching.
func (s *Service) computeAccess(userID uint, pageID uint) string {
	currentID := pageID
	for i := 0; i < 50; i++ { // safety limit: max 50 levels deep
		var page db.Page
		if err := s.DB.First(&page, currentID).Error; err != nil {
			return ""
		}

		// Check if user is workspace owner
		var member db.WorkspaceMember
		if err := s.DB.Where("workspace_id = ? AND user_id = ?", page.WorkspaceID, userID).First(&member).Error; err == nil {
			if member.Role == "owner" {
				return "editor"
			}
		}

		// Check direct permission on this page
		var perm db.PagePermission
		if err := s.DB.Where("page_id = ? AND subject_type = ? AND subject_id = ?", currentID, "user", userID).First(&perm).Error; err == nil {
			return perm.Role
		}

		// Move up to parent
		if page.ParentPageID == nil {
			break
		}
		currentID = *page.ParentPageID
	}

	return ""
}

// InvalidateCache deletes all cached permission entries for the given page.
// It uses SCAN to find keys matching perm:*:{pageID} and deletes them.
func (s *Service) InvalidateCache(pageID uint) {
	if s.RedisClient == nil {
		return
	}

	ctx := context.Background()
	pattern := fmt.Sprintf("perm:*:%d", pageID)

	iter := s.RedisClient.Scan(ctx, 0, pattern, 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		log.Printf("permission: redis scan error: %v", err)
		return
	}

	if len(keys) > 0 {
		if err := s.RedisClient.Del(ctx, keys...).Err(); err != nil {
			log.Printf("permission: redis del error: %v", err)
		}
	}
}

// CanEdit returns true if the user can edit the page.
func (s *Service) CanEdit(userID uint, pageID uint) bool {
	return s.CheckAccess(userID, pageID) == "editor"
}

// CanComment returns true if the user can comment on the page.
func (s *Service) CanComment(userID uint, pageID uint) bool {
	role := s.CheckAccess(userID, pageID)
	return role == "editor" || role == "commenter"
}

// CanView returns true if the user can view the page.
func (s *Service) CanView(userID uint, pageID uint) bool {
	return s.CheckAccess(userID, pageID) != ""
}
