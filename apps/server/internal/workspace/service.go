package workspace

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

func (s *Service) Create(name string, ownerID uint) (*db.Workspace, error) {
	ws := &db.Workspace{Name: name}

	tx := s.DB.Begin()

	if err := tx.Create(ws).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	member := &db.WorkspaceMember{
		WorkspaceID: ws.ID,
		UserID:      ownerID,
		Role:        "owner",
	}
	if err := tx.Create(member).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	return ws, nil
}

func (s *Service) GetByID(id uint) (*db.Workspace, error) {
	var ws db.Workspace
	if err := s.DB.First(&ws, id).Error; err != nil {
		return nil, errors.New("workspace not found")
	}
	return &ws, nil
}

func (s *Service) ListByUser(userID uint) ([]db.Workspace, error) {
	var members []db.WorkspaceMember
	if err := s.DB.Preload("Workspace").Where("user_id = ?", userID).Find(&members).Error; err != nil {
		return nil, err
	}

	workspaces := make([]db.Workspace, len(members))
	for i, m := range members {
		workspaces[i] = m.Workspace
	}
	return workspaces, nil
}
