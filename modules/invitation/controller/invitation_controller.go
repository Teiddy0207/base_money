package controller

import (
	"go-api-starter/core/controller"
	"go-api-starter/core/errors"
	"go-api-starter/core/logger"
	"go-api-starter/core/utils"
	"go-api-starter/modules/invitation/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type InvitationController struct {
	controller.BaseController
	service *service.InvitationService
}

func NewInvitationController(service *service.InvitationService) *InvitationController {
	return &InvitationController{
		BaseController: controller.NewBaseController(),
		service:        service,
	}
}

// GetUserIDFromContext retrieves user ID from context
func (c *InvitationController) GetUserIDFromContext(ctx echo.Context) (uuid.UUID, error) {
	// Get token data from middleware
	tokenData := ctx.Get("token_data")
	if tokenData == nil {
		return uuid.Nil, errors.NewAppError(errors.ErrUnauthorized, "Token data not found in context", nil)
	}

	// Cast to TokenClaims
	claims, ok := tokenData.(*utils.TokenClaims)
	if !ok {
		return uuid.Nil, errors.NewAppError(errors.ErrUnauthorized, "Invalid token data format", nil)
	}

	return claims.UserID, nil
}

// GetPendingInvitations returns all pending invitations for the current user
func (c *InvitationController) GetPendingInvitations(ctx echo.Context) error {
	logger.Info("InvitationController:GetPendingInvitations:Start")

	userID, err := c.GetUserIDFromContext(ctx)
	if err != nil {
		logger.Error("InvitationController:GetPendingInvitations:Unauthorized", "error", err)
		return c.BadRequest(errors.ErrUnauthorized, "Unauthorized", nil)
	}

	logger.Info("InvitationController:GetPendingInvitations:CallingService", "user_id", userID)
	if c.service == nil {
		logger.Error("InvitationController:GetPendingInvitations:ServiceIsNil")
		return c.InternalServerError(errors.ErrInternalServer, "Service configuration error", nil)
	}

	response, err := c.service.GetPendingInvitations(ctx.Request().Context(), userID)
	if err != nil {
		logger.Error("InvitationController:GetPendingInvitations:ServiceError", "error", err)
		return c.InternalServerError(errors.ErrInternalServer, err.Error(), nil)
	}

	return c.SuccessResponse(ctx, response, "Pending invitations retrieved")
}

// AcceptInvitation accepts an invitation
func (c *InvitationController) AcceptInvitation(ctx echo.Context) error {
	userID, err := c.GetUserIDFromContext(ctx)
	if err != nil {
		return c.BadRequest(errors.ErrUnauthorized, "Unauthorized", nil)
	}

	idStr := ctx.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.BadRequest(errors.ErrInvalidRequestData, "Invalid invitation ID", nil)
	}

	resp, err := c.service.AcceptInvitation(ctx.Request().Context(), id, userID)
	if err != nil {
		return c.InternalServerError(errors.ErrInternalServer, err.Error(), nil)
	}

	return c.SuccessResponse(ctx, resp, "Invitation accepted")
}

// DeclineInvitation declines an invitation
func (c *InvitationController) DeclineInvitation(ctx echo.Context) error {
	userID, err := c.GetUserIDFromContext(ctx)
	if err != nil {
		return c.BadRequest(errors.ErrUnauthorized, "Unauthorized", nil)
	}

	idStr := ctx.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.BadRequest(errors.ErrInvalidRequestData, "Invalid invitation ID", nil)
	}

	if err := c.service.DeclineInvitation(ctx.Request().Context(), id, userID); err != nil {
		return c.InternalServerError(errors.ErrInternalServer, err.Error(), nil)
	}

	return c.SuccessResponse(ctx, nil, "Invitation declined")
}

// CountPending counts pending invitations
func (c *InvitationController) CountPending(ctx echo.Context) error {
	userID, err := c.GetUserIDFromContext(ctx)
	if err != nil {
		return c.BadRequest(errors.ErrUnauthorized, "Unauthorized", nil)
	}

	count, err := c.service.CountPending(ctx.Request().Context(), userID)
	if err != nil {
		return c.InternalServerError(errors.ErrInternalServer, err.Error(), nil)
	}

	return c.SuccessResponse(ctx, map[string]int{"count": count}, "Pending count retrieved")
}
