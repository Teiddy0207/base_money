package controller

import (
	"go-api-starter/core/errors"
	"go-api-starter/core/params"
	"go-api-starter/core/utils"
	"go-api-starter/modules/product/dto"
	"go-api-starter/modules/product/validator"

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

	err := controller.ProductService.PrivateCreateGroup(ctx, requestData)
	if err != nil {
		return controller.InternalServerError(errors.ErrInternalServer, "create group failed", err)
	}

	return controller.SuccessResponse(c, nil, "create group success")
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

	groups, err := controller.ProductService.PrivateGetGroups(ctx, *queryParams)
	if err != nil {
		return controller.ErrorResponse(c, err)
	}

	return controller.SuccessResponse(c, groups, "get groups success")
}

func (controller *ProductController) PublicGetGroups(c echo.Context) error {
	ctx := c.Request().Context()

	queryParams := params.NewQueryParams(c)

	groups, err := controller.ProductService.PublicGetGroups(ctx, *queryParams)
	if err != nil {
		return controller.ErrorResponse(c, err)
	}

	return controller.SuccessResponse(c, groups, "get groups success")
}


