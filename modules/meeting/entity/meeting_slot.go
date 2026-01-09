package entity

import (
	"time"

	"github.com/google/uuid"
)

// EventSlot represents a suggested time slot for an event
type EventSlot struct {
	ID                uuid.UUID `db:"id" json:"id"`
	EventID           uuid.UUID `db:"event_id" json:"event_id"`
	StartTime         time.Time `db:"start_time" json:"start_time"`
	EndTime           time.Time `db:"end_time" json:"end_time"`
	AvailableCount    int       `db:"available_count" json:"available_count"`
	TotalParticipants int       `db:"total_participants" json:"total_participants"`
	Score             int       `db:"score" json:"score"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
}

// TimeSlot represents a generic time range (used for free/busy calculations)
type TimeSlot struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}
