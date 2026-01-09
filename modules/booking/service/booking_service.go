package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"go-api-starter/core/config"
	"go-api-starter/core/constants"
	"go-api-starter/core/errors"
	"go-api-starter/core/logger"
	"go-api-starter/core/params"
	authservice "go-api-starter/modules/auth/service"
	"go-api-starter/modules/booking/dto"

	"github.com/google/uuid"
)

type BookingService interface {
	GetPersonalBookingURL(ctx context.Context, userID uuid.UUID) (*dto.PersonalBookingURLResponse, *errors.AppError)
	GetWeekStatistics(ctx context.Context, userID uuid.UUID) (*dto.WeekStatisticsResponse, *errors.AppError)
}

type bookingService struct {
	authService authservice.AuthServiceInterface
}

func NewBookingService(authService authservice.AuthServiceInterface) BookingService {
	return &bookingService{
		authService: authService,
	}
}

// GetPersonalBookingURL returns the personal booking page URL for the authenticated user
func (s *bookingService) GetPersonalBookingURL(ctx context.Context, userID uuid.UUID) (*dto.PersonalBookingURLResponse, *errors.AppError) {
	logger.Info("BookingService:GetPersonalBookingURL:Start", "user_id", userID)

	// Get Google social login for this user (booking page primarily uses Google Calendar)
	socialLogin, appErr := s.authService.GetSocialLoginByUserAndProviderName(ctx, userID, "google")
	if appErr != nil {
		logger.Error("BookingService:GetPersonalBookingURL:GetSocialLoginByUserAndProviderName:Error", "error", appErr, "user_id", userID)
		return nil, errors.NewAppError(errors.ErrNotFound, "Chưa kết nối Google Calendar. Vui lòng kết nối Google Calendar trước.", nil)
	}

	if socialLogin == nil {
		logger.Warn("BookingService:GetPersonalBookingURL:SocialLoginNotFound", "user_id", userID)
		return nil, errors.NewAppError(errors.ErrNotFound, "Chưa kết nối Google Calendar. Vui lòng kết nối Google Calendar trước.", nil)
	}

	// Get server host from config
	cfg, isInitialized := config.GetSafe()
	if !isInitialized {
		logger.Error("BookingService:GetPersonalBookingURL:ConfigNotInitialized")
		return nil, errors.NewAppError(errors.ErrInternalServer, "Server configuration error", nil)
	}

	// Build URL: http://localhost:7070/personal-booking/{social_login_id}
	// or use config server host if available
	host := fmt.Sprintf("http://localhost:%d", cfg.Server.Port)
	if cfg.Server.Host != "" && cfg.Server.Host != "0.0.0.0" {
		host = fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.Port)
	}

	bookingURL := fmt.Sprintf("%s/personal-booking/%s", host, socialLogin.ID.String())

	logger.Info("BookingService:GetPersonalBookingURL:Success", "user_id", userID, "social_login_id", socialLogin.ID, "url", bookingURL)

	return &dto.PersonalBookingURLResponse{
		URL: bookingURL,
	}, nil
}

// GetWeekStatistics returns weekly event statistics (total events and total duration)
func (s *bookingService) GetWeekStatistics(ctx context.Context, userID uuid.UUID) (*dto.WeekStatisticsResponse, *errors.AppError) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultTimeout)
	defer cancel()

	logger.Info("BookingService:GetWeekStatistics:Start", "user_id", userID)

	// Calculate start and end of current week (Monday to Sunday)
	now := time.Now()
	weekday := int(now.Weekday())
	// Convert Sunday (0) to 7, Monday (1) to 1, etc.
	if weekday == 0 {
		weekday = 7
	}

	// Get Monday of current week
	monday := now.AddDate(0, 0, -(weekday - 1))
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())

	// Get Sunday of current week (end of week)
	sunday := monday.AddDate(0, 0, 6)
	sunday = time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 23, 59, 59, 0, sunday.Location())

	// Format as RFC3339 for Google Calendar API
	timeMin := monday.Format(time.RFC3339)
	timeMax := sunday.Format(time.RFC3339)

	logger.Info("BookingService:GetWeekStatistics:WeekRange", "user_id", userID, "monday", timeMin, "sunday", timeMax)

	// Get all events for this week (use large page size to get all events)
	queryParams := params.QueryParams{
		PageNumber: 1,
		PageSize:   1000, // Large enough to get all events
	}

	eventsResult, appErr := s.authService.GetGoogleCalendarEvents(ctx, userID, queryParams, timeMin, timeMax)
	if appErr != nil {
		logger.Error("BookingService:GetWeekStatistics:GetGoogleCalendarEvents:Error", "error", appErr, "user_id", userID)
		return nil, appErr
	}

	if eventsResult == nil {
		return &dto.WeekStatisticsResponse{
			TotalEvents:          0,
			TotalDurationMinutes: 0,
			TotalDurationHours:   0,
		}, nil
	}

	// Calculate statistics
	totalEvents := 0
	totalDurationMinutes := 0

	for _, event := range eventsResult.Items {
		// Parse start and end times
		var startTime, endTime time.Time
		var err error

		if event.Start.DateTime != "" {
			startTime, err = time.Parse(time.RFC3339, event.Start.DateTime)
		} else if event.Start.Date != "" {
			startTime, err = time.Parse("2006-01-02", event.Start.Date)
		}

		if event.End.DateTime != "" {
			endTime, err = time.Parse(time.RFC3339, event.End.DateTime)
		} else if event.End.Date != "" {
			endTime, err = time.Parse("2006-01-02", event.End.Date)
			// For all-day events, end date is exclusive, so add 1 day
			endTime = endTime.AddDate(0, 0, 1)
		}

		if err != nil {
			logger.Warn("BookingService:GetWeekStatistics:ParseTimeError", "error", err, "event_id", event.ID)
			continue
		}

		// Calculate duration in minutes
		duration := endTime.Sub(startTime)
		durationMinutes := int(duration.Minutes())

		totalEvents++
		totalDurationMinutes += durationMinutes
	}

	// Convert to hours (rounded to 2 decimal places)
	totalDurationHours := math.Round(float64(totalDurationMinutes)/60.0*100) / 100

	logger.Info("BookingService:GetWeekStatistics:Success",
		"user_id", userID,
		"total_events", totalEvents,
		"total_duration_minutes", totalDurationMinutes,
		"total_duration_hours", totalDurationHours)

	return &dto.WeekStatisticsResponse{
		TotalEvents:          totalEvents,
		TotalDurationMinutes: totalDurationMinutes,
		TotalDurationHours:   totalDurationHours,
	}, nil
}
