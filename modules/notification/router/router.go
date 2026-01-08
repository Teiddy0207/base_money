package router

import (
	"go-api-starter/core/middleware"
	"go-api-starter/modules/notification/controller"

	"github.com/labstack/echo/v4"
)

type NotificationRouter struct {
	controller *controller.NotificationController
}

func NewNotificationRouter(controller *controller.NotificationController) *NotificationRouter {
	return &NotificationRouter{controller: controller}
}

func (r *NotificationRouter) Register(e *echo.Group, mw *middleware.Middleware) {
	group := e.Group("/notifications", mw.AuthMiddleware())
	group.GET("", r.controller.GetMyNotifications)
	group.GET("/unread-count", r.controller.CountUnread)
	group.PUT("/mark-read", r.controller.MarkAsRead)
	group.PUT("/mark-all-read", r.controller.MarkAllAsRead)
}
