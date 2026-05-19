package user

import (
	"errors"

	"github.com/bin-ke/my-notion/pkg/db"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	DB *gorm.DB
}

func NewService(database *gorm.DB) *Service {
	return &Service{DB: database}
}

func (s *Service) Create(email, name, password string) (*db.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &db.User{
		Email:        email,
		Name:         name,
		PasswordHash: string(hash),
	}

	if err := s.DB.Create(user).Error; err != nil {
		return nil, errors.New("email already registered")
	}

	return user, nil
}

func (s *Service) FindByEmail(email string) (*db.User, error) {
	var user db.User
	if err := s.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

func (s *Service) FindByID(id uint) (*db.User, error) {
	var user db.User
	if err := s.DB.First(&user, id).Error; err != nil {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

func (s *Service) CheckPassword(user *db.User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}
