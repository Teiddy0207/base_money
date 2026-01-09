package controller

import (
	"go-api-starter/core/constants"
	"go-api-starter/core/controller"
	"go-api-starter/core/errors"
	"go-api-starter/core/utils"
	"go-api-starter/modules/meeting/dto"
	"go-api-starter/modules/meeting/service"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// MeetingController handles event HTTP requests
type MeetingController struct {
	controller.BaseController
	MeetingService service.MeetingServiceInterface
}

// NewMeetingController creates a new controller
func NewMeetingController(svc service.MeetingServiceInterface) *MeetingController {
	return &MeetingController{
		BaseController: controller.NewBaseController(),
		MeetingService: svc,
	}
}

// getUserIDFromContext extracts user ID from JWT context
func (c *MeetingController) getUserIDFromContext(ctx echo.Context) (uuid.UUID, error) {
	tokenData := ctx.Get(constants.ContextTokenData)
	if tokenData == nil {
		return uuid.Nil, errors.NewAppError(errors.ErrUnauthorized, "User not authenticated", nil)
	}

	claims, ok := tokenData.(*utils.TokenClaims)
	if !ok {
		return uuid.Nil, errors.NewAppError(errors.ErrUnauthorized, "Invalid token data", nil)
	}

	return claims.UserID, nil
}

// CreateEvent handles POST /events
// @Summary Tạo sự kiện hẹn
// @Description Tạo một sự kiện hẹn mới với người tham gia
// @Tags Meeting
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.CreateEventRequest true "Thông tin sự kiện"
// @Success 200 {object} dto.EventResponse
// @Failure 400 {object} errors.AppError
// @Failure 401 {object} errors.AppError
// @Router /private/meetings [post]
func (c *MeetingController) CreateEvent(ctx echo.Context) error {
	hostID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		return c.Unauthorized(errors.ErrUnauthorized, "User not authenticated")
	}

	var req dto.CreateEventRequest
	if err := ctx.Bind(&req); err != nil {
		return c.BadRequest(errors.ErrInvalidInput, "Invalid request body")
	}

	result, appErr := c.MeetingService.CreateEvent(ctx.Request().Context(), hostID, &req)
	if appErr != nil {
		return c.InternalServerError(appErr.Code, appErr.Message)
	}

	return c.SuccessResponse(ctx, result, "Event created successfully")
}

// GetEvent handles GET /events/:id
// @Summary Lấy thông tin sự kiện
// @Description Lấy chi tiết một sự kiện hẹn theo ID
// @Tags Meeting
// @Security BearerAuth
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} dto.EventResponse
// @Failure 404 {object} errors.AppError
// @Router /private/meetings/{id} [get]
func (c *MeetingController) GetEvent(ctx echo.Context) error {
	eventID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return c.BadRequest(errors.ErrInvalidInput, "Invalid event ID")
	}

	result, appErr := c.MeetingService.GetEventByID(ctx.Request().Context(), eventID)
	if appErr != nil {
		return c.NotFound(appErr.Code, appErr.Message)
	}

	return c.SuccessResponse(ctx, result, "Success")
}

// GetMyEvents handles GET /events
// @Summary Lấy danh sách sự kiện
// @Description Lấy danh sách sự kiện của người dùng
// @Tags Meeting
// @Security BearerAuth
// @Produce json
// @Success 200 {array} dto.EventResponse
// @Failure 401 {object} errors.AppError
// @Router /private/meetings [get]
func (c *MeetingController) GetMyEvents(ctx echo.Context) error {
	hostID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		return c.Unauthorized(errors.ErrUnauthorized, "User not authenticated")
	}

	result, appErr := c.MeetingService.GetMyEvents(ctx.Request().Context(), hostID)
	if appErr != nil {
		return c.InternalServerError(appErr.Code, appErr.Message)
	}

	return c.SuccessResponse(ctx, result, "Success")
}

// UpdateEvent handles PUT /events/:id
// @Summary Cập nhật sự kiện
// @Description Cập nhật thông tin sự kiện hẹn
// @Tags Meeting
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Event ID"
// @Param request body dto.UpdateEventRequest true "Thông tin cập nhật"
// @Success 200 {object} dto.EventResponse
// @Failure 400 {object} errors.AppError
// @Router /private/meetings/{id} [put]
func (c *MeetingController) UpdateEvent(ctx echo.Context) error {
	hostID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		return c.Unauthorized(errors.ErrUnauthorized, "User not authenticated")
	}

	eventID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return c.BadRequest(errors.ErrInvalidInput, "Invalid event ID")
	}

	var req dto.UpdateEventRequest
	if err := ctx.Bind(&req); err != nil {
		return c.BadRequest(errors.ErrInvalidInput, "Invalid request body")
	}

	result, appErr := c.MeetingService.UpdateEvent(ctx.Request().Context(), eventID, hostID, &req)
	if appErr != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": appErr.Message})
	}

	return c.SuccessResponse(ctx, result, "Event updated successfully")
}

// DeleteEvent handles DELETE /events/:id
// @Summary Xóa sự kiện
// @Description Xóa một sự kiện hẹn
// @Tags Meeting
// @Security BearerAuth
// @Param id path string true "Event ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errors.AppError
// @Router /private/meetings/{id} [delete]
func (c *MeetingController) DeleteEvent(ctx echo.Context) error {
	hostID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		return c.Unauthorized(errors.ErrUnauthorized, "User not authenticated")
	}

	eventID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return c.BadRequest(errors.ErrInvalidInput, "Invalid event ID")
	}

	appErr := c.MeetingService.DeleteEvent(ctx.Request().Context(), eventID, hostID)
	if appErr != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": appErr.Message})
	}

	return c.SuccessResponse(ctx, nil, "Event deleted successfully")
}

// FindSlots handles POST /events/:id/find-slots
// @Summary Tìm khung giờ rảnh
// @Description Tìm các khung giờ phù hợp cho sự kiện
// @Tags Meeting
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Event ID"
// @Param request body dto.FindSlotsRequest true "Tham số tìm kiếm"
// @Success 200 {object} dto.FindSlotsResponse
// @Failure 400 {object} errors.AppError
// @Router /private/meetings/{id}/find-slots [post]
func (c *MeetingController) FindSlots(ctx echo.Context) error {
	eventID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return c.BadRequest(errors.ErrInvalidInput, "Invalid event ID")
	}

	var req dto.FindSlotsRequest
	if err := ctx.Bind(&req); err != nil {
		req = dto.FindSlotsRequest{SearchDays: 7}
	}

	result, appErr := c.MeetingService.FindSlots(ctx.Request().Context(), eventID, &req)
	if appErr != nil {
		return c.InternalServerError(appErr.Code, appErr.Message)
	}

	return c.SuccessResponse(ctx, result, "Slots found")
}

// SelectSlot handles POST /events/:id/select-slot
// @Summary Chọn khung giờ
// @Description Chọn khung giờ cho sự kiện và tạo lịch
// @Tags Meeting
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Event ID"
// @Param request body dto.SelectSlotRequest true "Thông tin khung giờ"
// @Success 200 {object} dto.EventResponse
// @Failure 400 {object} errors.AppError
// @Router /private/meetings/{id}/select-slot [post]
func (c *MeetingController) SelectSlot(ctx echo.Context) error {
	hostID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		return c.Unauthorized(errors.ErrUnauthorized, "User not authenticated")
	}

	eventID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return c.BadRequest(errors.ErrInvalidInput, "Invalid event ID")
	}

	var req dto.SelectSlotRequest
	if err := ctx.Bind(&req); err != nil {
		return c.BadRequest(errors.ErrInvalidInput, "Invalid request body")
	}

	result, appErr := c.MeetingService.SelectSlot(ctx.Request().Context(), eventID, hostID, &req)
	if appErr != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": appErr.Message})
	}

	return c.SuccessResponse(ctx, result, "Slot selected successfully")
}
