package auth

import (
	"time"
)

type User struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Email        string    `json:"email" gorm:"uniqueIndex;not null"`
	Username     string    `json:"username" gorm:"uniqueIndex;not null"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	PasswordHash string    `json:"-" gorm:"not null"`
	Avatar       string    `json:"avatar"`
	IsActive     bool      `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	LastLoginAt  time.Time `json:"last_login_at"`
}

// TableName sets the table name for GORM
func (User) TableName() string {
	return "users"
}

type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Username  string `json:"username" validate:"required,min=3,max=50"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	User         *User  `json:"user"`
	ExpiresIn    int64  `json:"expires_in"`
}
