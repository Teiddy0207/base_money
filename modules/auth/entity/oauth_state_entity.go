package entity

import (
	"time"
	"go-api-starter/core/entity"
)

type OAuthState struct {
	State     string    `db:"state"`
	ExpiresAt time.Time `db:"expires_at"`
	entity.BaseEntity
}



