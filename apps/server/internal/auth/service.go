package auth

import (
	"errors"
	"os"
	"time"

	"github.com/bin-ke/my-notion/internal/user"
	"github.com/bin-ke/my-notion/pkg/db"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type Service struct {
	UserService *user.Service
}

func NewService(database *gorm.DB) *Service {
	return &Service{
		UserService: user.NewService(database),
	}
}

type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func (s *Service) Register(email, name, password string) (*db.User, error) {
	return s.UserService.Create(email, name, password)
}

func (s *Service) Login(email, password string) (string, *db.User, error) {
	u, err := s.UserService.FindByEmail(email)
	if err != nil {
		return "", nil, errors.New("invalid email or password")
	}

	if !s.UserService.CheckPassword(u, password) {
		return "", nil, errors.New("invalid email or password")
	}

	token, err := s.generateToken(u)
	if err != nil {
		return "", nil, err
	}

	return token, u, nil
}

func (s *Service) generateToken(u *db.User) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}

	claims := &Claims{
		UserID: u.ID,
		Email:  u.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
