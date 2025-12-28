package dto

import (
	"time"
	"go-api-starter/core/dto"

	"github.com/google/uuid"
)

type GroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type GroupResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PaginatedGroupResponse = dto.Pagination[GroupResponse]


