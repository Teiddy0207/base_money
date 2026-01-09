package controller

import (
	coreController "go-api-starter/core/controller"
	"net/http"

	"github.com/labstack/echo/v4"
)

// SearchSocialUsers searches for users from social_logins table
// @Summary Tìm kiếm người dùng
// @Description Tìm kiếm người dùng theo từ khóa (tên, email)
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param q query string true "Từ khóa tìm kiếm"
// @Success 200 {array} repository.SocialUserResult
// @Failure 401 {object} errors.AppError
// @Failure 500 {object} errors.AppError
// @Router /private/auth/users/search [get]
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
// @Summary Lấy tất cả người dùng
// @Description Lấy danh sách tất cả người dùng đã đăng ký qua social login
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {array} repository.SocialUserResult
// @Failure 401 {object} errors.AppError
// @Failure 500 {object} errors.AppError
// @Router /private/auth/users/social [get]
func (a *AuthController) GetAllSocialUsers(c echo.Context) error {
	users, err := a.AuthService.GetAllSocialUsers(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, coreController.NewErrorResponse(
			http.StatusInternalServerError, 5000, "Failed to get users"))
	}

	return c.JSON(http.StatusOK, coreController.NewSuccessResponse(http.StatusOK, users, "Success"))
}
