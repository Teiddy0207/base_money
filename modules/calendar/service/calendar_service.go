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
	authservice "go-api-starter/modules/auth/service"
	"go-api-starter/modules/calendar/dto"
	"go-api-starter/modules/calendar/entity"
	"go-api-starter/modules/calendar/repository"

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
}

type calendarService struct {
	repo        repository.CalendarRepository
	authService authservice.AuthServiceInterface
}

func NewCalendarService(repo repository.CalendarRepository, authSvc authservice.AuthServiceInterface) CalendarService {
	return &calendarService{repo: repo, authService: authSvc}
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
	accessToken, appErr := s.authService.GetGoogleAccessToken(ctx, userID)
	if appErr != nil {
		return nil, appErr
	}

	// Call Google Calendar FreeBusy API
	busySlots, err := s.callGoogleFreeBusy(accessToken, "", startTime, endTime)
	if err != nil {
		return nil, err
	}

	return busySlots, nil
}

// GetFreeBusyForUsers gets free/busy info for multiple users
func (s *calendarService) GetFreeBusyForUsers(ctx context.Context, userIDs []uuid.UUID, startTime, endTime time.Time) ([]dto.UserFreeBusy, error) {
	var results []dto.UserFreeBusy
	for _, uid := range userIDs {
		accessToken, appErr := s.authService.GetGoogleAccessToken(ctx, uid)
		if appErr != nil {
			logger.Error("Failed to get token for user", "user_id", uid, "error", appErr)
			continue
		}
		busySlots, err := s.callGoogleFreeBusy(accessToken, "", startTime, endTime)
		if err != nil {
			logger.Error("Failed to get free/busy for user", "user_id", uid, "error", err)
			continue
		}

		results = append(results, dto.UserFreeBusy{
			UserID:    uid.String(),
			Email:     "",
			BusySlots: busySlots,
		})
	}

	return results, nil
}

// CreateEvent creates an event on Google Calendar
func (s *calendarService) CreateEvent(ctx context.Context, userID uuid.UUID, req *dto.CreateEventRequest) (*dto.CreateEventResponse, error) {
	accessToken, appErr := s.authService.GetGoogleAccessToken(ctx, userID)
	if appErr != nil {
		return nil, appErr
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

	return &dto.CreateEventResponse{
		EventID:     result["id"].(string),
		Title:       req.Title,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		MeetingLink: req.MeetingLink,
	}, nil
}

// ensureValidToken refreshes token if expired
func (s *calendarService) ensureValidToken(ctx context.Context, conn *entity.CalendarConnection) (string, error) {
	if time.Now().Before(conn.TokenExpiresAt.Add(-5 * time.Minute)) {
		return conn.AccessToken, nil
	}

	// Token expired, refresh it
	cfg, _ := config.GetSafe()

	data := url.Values{}
	data.Set("client_id", cfg.GoogleAPI.ClientID)
	data.Set("client_secret", cfg.GoogleAPI.ClientSecret)
	data.Set("refresh_token", conn.RefreshToken)
	data.Set("grant_type", "refresh_token")

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	accessToken := result["access_token"].(string)
	expiresIn := int(result["expires_in"].(float64))

	conn.AccessToken = accessToken
	conn.TokenExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)

	if err := s.repo.UpdateConnection(ctx, conn); err != nil {
		logger.Error("Failed to update token", "error", err)
	}

	return accessToken, nil
}

// callGoogleFreeBusy calls Google Calendar FreeBusy API
func (s *calendarService) callGoogleFreeBusy(accessToken, email string, startTime, endTime time.Time) ([]dto.TimeSlot, error) {
	payload := map[string]interface{}{
		"timeMin": startTime.Format(time.RFC3339),
		"timeMax": endTime.Format(time.RFC3339),
		"items": []map[string]string{
			{"id": "primary"},
		},
	}

	payloadJSON, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", googleFreeBusyAPI, strings.NewReader(string(payloadJSON)))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	logger.Info("CalendarService:FreeBusy:Request", "time_min", startTime.Format(time.RFC3339), "time_max", endTime.Format(time.RFC3339))
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	logger.Info("CalendarService:FreeBusy:Response", "status", resp.StatusCode)
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
	if cal, ok := result.Calendars["primary"]; ok {
		for _, busy := range cal.Busy {
			busySlots = append(busySlots, dto.TimeSlot{
				Start: busy.Start,
				End:   busy.End,
			})
		}
	}

	logger.Info("CalendarService:FreeBusy:BusyCount", "count", len(busySlots))
	return busySlots, nil
}
