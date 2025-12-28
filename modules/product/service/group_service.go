package service

import (
	"context"
	"database/sql"
	"go-api-starter/core/constants"
	"go-api-starter/core/errors"
	"go-api-starter/core/logger"
	"go-api-starter/core/params"
	"go-api-starter/modules/product/dto"
	"go-api-starter/modules/product/mapper"

	"github.com/google/uuid"
)

func (s *ProductService) PrivateCreateGroup(ctx context.Context, req *dto.GroupRequest) *errors.AppError {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	group := mapper.ToGroupEntity(req)

	err := s.repo.PrivateCreateGroup(ctx, group)
	if err != nil {
		return errors.NewAppError(errors.ErrCreateFailed, "create group failed", err)
	}
	return nil
}

func (s *ProductService) PrivateGetGroupById(ctx context.Context, id uuid.UUID) (*dto.GroupResponse, *errors.AppError) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	group, err := s.repo.PrivateGetGroupById(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Error("ProductService:PrivateGetGroupById:GroupNotFound: ", err)
			return nil, errors.NewAppError(errors.ErrNotFound, "group not found", err)
		}
		return nil, errors.NewAppError(errors.ErrGetFailed, "get group failed", err)
	}
	if group == nil {
		return nil, errors.NewAppError(errors.ErrNotFound, "group not found", nil)
	}
	return mapper.ToGroupResponse(group), nil
}

func (s *ProductService) PrivateGetGroups(ctx context.Context, params params.QueryParams) (*dto.PaginatedGroupResponse, *errors.AppError) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	groups, err := s.repo.PrivateGetGroups(ctx, params)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrGetFailed, "get groups failed", err)
	}
	return mapper.ToGroupPaginationResponse(groups), nil
}

func (s *ProductService) PrivateUpdateGroup(ctx context.Context, req *dto.GroupRequest, id uuid.UUID) *errors.AppError {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	group := mapper.ToGroupEntity(req)

	err := s.repo.PrivateUpdateGroup(ctx, group, id)
	if err != nil {
		return errors.NewAppError(errors.ErrUpdateFailed, "update group failed", err)
	}

	return nil
}

func (s *ProductService) PrivateDeleteGroup(ctx context.Context, id uuid.UUID) *errors.AppError {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	err := s.repo.PrivateDeleteGroup(ctx, id)
	if err != nil {
		return errors.NewAppError(errors.ErrDeleteFailed, "delete group failed", err)
	}

	return nil
}

func (s *ProductService) PublicGetGroups(ctx context.Context, params params.QueryParams) (*dto.PaginatedGroupResponse, *errors.AppError) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	groups, err := s.repo.PrivateGetGroups(ctx, params)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrGetFailed, "get groups failed", err)
	}
	
	if groups == nil {
		return &dto.PaginatedGroupResponse{
			Items:      []dto.GroupResponse{},
			TotalItems: 0,
			TotalPages: 0,
			PageNumber: params.PageNumber,
			PageSize:   params.PageSize,
		}, nil
	}
	
	return mapper.ToGroupPaginationResponse(groups), nil
}

func (s *ProductService) PublicGetGroupById(ctx context.Context, id uuid.UUID) (*dto.GroupResponse, *errors.AppError) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	group, err := s.repo.PrivateGetGroupById(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Error("ProductService:PublicGetGroupById:GroupNotFound: ", err)
			return nil, errors.NewAppError(errors.ErrNotFound, "group not found", err)
		}
		return nil, errors.NewAppError(errors.ErrGetFailed, "get group failed", err)
	}
	if group == nil {
		return nil, errors.NewAppError(errors.ErrNotFound, "group not found", nil)
	}
	return mapper.ToGroupResponse(group), nil
}



