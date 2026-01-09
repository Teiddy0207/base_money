package router

import (
	"go-api-starter/modules/booking/controller"

	"github.com/labstack/echo/v4"
)

type BookingRouter struct {
	Controller *controller.BookingController
}

func NewBookingRouter(ctrl *controller.BookingController) *BookingRouter {
	return &BookingRouter{Controller: ctrl}
}

func (r *BookingRouter) Setup(e *echo.Echo, mw interface{}) {
	e.GET("/p/:slug", r.Controller.PublicPage)
	e.GET("/personal-booking/:id", r.Controller.PublicPersonalPage)
	e.GET("/api/v1/public/booking/:slug/free", r.Controller.PublicFreeSlots)
	e.POST("/api/v1/public/booking/:slug/schedule", r.Controller.PublicSchedule)
	e.POST("/api/v1/public/booking/:id/suggested-slots", r.Controller.PublicSuggestedSlots)
	e.GET("/api/v1/public/booking/requests/:id/accept", r.Controller.PublicTokenAccept)
	e.GET("/api/v1/public/booking/requests/:id/decline", r.Controller.PublicTokenDecline)
	// Private booking approval routes
	if mw != nil {
		if m, ok := mw.(interface {
			AuthMiddleware() echo.MiddlewareFunc
		}); ok {
			v1 := e.Group("/api/v1")
			priv := v1.Group("/private", m.AuthMiddleware())
			req := priv.Group("/booking/requests")
			req.GET("", r.Controller.PrivateListPending)
			req.POST("/:id/accept", r.Controller.PrivateAcceptRequest)
			req.POST("/:id/decline", r.Controller.PrivateDeclineRequest)
			
			// Personal booking URL endpoint
			booking := priv.Group("/booking")
			booking.GET("/personal-url", r.Controller.GetPersonalBookingURL)
		}
	}
}

