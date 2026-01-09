package entity

import (
	"time"

	"github.com/google/uuid"
)

// ParticipantStatus represents the status of a participant
type ParticipantStatus string

const (
	ParticipantStatusPending  ParticipantStatus = "pending"
	ParticipantStatusAccepted ParticipantStatus = "accepted"
	ParticipantStatusDeclined ParticipantStatus = "declined"
)

// UserEvent represents a participant in an event (from user_events table)
type UserEvent struct {
	UserID               uuid.UUID         `db:"user_id" json:"user_id"`
	EventID              uuid.UUID         `db:"event_id" json:"event_id"`
	Status               ParticipantStatus `db:"status" json:"status"`
	HasCalendarConnected bool              `db:"has_calendar_connected" json:"has_calendar_connected"`
	CreatedAt            time.Time         `db:"created_at" json:"created_at"`
}
