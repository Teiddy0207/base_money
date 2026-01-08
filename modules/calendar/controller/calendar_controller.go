package controller

import (
	"net/http"
	"time"

	"go-api-starter/core/errors"
	"go-api-starter/core/utils"
	"go-api-starter/modules/calendar/dto"
	"go-api-starter/modules/calendar/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type CalendarController struct {
	service service.CalendarService
}

func NewCalendarController(service service.CalendarService) *CalendarController {
	return &CalendarController{service: service}
}

// GetConnections returns all calendar connections for the current user
// GET /api/v1/private/calendar/connections
func (c *CalendarController) GetConnections(ctx echo.Context) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, errors.NewAppError(errors.ErrUnauthorized, "Invalid user", nil))
	}

	connections, err := c.service.GetConnections(ctx.Request().Context(), userID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, errors.NewAppError(errors.ErrInternalServer, "Failed to get connections", err))
	}

	return ctx.JSON(http.StatusOK, dto.CalendarConnectionListResponse{
		Connections: connections,
	})
}

// DisconnectCalendar disconnects a calendar provider
// DELETE /api/v1/private/calendar/connections/:provider
func (c *CalendarController) DisconnectCalendar(ctx echo.Context) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, errors.NewAppError(errors.ErrUnauthorized, "Invalid user", nil))
	}

	provider := ctx.Param("provider")
	if provider != dto.ProviderGoogle && provider != dto.ProviderOutlook {
		return ctx.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "Invalid provider", nil))
	}

	if err := c.service.DisconnectCalendar(ctx.Request().Context(), userID, provider); err != nil {
		return ctx.JSON(http.StatusInternalServerError, errors.NewAppError(errors.ErrInternalServer, "Failed to disconnect", err))
	}

	return ctx.JSON(http.StatusOK, map[string]string{"message": "Disconnected successfully"})
}

// GetFreeBusy returns free/busy information
// GET /api/v1/private/calendar/free-busy?start_time=...&end_time=...&user_ids=...
func (c *CalendarController) GetFreeBusy(ctx echo.Context) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, errors.NewAppError(errors.ErrUnauthorized, "Invalid user", nil))
	}

	startTimeStr := ctx.QueryParam("start_time")
	endTimeStr := ctx.QueryParam("end_time")

	if startTimeStr == "" || endTimeStr == "" {
		return ctx.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "start_time and end_time are required", nil))
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "Invalid start_time format", nil))
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "Invalid end_time format", nil))
	}

	// Check if querying for multiple users
	userIDsParam := ctx.QueryParam("user_ids")
	if userIDsParam != "" {
		// Parse user IDs and get free/busy for all
		var userIDs []uuid.UUID
		// Simple comma-separated parsing
		for _, idStr := range splitAndTrim(userIDsParam, ",") {
			if id, err := uuid.Parse(idStr); err == nil {
				userIDs = append(userIDs, id)
			}
		}

		results, err := c.service.GetFreeBusyForUsers(ctx.Request().Context(), userIDs, startTime, endTime)
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, errors.NewAppError(errors.ErrInternalServer, "Failed to get free/busy", err))
		}

		return ctx.JSON(http.StatusOK, dto.FreeBusyResponse{Users: results})
	}

	// Get free/busy for current user only
	busySlots, err := c.service.GetFreeBusy(ctx.Request().Context(), userID, startTime, endTime)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, errors.NewAppError(errors.ErrInternalServer, err.Error(), err))
	}

	return ctx.JSON(http.StatusOK, dto.FreeBusyResponse{
		Users: []dto.UserFreeBusy{
			{
				UserID:    userID.String(),
				BusySlots: busySlots,
			},
		},
	})
}

// CreateEvent creates a calendar event
// POST /api/v1/private/calendar/events
func (c *CalendarController) CreateEvent(ctx echo.Context) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, errors.NewAppError(errors.ErrUnauthorized, "Invalid user", nil))
	}

	var req dto.CreateEventRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "Invalid request body", nil))
	}

	// Set default timezone if not provided
	if req.Timezone == "" {
		req.Timezone = "Asia/Ho_Chi_Minh"
	}

	result, err := c.service.CreateEvent(ctx.Request().Context(), userID, &req)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, errors.NewAppError(errors.ErrInternalServer, err.Error(), err))
	}

	return ctx.JSON(http.StatusCreated, result)
}

// DeleteEvent deletes or declines a calendar event
// DELETE /api/v1/private/calendar/events/:id
func (c *CalendarController) DeleteEvent(ctx echo.Context) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, errors.NewAppError(errors.ErrUnauthorized, "Invalid user", nil))
	}

	eventID := ctx.Param("id")
	if eventID == "" {
		return ctx.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "Event ID is required", nil))
	}

	if err := c.service.DeleteEvent(ctx.Request().Context(), userID, eventID); err != nil {
		return ctx.JSON(http.StatusInternalServerError, errors.NewAppError(errors.ErrInternalServer, err.Error(), err))
	}

	return ctx.JSON(http.StatusOK, map[string]string{"message": "Event deleted successfully"})
}

// GetSuggestedSlots finds available meeting time slots
// POST /api/v1/private/calendar/suggested-slots
func (c *CalendarController) GetSuggestedSlots(ctx echo.Context) error {
	// Get current user (event creator)
	currentUserID, err := getUserIDFromContext(ctx)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, errors.NewAppError(errors.ErrUnauthorized, "Invalid user", nil))
	}

	var req dto.SuggestedSlotsRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "Invalid request body", nil))
	}

	// Always include current user (event creator) in the search
	// Prepend current user ID to ensure creator's calendar is checked
	allUserIDs := []string{currentUserID.String()}
	for _, uid := range req.UserIDs {
		if uid != currentUserID.String() { // Avoid duplicates
			allUserIDs = append(allUserIDs, uid)
		}
	}
	req.UserIDs = allUserIDs

	result, err := c.service.FindAvailableSlots(ctx.Request().Context(), &req)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, errors.NewAppError(errors.ErrInternalServer, err.Error(), err))
	}

	return ctx.JSON(http.StatusOK, result)
}

// Helper function to get user ID from JWT context
func getUserIDFromContext(ctx echo.Context) (uuid.UUID, error) {
	token := ctx.Request().Header.Get("Authorization")
	if token == "" {
		return uuid.Nil, errors.NewAppError(errors.ErrUnauthorized, "No token provided", nil)
	}

	// Remove "Bearer " prefix
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	tokenData, err := utils.ValidateAndParseToken(token)
	if err != nil {
		return uuid.Nil, err
	}

	return tokenData.UserID, nil
}

// Helper function to split string and trim spaces
func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range splitString(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitString(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
