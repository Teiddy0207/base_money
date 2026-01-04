package router

import (
	"go-api-starter/core/middleware"
	"go-api-starter/modules/calendar/controller"

	"github.com/labstack/echo/v4"
)

type CalendarRouter struct {
	controller *controller.CalendarController
}

func NewCalendarRouter(controller *controller.CalendarController) *CalendarRouter {
	return &CalendarRouter{
		controller: controller,
	}
}

func (r *CalendarRouter) Setup(e *echo.Echo, mw *middleware.Middleware) {
	v1 := e.Group("/api/v1")

	// Private routes (require authentication)
	calendarRoutes := v1.Group("/private/calendar")
	calendarRoutes.Use(mw.AuthMiddleware())

	// Calendar connections
	calendarRoutes.GET("/connections", r.controller.GetConnections)
	calendarRoutes.DELETE("/connections/:provider", r.controller.DisconnectCalendar)

	// Free/Busy
	calendarRoutes.GET("/free-busy", r.controller.GetFreeBusy)

	// Events
	calendarRoutes.POST("/events", r.controller.CreateEvent)

	// User-specific busy view
	userRoutes := v1.Group("/private/users")
	userRoutes.Use(mw.AuthMiddleware())
	userRoutes.GET("/:id/calendar/busy", r.controller.GetUserBusy)
}
