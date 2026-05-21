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

func (s *Service) Update(id uint, name string) (*db.Workspace, error) {
	var ws db.Workspace
	if err := s.DB.First(&ws, id).Error; err != nil {
		return nil, errors.New("workspace not found")
	}
	if name != "" {
		ws.Name = name
	}
	if err := s.DB.Save(&ws).Error; err != nil {
		return nil, err
	}
	return &ws, nil
}

func (s *Service) Delete(id uint) error {
	return s.DB.Delete(&db.Workspace{}, id).Error
}

func (s *Service) IsOwner(workspaceID, userID uint) bool {
	var member db.WorkspaceMember
	err := s.DB.Where("workspace_id = ? AND user_id = ? AND role = ?", workspaceID, userID, "owner").First(&member).Error
	return err == nil
}

func (s *Service) ListMembers(workspaceID uint) ([]db.WorkspaceMember, error) {
	var members []db.WorkspaceMember
	err := s.DB.Preload("User").Where("workspace_id = ?", workspaceID).Find(&members).Error
	return members, err
}

func (s *Service) AddMember(workspaceID, userID uint, role string) error {
	member := db.WorkspaceMember{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        role,
	}
	return s.DB.Create(&member).Error
}

func (s *Service) RemoveMember(workspaceID, userID uint) error {
	return s.DB.Where("workspace_id = ? AND user_id = ?", workspaceID, userID).Delete(&db.WorkspaceMember{}).Error
}
