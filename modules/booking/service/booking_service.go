package service

import (
	"context"
	"fmt"

	"go-api-starter/core/config"
	"go-api-starter/core/errors"
	"go-api-starter/core/logger"
	authservice "go-api-starter/modules/auth/service"
	"go-api-starter/modules/booking/dto"

	"github.com/google/uuid"
)

type BookingService interface {
	GetPersonalBookingURL(ctx context.Context, userID uuid.UUID) (*dto.PersonalBookingURLResponse, *errors.AppError)
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

