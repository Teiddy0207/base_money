package mapper

import (
	"go-api-starter/modules/product/dto"
	"go-api-starter/modules/product/entity"

	"github.com/google/uuid"
)

func ToUserGroupEntity(userID uuid.UUID, groupID uuid.UUID) *entity.UserGroup {
	return &entity.UserGroup{
		UserID:  userID,
		GroupID: groupID,
	}
}

func ToUserGroupResponse(entity *entity.UserGroup) *dto.UserGroupResponse {
	if entity == nil {
		return nil
	}

	return &dto.UserGroupResponse{
		ID:        entity.ID,
		UserID:    entity.UserID,
		GroupID:   entity.GroupID,
		CreatedAt: entity.CreatedAt,
	}
}

