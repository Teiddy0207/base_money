package mapper

import (
	"go-api-starter/modules/product/dto"
	"go-api-starter/modules/product/entity"
)

func ToGroupEntity(req *dto.GroupRequest) *entity.Group {
	return &entity.Group{
		Name:        req.Name,
		Description: req.Description,
	}
}

func ToGroupResponse(entity *entity.Group) *dto.GroupResponse {
	response := &dto.GroupResponse{
		ID:          entity.ID,
		Name:        entity.Name,
		Description: entity.Description,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}

	return response
}

func ToGroupPaginationResponse(entity *entity.PaginatedGroupResponse) *dto.PaginatedGroupResponse {
	if entity == nil {
		return &dto.PaginatedGroupResponse{
			Items:      []dto.GroupResponse{},
			TotalItems: 0,
			TotalPages: 0,
			PageNumber: 0,
			PageSize:   0,
		}
	}
	// Convert từng group entity sang group response
	groupResponses := make([]dto.GroupResponse, len(entity.Items))
	for i, group := range entity.Items {
		groupResponses[i] = *ToGroupResponse(&group)
	}

	// Tính total pages
	totalPages := 0
	if entity.PageSize > 0 {
		totalPages = (entity.TotalItems + entity.PageSize - 1) / entity.PageSize
	}

	return &dto.PaginatedGroupResponse{
		Items:      groupResponses,
		TotalItems: entity.TotalItems,
		TotalPages: totalPages,
		PageNumber: entity.PageNumber,
		PageSize:   entity.PageSize,
	}
}


