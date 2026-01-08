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

func ToUserGroupResponseWithRelations(relation *dto.UserGroupWithRelations) *dto.UserGroupResponse {
	if relation == nil {
		return nil
	}

	response := &dto.UserGroupResponse{
		ID:        relation.ID,
		UserID:    relation.UserID,
		GroupID:   relation.GroupID,
		CreatedAt: relation.CreatedAt,
	}

	// Sử dụng UserIDFromUser (social_logins.id) nếu có, nếu không thì dùng UserID (user_groups.user_id)
	userID := relation.UserID
	if relation.UserIDFromUser != uuid.Nil {
		userID = relation.UserIDFromUser
	}

	// Luôn tạo UserInfo với thông tin từ social_logins
	// COALESCE trong query đảm bảo ProviderName và ProviderEmail không null (empty string nếu không có)
	response.User = &dto.UserInfo{
		ID:            userID,
		ProviderName:  relation.ProviderName,  // Đã được COALESCE thành empty string nếu null
		ProviderEmail: relation.ProviderEmail, // Đã được COALESCE thành empty string nếu null
	}

	// Thêm Group info nếu có
	if relation.GroupIDFromGroup != uuid.Nil {
		response.Group = &dto.GroupInfo{
			ID:          relation.GroupIDFromGroup,
			Name:        relation.GroupName,
			Description: relation.GroupDescription,
		}
	}

	return response
}

