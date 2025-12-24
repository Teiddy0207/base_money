package dto

import (
	"time"
	"go-api-starter/core/dto"

	"github.com/google/uuid"
)

type GroupRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Thumbnail   string `json:"thumbnail"`
	SortOrder   int    `json:"sort_order"`
	IsActive    bool   `json:"is_active"`
}

type GroupResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Thumbnail   string    `json:"thumbnail"`
	SortOrder   int       `json:"sort_order"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PaginatedGroupResponse = dto.Pagination[GroupResponse]


