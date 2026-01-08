package invitation

import (
	"go-api-starter/core/database"
	"go-api-starter/core/middleware"
	authRepo "go-api-starter/modules/auth/repository"
	"go-api-starter/modules/invitation/controller"
	"go-api-starter/modules/invitation/repository"
	"go-api-starter/modules/invitation/router"
	"go-api-starter/modules/invitation/service"
	notifService "go-api-starter/modules/notification/service"

	"github.com/labstack/echo/v4"
)

// Init initializes the invitation module and returns the service for use by other modules
func Init(g *echo.Group, db database.Database, mw *middleware.Middleware, notificationService *notifService.NotificationService) *service.InvitationService {
	repo := repository.NewInvitationRepository(db)
	authRepository := authRepo.NewAuthRepository(db)
	svc := service.NewInvitationService(repo, notificationService, authRepository)
	ctrl := controller.NewInvitationController(svc)
	r := router.NewInvitationRouter(ctrl)

	r.Register(g, mw)

	return svc
}
