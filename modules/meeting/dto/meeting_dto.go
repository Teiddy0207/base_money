package dto

import (
	"go-api-starter/modules/meeting/entity"
	"time"
)

// ===================== Request DTOs =====================

// CreateEventRequest for creating a new event
type CreateEventRequest struct {
	Title           string            `json:"title" validate:"required"`
	Description     string            `json:"description"`
	Address         string            `json:"address"`
	DurationMinutes int               `json:"duration_minutes" validate:"required,min=15,max=480"`
	Participants    []string          `json:"participants"` // user_ids
	Preferences     *EventPreferences `json:"preferences"`
}

// EventPreferences for scheduling preferences
type EventPreferences struct {
	OnlyBusinessHours bool   `json:"only_business_hours"`
	PreferMorning     bool   `json:"prefer_morning"`
	PreferAfternoon   bool   `json:"prefer_afternoon"`
	ExcludeWeekends   bool   `json:"exclude_weekends"`
	Timezone          string `json:"timezone"`
}

// UpdateEventRequest for updating event details
type UpdateEventRequest struct {
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	Address         string            `json:"address"`
	DurationMinutes int               `json:"duration_minutes" validate:"min=15,max=480"`
	Preferences     *EventPreferences `json:"preferences"`
}

// FindSlotsRequest for finding available time slots
type FindSlotsRequest struct {
	SearchStartDate string `json:"search_start_date"` // YYYY-MM-DD
	SearchEndDate   string `json:"search_end_date"`   // YYYY-MM-DD
	SearchDays      int    `json:"search_days"`       // Alternative: search next N days
}

// SelectSlotRequest for selecting a slot
type SelectSlotRequest struct {
	SlotID    string `json:"slot_id"`
	StartTime string `json:"start_time"` // RFC3339 format
	EndTime   string `json:"end_time"`   // RFC3339 format
}

// ===================== Response DTOs =====================

// EventResponse for event details
type EventResponse struct {
	ID              string                `json:"id"`
	HostID          string                `json:"host_id,omitempty"`
	Title           string                `json:"title"`
	Description     string                `json:"description,omitempty"`
	Address         string                `json:"address,omitempty"`
	DurationMinutes int                   `json:"duration_minutes"`
	Status          string                `json:"status"`
	Timezone        string                `json:"timezone"`
	StartDate       *time.Time            `json:"start_date,omitempty"`
	EndDate         *time.Time            `json:"end_date,omitempty"`
	MeetingLink     string                `json:"meeting_link,omitempty"`
	Preferences     *EventPreferences     `json:"preferences,omitempty"`
	Participants    []ParticipantResponse `json:"participants,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
}

// ParticipantResponse for participant status
type ParticipantResponse struct {
	UserID               string `json:"user_id"`
	EventID              string `json:"event_id"`
	Status               string `json:"status"`
	HasCalendarConnected bool   `json:"has_calendar_connected"`
}

// FindSlotsResponse for suggested time slots
type FindSlotsResponse struct {
	EventID      string                `json:"event_id"`
	Slots        []SuggestedSlotDTO    `json:"slots"`
	Participants []ParticipantResponse `json:"participants"`
}

// SuggestedSlotDTO for a single suggested slot
type SuggestedSlotDTO struct {
	ID             string    `json:"id"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	Score          int       `json:"score"`
	AvailableCount int       `json:"available_count"`
	TotalCount     int       `json:"total_count"`
	DayOfWeek      string    `json:"day_of_week"`
	FormattedDate  string    `json:"formatted_date"`
	FormattedTime  string    `json:"formatted_time"`
}

// PaginatedEventResponse for paginated events list
type PaginatedEventResponse struct {
	Items      []EventResponse `json:"items"`
	TotalItems int             `json:"total_items"`
	PageNumber int             `json:"page_number"`
	PageSize   int             `json:"page_size"`
}

// ===================== Mapper Functions =====================

// ToEventResponse maps entity to DTO
func ToEventResponse(e *entity.Event, participants []entity.UserEvent) *EventResponse {
	resp := &EventResponse{
		ID:              e.ID.String(),
		Title:           e.Title,
		DurationMinutes: e.DurationMinutes,
		Status:          string(e.Status),
		Timezone:        e.Timezone,
		StartDate:       e.StartDate,
		EndDate:         e.EndDate,
		CreatedAt:       e.CreatedAt,
	}

	if e.HostID != nil {
		resp.HostID = e.HostID.String()
	}
	if e.Description != nil {
		resp.Description = *e.Description
	}
	if e.Address != nil {
		resp.Address = *e.Address
	}
	if e.MeetingLink != nil {
		resp.MeetingLink = *e.MeetingLink
	}

	// Map participants
	for _, p := range participants {
		pResp := ParticipantResponse{
			UserID:               p.UserID.String(),
			EventID:              p.EventID.String(),
			Status:               string(p.Status),
			HasCalendarConnected: p.HasCalendarConnected,
		}
		resp.Participants = append(resp.Participants, pResp)
	}

	return resp
}

// ToSlotDTO maps entity slot to DTO
func ToSlotDTO(s *entity.EventSlot) *SuggestedSlotDTO {
	days := []string{"Chủ nhật", "Thứ 2", "Thứ 3", "Thứ 4", "Thứ 5", "Thứ 6", "Thứ 7"}

	return &SuggestedSlotDTO{
		ID:             s.ID.String(),
		StartTime:      s.StartTime,
		EndTime:        s.EndTime,
		Score:          s.Score,
		AvailableCount: s.AvailableCount,
		TotalCount:     s.TotalParticipants,
		DayOfWeek:      days[int(s.StartTime.Weekday())],
		FormattedDate:  s.StartTime.Format("02/01/2006"),
		FormattedTime:  s.StartTime.Format("15h04") + " - " + s.EndTime.Format("15h04"),
	}
}
