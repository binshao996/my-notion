package database

import (
	"encoding/json"
	"fmt"

	"github.com/bin-ke/my-notion/pkg/db"
	"gorm.io/gorm"
)

type RollupService struct {
	DB *gorm.DB
}

type RollupConfig struct {
	RelationPropertyID uint   `json:"relation_property_id"`
	TargetPropertyID   uint   `json:"target_property_id"`
	Aggregation        string `json:"aggregation"`
}

func NewRollupService(d *gorm.DB) *RollupService {
	return &RollupService{DB: d}
}

// ComputeRollup reads the relation values for this record, fetches the target
// property values on the related records, and applies the aggregation.
// Returns a JSONB value string suitable for storing in property_values.value.
func (s *RollupService) ComputeRollup(recordID uint, configJSON string) (string, error) {
	var cfg RollupConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return "{}", err
	}

	// Load the relation property value for this record
	var relPV db.PropertyValue
	if err := s.DB.Where("record_id = ? AND property_id = ?", recordID, cfg.RelationPropertyID).First(&relPV).Error; err != nil {
		if cfg.Aggregation == "count" {
			return `{"number": 0}`, nil
		}
		return `{"text": ""}`, nil
	}

	var relValue struct {
		Relation []uint `json:"relation"`
	}
	if err := json.Unmarshal([]byte(relPV.Value), &relValue); err != nil || len(relValue.Relation) == 0 {
		if cfg.Aggregation == "count" {
			return `{"number": 0}`, nil
		}
		return `{"text": ""}`, nil
	}

	// Load target property values for the related records
	var targetValues []db.PropertyValue
	s.DB.Where("record_id IN ? AND property_id = ?", relValue.Relation, cfg.TargetPropertyID).Find(&targetValues)

	switch cfg.Aggregation {
	case "count":
		return fmt.Sprintf(`{"number": %d}`, len(relValue.Relation)), nil

	case "sum", "average", "min", "max":
		nums := extractNumbers(targetValues)
		if len(nums) == 0 {
			return `{"number": 0}`, nil
		}
		switch cfg.Aggregation {
		case "sum":
			total := 0.0
			for _, n := range nums {
				total += n
			}
			return fmt.Sprintf(`{"number": %f}`, total), nil
		case "average":
			total := 0.0
			for _, n := range nums {
				total += n
			}
			return fmt.Sprintf(`{"number": %f}`, total/float64(len(nums))), nil
		case "min":
			m := nums[0]
			for _, n := range nums {
				if n < m {
					m = n
				}
			}
			return fmt.Sprintf(`{"number": %f}`, m), nil
		case "max":
			m := nums[0]
			for _, n := range nums {
				if n > m {
					m = n
				}
			}
			return fmt.Sprintf(`{"number": %f}`, m), nil
		}

	case "show_original":
		for _, pv := range targetValues {
			if pv.Value != "" && pv.Value != "{}" {
				return pv.Value, nil
			}
		}
		return `{"text": ""}`, nil
	}

	return "{}", nil
}

func extractNumbers(values []db.PropertyValue) []float64 {
	var nums []float64
	for _, pv := range values {
		var v map[string]interface{}
		if err := json.Unmarshal([]byte(pv.Value), &v); err != nil {
			continue
		}
		if n, ok := v["number"]; ok {
			switch num := n.(type) {
			case float64:
				nums = append(nums, num)
			case json.Number:
				if f, err := num.Float64(); err == nil {
					nums = append(nums, f)
				}
			}
		}
	}
	return nums
}
