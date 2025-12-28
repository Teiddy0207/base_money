package calendar

import (
	"go-api-starter/core/cache"
	"go-api-starter/core/database"
	"go-api-starter/core/middleware"
	"go-api-starter/modules/calendar/controller"
	"go-api-starter/modules/calendar/repository"
	"go-api-starter/modules/calendar/router"
	"go-api-starter/modules/calendar/service"

	"github.com/labstack/echo/v4"
)

func Init(e *echo.Echo, db database.Database, cache cache.Cache) {
	// Initialize layers
	repo := repository.NewCalendarRepository(db)
	calendarService := service.NewCalendarService(repo)
	calendarController := controller.NewCalendarController(calendarService)

	// Get middleware for auth
	mw := middleware.NewMiddleware(nil) // Auth service will be injected separately

	// Setup routes
	router.NewCalendarRouter(calendarController).Setup(e, mw)
}
