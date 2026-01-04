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

// Single user busy response for strict free/busy view
type UserBusyResponse struct {
	UserID string       `json:"user_id"`
	User   BusyUserInfo `json:"user"`
	Busy   []TimeSlot   `json:"busy"`
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

// BusyUserInfo returns basic relational information for the target user
type BusyUserInfo struct {
	ID               string  `json:"id"`
	ProviderUsername *string `json:"provider_username,omitempty"`
	ProviderEmail    *string `json:"provider_email,omitempty"`
}
