package dto

// Provider constants
const (
	ProviderGoogle  = "google"
	ProviderOutlook = "outlook"
)

// ========== Calendar Connection DTOs ==========

// CalendarConnectionResponse represents a calendar connection
type CalendarConnectionResponse struct {
	ID            string `json:"id"`
	Provider      string `json:"provider"`
	CalendarEmail string `json:"calendar_email"`
	IsActive      bool   `json:"is_active"`
	ConnectedAt   string `json:"connected_at"`
}

// CalendarConnectionListResponse represents list of connections
type CalendarConnectionListResponse struct {
	Connections []CalendarConnectionResponse `json:"connections"`
}

// ========== Free/Busy DTOs ==========

// FreeBusyRequest request for free/busy info
type FreeBusyRequest struct {
	StartTime string   `json:"start_time" validate:"required"` // RFC3339
	EndTime   string   `json:"end_time" validate:"required"`   // RFC3339
	UserIDs   []string `json:"user_ids,omitempty"`
}

// TimeSlot represents a time period
type TimeSlot struct {
	Start string `json:"start"` // RFC3339
	End   string `json:"end"`   // RFC3339
}

// UserFreeBusy represents free/busy info for a user
type UserFreeBusy struct {
	UserID    string     `json:"user_id"`
	Email     string     `json:"email"`
	BusySlots []TimeSlot `json:"busy_slots"`
}

// FreeBusyResponse response with free/busy info
type FreeBusyResponse struct {
	Users []UserFreeBusy `json:"users"`
}

// ========== Calendar Event DTOs ==========

// CreateEventRequest request to create a calendar event
type CreateEventRequest struct {
	Title       string   `json:"title" validate:"required"`
	Description string   `json:"description"`
	StartTime   string   `json:"start_time" validate:"required"` // RFC3339
	EndTime     string   `json:"end_time" validate:"required"`   // RFC3339
	Timezone    string   `json:"timezone"`
	Attendees   []string `json:"attendees"` // Email addresses
	MeetingLink string   `json:"meeting_link"`
}

// CreateEventResponse response after creating event
type CreateEventResponse struct {
	EventID     string `json:"event_id"`
	Title       string `json:"title"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	MeetingLink string `json:"meeting_link,omitempty"`
}

// ========== OAuth DTOs ==========

// OAuthURLResponse response with OAuth URL
type OAuthURLResponse struct {
	URL   string `json:"url"`
	State string `json:"state"`
}

// ========== Suggested Slots DTOs ==========

// SuggestedSlotsRequest request for finding available meeting slots
type SuggestedSlotsRequest struct {
	UserIDs          []string `json:"user_ids"`
	DurationMinutes  int      `json:"duration_minutes"`
	DaysAhead        int      `json:"days_ahead"`         // default 7
	WorkingHoursOnly bool     `json:"working_hours_only"` // 8:00-18:00
	StartDate        string   `json:"start_date"`         // optional, RFC3339 date (YYYY-MM-DD)
	TimePreference   string   `json:"time_preference"`    // "morning", "afternoon", "evening", "" for no preference
}

// SuggestedSlot represents a suggested meeting time slot
type SuggestedSlot struct {
	StartTime      string `json:"start_time"` // RFC3339
	EndTime        string `json:"end_time"`   // RFC3339
	Score          int    `json:"score"`      // 0-100 based on availability
	AvailableCount int    `json:"available_count"`
	TotalCount     int    `json:"total_count"`
}

// DisconnectedUser represents a user without calendar connection
type DisconnectedUser struct {
	UserID string `json:"user_id"`
	Email  string `json:"email,omitempty"`
	Name   string `json:"name,omitempty"`
}

// SuggestedSlotsResponse response with suggested slots and connection status
type SuggestedSlotsResponse struct {
	Slots             []SuggestedSlot    `json:"slots"`
	ConnectedCount    int                `json:"connected_count"`
	DisconnectedCount int                `json:"disconnected_count"`
	TotalParticipants int                `json:"total_participants"`
	DisconnectedUsers []DisconnectedUser `json:"disconnected_users,omitempty"`
	Warning           string             `json:"warning,omitempty"`
}
