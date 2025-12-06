package mapper

import "go-api-starter/modules/auth/dto"

func ToGoogleCalendarEventsDTO(items []dto.GoogleCalendarEvent) []dto.GoogleCalendarEvent {
	return items
}

func ToPaginatedGoogleCalendarDTO(items []dto.GoogleCalendar, totalItems int, pageNumber int, pageSize int) *dto.PaginatedGoogleCalendarDTO {
	totalPages := 0
	if pageSize > 0 {
		totalPages = (totalItems + pageSize - 1) / pageSize
	}

	return &dto.PaginatedGoogleCalendarDTO{
		Items:      items,
		TotalItems: totalItems,
		TotalPages: totalPages,
		PageNumber: pageNumber,
		PageSize:   pageSize,
	}
}
