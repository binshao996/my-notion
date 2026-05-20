package permission

import (
	"github.com/bin-ke/my-notion/pkg/db"
	"gorm.io/gorm"
)

type Service struct {
	DB *gorm.DB
}

func NewService(database *gorm.DB) *Service {
	return &Service{DB: database}
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
	return &perm, nil
}

// Remove deletes a permission by its ID.
func (s *Service) Remove(permissionID uint) error {
	return s.DB.Delete(&db.PagePermission{}, permissionID).Error
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
func (s *Service) CheckAccess(userID uint, pageID uint) string {
	// Walk up the parent chain
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
