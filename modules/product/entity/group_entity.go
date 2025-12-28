package entity

import (
	"go-api-starter/core/entity"
)

type Group struct {
	Name string `db:"name"`

	Description string `db:"description"`

	entity.BaseEntity
}

type PaginatedGroupResponse = entity.Pagination[Group]


