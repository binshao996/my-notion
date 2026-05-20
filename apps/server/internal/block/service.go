package block

import (
	"encoding/json"

	"github.com/bin-ke/my-notion/internal/search"
	"github.com/bin-ke/my-notion/pkg/db"
	"github.com/bin-ke/my-notion/pkg/position"
	"gorm.io/gorm"
)

type Service struct {
	DB            *gorm.DB
	SearchService *search.Service
}

func NewService(database *gorm.DB) *Service {
	return &Service{DB: database}
}

func (s *Service) GetByPage(pageID uint) ([]db.Block, error) {
	var blocks []db.Block
	err := s.DB.Where("page_id = ?", pageID).Order("position ASC").Find(&blocks).Error
	return blocks, err
}

func (s *Service) BatchSave(pageID uint, blocks []db.Block) ([]db.Block, error) {
	tx := s.DB.Begin()

	// Delete existing blocks for this page
	if err := tx.Where("page_id = ?", pageID).Delete(&db.Block{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Assign new positions and insert
	positions := position.GenerateN(len(blocks))
	for i := range blocks {
		blocks[i].ID = 0 // force new insert
		blocks[i].PageID = pageID
		blocks[i].Position = positions[i]
	}

	if len(blocks) > 0 {
		if err := tx.Create(&blocks).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	tx.Commit()

	// Index blocks for search
	if s.SearchService != nil {
		var page db.Page
		if err := s.DB.First(&page, pageID).Error; err == nil {
			for _, b := range blocks {
				text := search.ExtractText(b.Props)
				if text != "" {
					s.SearchService.IndexBlock(b.ID, pageID, page.WorkspaceID, b.Type, text)
				}
			}
		}
	}

	return blocks, nil
}

type OpRequest struct {
	Type     string          `json:"type"`
	BlockID  uint            `json:"block_id,omitempty"`
	Position string          `json:"position,omitempty"`
	Props    json.RawMessage `json:"props,omitempty"`
}

func (s *Service) ApplyOps(pageID uint, ops []OpRequest) error {
	// Stub for M4 collaboration. M1 uses BatchSave instead.
	return nil
}
