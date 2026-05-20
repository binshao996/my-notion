package database

import (
	"errors"

	"github.com/bin-ke/my-notion/pkg/db"
	"gorm.io/gorm"
)

type Service struct {
	DB *gorm.DB
}

func NewService(database *gorm.DB) *Service {
	return &Service{DB: database}
}

func (s *Service) Create(workspaceID uint, name string, createdBy uint) (*db.Database, error) {
	tx := s.DB.Begin()

	page := db.Page{
		WorkspaceID: workspaceID,
		Title:       name,
		CreatedBy:   createdBy,
	}
	if err := tx.Create(&page).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	database := db.Database{
		WorkspaceID: workspaceID,
		PageID:      page.ID,
		Name:        name,
	}
	if err := tx.Create(&database).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	prop := db.Property{
		DatabaseID: database.ID,
		Name:       "Name",
		Type:       "title",
		Position:   "a0",
	}
	if err := tx.Create(&prop).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	return &database, nil
}

func (s *Service) GetByID(id uint) (*db.Database, error) {
	var database db.Database
	if err := s.DB.First(&database, id).Error; err != nil {
		return nil, errors.New("database not found")
	}
	return &database, nil
}

func (s *Service) GetByPageID(pageID uint) (*db.Database, error) {
	var database db.Database
	if err := s.DB.Where("page_id = ?", pageID).First(&database).Error; err != nil {
		return nil, errors.New("database not found")
	}
	return &database, nil
}

func (s *Service) Update(id uint, updates map[string]interface{}) (*db.Database, error) {
	var database db.Database
	if err := s.DB.First(&database, id).Error; err != nil {
		return nil, errors.New("database not found")
	}
	if err := s.DB.Model(&database).Updates(updates).Error; err != nil {
		return nil, err
	}
	s.DB.First(&database, id)
	return &database, nil
}

func (s *Service) Delete(id uint) error {
	tx := s.DB.Begin()

	var database db.Database
	if err := tx.First(&database, id).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete property values for records in this database
	if err := tx.Where("record_id IN (SELECT id FROM records WHERE database_id = ?)", id).Delete(&db.PropertyValue{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete all records
	if err := tx.Where("database_id = ?", id).Delete(&db.Record{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete all properties
	if err := tx.Where("database_id = ?", id).Delete(&db.Property{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete all views
	if err := tx.Where("database_id = ?", id).Delete(&db.View{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete the container page
	if err := tx.Delete(&db.Page{}, database.PageID).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete the database itself
	if err := tx.Delete(&database).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}
