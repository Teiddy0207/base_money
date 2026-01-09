package router

import (
	"go-api-starter/core/middleware"
	"go-api-starter/modules/meeting/controller"

	"github.com/labstack/echo/v4"
)

// MeetingRouter handles event routes
type MeetingRouter struct {
	MeetingController *controller.MeetingController
}

// NewMeetingRouter creates a new router
func NewMeetingRouter(meetingController *controller.MeetingController) *MeetingRouter {
	return &MeetingRouter{
		MeetingController: meetingController,
	}
}

// Setup registers event routes
func (r *MeetingRouter) Setup(e *echo.Echo, mw *middleware.Middleware) {
	v1 := e.Group("/api/v1")
	privateRoutes := v1.Group("/private")

	// Event routes (all protected) - uses /events endpoint
	eventRoutes := privateRoutes.Group("/events", mw.AuthMiddleware())

	// CRUD
	eventRoutes.POST("", r.MeetingController.CreateEvent)
	eventRoutes.GET("", r.MeetingController.GetMyEvents)
	eventRoutes.GET("/:id", r.MeetingController.GetEvent)
	eventRoutes.PUT("/:id", r.MeetingController.UpdateEvent)
	eventRoutes.DELETE("/:id", r.MeetingController.DeleteEvent)

	// Slot finding
	eventRoutes.POST("/:id/find-slots", r.MeetingController.FindSlots)
	eventRoutes.POST("/:id/select-slot", r.MeetingController.SelectSlot)

	// Also register /meetings endpoint for backward compatibility
	meetingRoutes := privateRoutes.Group("/meetings", mw.AuthMiddleware())
	meetingRoutes.POST("", r.MeetingController.CreateEvent)
	meetingRoutes.GET("", r.MeetingController.GetMyEvents)
	meetingRoutes.GET("/:id", r.MeetingController.GetEvent)
	meetingRoutes.PUT("/:id", r.MeetingController.UpdateEvent)
	meetingRoutes.DELETE("/:id", r.MeetingController.DeleteEvent)
	meetingRoutes.POST("/:id/find-slots", r.MeetingController.FindSlots)
	meetingRoutes.POST("/:id/select-slot", r.MeetingController.SelectSlot)
}
