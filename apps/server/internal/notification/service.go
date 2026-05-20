package notification

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

// Create creates a new notification.
func (s *Service) Create(userID uint, notifType string, actorID uint, targetPageID uint, commentID *uint) (*db.Notification, error) {
	n := db.Notification{
		UserID:       userID,
		Type:         notifType,
		ActorID:      actorID,
		TargetPageID: targetPageID,
		CommentID:    commentID,
	}
	if err := s.DB.Create(&n).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

// ListByUser returns the most recent 50 notifications for a user.
func (s *Service) ListByUser(userID uint) ([]db.Notification, error) {
	var notifications []db.Notification
	err := s.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(50).
		Find(&notifications).Error
	return notifications, err
}

// MarkRead marks a single notification as read.
func (s *Service) MarkRead(notificationID uint, userID uint) error {
	return s.DB.Model(&db.Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Update("read", true).Error
}

// MarkAllRead marks all notifications for a user as read.
func (s *Service) MarkAllRead(userID uint) error {
	return s.DB.Model(&db.Notification{}).
		Where("user_id = ?", userID).
		Update("read", true).Error
}

// UnreadCount returns the number of unread notifications for a user.
func (s *Service) UnreadCount(userID uint) (int64, error) {
	var count int64
	err := s.DB.Model(&db.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Count(&count).Error
	return count, err
}
