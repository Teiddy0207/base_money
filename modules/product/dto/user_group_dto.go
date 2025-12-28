package dto

import (
	"time"

	"github.com/google/uuid"
)

type AddUsersToGroupRequest struct {
	GroupID uuid.UUID   `json:"group_id"`
	UserIDs []uuid.UUID `json:"user_ids"`
}

type RemoveUserFromGroupRequest struct {
	GroupID uuid.UUID `json:"group_id"`
	UserID  uuid.UUID `json:"user_id"`
}

type UserGroupResponse struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	GroupID   uuid.UUID `json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
}

type GroupUsersResponse struct {
	GroupID uuid.UUID           `json:"group_id"`
	Users   []UserGroupResponse `json:"users"`
}

