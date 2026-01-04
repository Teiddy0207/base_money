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

func (s *ProductService) PrivateCreateGroup(ctx context.Context, req *dto.GroupRequest) (*dto.GroupResponse, *errors.AppError) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	group := mapper.ToGroupEntity(req)

	created, err := s.repo.PrivateCreateGroup(ctx, group)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrCreateFailed, "create group failed", err)
	}
	return mapper.ToGroupResponse(created), nil
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

	logger.Info("ProductService:PrivateGetGroups:Request", "page_number", params.PageNumber, "page_size", params.PageSize, "search", params.Search)
	groups, err := s.repo.PrivateGetGroups(ctx, params)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrGetFailed, "get groups failed", err)
	}
	logger.Info("ProductService:PrivateGetGroups:Result", "total_items", groups.TotalItems)
	return mapper.ToGroupPaginationResponse(groups), nil
}

func (s *ProductService) PrivateGetGroupsWhereMember(ctx context.Context, memberID uuid.UUID, params params.QueryParams) (*dto.PaginatedGroupResponse, *errors.AppError) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()
	logger.Info("ProductService:PrivateGetGroupsWhereMember:Request", "member_id", memberID, "page_number", params.PageNumber, "page_size", params.PageSize, "search", params.Search)
	groups, err := s.repo.PrivateGetGroupsWhereMember(ctx, memberID, params)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrGetFailed, "get groups failed", err)
	}
	logger.Info("ProductService:PrivateGetGroupsWhereMember:Result", "total_items", groups.TotalItems)
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

// UserGroup service methods - Quản lý user trong group

func (s *ProductService) PrivateAddUsersToGroup(ctx context.Context, req *dto.AddUsersToGroupRequest) *errors.AppError {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	// Kiểm tra group có tồn tại không
	_, err := s.repo.PrivateGetGroupById(ctx, req.GroupID)
	if err != nil {
		return errors.NewAppError(errors.ErrNotFound, "group not found", err)
	}

	err = s.repo.PrivateAddUsersToGroup(ctx, req.GroupID, req.UserIDs)
	if err != nil {
		return errors.NewAppError(errors.ErrCreateFailed, "add users to group failed", err)
	}

	return nil
}

func (s *ProductService) PrivateRemoveUserFromGroup(ctx context.Context, req *dto.RemoveUserFromGroupRequest) *errors.AppError {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	err := s.repo.PrivateRemoveUserFromGroup(ctx, req.GroupID, req.UserID)
	if err != nil {
		return errors.NewAppError(errors.ErrDeleteFailed, "remove user from group failed", err)
	}

	return nil
}

func (s *ProductService) PrivateGetUsersByGroupId(ctx context.Context, groupID uuid.UUID) (*dto.GroupUsersResponse, *errors.AppError) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	userGroupsWithRelations, group, err := s.repo.PrivateGetUsersByGroupIdWithRelations(ctx, groupID)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrGetFailed, "get users by group id failed", err)
	}

	// Convert to DTO với dữ liệu quan hệ
	userResponses := make([]dto.UserGroupResponse, len(userGroupsWithRelations))
	for i, relation := range userGroupsWithRelations {
		userResponses[i] = *mapper.ToUserGroupResponseWithRelations(&relation)
	}

	// Map Group info
	var groupInfo *dto.GroupInfo
	if group != nil {
		groupInfo = &dto.GroupInfo{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
		}
	}

	return &dto.GroupUsersResponse{
		GroupID: groupID,
		Group:   groupInfo,
		Users:   userResponses,
	}, nil
}

func (s *ProductService) PrivateGetGroupsByUserId(ctx context.Context, userID uuid.UUID) ([]dto.UserGroupResponse, *errors.AppError) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	userGroups, err := s.repo.PrivateGetGroupsByUserId(ctx, userID)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrGetFailed, "get groups by user id failed", err)
	}

	// Convert entity to DTO
	groupResponses := make([]dto.UserGroupResponse, len(userGroups))
	for i, ug := range userGroups {
		groupResponses[i] = *mapper.ToUserGroupResponse(&ug)
	}

	return groupResponses, nil
}

func (s *ProductService) PrivateAreUsersInSameGroup(ctx context.Context, userA uuid.UUID, userB uuid.UUID) (bool, *errors.AppError) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()
	exists, err := s.repo.PrivateAreUsersInSameGroup(ctx, userA, userB)
	if err != nil {
		return false, errors.NewAppError(errors.ErrGetFailed, "check same group failed", err)
	}
	return exists, nil
}
