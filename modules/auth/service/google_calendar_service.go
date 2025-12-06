package service

import (
	"context"
	"encoding/json"
	"fmt"
	"go-api-starter/core/config"
	"go-api-starter/core/errors"
	"go-api-starter/core/logger"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleCalendar represents a Google Calendar
type GoogleCalendar struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	TimeZone    string `json:"timeZone"`
}

// GoogleCalendarEvent represents a Google Calendar event
type GoogleCalendarEvent struct {
	ID          string    `json:"id"`
	Summary     string    `json:"summary"`
	Description string    `json:"description"`
	Start       EventTime `json:"start"`
	End         EventTime `json:"end"`
	Location    string    `json:"location"`
	Status      string    `json:"status"`
}

// EventTime represents event start/end time
type EventTime struct {
	DateTime string `json:"dateTime"`
	Date     string `json:"date"`
	TimeZone string `json:"timeZone"`
}

// GetGoogleCalendarEvents retrieves calendar events for a user
func (service *AuthService) GetGoogleCalendarEvents(ctx context.Context, userID uuid.UUID, timeMin string, timeMax string) ([]GoogleCalendarEvent, *errors.AppError) {
	// Get Google access token for user
	googleToken, err := service.getGoogleTokenForUser(ctx, userID)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrUnauthorized, "Google OAuth token not found. Please login with Google again", nil)
	}

	// Build Calendar API URL
	apiURL := "https://www.googleapis.com/calendar/v3/calendars/primary/events"
	params := url.Values{}
	params.Add("singleEvents", "true")
	params.Add("orderBy", "startTime")
	
	if timeMin != "" {
		params.Add("timeMin", timeMin)
	} else {
		// Default: events from now
		params.Add("timeMin", time.Now().Format(time.RFC3339))
	}
	
	if timeMax != "" {
		params.Add("timeMax", timeMax)
	}

	apiURL += "?" + params.Encode()

	// Make request to Google Calendar API
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		logger.Error("AuthService:GetGoogleCalendarEvents:NewRequest:Error", "error", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to create request", err)
	}

	req.Header.Set("Authorization", "Bearer "+googleToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("AuthService:GetGoogleCalendarEvents:DoRequest:Error", "error", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to fetch calendar events", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Error("AuthService:GetGoogleCalendarEvents:APIError", "status", resp.StatusCode, "body", string(body))
		return nil, errors.NewAppError(errors.ErrInternalServer, fmt.Sprintf("Google Calendar API error: %d", resp.StatusCode), nil)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("AuthService:GetGoogleCalendarEvents:ReadBody:Error", "error", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to read response", err)
	}

	var calendarResponse struct {
		Items []GoogleCalendarEvent `json:"items"`
	}
	if err := json.Unmarshal(body, &calendarResponse); err != nil {
		logger.Error("AuthService:GetGoogleCalendarEvents:Unmarshal:Error", "error", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to parse response", err)
	}

	return calendarResponse.Items, nil
}

// GetGoogleCalendarList retrieves list of calendars for a user
func (service *AuthService) GetGoogleCalendarList(ctx context.Context, userID uuid.UUID) ([]GoogleCalendar, *errors.AppError) {
	// Get Google access token for user
	googleToken, err := service.getGoogleTokenForUser(ctx, userID)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrUnauthorized, "Google OAuth token not found. Please login with Google again", nil)
	}

	// Make request to Google Calendar API
	apiURL := "https://www.googleapis.com/calendar/v3/users/me/calendarList"
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		logger.Error("AuthService:GetGoogleCalendarList:NewRequest:Error", "error", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to create request", err)
	}

	req.Header.Set("Authorization", "Bearer "+googleToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("AuthService:GetGoogleCalendarList:DoRequest:Error", "error", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to fetch calendar list", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Error("AuthService:GetGoogleCalendarList:APIError", "status", resp.StatusCode, "body", string(body))
		return nil, errors.NewAppError(errors.ErrInternalServer, fmt.Sprintf("Google Calendar API error: %d", resp.StatusCode), nil)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("AuthService:GetGoogleCalendarList:ReadBody:Error", "error", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to read response", err)
	}

	var calendarListResponse struct {
		Items []GoogleCalendar `json:"items"`
	}
	if err := json.Unmarshal(body, &calendarListResponse); err != nil {
		logger.Error("AuthService:GetGoogleCalendarList:Unmarshal:Error", "error", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to parse response", err)
	}

	return calendarListResponse.Items, nil
}

// getGoogleTokenForUser retrieves Google access token for a user
func (service *AuthService) getGoogleTokenForUser(ctx context.Context, userID uuid.UUID) (string, error) {
	// Get from in-memory storage
	googleToken, exists := service.googleTokens[userID]
	if !exists || googleToken == nil {
		return "", fmt.Errorf("Google token not found for user %s. Please login with Google again", userID)
	}

	// Check if token is expired
	if time.Now().After(googleToken.ExpiresAt) {
		// Token expired, try to refresh
		if googleToken.RefreshToken != "" {
			newToken, err := service.refreshGoogleToken(ctx, googleToken.RefreshToken)
			if err != nil {
				return "", fmt.Errorf("failed to refresh Google token: %w", err)
			}
			// Update stored token
			service.googleTokens[userID] = newToken
			return newToken.AccessToken, nil
		}
		return "", fmt.Errorf("Google token expired and no refresh token available")
	}

	return googleToken.AccessToken, nil
}

// refreshGoogleToken refreshes an expired Google access token
func (service *AuthService) refreshGoogleToken(ctx context.Context, refreshToken string) (*GoogleToken, error) {
	cfg, ok := config.GetSafe()
	if !ok {
		return nil, fmt.Errorf("config not initialized")
	}

	oauthConfig := &oauth2.Config{
		ClientID:     cfg.GoogleAPI.ClientID,
		ClientSecret: cfg.GoogleAPI.ClientSecret,
		Endpoint:     google.Endpoint,
	}

	tokenSource := oauthConfig.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
	})

	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, err
	}

	return &GoogleToken{
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken,
		ExpiresAt:    newToken.Expiry,
	}, nil
}

