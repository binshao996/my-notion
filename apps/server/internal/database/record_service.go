package database

import (
	"encoding/json"
	"errors"

	"github.com/bin-ke/my-notion/pkg/db"
	"github.com/bin-ke/my-notion/pkg/position"
	"gorm.io/gorm"
)

type RecordService struct {
	DB *gorm.DB
}

func NewRecordService(database *gorm.DB) *RecordService {
	return &RecordService{DB: database}
}

func (s *RecordService) Create(databaseID uint, propertyValues map[uint]string, createdBy uint) (*db.Record, error) {
	// Load database first to get workspaceID
	var database db.Database
	if err := s.DB.First(&database, databaseID).Error; err != nil {
		return nil, err
	}

	tx := s.DB.Begin()

	// Extract title from property values
	title := "Untitled"
	var titleProperty db.Property
	if err := tx.Where("database_id = ? AND type = ?", databaseID, "title").First(&titleProperty).Error; err == nil {
		if val, ok := propertyValues[titleProperty.ID]; ok && val != "" {
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(val), &m); err == nil {
				if t, ok := m["text"]; ok {
					if ts, ok := t.(string); ok {
						title = ts
					}
				}
			}
		}
	}

	// Create container page
	page := db.Page{
		WorkspaceID: database.WorkspaceID,
		Title:       title,
		CreatedBy:   createdBy,
	}
	if err := tx.Create(&page).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Determine position
	pos := "a0"
	var lastRecord db.Record
	if err := tx.Where("database_id = ?", databaseID).Order("position DESC").Limit(1).First(&lastRecord).Error; err == nil {
		pos = position.Between(lastRecord.Position, "")
	}

	// Create record
	record := db.Record{
		DatabaseID: databaseID,
		PageID:     page.ID,
		Position:   pos,
	}
	if err := tx.Create(&record).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Create property values
	for propID, value := range propertyValues {
		pv := db.PropertyValue{
			RecordID:   record.ID,
			PropertyID: propID,
			Value:      value,
		}
		if err := tx.Create(&pv).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	tx.Commit()
	return &record, nil
}

func (s *RecordService) Update(id uint, propertyValues map[uint]string) error {
	for propID, value := range propertyValues {
		var pv db.PropertyValue
		if err := s.DB.Where("record_id = ? AND property_id = ?", id, propID).
			Assign(db.PropertyValue{Value: value}).
			FirstOrCreate(&pv).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *RecordService) Delete(id uint) error {
	tx := s.DB.Begin()

	// Delete property values for this record
	if err := tx.Where("record_id = ?", id).Delete(&db.PropertyValue{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Load record to get page ID
	var record db.Record
	if err := tx.First(&record, id).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete record
	if err := tx.Delete(&record).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete container page
	if err := tx.Delete(&db.Page{}, record.PageID).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (s *RecordService) ListByDatabase(databaseID uint) ([]db.Record, error) {
	var records []db.Record
	if err := s.DB.Where("database_id = ?", databaseID).Order("position ASC").Find(&records).Error; err != nil {
		return nil, err
	}

	// Resolve rollup property values for all records
	var rollupProps []db.Property
	s.DB.Where("database_id = ? AND type = ?", databaseID, "rollup").Find(&rollupProps)
	if len(rollupProps) > 0 {
		rs := NewRollupService(s.DB)
		for _, record := range records {
			for _, prop := range rollupProps {
				computedValue, err := rs.ComputeRollup(record.ID, prop.Config)
				if err != nil {
					continue
				}
				// Upsert the computed value
				s.DB.Where("record_id = ? AND property_id = ?", record.ID, prop.ID).
					Assign(db.PropertyValue{Value: computedValue}).
					FirstOrCreate(&db.PropertyValue{})
			}
		}
	}

	return records, nil
}

func (s *RecordService) GetByID(id uint) (*db.Record, []db.PropertyValue, []db.Property, error) {
	var record db.Record
	if err := s.DB.First(&record, id).Error; err != nil {
		return nil, nil, nil, errors.New("record not found")
	}

	var values []db.PropertyValue
	s.DB.Where("record_id = ?", id).Find(&values)

	var properties []db.Property
	s.DB.Where("database_id = ?", record.DatabaseID).Order("position ASC").Find(&properties)

	return &record, values, properties, nil
}
