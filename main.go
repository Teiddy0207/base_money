package main

import (
	"go-api-starter/core/logger"
	"go-api-starter/core/server"

	_ "go-api-starter/docs" // Swagger docs
)

// @title SmartSchedule API
// @version 1.0
// @description API Backend cho ứng dụng SmartSchedule - Lịch hẹn thông minh
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@smartschedule.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:7070
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT Bearer token. Example: "Bearer {token}"

func main() {
	if err := server.Run(); err != nil {
		logger.Error("run server error", err)
	}
}
