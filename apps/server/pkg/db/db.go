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

type Page struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	WorkspaceID  uint      `gorm:"not null;index" json:"workspace_id"`
	ParentPageID *uint     `gorm:"index" json:"parent_page_id"`
	Title        string    `gorm:"not null;default:''" json:"title"`
	Icon         string    `json:"icon"`
	Cover        string    `json:"cover"`
	CreatedBy    uint      `gorm:"not null" json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Archived     bool      `gorm:"default:false" json:"archived"`
}

type Block struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	PageID        uint      `gorm:"not null;index" json:"page_id"`
	ParentBlockID *uint     `gorm:"index" json:"parent_block_id"`
	Type          string    `gorm:"not null" json:"type"`
	Position      string    `gorm:"not null" json:"position"`
	Props         string    `gorm:"type:jsonb;not null;default:'{}'" json:"props"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Database struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	WorkspaceID uint      `gorm:"not null;index" json:"workspace_id"`
	PageID      uint      `gorm:"not null;uniqueIndex" json:"page_id"`
	Name        string    `gorm:"not null;default:''" json:"name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Property struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	DatabaseID uint      `gorm:"not null;index" json:"database_id"`
	Name       string    `gorm:"not null" json:"name"`
	Type       string    `gorm:"not null" json:"type"`
	Config     string    `gorm:"type:jsonb;not null;default:'{}'" json:"config"`
	Position   string    `gorm:"not null" json:"position"`
	CreatedAt  time.Time `json:"created_at"`
}

type Record struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	DatabaseID uint      `gorm:"not null;index" json:"database_id"`
	PageID     uint      `gorm:"not null;uniqueIndex" json:"page_id"`
	Position   string    `gorm:"not null" json:"position"`
	CreatedAt  time.Time `json:"created_at"`
}

type PropertyValue struct {
	RecordID   uint   `gorm:"primaryKey;autoIncrement:false" json:"record_id"`
	PropertyID uint   `gorm:"primaryKey;autoIncrement:false" json:"property_id"`
	Value      string `gorm:"type:jsonb;not null;default:'{}'" json:"value"`
}

type View struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	DatabaseID uint      `gorm:"not null;index" json:"database_id"`
	Name       string    `gorm:"not null;default:''" json:"name"`
	Type       string    `gorm:"not null" json:"type"`
	Config     string    `gorm:"type:jsonb;not null;default:'{}'" json:"config"`
	Position   string    `gorm:"not null" json:"position"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type PagePermission struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PageID      uint      `gorm:"not null;index" json:"page_id"`
	SubjectType string    `gorm:"not null;default:user" json:"subject_type"`
	SubjectID   uint      `gorm:"not null" json:"subject_id"`
	Role        string    `gorm:"not null;default:viewer" json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

type ShareToken struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Token     string     `gorm:"uniqueIndex;not null" json:"token"`
	PageID    uint       `gorm:"not null;index" json:"page_id"`
	Role      string     `gorm:"not null;default:viewer" json:"role"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedBy uint       `gorm:"not null" json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
}

type Comment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PageID    uint      `gorm:"not null;index" json:"page_id"`
	BlockID   *uint     `gorm:"index" json:"block_id"`
	AuthorID  uint      `gorm:"not null" json:"author_id"`
	Content   string    `gorm:"type:jsonb;not null;default:'{}'" json:"content"`
	Resolved  bool      `gorm:"default:false" json:"resolved"`
	ParentID  *uint     `gorm:"index" json:"parent_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Author    User      `gorm:"foreignKey:AuthorID" json:"-"`
}

type Notification struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"not null;index" json:"user_id"`
	Type         string    `gorm:"not null" json:"type"`
	ActorID      uint      `json:"actor_id"`
	TargetPageID uint      `json:"target_page_id"`
	CommentID    *uint     `json:"comment_id"`
	Read         bool      `gorm:"default:false" json:"read"`
	CreatedAt    time.Time `json:"created_at"`
}

func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&User{}, &Workspace{}, &WorkspaceMember{}, &Page{}, &Block{}, &Database{}, &Property{}, &Record{}, &PropertyValue{}, &View{}, &PagePermission{}, &ShareToken{}, &Comment{}, &Notification{}); err != nil {
		return nil, err
	}

	return db, nil
}
