package service

import (
	"context"
	"encoding/json"
	"fmt"
	"go-api-starter/core/config"
	"go-api-starter/core/constants"
	"go-api-starter/core/errors"
	"go-api-starter/core/logger"
	"go-api-starter/core/params"
	"go-api-starter/modules/auth/dto"
	"go-api-starter/modules/auth/mapper"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func (service *AuthService) GetGoogleCalendarEvents(ctx context.Context, userID uuid.UUID, timeMin string, timeMax string) ([]dto.GoogleCalendarEvent, *errors.AppError) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultTimeout)
	defer cancel()

	googleToken, err := service.getGoogleTokenForUser(ctx, userID)
	if err != nil {
		logger.Error("AuthService:GetGoogleCalendarEvents:GetGoogleToken:Error", "error", err, "user_id", userID)
		return nil, errors.NewAppError(errors.ErrUnauthorized, "Google OAuth token not found. Please login with Google again", nil)
	}

	apiURL := service.buildCalendarEventsURL(timeMin, timeMax)
	events, appErr := service.fetchGoogleCalendarEvents(ctx, apiURL, googleToken)
	if appErr != nil {
		return nil, appErr
	}

	return mapper.ToGoogleCalendarEventsDTO(events), nil
}

func (service *AuthService) GetGoogleCalendarList(ctx context.Context, userID uuid.UUID, params params.QueryParams) (*dto.PaginatedGoogleCalendarDTO, *errors.AppError) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultTimeout)
	defer cancel()

	googleToken, err := service.getGoogleTokenForUser(ctx, userID)
	if err != nil {
		logger.Error("AuthService:GetGoogleCalendarList:GetGoogleToken:Error", "error", err, "user_id", userID)
		return nil, errors.NewAppError(errors.ErrUnauthorized, "Google OAuth token not found. Please login with Google again", nil)
	}

	allCalendars, appErr := service.fetchGoogleCalendarList(ctx, googleToken)
	if appErr != nil {
		return nil, appErr
	}

	totalItems := len(allCalendars)
	offset := (params.PageNumber - 1) * params.PageSize
	end := offset + params.PageSize

	if offset > totalItems {
		return mapper.ToPaginatedGoogleCalendarDTO([]dto.GoogleCalendar{}, totalItems, params.PageNumber, params.PageSize), nil
	}

	if end > totalItems {
		end = totalItems
	}

	paginatedItems := allCalendars[offset:end]

	return mapper.ToPaginatedGoogleCalendarDTO(paginatedItems, totalItems, params.PageNumber, params.PageSize), nil
}

func (service *AuthService) fetchGoogleCalendarList(ctx context.Context, accessToken string) ([]dto.GoogleCalendar, *errors.AppError) {
	apiURL := "https://www.googleapis.com/calendar/v3/users/me/calendarList"
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		logger.Error("AuthService:fetchGoogleCalendarList:NewRequest:Error", "error", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to create request", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("AuthService:fetchGoogleCalendarList:DoRequest:Error", "error", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to fetch calendar list", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Error("AuthService:fetchGoogleCalendarList:APIError", "status", resp.StatusCode, "body", string(body))
		return nil, errors.NewAppError(errors.ErrInternalServer, fmt.Sprintf("Google Calendar API error: %d", resp.StatusCode), nil)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("AuthService:fetchGoogleCalendarList:ReadBody:Error", "error", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to read response", err)
	}

	var calendarListResponse struct {
		Items []dto.GoogleCalendar `json:"items"`
	}
	if err := json.Unmarshal(body, &calendarListResponse); err != nil {
		logger.Error("AuthService:fetchGoogleCalendarList:Unmarshal:Error", "error", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to parse response", err)
	}

	return calendarListResponse.Items, nil
}

func (service *AuthService) buildCalendarEventsURL(timeMin string, timeMax string) string {
	apiURL := "https://www.googleapis.com/calendar/v3/calendars/primary/events"
	params := url.Values{}
	params.Add("singleEvents", "true")
	params.Add("orderBy", "startTime")

	if timeMin != "" {
		params.Add("timeMin", timeMin)
	} else {
		params.Add("timeMin", time.Now().Format(time.RFC3339))
	}

	if timeMax != "" {
		params.Add("timeMax", timeMax)
	}

	return apiURL + "?" + params.Encode()
}

func (service *AuthService) fetchGoogleCalendarEvents(ctx context.Context, apiURL string, accessToken string) ([]dto.GoogleCalendarEvent, *errors.AppError) {
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		logger.Error("AuthService:fetchGoogleCalendarEvents:NewRequest:Error", "error", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to create request", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("AuthService:fetchGoogleCalendarEvents:DoRequest:Error", "error", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to fetch calendar events", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Error("AuthService:fetchGoogleCalendarEvents:APIError", "status", resp.StatusCode, "body", string(body))
		return nil, errors.NewAppError(errors.ErrInternalServer, fmt.Sprintf("Google Calendar API error: %d", resp.StatusCode), nil)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("AuthService:fetchGoogleCalendarEvents:ReadBody:Error", "error", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to read response", err)
	}

	var calendarResponse struct {
		Items []dto.GoogleCalendarEvent `json:"items"`
	}
	if err := json.Unmarshal(body, &calendarResponse); err != nil {
		logger.Error("AuthService:fetchGoogleCalendarEvents:Unmarshal:Error", "error", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to parse response", err)
	}

	return calendarResponse.Items, nil
}

func (service *AuthService) getGoogleTokenForUser(ctx context.Context, userID uuid.UUID) (string, error) {
	provider, err := service.repo.GetOAuthProviderByName(ctx, "google")
	if err != nil {
		return "", fmt.Errorf("failed to get Google provider: %w", err)
	}
	if provider == nil {
		return "", fmt.Errorf("Google provider not found in database")
	}

	socialLogin, err := service.repo.GetSocialLoginByUserIDAndProvider(ctx, userID, provider.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get social login: %w", err)
	}
	if socialLogin == nil || socialLogin.AccessToken == nil {
		return "", fmt.Errorf("Google token not found for user %s. Please login with Google again", userID)
	}

	accessToken := *socialLogin.AccessToken
	refreshToken := ""
	if socialLogin.RefreshToken != nil {
		refreshToken = *socialLogin.RefreshToken
	}

	var expiresAt time.Time
	if socialLogin.TokenExpiresAt != nil {
		expiresAt = *socialLogin.TokenExpiresAt
	}

	if !expiresAt.IsZero() && time.Now().After(expiresAt) {
		if refreshToken == "" {
			return "", fmt.Errorf("Google token expired and no refresh token available")
		}

		newToken, err := service.refreshGoogleToken(ctx, refreshToken)
		if err != nil {
			return "", fmt.Errorf("failed to refresh Google token: %w", err)
		}

		expiresAtTime := newToken.ExpiresAt
		socialLogin.AccessToken = &newToken.AccessToken
		if newToken.RefreshToken != "" {
			socialLogin.RefreshToken = &newToken.RefreshToken
		}
		socialLogin.TokenExpiresAt = &expiresAtTime
		socialLogin.LastLoginAt = &expiresAtTime

		if err := service.repo.SaveOrUpdateSocialLogin(ctx, socialLogin); err != nil {
			logger.Error("AuthService:getGoogleTokenForUser:SaveOrUpdateSocialLogin:Error", "error", err)
			return "", fmt.Errorf("failed to update refreshed token: %w", err)
		}

		return newToken.AccessToken, nil
	}

	return accessToken, nil
}

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
