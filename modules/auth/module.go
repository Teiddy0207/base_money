package auth

import (
	"context"
	"go-api-starter/core/cache"
	"go-api-starter/core/config"
	"go-api-starter/core/database"
	"go-api-starter/core/logger"
	"go-api-starter/core/middleware"
	"go-api-starter/modules/auth/controller"
	"go-api-starter/modules/auth/repository"
	"go-api-starter/modules/auth/router"
	"go-api-starter/modules/auth/service"

	"github.com/labstack/echo/v4"
)

func Init(e *echo.Echo, db database.Database, cache cache.Cache) {
	repo := repository.NewAuthRepository(db)
	authService := service.NewAuthService(repo, cache)
	controller := controller.NewAuthController(authService)
	middleware := middleware.NewMiddleware(authService)

	seedGoogleProvider(repo)

	router.NewAuthRouter(*controller).Setup(e, middleware)
}

func seedGoogleProvider(repo repository.AuthRepositoryInterface) {
	cfg, ok := config.GetSafe()
	if !ok {
		logger.Warn("Auth:SeedGoogleProvider:ConfigNotInitialized")
		return
	}

	if cfg.GoogleAPI.ClientID == "" || cfg.GoogleAPI.ClientSecret == "" || cfg.GoogleAPI.RedirectURI == "" {
		logger.Info("Auth:SeedGoogleProvider:Skipped", "reason", "Google OAuth credentials not configured in env")
		return
	}

	ctx := context.Background()
	if err := repo.SeedGoogleProvider(ctx, cfg.GoogleAPI.ClientID, cfg.GoogleAPI.ClientSecret, cfg.GoogleAPI.RedirectURI); err != nil {
		logger.Error("Auth:SeedGoogleProvider:Error", "error", err)
	}
}

// GetService creates and returns an AuthService instance for use by other modules
func GetService(db database.Database, cache cache.Cache) service.AuthServiceInterface {
	repo := repository.NewAuthRepository(db)
	return service.NewAuthService(repo, cache)
}
