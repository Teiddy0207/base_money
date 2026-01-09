package dto

import "go-api-starter/core/dto"

type GoogleCalendar struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	TimeZone    string `json:"timeZone"`
}

type GoogleCalendarEvent struct {
	ID          string          `json:"id"`
	Summary     string          `json:"summary"`
	Description string          `json:"description"`
	Start       EventTime       `json:"start"`
	End         EventTime       `json:"end"`
	Location    string          `json:"location"`
	Status      string          `json:"status"`
	Attendees   []EventAttendee `json:"attendees"`
	Organizer   EventOrganizer  `json:"organizer"`
}

type EventAttendee struct {
	Email          string `json:"email"`
	ResponseStatus string `json:"responseStatus"`
	Self           bool   `json:"self"`
}

type EventOrganizer struct {
	Email string `json:"email"`
	Self  bool   `json:"self"`
}

type EventTime struct {
	DateTime string `json:"dateTime"`
	Date     string `json:"date"`
	TimeZone string `json:"timeZone"`
}

type BusyTimeSlot struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type GoogleCalendarBusyResponse struct {
	Busy []BusyTimeSlot `json:"busy"`
}

type PaginatedGoogleCalendarDTO = dto.Pagination[GoogleCalendar]
type PaginatedGoogleCalendarEventDTO = dto.Pagination[GoogleCalendarEvent]
