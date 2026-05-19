package db

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	Name         string    `gorm:"not null" json:"name"`
	AvatarURL    string    `json:"avatar_url"`
	PasswordHash string    `gorm:"not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Workspace struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Icon      string    `json:"icon"`
	CreatedAt time.Time `json:"created_at"`
}

type WorkspaceMember struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	WorkspaceID uint      `gorm:"not null;index" json:"workspace_id"`
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	Role        string    `gorm:"not null;default:member" json:"role"`
	JoinedAt    time.Time `json:"joined_at"`
	Workspace   Workspace `gorm:"foreignKey:WorkspaceID" json:"-"`
	User        User      `gorm:"foreignKey:UserID" json:"-"`
}

func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&User{}, &Workspace{}, &WorkspaceMember{}); err != nil {
		return nil, err
	}

	return db, nil
}
