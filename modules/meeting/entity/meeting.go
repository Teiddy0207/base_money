package entity

import (
	"time"

	"github.com/google/uuid"
)

// EventStatus represents the status of an event
type EventStatus string

const (
	EventStatusPending   EventStatus = "pending"
	EventStatusScheduled EventStatus = "scheduled"
	EventStatusCancelled EventStatus = "cancelled"
)

// Event represents a scheduled event (extends existing events table)
type Event struct {
	ID              uuid.UUID   `db:"id" json:"id"`
	HostID          *uuid.UUID  `db:"host_id" json:"host_id,omitempty"`
	Title           string      `db:"title" json:"title"`
	Description     *string     `db:"description" json:"description,omitempty"`
	Address         *string     `db:"address" json:"address,omitempty"`
	DurationMinutes int         `db:"duration_minutes" json:"duration_minutes"`
	Status          EventStatus `db:"status" json:"status"`
	Timezone        string      `db:"timezone" json:"timezone"`
	StartDate       *time.Time  `db:"start_date" json:"start_date,omitempty"`
	EndDate         *time.Time  `db:"end_date" json:"end_date,omitempty"`
	MeetingLink     *string     `db:"meeting_link" json:"meeting_link,omitempty"`
	Preferences     *string     `db:"preferences" json:"preferences,omitempty"` // JSONB as string
	CreatedAt       time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time   `db:"updated_at" json:"updated_at"`
}

// EventPreferences represents event scheduling preferences
type EventPreferences struct {
	OnlyBusinessHours bool   `json:"only_business_hours"`
	PreferMorning     bool   `json:"prefer_morning"`
	PreferAfternoon   bool   `json:"prefer_afternoon"`
	ExcludeWeekends   bool   `json:"exclude_weekends"`
	Timezone          string `json:"timezone"`
}
