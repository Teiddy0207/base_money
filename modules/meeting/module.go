package meeting

import (
	"go-api-starter/core/database"
	"go-api-starter/core/middleware"
	"go-api-starter/modules/meeting/controller"
	"go-api-starter/modules/meeting/repository"
	"go-api-starter/modules/meeting/router"
	"go-api-starter/modules/meeting/service"

	"github.com/labstack/echo/v4"
)

// Init initializes the meeting module and registers routes
func Init(e *echo.Echo, db database.Database, mw *middleware.Middleware) {
	repo := repository.NewMeetingRepository(db)
	svc := service.NewMeetingService(repo)
	ctrl := controller.NewMeetingController(svc)
	rtr := router.NewMeetingRouter(ctrl)

	rtr.Setup(e, mw)
}
