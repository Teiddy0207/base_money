package router

import (
	"go-api-starter/core/middleware"
	"go-api-starter/modules/invitation/controller"

	"github.com/labstack/echo/v4"
)

type InvitationRouter struct {
	controller *controller.InvitationController
}

func NewInvitationRouter(controller *controller.InvitationController) *InvitationRouter {
	return &InvitationRouter{
		controller: controller,
	}
}

func (r *InvitationRouter) Register(g *echo.Group, mw *middleware.Middleware) {
	invitations := g.Group("/invitations")
	invitations.Use(mw.AuthMiddleware())

	invitations.GET("", r.controller.GetPendingInvitations)
	invitations.GET("/count", r.controller.CountPending)
	invitations.POST("/:id/accept", r.controller.AcceptInvitation)
	invitations.POST("/:id/decline", r.controller.DeclineInvitation)
}
