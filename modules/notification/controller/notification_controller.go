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
