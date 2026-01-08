package repository

import (
	"context"
	"database/sql"
	"go-api-starter/core/logger"
)

// SocialUserResult represents a user from social_logins for search
type SocialUserResult struct {
	ID           string `db:"id" json:"id"`                        // social_logins.id - dùng để thêm vào group
	UserID       string `db:"user_id" json:"user_id"`              // users.id - giữ lại để tương thích
	Email        string `db:"provider_email" json:"email"`
	DisplayName  string `db:"provider_username" json:"display_name"`
}

// SearchSocialUsers searches for users from social_logins table
func (r *AuthRepository) SearchSocialUsers(ctx context.Context, query string) ([]SocialUserResult, error) {
	var users []SocialUserResult

	sqlQuery := `
		SELECT DISTINCT ON (user_id)
			id::text as id,
			user_id::text as user_id,
			COALESCE(provider_email, '') as provider_email,
			COALESCE(provider_username, '') as provider_username
		FROM social_logins
		WHERE is_active = true
		  AND (
			provider_email ILIKE $1 
			OR provider_username ILIKE $1
		  )
		ORDER BY user_id, last_login_at DESC
		LIMIT 20
	`

	searchPattern := "%" + query + "%"
	err := r.DB.SelectContext(ctx, &users, sqlQuery, searchPattern)
	if err != nil {
		if err == sql.ErrNoRows {
			return []SocialUserResult{}, nil
		}
		logger.Error("AuthRepository:SearchSocialUsers:Error:", err)
		return nil, err
	}

	return users, nil
}

// GetAllSocialUsers gets all users from social_logins table
func (r *AuthRepository) GetAllSocialUsers(ctx context.Context) ([]SocialUserResult, error) {
	var users []SocialUserResult

	sqlQuery := `
		SELECT DISTINCT ON (user_id)
			id::text as id,
			user_id::text as user_id,
			COALESCE(provider_email, '') as provider_email,
			COALESCE(provider_username, '') as provider_username
		FROM social_logins
		WHERE is_active = true
		ORDER BY user_id, last_login_at DESC
		LIMIT 50
	`

	err := r.DB.SelectContext(ctx, &users, sqlQuery)
	if err != nil {
		if err == sql.ErrNoRows {
			return []SocialUserResult{}, nil
		}
		logger.Error("AuthRepository:GetAllSocialUsers:Error:", err)
		return nil, err
	}

	return users, nil
}
