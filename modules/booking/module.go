package booking

import (
	"go-api-starter/core/cache"
	"go-api-starter/core/database"
	"go-api-starter/core/middleware"
	authRepository "go-api-starter/modules/auth/repository"
	authService "go-api-starter/modules/auth/service"
	"go-api-starter/modules/booking/controller"
	bookingService "go-api-starter/modules/booking/service"
	"go-api-starter/modules/booking/router"
	calRepository "go-api-starter/modules/calendar/repository"
	calService "go-api-starter/modules/calendar/service"
	invitService "go-api-starter/modules/invitation/service"
	notifService "go-api-starter/modules/notification/service"
	meetRepository "go-api-starter/modules/meeting/repository"

	"github.com/labstack/echo/v4"
)

func Init(e *echo.Echo, db database.Database, cache cache.Cache, notifSvc *notifService.NotificationService, invitSvc *invitService.InvitationService) {
	calRepo := calRepository.NewCalendarRepository(db)
	authRepo := authRepository.NewAuthRepository(db)
	authSvc := authService.NewAuthService(authRepo, cache)
	calSvc := calService.NewCalendarService(calRepo, authRepo, notifSvc, invitSvc)
	meetRepo := meetRepository.NewMeetingRepository(db)
	
	// Initialize booking service
	bookingSvc := bookingService.NewBookingService(authSvc)
	
	ctrl := controller.NewBookingController(calSvc, authSvc, meetRepo, notifSvc, bookingSvc)
	mw := middleware.NewMiddleware(authSvc)
	router.NewBookingRouter(ctrl).Setup(e, mw)
}
