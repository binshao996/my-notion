package database

import (
	"errors"

	"github.com/bin-ke/my-notion/pkg/db"
	"github.com/bin-ke/my-notion/pkg/position"
	"gorm.io/gorm"
)

type PropertyService struct {
	DB *gorm.DB
}

func NewPropertyService(database *gorm.DB) *PropertyService {
	return &PropertyService{DB: database}
}

func (s *PropertyService) Create(databaseID uint, name, propType, config string) (*db.Property, error) {
	pos := "a0"
	var lastProp db.Property
	if err := s.DB.Where("database_id = ?", databaseID).Order("position DESC").Limit(1).First(&lastProp).Error; err == nil {
		pos = position.Between(lastProp.Position, "")
	}

	prop := db.Property{
		DatabaseID: databaseID,
		Name:       name,
		Type:       propType,
		Config:     config,
		Position:   pos,
	}
	if err := s.DB.Create(&prop).Error; err != nil {
		return nil, err
	}
	return &prop, nil
}

func (s *PropertyService) Update(id uint, updates map[string]interface{}) (*db.Property, error) {
	var prop db.Property
	if err := s.DB.First(&prop, id).Error; err != nil {
		return nil, errors.New("property not found")
	}
	if err := s.DB.Model(&prop).Updates(updates).Error; err != nil {
		return nil, err
	}
	s.DB.First(&prop, id)
	return &prop, nil
}

func (s *PropertyService) Delete(id uint) error {
	tx := s.DB.Begin()

	if err := tx.Where("property_id = ?", id).Delete(&db.PropertyValue{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Delete(&db.Property{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (s *PropertyService) ListByDatabase(databaseID uint) ([]db.Property, error) {
	var props []db.Property
	err := s.DB.Where("database_id = ?", databaseID).Order("position ASC").Find(&props).Error
	return props, err
}
