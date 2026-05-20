package comment

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/bin-ke/my-notion/internal/notification"
	"github.com/bin-ke/my-notion/pkg/db"
	"gorm.io/gorm"
)

type Service struct {
	DB                  *gorm.DB
	NotificationService *notification.Service
}

func NewService(database *gorm.DB, notifService *notification.Service) *Service {
	return &Service{DB: database, NotificationService: notifService}
}

// Create creates a comment and triggers notifications for @mentions.
func (s *Service) Create(pageID uint, blockID *uint, authorID uint, contentJSON string, parentID *uint) (*db.Comment, error) {
	comment := db.Comment{
		PageID:   pageID,
		BlockID:  blockID,
		AuthorID: authorID,
		Content:  contentJSON,
		ParentID: parentID,
	}

	tx := s.DB.Begin()

	if err := tx.Create(&comment).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Preload author
	tx.Preload("Author").First(&comment, comment.ID)

	tx.Commit()

	// Parse @mentions and create notifications (non-blocking)
	go s.notifyMentions(&comment)

	// Notify page watchers (simplified: notify all users who have permissions on this page)
	go s.notifyNewComment(&comment)

	return &comment, nil
}

// Update edits a comment's content.
func (s *Service) Update(commentID uint, contentJSON string) (*db.Comment, error) {
	var comment db.Comment
	if err := s.DB.First(&comment, commentID).Error; err != nil {
		return nil, errors.New("comment not found")
	}

	comment.Content = contentJSON
	if err := s.DB.Save(&comment).Error; err != nil {
		return nil, err
	}

	s.DB.Preload("Author").First(&comment, commentID)
	return &comment, nil
}

// Resolve toggles the resolved flag on a comment.
func (s *Service) Resolve(commentID uint) (*db.Comment, error) {
	var comment db.Comment
	if err := s.DB.First(&comment, commentID).Error; err != nil {
		return nil, errors.New("comment not found")
	}

	comment.Resolved = !comment.Resolved
	if err := s.DB.Save(&comment).Error; err != nil {
		return nil, err
	}

	s.DB.Preload("Author").First(&comment, commentID)
	return &comment, nil
}

// Delete removes a comment. If it's a parent comment, all replies are also deleted.
func (s *Service) Delete(commentID uint) error {
	// Delete child comments first
	s.DB.Where("parent_id = ?", commentID).Delete(&db.Comment{})
	return s.DB.Delete(&db.Comment{}, commentID).Error
}

// ListByPage returns all comments for a page, ordered by creation time. Top-level
// comments first, threaded by parent_id.
func (s *Service) ListByPage(pageID uint) ([]db.Comment, error) {
	var comments []db.Comment
	err := s.DB.Where("page_id = ?", pageID).
		Preload("Author").
		Order("created_at ASC").
		Find(&comments).Error
	return comments, err
}

// notifyMentions scans comment content for @username patterns and creates
// notifications for each mentioned user.
func (s *Service) notifyMentions(comment *db.Comment) {
	var content struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(comment.Content), &content); err != nil {
		return
	}

	words := strings.Fields(content.Text)
	for _, word := range words {
		if strings.HasPrefix(word, "@") && len(word) > 1 {
			username := word[1:]
			// Strip trailing punctuation
			username = strings.TrimRight(username, ".,;:!?)")

			// Look up user by name
			var user db.User
			if err := s.DB.Where("name = ?", username).First(&user).Error; err == nil {
				// Don't notify self
				if user.ID != comment.AuthorID {
					s.NotificationService.Create(
						user.ID,
						"mention",
						comment.AuthorID,
						comment.PageID,
						&comment.ID,
					)
				}
			}
		}
	}
}

// notifyNewComment creates notifications for users who have permissions on this page.
func (s *Service) notifyNewComment(comment *db.Comment) {
	var perms []db.PagePermission
	s.DB.Where("page_id = ?", comment.PageID).Find(&perms)
	for _, perm := range perms {
		if perm.SubjectID != comment.AuthorID {
			s.NotificationService.Create(
				perm.SubjectID,
				"comment",
				comment.AuthorID,
				comment.PageID,
				&comment.ID,
			)
		}
	}
}
