package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"go-api-starter/core/logger"
	"go-api-starter/core/params"
	"go-api-starter/modules/product/entity"

	"github.com/google/uuid"
)

func (r *ProductRepository) PrivateCreateGroup(ctx context.Context, group *entity.Group) error {
	query := `
		INSERT INTO groups (name, slug, description, thumbnail, sort_order, is_active)
		VALUES (:name, :slug, :description, :thumbnail, :sort_order, :is_active)
	`
	_, err := r.DB.NamedExecContext(ctx, query, group)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		logger.Error("ProductRepository:PrivateCreateGroup", err)
		return err
	}
	return nil
}

func (r *ProductRepository) PrivateUpdateGroup(ctx context.Context, group *entity.Group, id uuid.UUID) error {
	query := `
		UPDATE groups
		SET name = $1, slug = $2, description = $3, thumbnail = $4, 
		    sort_order = $5, is_active = $6, updated_at = now()
		WHERE id = $7
	`

	result, err := r.DB.SQLx().ExecContext(ctx, query,
		group.Name,
		group.Slug,
		group.Description,
		group.Thumbnail,
		group.SortOrder,
		group.IsActive,
		id,
	)

	if err != nil {
		logger.Error("ProductRepository:PrivateUpdateGroup", err)
		return err
	}

	// Kiểm tra xem có record nào được update không
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logger.Error("ProductRepository:PrivateUpdateGroup - RowsAffected", err)
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("group with id %s not found", id)
	}

	return nil
}

func (r *ProductRepository) PrivateDeleteGroup(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM groups
		WHERE id = :id
	`
	_, err := r.DB.NamedExecContext(ctx, query, map[string]any{"id": id})
	if err != nil {
		logger.Error("ProductRepository:PrivateDeleteGroup", err)
		return err
	}
	return nil
}

func (r *ProductRepository) PrivateGetGroupById(ctx context.Context, id uuid.UUID) (*entity.Group, error) {
	var group entity.Group
	query := `
		SELECT 
			id, 
			name, 
			slug, 
			description, 
			thumbnail, 
			sort_order, 
			is_active, 
			created_at, 
			updated_at
		FROM groups
		WHERE id = $1
	`
	err := r.DB.GetContext(ctx, &group, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		logger.Error("ProductRepository:PrivateGetGroupById", err)
		return nil, err
	}
	return &group, nil
}

func (r *ProductRepository) PrivateGetGroups(ctx context.Context, params params.QueryParams) (*entity.PaginatedGroupResponse, error) {
	// Tính offset cho pagination
	offset := (params.PageNumber - 1) * params.PageSize

	// Base query để lấy groups
	baseQuery := `FROM groups`

	// Thêm điều kiện search nếu có
	var whereClause string
	var args []interface{}

	conditions := []string{}
	argIndex := 1

	if params.Search != "" {
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argIndex))
		args = append(args, "%"+params.Search+"%")
		argIndex++
	}

	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Query để đếm tổng số records
	countQuery := "SELECT COUNT(*) " + baseQuery + whereClause

	var totalItems int
	err := r.DB.GetContext(ctx, &totalItems, countQuery, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		logger.Error("ProductRepository:PrivateGetGroups - Count", err)
		return nil, err
	}

	// Query để lấy data với pagination
	dataQuery := `
		SELECT 
			id, 
			name, 
			slug, 
			description, 
			thumbnail, 
			sort_order, 
			is_active, 
			created_at, 
			updated_at
	` + baseQuery + whereClause + `
		ORDER BY sort_order ASC, created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argIndex) + ` OFFSET $` + fmt.Sprintf("%d", argIndex+1)

	// Thêm params cho pagination
	args = append(args, params.PageSize, offset)

	var groups []entity.Group
	err = r.DB.SelectContext(ctx, &groups, dataQuery, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		logger.Error("ProductRepository:PrivateGetGroups - Select", err)
		return nil, err
	}

	// Tạo response pagination
	response := &entity.PaginatedGroupResponse{
		Items:      groups,
		TotalItems: totalItems,
		PageNumber: params.PageNumber,
		PageSize:   params.PageSize,
	}

	return response, nil
}


