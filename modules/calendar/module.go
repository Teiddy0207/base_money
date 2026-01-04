package calendar

import (
	"go-api-starter/core/cache"
	"go-api-starter/core/database"
	"go-api-starter/core/middleware"
	authRepository "go-api-starter/modules/auth/repository"
	authService "go-api-starter/modules/auth/service"
	"go-api-starter/modules/calendar/controller"
	"go-api-starter/modules/calendar/repository"
	"go-api-starter/modules/calendar/router"
	"go-api-starter/modules/calendar/service"
	productRepository "go-api-starter/modules/product/repository"
	productService "go-api-starter/modules/product/service"

	"github.com/labstack/echo/v4"
)

func Init(e *echo.Echo, db database.Database, cache cache.Cache) {
	// Initialize layers
	repo := repository.NewCalendarRepository(db)
	authRepo := authRepository.NewAuthRepository(db)
	authSvc := authService.NewAuthService(authRepo, cache)
	calendarService := service.NewCalendarService(repo, authSvc)
	productRepo := productRepository.NewProductRepository(db)
	productSvc := productService.NewProductService(productRepo)
	calendarController := controller.NewCalendarController(calendarService, productSvc, authSvc)

	// Get middleware for auth
	mw := middleware.NewMiddleware(authSvc)

	// Setup routes
	router.NewCalendarRouter(calendarController).Setup(e, mw)
}
