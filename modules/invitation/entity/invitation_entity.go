package entity

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type InvitationStatus string

const (
	InvitationStatusPending  InvitationStatus = "pending"
	InvitationStatusAccepted InvitationStatus = "accepted"
	InvitationStatusDeclined InvitationStatus = "declined"
)

type EventData struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Location    string `json:"location"`
	MeetingLink string `json:"meeting_link"`
	Timezone    string `json:"timezone"`
}

func (e EventData) Value() (driver.Value, error) {
	return json.Marshal(e)
}

func (e *EventData) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, e)
}

type EventInvitation struct {
	ID            uuid.UUID        `db:"id" json:"id"`
	EventGoogleID string           `db:"event_google_id" json:"event_google_id"`
	CreatorID     uuid.UUID        `db:"creator_id" json:"creator_id"`
	InviteeID     uuid.UUID        `db:"invitee_id" json:"invitee_id"`
	Status        InvitationStatus `db:"status" json:"status"`
	EventData     EventData        `db:"event_data" json:"event_data"`
	RespondedAt   *time.Time       `db:"responded_at" json:"responded_at"`
	CreatedAt     time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time        `db:"updated_at" json:"updated_at"`
}
