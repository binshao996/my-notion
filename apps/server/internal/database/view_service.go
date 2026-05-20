package database

import (
	"errors"

	"github.com/bin-ke/my-notion/pkg/db"
	"github.com/bin-ke/my-notion/pkg/position"
	"gorm.io/gorm"
)

type ViewService struct {
	DB *gorm.DB
}

func NewViewService(database *gorm.DB) *ViewService {
	return &ViewService{DB: database}
}

func (s *ViewService) Create(databaseID uint, name, viewType string) (*db.View, error) {
	pos := "a0"
	var lastView db.View
	if err := s.DB.Where("database_id = ?", databaseID).Order("position DESC").Limit(1).First(&lastView).Error; err == nil {
		pos = position.Between(lastView.Position, "")
	}

	view := db.View{
		DatabaseID: databaseID,
		Name:       name,
		Type:       viewType,
		Config:     "{}",
		Position:   pos,
	}
	if err := s.DB.Create(&view).Error; err != nil {
		return nil, err
	}
	return &view, nil
}

func (s *ViewService) Update(id uint, updates map[string]interface{}) (*db.View, error) {
	var view db.View
	if err := s.DB.First(&view, id).Error; err != nil {
		return nil, errors.New("view not found")
	}
	if err := s.DB.Model(&view).Updates(updates).Error; err != nil {
		return nil, err
	}
	s.DB.First(&view, id)
	return &view, nil
}

func (s *ViewService) Delete(id uint) error {
	return s.DB.Delete(&db.View{}, id).Error
}

func (s *ViewService) ListByDatabase(databaseID uint) ([]db.View, error) {
	var views []db.View
	err := s.DB.Where("database_id = ?", databaseID).Order("position ASC").Find(&views).Error
	return views, err
}
