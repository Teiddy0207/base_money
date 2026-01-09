package dto

import (
	"time"

	"github.com/google/uuid"
)

type EventDataDTO struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Location    string `json:"location"`
	MeetingLink string `json:"meeting_link"`
	Timezone    string `json:"timezone"`
}

type CreateInvitationRequest struct {
	EventGoogleID string       `json:"event_google_id"`
	CreatorID     uuid.UUID    `json:"creator_id"`
	InviteeIDs    []uuid.UUID  `json:"invitee_ids"`
	EventData     EventDataDTO `json:"event_data"`
}

type InvitationResponse struct {
	ID            uuid.UUID    `json:"id"`
	EventGoogleID string       `json:"event_google_id"`
	CreatorID     uuid.UUID    `json:"creator_id"`
	Status        string       `json:"status"`
	EventData     EventDataDTO `json:"event_data"`
	CreatedAt     time.Time    `json:"created_at"`
}

type PendingInvitationsResponse struct {
	Invitations []InvitationResponse `json:"invitations"`
	Total       int                  `json:"total"`
}
