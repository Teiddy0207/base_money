package entity

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"go-api-starter/core/entity"

	"github.com/google/uuid"
)

type Notification struct {
	UserID  uuid.UUID `db:"user_id" json:"user_id"`
	Title   string    `db:"title" json:"title"`
	Message string    `db:"message" json:"message"`
	Type    string    `db:"type" json:"type"`
	Data    JSONB     `db:"data" json:"data"`
	IsRead  bool      `db:"is_read" json:"is_read"`
	entity.BaseEntity
}

type JSONB map[string]interface{}

func (a JSONB) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *JSONB) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

type PaginatedNotificationEntity = entity.Pagination[Notification]
