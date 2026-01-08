package controller

import (
	"go-api-starter/core/errors"
	"go-api-starter/core/logger"
	"go-api-starter/core/params"
	"go-api-starter/core/utils"
	"go-api-starter/modules/product/dto"
	"go-api-starter/modules/product/validator"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (controller *ProductController) PrivateCreateGroup(c echo.Context) error {
	ctx := c.Request().Context()

	requestData := new(dto.GroupRequest)
	if err := c.Bind(requestData); err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid request data", nil)
	}

	validationResult := validator.ValidateGroupRequest(requestData)
	if validationResult.HasError() {
		return controller.BadRequest(errors.ErrInvalidInput, "Invalid request data", validationResult)
	}

	created, appErr := controller.ProductService.PrivateCreateGroup(ctx, requestData)
	if appErr != nil {
		return controller.InternalServerError(errors.ErrInternalServer, "create group failed", appErr)
	}
	token, err := utils.GetTokenFromHeader(c)
	if err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid token", nil)
	}
	tokenData, err := utils.ValidateAndParseToken(token)
	if err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid token", nil)
	}
	sl, slErr := controller.AuthService.GetSocialLoginByUserAndProviderName(ctx, tokenData.UserID, "google")
	if slErr == nil && sl != nil {
		_ = controller.ProductService.PrivateAddUsersToGroup(ctx, &dto.AddUsersToGroupRequest{
			GroupID: created.ID,
			UserIDs: []uuid.UUID{sl.ID},
		})
	}
	// Wrap response trong data.group theo yêu cầu frontend
	responseData := map[string]interface{}{
		"group": created,
	}
	return controller.SuccessResponse(c, responseData, "create group success")
}

func (controller *ProductController) PrivateGetGroupById(c echo.Context) error {
	ctx := c.Request().Context()

	groupId := utils.ToUUID(c.Param("id"))

	group, errGet := controller.ProductService.PrivateGetGroupById(ctx, groupId)
	if errGet != nil {
		return controller.ErrorResponse(c, errGet)
	}

	return controller.SuccessResponse(c, group, "get group success")
}

func (controller *ProductController) PrivateUpdateGroup(c echo.Context) error {
	ctx := c.Request().Context()

	groupId := utils.ToUUID(c.Param("id"))

	requestData := new(dto.GroupRequest)
	if err := c.Bind(requestData); err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid request data", nil)
	}

	validationResult := validator.ValidateGroupRequest(requestData)
	if validationResult.HasError() {
		return controller.BadRequest(errors.ErrInvalidInput, "Invalid request data", validationResult)
	}

	errUpdate := controller.ProductService.PrivateUpdateGroup(ctx, requestData, groupId)
	if errUpdate != nil {
		return controller.ErrorResponse(c, errUpdate)
	}

	return controller.SuccessResponse(c, nil, "update group success")
}

func (controller *ProductController) PrivateDeleteGroup(c echo.Context) error {
	ctx := c.Request().Context()

	groupId := utils.ToUUID(c.Param("id"))
	errDelete := controller.ProductService.PrivateDeleteGroup(ctx, groupId)
	if errDelete != nil {
		return controller.ErrorResponse(c, errDelete)
	}

	return controller.SuccessResponse(c, nil, "delete group success")
}

func (controller *ProductController) PrivateGetGroups(c echo.Context) error {
	ctx := c.Request().Context()

	queryParams := params.NewQueryParams(c)

	token, err := utils.GetTokenFromHeader(c)
	if err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid token", nil)
	}
	tokenData, err := utils.ValidateAndParseToken(token)
	if err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid token", nil)
	}
	sl, appErrSL := controller.AuthService.GetSocialLoginByUserAndProviderName(ctx, tokenData.UserID, "google")
	if appErrSL != nil || sl == nil {
		return controller.Forbidden(errors.ErrForbidden, "forbidden", nil)
	}
	logger.Info("ProductController:PrivateGetGroups:Request", "user_id", tokenData.UserID, "social_login_id", sl.ID, "page_number", queryParams.PageNumber, "page_size", queryParams.PageSize, "search", queryParams.Search)
	groups, appErr := controller.ProductService.PrivateGetGroupsWhereMember(ctx, sl.ID, *queryParams)
	if appErr != nil {
		logger.Error("ProductController:PrivateGetGroups:ServiceError", "error", appErr)
		return controller.ErrorResponse(c, appErr)
	}
	logger.Info("ProductController:PrivateGetGroups:Result", "total_items", groups.TotalItems, "items_count", len(groups.Items))
	return controller.SuccessResponse(c, groups, "get groups success")
}

func (controller *ProductController) PublicGetGroupById(c echo.Context) error {
	ctx := c.Request().Context()

	groupId := utils.ToUUID(c.Param("id"))

	group, errGet := controller.ProductService.PublicGetGroupById(ctx, groupId)
	if errGet != nil {
		return controller.ErrorResponse(c, errGet)
	}

	return controller.SuccessResponse(c, group, "get group success")
}

// UserGroup controller methods - Quản lý user trong group

func (controller *ProductController) PrivateAddUsersToGroup(c echo.Context) error {
	ctx := c.Request().Context()

	requestData := new(dto.AddUsersToGroupRequest)
	if err := c.Bind(requestData); err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid request data", nil)
	}

	validationResult := validator.ValidateAddUsersToGroupRequest(requestData)
	if validationResult.HasError() {
		return controller.BadRequest(errors.ErrInvalidInput, "Invalid request data", validationResult)
	}

	// Convert user_ids từ users.id sang social_logins.id nếu cần
	// Đảm bảo user_ids luôn là social_logins.id để query groups đúng
	convertedUserIDs := make([]uuid.UUID, 0, len(requestData.UserIDs))
	for _, userID := range requestData.UserIDs {
		// Thử lấy social_login bằng ID trực tiếp (nếu userID đã là social_logins.id)
		sl, err := controller.AuthService.GetSocialLoginByID(ctx, userID)
		if err == nil && sl != nil {
			// userID đã là social_logins.id
			convertedUserIDs = append(convertedUserIDs, userID)
		} else {
			// userID có thể là users.id, thử convert sang social_logins.id
			sl, err := controller.AuthService.GetSocialLoginByUserAndProviderName(ctx, userID, "google")
			if err == nil && sl != nil {
				convertedUserIDs = append(convertedUserIDs, sl.ID)
			} else {
				// Nếu không tìm thấy, giữ nguyên (có thể là social_logins.id nhưng không active)
				convertedUserIDs = append(convertedUserIDs, userID)
			}
		}
	}

	// Cập nhật requestData với user_ids đã convert
	requestData.UserIDs = convertedUserIDs

	err := controller.ProductService.PrivateAddUsersToGroup(ctx, requestData)
	if err != nil {
		return controller.ErrorResponse(c, err)
	}

	return controller.SuccessResponse(c, nil, "add users to group success")
}

func (controller *ProductController) PrivateRemoveUserFromGroup(c echo.Context) error {
	ctx := c.Request().Context()

	requestData := new(dto.RemoveUserFromGroupRequest)
	if err := c.Bind(requestData); err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid request data", nil)
	}

	validationResult := validator.ValidateRemoveUserFromGroupRequest(requestData)
	if validationResult.HasError() {
		return controller.BadRequest(errors.ErrInvalidInput, "Invalid request data", validationResult)
	}

	err := controller.ProductService.PrivateRemoveUserFromGroup(ctx, requestData)
	if err != nil {
		return controller.ErrorResponse(c, err)
	}

	return controller.SuccessResponse(c, nil, "remove user from group success")
}

func (controller *ProductController) PrivateGetUsersByGroupId(c echo.Context) error {
	ctx := c.Request().Context()

	groupId := utils.ToUUID(c.Param("id"))

	response, err := controller.ProductService.PrivateGetUsersByGroupId(ctx, groupId)
	if err != nil {
		return controller.ErrorResponse(c, err)
	}

	return controller.SuccessResponse(c, response, "get users by group id success")
}

func (controller *ProductController) PrivateGetGroupsByUserId(c echo.Context) error {
	ctx := c.Request().Context()

	userId := utils.ToUUID(c.Param("id"))

	response, err := controller.ProductService.PrivateGetGroupsByUserId(ctx, userId)
	if err != nil {
		return controller.ErrorResponse(c, err)
	}

	return controller.SuccessResponse(c, response, "get groups by user id success")
}
