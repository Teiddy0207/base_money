package repository

import (
	"context"
	"database/sql"
	"go-api-starter/core/logger"
	"go-api-starter/modules/auth/entity"

	"github.com/google/uuid"
)

func (r *AuthRepository) GetSocialLoginByUserIDAndProvider(ctx context.Context, userID uuid.UUID, providerID uuid.UUID) (*entity.SocialLogin, error) {
	var socialLogin entity.SocialLogin
	query := `
		SELECT * FROM social_logins 
		WHERE user_id = $1 AND provider_id = $2 AND is_active = true
	`
	err := r.DB.GetContext(ctx, &socialLogin, query, userID, providerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		logger.Error("AuthRepository:GetSocialLoginByUserIDAndProvider:Error", "error", err, "user_id", userID, "provider_id", providerID)
		return nil, err
	}
	return &socialLogin, nil
}

func (r *AuthRepository) SaveOrUpdateSocialLogin(ctx context.Context, socialLogin *entity.SocialLogin) error {
	query := `
		INSERT INTO social_logins (
			user_id, provider_id, provider_user_id, provider_username, 
			provider_email, access_token, refresh_token, token_expires_at, 
			last_login_at, is_active, created_at, updated_at
		)
		VALUES (
			:user_id, :provider_id, :provider_user_id, :provider_username,
			:provider_email, :access_token, :refresh_token, :token_expires_at,
			:last_login_at, :is_active, NOW(), NOW()
		)
		ON CONFLICT (user_id, provider_id)
		DO UPDATE SET
			provider_user_id = EXCLUDED.provider_user_id,
			provider_username = EXCLUDED.provider_username,
			provider_email = EXCLUDED.provider_email,
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			token_expires_at = EXCLUDED.token_expires_at,
			last_login_at = EXCLUDED.last_login_at,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
	`
	_, err := r.DB.NamedExecContext(ctx, query, socialLogin)
	if err != nil {
		logger.Error("AuthRepository:SaveOrUpdateSocialLogin:Error", "error", err)
		return err
	}
	return nil
}

func (r *AuthRepository) GetOAuthProviderByName(ctx context.Context, name string) (*entity.OAuthProvider, error) {
	var provider entity.OAuthProvider
	query := `SELECT * FROM oauth_providers WHERE name = $1 AND is_active = true`
	err := r.DB.GetContext(ctx, &provider, query, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		logger.Error("AuthRepository:GetOAuthProviderByName:Error", "error", err, "name", name)
		return nil, err
	}
	return &provider, nil
}

func (r *AuthRepository) SeedGoogleProvider(ctx context.Context, clientID string, clientSecret string, redirectURI string) error {
	if clientID == "" || clientSecret == "" || redirectURI == "" {
		logger.Info("AuthRepository:SeedGoogleProvider:Skipped", "reason", "Google OAuth credentials not configured")
		return nil
	}

	query := `
		INSERT INTO oauth_providers (name, display_name, client_id, client_secret, redirect_uri, scopes, is_active, created_at, updated_at)
		VALUES (
			'google',
			'Google',
			$1,
			$2,
			$3,
			ARRAY[
				'https://www.googleapis.com/auth/userinfo.email',
				'https://www.googleapis.com/auth/userinfo.profile',
				'https://www.googleapis.com/auth/calendar.readonly'
			],
			true,
			NOW(),
			NOW()
		)
		ON CONFLICT (name) DO UPDATE SET
			client_id = EXCLUDED.client_id,
			client_secret = EXCLUDED.client_secret,
			redirect_uri = EXCLUDED.redirect_uri,
			updated_at = NOW()
	`
	err := r.DB.ExecContext(ctx, query, clientID, clientSecret, redirectURI)
	if err != nil {
		logger.Error("AuthRepository:SeedGoogleProvider:Error", "error", err)
		return err
	}
	logger.Info("AuthRepository:SeedGoogleProvider:Success", "provider", "google")
	return nil
}