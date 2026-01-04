package repository

import (
	"context"
	"database/sql"
	"fmt"
	coreentity "go-api-starter/core/entity"
	"go-api-starter/core/logger"
	"go-api-starter/core/params"
	"go-api-starter/modules/product/dto"
	"go-api-starter/modules/product/entity"
	"strings"

	"github.com/google/uuid"
)

func (r *ProductRepository) PrivateCreateGroup(ctx context.Context, group *entity.Group) (*entity.Group, error) {
	query := `
		INSERT INTO groups (name, description)
		VALUES (:name, :description)
		RETURNING id, created_at, updated_at
	`
	rows, err := r.DB.NamedQueryContext(ctx, query, group)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		logger.Error("ProductRepository:PrivateCreateGroup", err)
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&group.ID, &group.CreatedAt, &group.UpdatedAt)
		if err != nil {
			logger.Error("ProductRepository:PrivateCreateGroup:Scan", err)
			return nil, err
		}
	}
	return group, nil
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
	offset := (params.PageNumber - 1) * params.PageSize

	baseQuery := `FROM groups`

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

	logger.Info("ProductRepository:PrivateGetGroups:Result", "total_items", totalItems, "items_count", len(groups))
	response := &entity.PaginatedGroupResponse{
		Items:      groups,
		TotalItems: totalItems,
		PageNumber: params.PageNumber,
		PageSize:   params.PageSize,
	}

	return response, nil
}

func (r *ProductRepository) PrivateGetGroupsWhereMember(ctx context.Context, memberID uuid.UUID, params params.QueryParams) (*entity.PaginatedGroupResponse, error) {
	offset := (params.PageNumber - 1) * params.PageSize
	baseQuery := `
		FROM groups g
		INNER JOIN user_groups ug ON ug.group_id = g.id
		WHERE ug.user_id = $1
	`
	var args []interface{}
	args = append(args, memberID)
	if params.Search != "" {
		baseQuery += " AND g.name ILIKE $2"
		args = append(args, "%"+params.Search+"%")
	}
	countQuery := "SELECT COUNT(DISTINCT g.id) " + baseQuery
	var totalItems int
	err := r.DB.GetContext(ctx, &totalItems, countQuery, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return &entity.PaginatedGroupResponse{
				Items:      []entity.Group{},
				TotalItems: 0,
				PageNumber: params.PageNumber,
				PageSize:   params.PageSize,
			}, nil
		}
		logger.Error("ProductRepository:PrivateGetGroupsWhereMember - Count", err)
		return nil, err
	}
	dataQuery := `
		SELECT 
			g.id, 
			g.name, 
			g.description, 
			g.created_at, 
			g.updated_at
	` + baseQuery + `
		ORDER BY g.created_at DESC
		LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
	args = append(args, params.PageSize, offset)
	var groups []entity.Group
	err = r.DB.SelectContext(ctx, &groups, dataQuery, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return &entity.PaginatedGroupResponse{
				Items:      []entity.Group{},
				TotalItems: 0,
				PageNumber: params.PageNumber,
				PageSize:   params.PageSize,
			}, nil
		}
		logger.Error("ProductRepository:PrivateGetGroupsWhereMember - Select", err)
		return nil, err
	}
	logger.Info("ProductRepository:PrivateGetGroupsWhereMember:Result", "member_id", memberID, "total_items", totalItems, "items_count", len(groups))
	response := &entity.PaginatedGroupResponse{
		Items:      groups,
		TotalItems: totalItems,
		PageNumber: params.PageNumber,
		PageSize:   params.PageSize,
	}
	return response, nil
}
func (r *ProductRepository) PrivateAddUsersToGroup(ctx context.Context, groupID uuid.UUID, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}

	tx, err := r.DB.SQLx().BeginTxx(ctx, nil)
	if err != nil {
		logger.Error("ProductRepository:PrivateAddUsersToGroup - BeginTx", err)
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO user_groups (id, user_id, group_id, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, group_id) DO NOTHING
	`

	for _, userID := range userIDs {
		newID := uuid.New()
		_, err := tx.ExecContext(ctx, query, newID, userID, groupID)
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
			ug.id as ug_id,
			ug.user_id as ug_user_id,
			ug.group_id as ug_group_id,
			ug.created_at as ug_created_at,
			g.id as g_id,
			g.name as g_name,
			g.description as g_description,
			sl.id as u_id,
			sl.provider_username as sl_provider_name,
			sl.provider_email as sl_provider_email
		FROM user_groups ug
		LEFT JOIN groups g ON g.id = ug.group_id
		LEFT JOIN social_logins sl ON sl.id = ug.user_id AND sl.is_active = true
		WHERE ug.group_id = $1
		ORDER BY ug.created_at DESC
	`

	var results []dto.UserGroupWithRelations
	err := r.DB.SelectContext(ctx, &results, query, groupID)
	if err != nil {
		if err == sql.ErrNoRows {
			return []entity.UserGroup{}, nil
		}
		logger.Error("ProductRepository:PrivateGetUsersByGroupId", err)
		return nil, err
	}

	userGroups := make([]entity.UserGroup, len(results))
	for i, result := range results {
		userGroups[i] = entity.UserGroup{
			ID:        result.ID,
			UserID:    result.UserID,
			GroupID:   result.GroupID,
			CreatedAt: result.CreatedAt,
		}
	}

	return userGroups, nil
}

func (r *ProductRepository) PrivateGetUsersByGroupIdWithRelations(ctx context.Context, groupID uuid.UUID) ([]dto.UserGroupWithRelations, *entity.Group, error) {
	query := `
		SELECT 
			ug.id as ug_id,
			ug.user_id as ug_user_id,
			ug.group_id as ug_group_id,
			ug.created_at as ug_created_at,
			g.id as g_id,
			g.name as g_name,
			g.description as g_description,
			sl.id as u_id,
			sl.provider_username as sl_provider_name,
			sl.provider_email as sl_provider_email
		FROM user_groups ug
		LEFT JOIN groups g ON g.id = ug.group_id
		LEFT JOIN social_logins sl ON sl.id = ug.user_id AND sl.is_active = true
		WHERE ug.group_id = $1
		ORDER BY ug.created_at DESC
	`

	var results []dto.UserGroupWithRelations
	err := r.DB.SelectContext(ctx, &results, query, groupID)
	if err != nil {
		if err == sql.ErrNoRows {
			return []dto.UserGroupWithRelations{}, nil, nil
		}
		logger.Error("ProductRepository:PrivateGetUsersByGroupIdWithRelations", err)
		return nil, nil, err
	}

	var group *entity.Group
	if len(results) > 0 {
		group = &entity.Group{
			BaseEntity: coreentity.BaseEntity{
				ID: results[0].GroupIDFromGroup,
			},
			Name:        results[0].GroupName,
			Description: results[0].GroupDescription,
		}
	} else {
		group, err = r.PrivateGetGroupById(ctx, groupID)
		if err != nil {
			logger.Error("ProductRepository:PrivateGetUsersByGroupIdWithRelations - GetGroupById", err)
			return results, nil, err
		}
	}

	return results, group, nil
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

func (r *ProductRepository) PrivateAreUsersInSameGroup(ctx context.Context, userA uuid.UUID, userB uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM user_groups ug1
			INNER JOIN user_groups ug2 ON ug1.group_id = ug2.group_id
			WHERE ug1.user_id = $1 AND ug2.user_id = $2
		)
	`
	var exists bool
	err := r.DB.GetContext(ctx, &exists, query, userA, userB)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		logger.Error("ProductRepository:PrivateAreUsersInSameGroup", err)
		return false, err
	}
	logger.Info("ProductRepository:PrivateAreUsersInSameGroup", "userA", userA, "userB", userB, "exists", exists)
	return exists, nil
}
