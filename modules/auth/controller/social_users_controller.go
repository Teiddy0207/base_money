package controller

import (
	coreController "go-api-starter/core/controller"
	"net/http"

	"github.com/labstack/echo/v4"
)

// SearchSocialUsers searches for users from social_logins table
// GET /api/v1/private/auth/users/search?q=keyword
func (a *AuthController) SearchSocialUsers(c echo.Context) error {
	query := c.QueryParam("q")

	users, err := a.AuthService.SearchSocialUsers(c.Request().Context(), query)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, coreController.NewErrorResponse(
			http.StatusInternalServerError, 5000, "Failed to search users"))
	}

	return c.JSON(http.StatusOK, coreController.NewSuccessResponse(http.StatusOK, users, "Success"))
}

// GetAllSocialUsers gets all users from social_logins table
// GET /api/v1/private/auth/users/social
func (a *AuthController) GetAllSocialUsers(c echo.Context) error {
	users, err := a.AuthService.GetAllSocialUsers(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, coreController.NewErrorResponse(
			http.StatusInternalServerError, 5000, "Failed to get users"))
	}

	return c.JSON(http.StatusOK, coreController.NewSuccessResponse(http.StatusOK, users, "Success"))
}
