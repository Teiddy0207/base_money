package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go-api-starter/core/config"
	"go-api-starter/core/errors"
	"go-api-starter/core/logger"
	authRepo "go-api-starter/modules/auth/repository"
	"go-api-starter/modules/calendar/dto"
	"go-api-starter/modules/calendar/entity"
	"go-api-starter/modules/calendar/repository"
	invitDto "go-api-starter/modules/invitation/dto"
	invitService "go-api-starter/modules/invitation/service"
	notifDto "go-api-starter/modules/notification/dto"
	notifService "go-api-starter/modules/notification/service"

	"github.com/google/uuid"
)

const (
	googleCalendarAPIBase = "https://www.googleapis.com/calendar/v3"
	googleFreeBusyAPI     = googleCalendarAPIBase + "/freeBusy"
	googleEventsAPI       = googleCalendarAPIBase + "/calendars/primary/events"
)

type CalendarService interface {
	// Connection management
	SaveGoogleConnection(ctx context.Context, userID uuid.UUID, accessToken, refreshToken string, expiresAt time.Time, email string) (*entity.CalendarConnection, error)
	GetConnections(ctx context.Context, userID uuid.UUID) ([]dto.CalendarConnectionResponse, error)
	DisconnectCalendar(ctx context.Context, userID uuid.UUID, provider string) error

	// Calendar operations
	GetFreeBusy(ctx context.Context, userID uuid.UUID, startTime, endTime time.Time) ([]dto.TimeSlot, error)
	GetFreeBusyForUsers(ctx context.Context, userIDs []uuid.UUID, startTime, endTime time.Time) ([]dto.UserFreeBusy, error)
	CreateEvent(ctx context.Context, userID uuid.UUID, req *dto.CreateEventRequest) (*dto.CreateEventResponse, error)
	DeleteEvent(ctx context.Context, userID uuid.UUID, eventID string) error
	FindAvailableSlots(ctx context.Context, req *dto.SuggestedSlotsRequest) (*dto.SuggestedSlotsResponse, error)
}

type calendarService struct {
	repo         repository.CalendarRepository
	userRepo     *authRepo.AuthRepository
	notifService *notifService.NotificationService
	invitService *invitService.InvitationService
}

func NewCalendarService(
	repo repository.CalendarRepository,
	userRepo *authRepo.AuthRepository,
	notifService *notifService.NotificationService,
	invitService *invitService.InvitationService,
) CalendarService {
	return &calendarService{
		repo:         repo,
		userRepo:     userRepo,
		notifService: notifService,
		invitService: invitService,
	}
}

// SaveGoogleConnection saves or updates a Google Calendar connection
func (s *calendarService) SaveGoogleConnection(ctx context.Context, userID uuid.UUID, accessToken, refreshToken string, expiresAt time.Time, email string) (*entity.CalendarConnection, error) {
	// Check if connection already exists
	existing, _ := s.repo.GetConnectionByUserAndProvider(ctx, userID, dto.ProviderGoogle)

	if existing != nil {
		// Update existing connection
		existing.AccessToken = accessToken
		existing.RefreshToken = refreshToken
		existing.TokenExpiresAt = expiresAt
		existing.CalendarEmail = email
		existing.IsActive = true

		if err := s.repo.UpdateConnection(ctx, existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	// Create new connection
	conn := &entity.CalendarConnection{
		UserID:         userID,
		Provider:       dto.ProviderGoogle,
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		TokenExpiresAt: expiresAt,
		CalendarEmail:  email,
		IsActive:       true,
	}

	return s.repo.CreateConnection(ctx, conn)
}

// GetConnections returns all calendar connections for a user
func (s *calendarService) GetConnections(ctx context.Context, userID uuid.UUID) ([]dto.CalendarConnectionResponse, error) {
	connections, err := s.repo.GetConnectionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var result []dto.CalendarConnectionResponse
	for _, conn := range connections {
		result = append(result, dto.CalendarConnectionResponse{
			ID:            conn.ID.String(),
			Provider:      conn.Provider,
			CalendarEmail: conn.CalendarEmail,
			IsActive:      conn.IsActive,
			ConnectedAt:   conn.CreatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

// DisconnectCalendar disconnects a calendar provider
func (s *calendarService) DisconnectCalendar(ctx context.Context, userID uuid.UUID, provider string) error {
	return s.repo.DeleteConnection(ctx, userID, provider)
}

// GetFreeBusy gets free/busy information from Google Calendar
func (s *calendarService) GetFreeBusy(ctx context.Context, userID uuid.UUID, startTime, endTime time.Time) ([]dto.TimeSlot, error) {
	conn, err := s.repo.GetConnectionByUserAndProvider(ctx, userID, dto.ProviderGoogle)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrNotFound, "No Google Calendar connected", err)
	}

	// Refresh token if expired
	accessToken, err := s.ensureValidToken(ctx, conn)
	if err != nil {
		return nil, err
	}

	// Call Google Calendar FreeBusy API
	busySlots, err := s.callGoogleFreeBusy(accessToken, conn.CalendarEmail, startTime, endTime)
	if err != nil {
		return nil, err
	}

	return busySlots, nil
}

// GetFreeBusyForUsers gets free/busy info for multiple users
func (s *calendarService) GetFreeBusyForUsers(ctx context.Context, userIDs []uuid.UUID, startTime, endTime time.Time) ([]dto.UserFreeBusy, error) {
	logger.Info("GetFreeBusyForUsers:Start", "user_ids", userIDs, "start_time", startTime, "end_time", endTime)

	connections, err := s.repo.GetConnectionsByUserIDs(ctx, userIDs)
	if err != nil {
		logger.Error("GetFreeBusyForUsers:GetConnections:Error", "error", err)
		return nil, err
	}

	logger.Info("GetFreeBusyForUsers:Connections", "count", len(connections))

	var results []dto.UserFreeBusy
	for _, conn := range connections {
		logger.Info("GetFreeBusyForUsers:ProcessingUser", "user_id", conn.UserID, "email", conn.CalendarEmail)

		accessToken, err := s.ensureValidToken(ctx, &conn)
		if err != nil {
			logger.Error("Failed to refresh token for user", "user_id", conn.UserID, "error", err)
			continue
		}

		busySlots, err := s.callGoogleFreeBusy(accessToken, conn.CalendarEmail, startTime, endTime)
		if err != nil {
			logger.Error("Failed to get free/busy for user", "user_id", conn.UserID, "error", err)
			continue
		}

		logger.Info("GetFreeBusyForUsers:BusySlotsReceived", "user_id", conn.UserID, "count", len(busySlots))
		for i, slot := range busySlots {
			logger.Info("GetFreeBusyForUsers:BusySlot", "user_id", conn.UserID, "index", i, "start", slot.Start, "end", slot.End)
		}

		results = append(results, dto.UserFreeBusy{
			UserID:    conn.UserID.String(),
			Email:     conn.CalendarEmail,
			BusySlots: busySlots,
		})
	}

	logger.Info("GetFreeBusyForUsers:Complete", "results_count", len(results))
	return results, nil
}

// CreateEvent creates an event on Google Calendar
func (s *calendarService) CreateEvent(ctx context.Context, userID uuid.UUID, req *dto.CreateEventRequest) (*dto.CreateEventResponse, error) {
	conn, err := s.repo.GetConnectionByUserAndProvider(ctx, userID, dto.ProviderGoogle)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrNotFound, "No Google Calendar connected", err)
	}

	accessToken, err := s.ensureValidToken(ctx, conn)
	if err != nil {
		return nil, err
	}

	// Build event payload
	event := map[string]interface{}{
		"summary":     req.Title,
		"description": req.Description,
		"start": map[string]string{
			"dateTime": req.StartTime,
			"timeZone": req.Timezone,
		},
		"end": map[string]string{
			"dateTime": req.EndTime,
			"timeZone": req.Timezone,
		},
	}

	if len(req.Attendees) > 0 {
		attendees := make([]map[string]string, len(req.Attendees))
		for i, email := range req.Attendees {
			attendees[i] = map[string]string{"email": email}
		}
		event["attendees"] = attendees
	}

	if req.MeetingLink != "" {
		event["hangoutLink"] = req.MeetingLink
	}

	// Call Google Calendar Events API
	eventJSON, _ := json.Marshal(event)
	httpReq, _ := http.NewRequest("POST", googleEventsAPI, strings.NewReader(string(eventJSON)))
	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInternalServer, "Failed to create event", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.NewAppError(errors.ErrInternalServer, fmt.Sprintf("Google API error: %s", string(body)), nil)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	eventID := result["id"].(string)

	// Create invitations for attendees if there are any
	logger.Info("CreateEvent:Attendees", "count", len(req.Attendees), "emails", req.Attendees)
	if len(req.Attendees) > 0 && s.invitService != nil {
		// Resolve attendee emails to user IDs
		inviteeIDs := make([]uuid.UUID, 0)
		for _, email := range req.Attendees {
			user, err := s.userRepo.GetUserByIdentifier(ctx, email)
			if err != nil {
				logger.Error("CreateEvent:GetUserByIdentifier:Error", "email", email, "error", err)
				continue
			}
			if user == nil {
				logger.Warn("CreateEvent:GetUserByIdentifier:UserNotFound", "email", email)
				continue
			}
			logger.Info("CreateEvent:FoundUser", "email", email, "user_id", user.ID)
			inviteeIDs = append(inviteeIDs, user.ID)
		}

		logger.Info("CreateEvent:InviteeIDs", "count", len(inviteeIDs), "ids", inviteeIDs)
		if len(inviteeIDs) > 0 {
			invitReq := &invitDto.CreateInvitationRequest{
				EventGoogleID: eventID,
				CreatorID:     userID,
				InviteeIDs:    inviteeIDs,
				EventData: invitDto.EventDataDTO{
					Title:       req.Title,
					Description: req.Description,
					StartTime:   req.StartTime,
					EndTime:     req.EndTime,
					Location:    "", // TODO: add location to CreateEventRequest if needed
					MeetingLink: req.MeetingLink,
					Timezone:    req.Timezone,
				},
			}
			if err := s.invitService.CreateInvitations(ctx, invitReq); err != nil {
				logger.Error("CreateEvent:CreateInvitations:Error:", err)
				// Don't fail the event creation for invitation errors
			} else {
				logger.Info("CreateEvent:CreateInvitations:Success", "count", len(inviteeIDs))
			}
		}
	}

	return &dto.CreateEventResponse{
		EventID:     eventID,
		Title:       req.Title,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		MeetingLink: req.MeetingLink,
	}, nil
}

// DeleteEvent deletes or declines an event based on user's role
func (s *calendarService) DeleteEvent(ctx context.Context, userID uuid.UUID, eventID string) error {
	conn, err := s.repo.GetConnectionByUserAndProvider(ctx, userID, dto.ProviderGoogle)
	if err != nil {
		return errors.NewAppError(errors.ErrNotFound, "No Google Calendar connected", err)
	}

	accessToken, err := s.ensureValidToken(ctx, conn)
	if err != nil {
		return err
	}

	// 1. Get event details to check if user is organizer
	eventURL := fmt.Sprintf("%s/%s", googleEventsAPI, eventID)
	getReq, _ := http.NewRequest("GET", eventURL, nil)
	getReq.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	getResp, err := client.Do(getReq)
	if err != nil {
		return errors.NewAppError(errors.ErrInternalServer, "Failed to get event", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		return errors.NewAppError(errors.ErrNotFound, "Event not found", nil)
	}

	var eventData struct {
		Organizer struct {
			Email string `json:"email"`
			Self  bool   `json:"self"`
		} `json:"organizer"`
		Attendees []struct {
			Email string `json:"email"`
		} `json:"attendees"`
		Summary string `json:"summary"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&eventData); err != nil {
		return errors.NewAppError(errors.ErrInternalServer, "Failed to parse event", err)
	}

	// 2. Check if current user is the organizer
	if eventData.Organizer.Self {
		// User is organizer - DELETE the event entirely
		deleteReq, _ := http.NewRequest("DELETE", eventURL, nil)
		deleteReq.Header.Set("Authorization", "Bearer "+accessToken)

		deleteResp, err := client.Do(deleteReq)
		if err != nil {
			return errors.NewAppError(errors.ErrInternalServer, "Failed to delete event", err)
		}
		defer deleteResp.Body.Close()

		if deleteResp.StatusCode != http.StatusNoContent && deleteResp.StatusCode != http.StatusOK {
			return errors.NewAppError(errors.ErrInternalServer, "Google API error when deleting", nil)
		}

		// Send notifications to attendees about cancellation
		if len(eventData.Attendees) > 0 && s.notifService != nil {
			organizerEmail := eventData.Organizer.Email
			eventTitle := eventData.Summary
			attendees := eventData.Attendees

			go func() {
				bgCtx := context.Background()
				logger.Info("DeleteEvent:SendingNotifications", "attendee_count", len(attendees), "organizer", organizerEmail)

				for _, att := range attendees {
					// Skip organizer
					if att.Email == organizerEmail {
						logger.Info("DeleteEvent:SkippingOrganizer", "email", att.Email)
						continue
					}

					logger.Info("DeleteEvent:LookingUpUser", "email", att.Email)
					user, err := s.userRepo.GetUserByIdentifier(bgCtx, att.Email)
					if err != nil {
						logger.Error("DeleteEvent:GetUser:Error", "email", att.Email, "error", err)
						continue
					}
					if user == nil {
						logger.Warn("DeleteEvent:UserNotFound", "email", att.Email)
						continue
					}

					// Create cancellation notification
					logger.Info("DeleteEvent:CreatingNotification", "user_id", user.ID, "email", att.Email)
					err = s.notifService.Create(bgCtx, &notifDto.CreateNotificationRequest{
						UserID:  user.ID,
						Title:   "Sự kiện đã bị hủy",
						Message: fmt.Sprintf("Sự kiện '%s' đã bị hủy bởi người tổ chức", eventTitle),
						Type:    "event_cancelled",
					})
					if err != nil {
						logger.Error("DeleteEvent:CreateNotification:Error", "error", err)
					} else {
						logger.Info("DeleteEvent:NotificationCreated", "user_id", user.ID)
					}
				}
			}()
		}

		return nil
	}

	// 3. User is attendee - PATCH to decline
	userEmail := conn.CalendarEmail
	patchPayload := map[string]interface{}{
		"attendees": []map[string]interface{}{
			{
				"email":          userEmail,
				"responseStatus": "declined",
			},
		},
	}

	jsonBody, _ := json.Marshal(patchPayload)
	patchReq, _ := http.NewRequest("PATCH", eventURL, strings.NewReader(string(jsonBody)))
	patchReq.Header.Set("Authorization", "Bearer "+accessToken)
	patchReq.Header.Set("Content-Type", "application/json")

	patchResp, err := client.Do(patchReq)
	if err != nil {
		return errors.NewAppError(errors.ErrInternalServer, "Failed to decline event", err)
	}
	defer patchResp.Body.Close()

	if patchResp.StatusCode != http.StatusOK {
		return errors.NewAppError(errors.ErrInternalServer, "Google API error when declining", nil)
	}

	return nil
}

// ensureValidToken refreshes token if expired
func (s *calendarService) ensureValidToken(ctx context.Context, conn *entity.CalendarConnection) (string, error) {
	if time.Now().Before(conn.TokenExpiresAt.Add(-5 * time.Minute)) {
		return conn.AccessToken, nil
	}

	logger.Info("ensureValidToken:RefreshingToken", "user_id", conn.UserID)

	// Token expired, refresh it
	cfg, _ := config.GetSafe()

	data := url.Values{}
	data.Set("client_id", cfg.GoogleAPI.ClientID)
	data.Set("client_secret", cfg.GoogleAPI.ClientSecret)
	data.Set("refresh_token", conn.RefreshToken)
	data.Set("grant_type", "refresh_token")

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		logger.Error("ensureValidToken:PostFormError", "error", err)
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Error("ensureValidToken:DecodeError", "error", err)
		return "", err
	}

	// Check for error in response
	if errMsg, ok := result["error"].(string); ok {
		errDesc, _ := result["error_description"].(string)
		logger.Error("ensureValidToken:GoogleError", "error", errMsg, "description", errDesc)
		return "", fmt.Errorf("Google token refresh error: %s - %s", errMsg, errDesc)
	}

	// Safe type assertions
	accessToken, ok := result["access_token"].(string)
	if !ok || accessToken == "" {
		logger.Error("ensureValidToken:NoAccessToken", "result", result)
		return "", fmt.Errorf("no access_token in response")
	}

	expiresInFloat, ok := result["expires_in"].(float64)
	if !ok {
		expiresInFloat = 3600 // Default 1 hour
	}
	expiresIn := int(expiresInFloat)

	conn.AccessToken = accessToken
	conn.TokenExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)

	if err := s.repo.UpdateConnection(ctx, conn); err != nil {
		logger.Error("Failed to update token", "error", err)
	}

	logger.Info("ensureValidToken:Success", "user_id", conn.UserID)
	return accessToken, nil
}

// callGoogleFreeBusy calls Google Calendar FreeBusy API
func (s *calendarService) callGoogleFreeBusy(accessToken, email string, startTime, endTime time.Time) ([]dto.TimeSlot, error) {
	payload := map[string]interface{}{
		"timeMin": startTime.Format(time.RFC3339),
		"timeMax": endTime.Format(time.RFC3339),
		"items": []map[string]string{
			{"id": email},
		},
	}

	payloadJSON, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", googleFreeBusyAPI, strings.NewReader(string(payloadJSON)))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google FreeBusy API error: %s", string(body))
	}

	var result struct {
		Calendars map[string]struct {
			Busy []struct {
				Start string `json:"start"`
				End   string `json:"end"`
			} `json:"busy"`
		} `json:"calendars"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var busySlots []dto.TimeSlot
	if cal, ok := result.Calendars[email]; ok {
		for _, busy := range cal.Busy {
			busySlots = append(busySlots, dto.TimeSlot{
				Start: busy.Start,
				End:   busy.End,
			})
		}
	}

	return busySlots, nil
}

// FindAvailableSlots finds available meeting slots for all participants
func (s *calendarService) FindAvailableSlots(ctx context.Context, req *dto.SuggestedSlotsRequest) (*dto.SuggestedSlotsResponse, error) {
	// Default values
	if req.DaysAhead <= 0 {
		req.DaysAhead = 7
	}
	if req.DurationMinutes <= 0 {
		req.DurationMinutes = 60
	}

	// Parse user IDs
	var userIDs []uuid.UUID
	for _, idStr := range req.UserIDs {
		if id, err := uuid.Parse(idStr); err == nil {
			userIDs = append(userIDs, id)
		}
	}

	if len(userIDs) == 0 {
		return &dto.SuggestedSlotsResponse{Slots: []dto.SuggestedSlot{}}, nil
	}

	// Calculate time range
	now := time.Now()
	var startTime, endTime time.Time

	// Use start_date if provided, otherwise start from now
	if req.StartDate != "" {
		// Parse start_date (YYYY-MM-DD format)
		parsedDate, err := time.Parse("2006-01-02", req.StartDate)
		if err == nil {
			startTime = time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, now.Location())
			// If start date is today and time has passed, start from next hour
			if startTime.Year() == now.Year() && startTime.Month() == now.Month() && startTime.Day() == now.Day() {
				startTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
			}
		} else {
			// Fallback to current hour
			startTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
		}
	} else {
		// Default: start from next hour today
		startTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
	}

	endTime = startTime.Add(time.Duration(req.DaysAhead) * 24 * time.Hour)

	// Get busy times for all users
	busyData, err := s.GetFreeBusyForUsers(ctx, userIDs, startTime, endTime)
	if err != nil {
		logger.Error("FindAvailableSlots:GetFreeBusyForUsers:Error", "error", err)
		// Continue with empty busy data
		busyData = []dto.UserFreeBusy{}
	}

	// Track connected vs disconnected users
	connectedUserIDs := make(map[string]bool)
	for _, userData := range busyData {
		connectedUserIDs[userData.UserID] = true
	}

	var disconnectedUsers []dto.DisconnectedUser
	for _, uid := range userIDs {
		if !connectedUserIDs[uid.String()] {
			disconnectedUsers = append(disconnectedUsers, dto.DisconnectedUser{
				UserID: uid.String(),
				// Email and Name would need to be fetched from users table if needed
			})
		}
	}

	connectedCount := len(busyData)
	disconnectedCount := len(disconnectedUsers)
	totalParticipants := len(userIDs)

	logger.Info("FindAvailableSlots:ConnectionStatus",
		"connected", connectedCount,
		"disconnected", disconnectedCount,
		"total", totalParticipants)

	logger.Info("FindAvailableSlots:BusyData", "user_count", len(busyData))
	for _, userData := range busyData {
		logger.Info("FindAvailableSlots:UserBusy", "user_id", userData.UserID, "busy_slots", len(userData.BusySlots))
	}

	// EMPLOYEE FREE TIME ALGORITHM:
	// 1. Collect all busy intervals from ALL users into one list
	// 2. Sort by start time
	// 3. Merge overlapping intervals
	// 4. Check candidate slots against merged busy intervals

	// Step 1: Collect all busy intervals
	type interval struct {
		start, end time.Time
	}
	var allBusyIntervals []interval
	for _, userData := range busyData {
		for _, busy := range userData.BusySlots {
			busyStart, err1 := time.Parse(time.RFC3339, busy.Start)
			busyEnd, err2 := time.Parse(time.RFC3339, busy.End)
			if err1 == nil && err2 == nil {
				allBusyIntervals = append(allBusyIntervals, interval{busyStart, busyEnd})
				logger.Info("FindAvailableSlots:BusyInterval",
					"user_id", userData.UserID,
					"start", busyStart.Format("2006-01-02 15:04"),
					"end", busyEnd.Format("2006-01-02 15:04"))
			}
		}
	}

	// Step 2: Sort by start time
	for i := 0; i < len(allBusyIntervals)-1; i++ {
		for j := i + 1; j < len(allBusyIntervals); j++ {
			if allBusyIntervals[j].start.Before(allBusyIntervals[i].start) {
				allBusyIntervals[i], allBusyIntervals[j] = allBusyIntervals[j], allBusyIntervals[i]
			}
		}
	}

	// Step 3: Merge overlapping intervals
	var mergedBusy []interval
	for _, curr := range allBusyIntervals {
		if len(mergedBusy) == 0 || mergedBusy[len(mergedBusy)-1].end.Before(curr.start) {
			// No overlap, add new interval
			mergedBusy = append(mergedBusy, curr)
		} else {
			// Overlap, extend the last interval's end if needed
			if curr.end.After(mergedBusy[len(mergedBusy)-1].end) {
				mergedBusy[len(mergedBusy)-1].end = curr.end
			}
		}
	}

	logger.Info("FindAvailableSlots:MergedIntervals", "count", len(mergedBusy))
	for i, mi := range mergedBusy {
		logger.Info("FindAvailableSlots:MergedBusy", "index", i,
			"start", mi.start.Format("2006-01-02 15:04"),
			"end", mi.end.Format("2006-01-02 15:04"))
	}

	// Generate candidate slots
	candidates := s.generateCandidateSlots(startTime, req.DaysAhead, req.DurationMinutes, req.WorkingHoursOnly)

	// Step 4: Check each candidate against merged busy intervals
	var slots []dto.SuggestedSlot
	for _, candidate := range candidates {
		isFree := true
		for _, busy := range mergedBusy {
			// Check overlap: slot overlaps with busy if NOT (slotEnd <= busyStart OR slotStart >= busyEnd)
			if !(candidate.end.Before(busy.start) || candidate.end.Equal(busy.start) ||
				candidate.start.After(busy.end) || candidate.start.Equal(busy.end)) {
				isFree = false
				break
			}
		}

		// If no busy data from any user, all slots are available
		if len(mergedBusy) == 0 && connectedCount > 0 {
			isFree = true
		}

		if !isFree {
			continue
		}

		slots = append(slots, dto.SuggestedSlot{
			StartTime:      candidate.start.Format(time.RFC3339),
			EndTime:        candidate.end.Format(time.RFC3339),
			Score:          100,
			AvailableCount: connectedCount,
			TotalCount:     connectedCount,
		})
	}

	// Filter and sort slots based on time preference
	logger.Info("FindAvailableSlots:Preference", "time_preference", req.TimePreference, "slots_before_filter", len(slots))

	if req.TimePreference != "" && len(slots) > 0 {
		// Helper to check if slot is in preferred time range
		isPreferred := func(slot dto.SuggestedSlot) bool {
			t, err := time.Parse(time.RFC3339, slot.StartTime)
			if err != nil {
				return false
			}
			hour := t.Hour()

			switch req.TimePreference {
			case "morning", "sáng", "sang":
				return hour >= 6 && hour < 12
			case "afternoon", "chiều", "chieu":
				return hour >= 12 && hour < 18
			case "evening", "tối", "toi":
				return hour >= 18 && hour < 23
			default:
				return true // No preference, all slots are valid
			}
		}

		// Filter to only preferred slots
		var preferredSlots []dto.SuggestedSlot
		for _, slot := range slots {
			if isPreferred(slot) {
				preferredSlots = append(preferredSlots, slot)
			}
		}

		logger.Info("FindAvailableSlots:FilteredSlots", "preferred_count", len(preferredSlots))

		// Use preferred slots if any, otherwise fall back to all slots
		if len(preferredSlots) > 0 {
			slots = preferredSlots
		}
	}

	// Limit to top 10 slots
	if len(slots) > 10 {
		slots = slots[:10]
	}

	// Generate warning message if there are disconnected users
	var warning string
	if disconnectedCount > 0 {
		if disconnectedCount == 1 {
			warning = "1 người tham gia chưa kết nối lịch. Kết quả đề xuất chỉ dựa trên những người đã kết nối."
		} else {
			warning = fmt.Sprintf("%d người tham gia chưa kết nối lịch. Kết quả đề xuất chỉ dựa trên những người đã kết nối.", disconnectedCount)
		}
	}

	return &dto.SuggestedSlotsResponse{
		Slots:             slots,
		ConnectedCount:    connectedCount,
		DisconnectedCount: disconnectedCount,
		TotalParticipants: totalParticipants,
		DisconnectedUsers: disconnectedUsers,
		Warning:           warning,
	}, nil
}

type candidateSlot struct {
	start time.Time
	end   time.Time
}

func (s *calendarService) generateCandidateSlots(startDate time.Time, daysAhead, durationMinutes int, workingHoursOnly bool) []candidateSlot {
	var slots []candidateSlot

	for day := 0; day < daysAhead; day++ {
		date := startDate.Add(time.Duration(day) * 24 * time.Hour)

		// Skip weekends
		if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
			continue
		}

		// Define working hours
		startHour := 8
		endHour := 18
		if !workingHoursOnly {
			startHour = 6 // Early morning
			endHour = 23  // Late night
		}

		// Generate slots every 30 minutes
		for hour := startHour; hour < endHour; hour++ {
			for minute := 0; minute < 60; minute += 30 {
				slotStart := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, date.Location())
				slotEnd := slotStart.Add(time.Duration(durationMinutes) * time.Minute)

				// Skip if slot ends after working hours
				if slotEnd.Hour() > endHour || (slotEnd.Hour() == endHour && slotEnd.Minute() > 0) {
					continue
				}

				slots = append(slots, candidateSlot{start: slotStart, end: slotEnd})
			}
		}
	}

	return slots
}

func (s *calendarService) isSlotBusy(slotStart, slotEnd time.Time, busySlots []dto.TimeSlot) bool {
	for _, busy := range busySlots {
		busyStart, err1 := time.Parse(time.RFC3339, busy.Start)
		busyEnd, err2 := time.Parse(time.RFC3339, busy.End)
		if err1 != nil || err2 != nil {
			continue
		}

		// Check overlap: slot overlaps with busy if NOT (slotEnd <= busyStart OR slotStart >= busyEnd)
		if !(slotEnd.Before(busyStart) || slotEnd.Equal(busyStart) || slotStart.After(busyEnd) || slotStart.Equal(busyEnd)) {
			return true
		}
	}
	return false
}
