package dto

import "go-api-starter/core/dto"

type GoogleCalendar struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	TimeZone    string `json:"timeZone"`
}

type GoogleCalendarEvent struct {
	ID          string    `json:"id"`
	Summary     string    `json:"summary"`
	Description string    `json:"description"`
	Start       EventTime `json:"start"`
	End         EventTime `json:"end"`
	Location    string    `json:"location"`
	Status      string    `json:"status"`
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
