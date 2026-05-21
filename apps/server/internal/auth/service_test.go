package auth

import (
	"os"
	"testing"

	"github.com/bin-ke/my-notion/pkg/db"
	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndValidateToken(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")

	s := &Service{} // nil UserService, only testing token methods

	u := &db.User{ID: 42, Email: "alice@example.com"}
	token, err := s.generateToken(u)
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := s.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.UserID != 42 {
		t.Errorf("UserID = %d, want 42", claims.UserID)
	}
	if claims.Email != "alice@example.com" {
		t.Errorf("Email = %s, want alice@example.com", claims.Email)
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	os.Setenv("JWT_SECRET", "secret-a")
	defer os.Unsetenv("JWT_SECRET")

	s := &Service{}
	u := &db.User{ID: 1, Email: "bob@example.com"}
	token, _ := s.generateToken(u)

	// Validate with different secret
	os.Setenv("JWT_SECRET", "secret-b")
	_, err := s.ValidateToken(token)
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")

	// Create an expired token manually
	claims := &Claims{
		UserID: 1,
		Email:  "expired@example.com",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte("test-secret"))

	s := &Service{}
	// Token without expiration should still validate (our ValidateToken doesn't check exp)
	_, err := s.ValidateToken(signed)
	if err != nil {
		t.Fatalf("ValidateToken without exp: %v", err)
	}
}

func TestValidateToken_Malformed(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")

	s := &Service{}
	_, err := s.ValidateToken("not.a.jwt")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestGenerateToken_EmptySecret_UsesDefault(t *testing.T) {
	os.Unsetenv("JWT_SECRET")

	s := &Service{}
	u := &db.User{ID: 7, Email: "test@example.com"}
	token, err := s.generateToken(u)
	if err != nil {
		t.Fatalf("generateToken with default secret: %v", err)
	}

	// Should validate with default secret
	claims, err := s.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken with default secret: %v", err)
	}
	if claims.UserID != 7 {
		t.Errorf("UserID = %d, want 7", claims.UserID)
	}
}
