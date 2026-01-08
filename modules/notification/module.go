package notification

import (
	"go-api-starter/core/database"
	"go-api-starter/core/middleware"
	"go-api-starter/modules/notification/controller"
	"go-api-starter/modules/notification/repository"
	"go-api-starter/modules/notification/router"
	"go-api-starter/modules/notification/service"

	"github.com/labstack/echo/v4"
)

func Init(e *echo.Group, db database.Database, mw *middleware.Middleware) *service.NotificationService {
	repo := repository.NewNotificationRepository(db)
	svc := service.NewNotificationService(repo)
	ctrl := controller.NewNotificationController(svc)

	router.NewNotificationRouter(ctrl).Register(e, mw)

	return svc
}
