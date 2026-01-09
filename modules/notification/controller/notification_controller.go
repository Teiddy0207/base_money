package controller

import (
	"go-api-starter/core/controller"
	"go-api-starter/core/errors"
	"go-api-starter/core/params"
	"go-api-starter/core/utils"
	"go-api-starter/modules/notification/dto"
	"go-api-starter/modules/notification/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type NotificationController struct {
	service *service.NotificationService
	controller.BaseController
}

func NewNotificationController(service *service.NotificationService) *NotificationController {
	return &NotificationController{
		service:        service,
		BaseController: controller.NewBaseController(),
	}
}

// GetMyNotifications retrieves user's notifications
// @Summary Lấy danh sách thông báo
// @Description Trả về danh sách thông báo của người dùng hiện tại
// @Tags Notification
// @Security BearerAuth
// @Produce json
// @Param page query int false "Số trang"
// @Param limit query int false "Số lượng mỗi trang"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} errors.AppError
// @Router /private/notifications [get]
func (c *NotificationController) GetMyNotifications(ctx echo.Context) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return c.Unauthorized(errors.ErrUnauthorized, "Unauthorized", nil)
	}

	queryParams := params.NewQueryParams(ctx)
	result, getErr := c.service.GetMyNotifications(ctx.Request().Context(), userID, *queryParams)
	if getErr != nil {
		return c.InternalServerError(errors.ErrInternalServer, "Failed to get notifications", getErr)
	}

	return c.SuccessResponse(ctx, result, "Notifications retrieved successfully")
}

// MarkAsRead marks specific notifications as read
// @Summary Đánh dấu đã đọc
// @Description Đánh dấu các thông báo cụ thể là đã đọc
// @Tags Notification
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.MarkAsReadRequest true "Danh sách ID thông báo"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errors.AppError
// @Failure 401 {object} errors.AppError
// @Router /private/notifications/mark-read [put]
func (c *NotificationController) MarkAsRead(ctx echo.Context) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return c.Unauthorized(errors.ErrUnauthorized, "Unauthorized", nil)
	}

	req := new(dto.MarkAsReadRequest)
	if err := ctx.Bind(req); err != nil {
		return c.BadRequest(errors.ErrInvalidRequestData, "Invalid request body", nil)
	}

	if err := c.service.MarkAsRead(ctx.Request().Context(), userID, req.IDs); err != nil {
		return c.InternalServerError(errors.ErrInternalServer, "Failed to mark as read", err)
	}

	return c.SuccessResponse(ctx, nil, "Marked as read successfully")
}

// MarkAllAsRead marks all notifications as read
// @Summary Đánh dấu tất cả đã đọc
// @Description Đánh dấu tất cả thông báo của người dùng là đã đọc
// @Tags Notification
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 401 {object} errors.AppError
// @Router /private/notifications/mark-all-read [put]
func (c *NotificationController) MarkAllAsRead(ctx echo.Context) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return c.Unauthorized(errors.ErrUnauthorized, "Unauthorized", nil)
	}

	if err := c.service.MarkAllAsRead(ctx.Request().Context(), userID); err != nil {
		return c.InternalServerError(errors.ErrInternalServer, "Failed to mark all as read", err)
	}

	return c.SuccessResponse(ctx, nil, "Marked all as read successfully")
}

// CountUnread counts unread notifications
// @Summary Đếm thông báo chưa đọc
// @Description Trả về số lượng thông báo chưa đọc
// @Tags Notification
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]int
// @Failure 401 {object} errors.AppError
// @Router /private/notifications/unread-count [get]
func (c *NotificationController) CountUnread(ctx echo.Context) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return c.Unauthorized(errors.ErrUnauthorized, "Unauthorized", nil)
	}

	count, err := c.service.CountUnread(ctx.Request().Context(), userID)
	if err != nil {
		return c.InternalServerError(errors.ErrInternalServer, "Failed to count unread", err)
	}

	return c.SuccessResponse(ctx, map[string]int{"count": count}, "Unread count retrieved")
}

// Helper function to get user ID from JWT context
func getUserIDFromContext(ctx echo.Context) (uuid.UUID, error) {
	token := ctx.Request().Header.Get("Authorization")
	if token == "" {
		return uuid.Nil, errors.NewAppError(errors.ErrUnauthorized, "No token provided", nil)
	}

	// Remove "Bearer " prefix
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	tokenData, err := utils.ValidateAndParseToken(token)
	if err != nil {
		return uuid.Nil, err
	}

	return tokenData.UserID, nil
}
