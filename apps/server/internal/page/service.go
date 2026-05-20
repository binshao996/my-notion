package page

import (
	"errors"

	"github.com/bin-ke/my-notion/internal/search"
	"github.com/bin-ke/my-notion/pkg/db"
	"gorm.io/gorm"
)

type Service struct {
	DB            *gorm.DB
	SearchService *search.Service
}

func NewService(database *gorm.DB) *Service {
	return &Service{DB: database}
}

func (s *Service) Create(workspaceID, createdBy uint, title string, parentPageID *uint) (*db.Page, error) {
	page := &db.Page{
		WorkspaceID:  workspaceID,
		Title:        title,
		CreatedBy:    createdBy,
		ParentPageID: parentPageID,
	}
	if err := s.DB.Create(page).Error; err != nil {
		return nil, err
	}

	if s.SearchService != nil {
		s.SearchService.IndexPage(page.ID, page.WorkspaceID, page.Title)
	}

	return page, nil
}

func (s *Service) GetByID(id uint) (*db.Page, error) {
	var page db.Page
	if err := s.DB.First(&page, id).Error; err != nil {
		return nil, errors.New("page not found")
	}
	return &page, nil
}

func (s *Service) Update(id uint, updates map[string]interface{}) (*db.Page, error) {
	var page db.Page
	if err := s.DB.First(&page, id).Error; err != nil {
		return nil, errors.New("page not found")
	}
	if err := s.DB.Model(&page).Updates(updates).Error; err != nil {
		return nil, err
	}

	if s.SearchService != nil {
		var updated db.Page
		if err := s.DB.First(&updated, id).Error; err == nil {
			if updated.Archived {
				s.SearchService.DeletePage(updated.ID)
			} else {
				s.SearchService.IndexPage(updated.ID, updated.WorkspaceID, updated.Title)
			}
		}
	}

	return &page, nil
}

func (s *Service) GetChildren(parentID uint) ([]db.Page, error) {
	var pages []db.Page
	err := s.DB.Where("parent_page_id = ? AND archived = false", parentID).
		Order("title ASC").Find(&pages).Error
	return pages, err
}

func (s *Service) GetWorkspaceTree(workspaceID uint) ([]db.Page, error) {
	var pages []db.Page
	err := s.DB.Where("workspace_id = ? AND parent_page_id IS NULL AND archived = false", workspaceID).
		Order("title ASC").Find(&pages).Error
	return pages, err
}
