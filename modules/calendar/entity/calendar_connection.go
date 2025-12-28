package entity

import (
	"time"

	"github.com/google/uuid"
	"go-api-starter/core/entity"
)

// CalendarConnection stores user's calendar provider connection
type CalendarConnection struct {
	entity.BaseEntity
	UserID         uuid.UUID  `db:"user_id" json:"user_id"`
	Provider       string     `db:"provider" json:"provider"` // "google" | "outlook"
	AccessToken    string     `db:"access_token" json:"-"`
	RefreshToken   string     `db:"refresh_token" json:"-"`
	TokenExpiresAt time.Time  `db:"token_expires_at" json:"token_expires_at"`
	CalendarEmail  string     `db:"calendar_email" json:"calendar_email"`
	IsActive       bool       `db:"is_active" json:"is_active"`
}

// TableName returns the table name for GORM
func (CalendarConnection) TableName() string {
	return "calendar_connections"
}
