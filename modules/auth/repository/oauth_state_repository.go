package repository

import (
	"context"
	"database/sql"
	"go-api-starter/core/logger"
	"go-api-starter/modules/auth/entity"
	"time"
)

// SaveOAuthState saves OAuth state token to database
func (r *AuthRepository) SaveOAuthState(ctx context.Context, state string, expiresAt time.Time) error {
	query := `
		INSERT INTO oauth_states (id, state, expires_at, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, NOW(), NOW())
		ON CONFLICT (state) 
		DO UPDATE SET expires_at = $2, updated_at = NOW()
	`
	err := r.DB.ExecContext(ctx, query, state, expiresAt)
	if err != nil {
		logger.Error("AuthRepository:SaveOAuthState:Error", "error", err, "state", state)
		return err
	}
	return nil
}

// GetOAuthState retrieves OAuth state token from database
func (r *AuthRepository) GetOAuthState(ctx context.Context, state string) (*entity.OAuthState, error) {
	var oauthState entity.OAuthState
	query := `
		SELECT id, state, expires_at, created_at, updated_at
		FROM oauth_states
		WHERE state = $1 AND expires_at > NOW()
	`
	err := r.DB.GetContext(ctx, &oauthState, query, state)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		logger.Error("AuthRepository:GetOAuthState:Error", "error", err, "state", state)
		return nil, err
	}
	return &oauthState, nil
}

// DeleteOAuthState deletes OAuth state token from database
func (r *AuthRepository) DeleteOAuthState(ctx context.Context, state string) error {
	query := `DELETE FROM oauth_states WHERE state = $1`
	err := r.DB.ExecContext(ctx, query, state)
	if err != nil {
		logger.Error("AuthRepository:DeleteOAuthState:Error", "error", err, "state", state)
		return err
	}
	return nil
}

// CleanupExpiredOAuthStates removes expired OAuth state tokens
func (r *AuthRepository) CleanupExpiredOAuthStates(ctx context.Context) error {
	query := `DELETE FROM oauth_states WHERE expires_at < NOW()`
	err := r.DB.ExecContext(ctx, query)
	if err != nil {
		logger.Error("AuthRepository:CleanupExpiredOAuthStates:Error", "error", err)
		return err
	}
	return nil
}



