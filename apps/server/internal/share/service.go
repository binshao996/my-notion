package share

import (
	"errors"
	"time"

	"github.com/bin-ke/my-notion/pkg/db"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	DB *gorm.DB
}

func NewService(database *gorm.DB) *Service {
	return &Service{DB: database}
}

// Create generates a new share token for a page.
func (s *Service) Create(pageID uint, role string, expiresAt *time.Time, createdBy uint) (*db.ShareToken, error) {
	if role == "" {
		role = "viewer"
	}

	token := db.ShareToken{
		Token:     uuid.New().String(),
		PageID:    pageID,
		Role:      role,
		ExpiresAt: expiresAt,
		CreatedBy: createdBy,
	}
	if err := s.DB.Create(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

// Revoke deletes a share token by its ID.
func (s *Service) Revoke(tokenID uint) error {
	return s.DB.Delete(&db.ShareToken{}, tokenID).Error
}

// ListByPage returns all active share tokens for a page.
func (s *Service) ListByPage(pageID uint) ([]db.ShareToken, error) {
	var tokens []db.ShareToken
	err := s.DB.Where("page_id = ?", pageID).Find(&tokens).Error
	return tokens, err
}

// ResolveToken validates a share token and returns the associated page data.
// Returns an error if the token is expired or not found.
func (s *Service) ResolveToken(tokenStr string) (*db.ShareToken, *db.Page, []db.Block, error) {
	var token db.ShareToken
	if err := s.DB.Where("token = ?", tokenStr).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, errors.New("share link not found")
		}
		return nil, nil, nil, err
	}

	// Check expiry
	if token.ExpiresAt != nil && time.Now().After(*token.ExpiresAt) {
		return nil, nil, nil, errors.New("share link has expired")
	}

	// Load the page
	var page db.Page
	if err := s.DB.First(&page, token.PageID).Error; err != nil {
		return nil, nil, nil, errors.New("page not found")
	}

	// Load the page's blocks
	var blocks []db.Block
	s.DB.Where("page_id = ?", page.ID).Order("position ASC").Find(&blocks)

	return &token, &page, blocks, nil
}
