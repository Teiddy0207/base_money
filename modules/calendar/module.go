package calendar

import (
	"go-api-starter/core/cache"
	"go-api-starter/core/database"
	"go-api-starter/core/middleware"
	authRepo "go-api-starter/modules/auth/repository"
	"go-api-starter/modules/calendar/controller"
	"go-api-starter/modules/calendar/repository"
	"go-api-starter/modules/calendar/router"
	"go-api-starter/modules/calendar/service"
	invitService "go-api-starter/modules/invitation/service"
	notifService "go-api-starter/modules/notification/service"

	"github.com/labstack/echo/v4"
)

func Init(e *echo.Echo, db database.Database, cache cache.Cache, notifService *notifService.NotificationService, invitationService *invitService.InvitationService) {
	// Initialize layers
	repo := repository.NewCalendarRepository(db)
	userRepo := authRepo.NewAuthRepository(db)

	calendarService := service.NewCalendarService(repo, userRepo, notifService, invitationService)
	calendarController := controller.NewCalendarController(calendarService)

	// Get middleware for auth
	mw := middleware.NewMiddleware(nil)

	// Setup routes
	router.NewCalendarRouter(calendarController).Setup(e, mw)
}
