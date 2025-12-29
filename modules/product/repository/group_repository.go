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
		INSERT INTO groups (name, description)
		VALUES (:name, :description)
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
		SET name = $1, description = $2, updated_at = now()
		WHERE id = $3
	`

	result, err := r.DB.SQLx().ExecContext(ctx, query,
		group.Name,
		group.Description,
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
			description, 
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
			description, 
			created_at, 
			updated_at
	` + baseQuery + whereClause + `
		ORDER BY created_at DESC
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

// UserGroup methods - Quản lý user trong group

func (r *ProductRepository) PrivateAddUsersToGroup(ctx context.Context, groupID uuid.UUID, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}

	// Sử dụng transaction để đảm bảo tính nhất quán
	tx, err := r.DB.SQLx().BeginTxx(ctx, nil)
	if err != nil {
		logger.Error("ProductRepository:PrivateAddUsersToGroup - BeginTx", err)
		return err
	}
	defer tx.Rollback()

	// Insert từng user vào group (bỏ qua nếu đã tồn tại)
	query := `
		INSERT INTO user_groups (user_id, group_id, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, group_id) DO NOTHING
	`

	for _, userID := range userIDs {
		_, err := tx.ExecContext(ctx, query, userID, groupID)
		if err != nil {
			logger.Error("ProductRepository:PrivateAddUsersToGroup - Insert", err)
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		logger.Error("ProductRepository:PrivateAddUsersToGroup - Commit", err)
		return err
	}

	return nil
}

func (r *ProductRepository) PrivateRemoveUserFromGroup(ctx context.Context, groupID uuid.UUID, userID uuid.UUID) error {
	query := `
		DELETE FROM user_groups
		WHERE group_id = $1 AND user_id = $2
	`

	result, err := r.DB.SQLx().ExecContext(ctx, query, groupID, userID)
	if err != nil {
		logger.Error("ProductRepository:PrivateRemoveUserFromGroup", err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logger.Error("ProductRepository:PrivateRemoveUserFromGroup - RowsAffected", err)
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user %s is not in group %s", userID, groupID)
	}

	return nil
}

func (r *ProductRepository) PrivateGetUsersByGroupId(ctx context.Context, groupID uuid.UUID) ([]entity.UserGroup, error) {
	query := `
		SELECT 
			id,
			user_id,
			group_id,
			created_at
		FROM user_groups
		WHERE group_id = $1
		ORDER BY created_at DESC
	`

	var userGroups []entity.UserGroup
	err := r.DB.SelectContext(ctx, &userGroups, query, groupID)
	if err != nil {
		if err == sql.ErrNoRows {
			return []entity.UserGroup{}, nil
		}
		logger.Error("ProductRepository:PrivateGetUsersByGroupId", err)
		return nil, err
	}

	return userGroups, nil
}

func (r *ProductRepository) PrivateGetGroupsByUserId(ctx context.Context, userID uuid.UUID) ([]entity.UserGroup, error) {
	query := `
		SELECT 
			id,
			user_id,
			group_id,
			created_at
		FROM user_groups
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	var userGroups []entity.UserGroup
	err := r.DB.SelectContext(ctx, &userGroups, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return []entity.UserGroup{}, nil
		}
		logger.Error("ProductRepository:PrivateGetGroupsByUserId", err)
		return nil, err
	}

	return userGroups, nil
}


