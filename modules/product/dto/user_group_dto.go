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

type UserInfo struct {
	ID            uuid.UUID `json:"id"`
	ProviderName  *string    `json:"provider_name"`
	ProviderEmail *string    `json:"provider_email"`
}

type GroupInfo struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

type UserGroupResponse struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	User      *UserInfo  `json:"user,omitempty"`
	GroupID   uuid.UUID  `json:"group_id"`
	Group     *GroupInfo `json:"group,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type GroupUsersResponse struct {
	GroupID uuid.UUID           `json:"group_id"`
	Group   *GroupInfo          `json:"group,omitempty"`
	Users   []UserGroupResponse `json:"users"`
}

// UserGroupWithRelations struct để lưu kết quả từ query join
type UserGroupWithRelations struct {
	// UserGroup fields
	ID        uuid.UUID `db:"ug_id"`
	UserID    uuid.UUID `db:"ug_user_id"`
	GroupID   uuid.UUID `db:"ug_group_id"`
	CreatedAt time.Time `db:"ug_created_at"`
	
	// Group fields
	GroupIDFromGroup   uuid.UUID  `db:"g_id"`
	GroupName          string     `db:"g_name"`
	GroupDescription   string     `db:"g_description"`
	
	// User fields
	UserIDFromUser     uuid.UUID  `db:"u_id"`
	
	// SocialLogin fields
	ProviderName       *string    `db:"sl_provider_name"`
	ProviderEmail      *string    `db:"sl_provider_email"`
}


