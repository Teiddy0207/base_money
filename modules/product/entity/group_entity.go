package entity

import (
	"go-api-starter/core/entity"
)

// Group represents a product group entity
// Đại diện cho nhóm sản phẩm
type Group struct {
	// Name is the display name of the group
	// Tên hiển thị của nhóm
	Name string `db:"name"`

	// Slug is the URL-friendly version of the group name
	// Slug là phiên bản thân thiện URL của tên nhóm
	Slug string `db:"slug"`

	// Description provides detailed information about the group
	// Mô tả cung cấp thông tin chi tiết về nhóm
	Description string `db:"description"`

	// Thumbnail is the group's thumbnail image URL
	// Thumbnail là URL hình ảnh thu nhỏ của nhóm
	Thumbnail string `db:"thumbnail"`

	// SortOrder determines the display order in lists
	// Thứ tự sắp xếp xác định thứ tự hiển thị trong danh sách
	SortOrder int `db:"sort_order"`

	// IsActive indicates whether this group is currently active
	// Cho biết nhóm này có đang hoạt động không
	IsActive bool `db:"is_active"`

	entity.BaseEntity
}

type PaginatedGroupResponse = entity.Pagination[Group]


