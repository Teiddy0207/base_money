package repository

import (
	"context"
	"database/sql"
	"time"

	"go-api-starter/core/database"
	"go-api-starter/modules/calendar/entity"

	"github.com/google/uuid"
)

type CalendarRepository interface {
	// Calendar Connections
	CreateConnection(ctx context.Context, conn *entity.CalendarConnection) (*entity.CalendarConnection, error)
	GetConnectionByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*entity.CalendarConnection, error)
	GetConnectionsByUserID(ctx context.Context, userID uuid.UUID) ([]entity.CalendarConnection, error)
	UpdateConnection(ctx context.Context, conn *entity.CalendarConnection) error
	DeleteConnection(ctx context.Context, userID uuid.UUID, provider string) error

	// Get connections by multiple user IDs (for free/busy lookup)
	GetConnectionsByUserIDs(ctx context.Context, userIDs []uuid.UUID) ([]entity.CalendarConnection, error)
}

type calendarRepository struct {
	db database.Database
}

func NewCalendarRepository(db database.Database) CalendarRepository {
	return &calendarRepository{db: db}
}

// CreateConnection creates a new calendar connection
func (r *calendarRepository) CreateConnection(ctx context.Context, conn *entity.CalendarConnection) (*entity.CalendarConnection, error) {
	query := `
		INSERT INTO calendar_connections (user_id, provider, access_token, refresh_token, token_expires_at, calendar_email, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(
		ctx, query,
		conn.UserID, conn.Provider, conn.AccessToken, conn.RefreshToken,
		conn.TokenExpiresAt, conn.CalendarEmail, conn.IsActive,
	).Scan(&conn.ID, &conn.CreatedAt, &conn.UpdatedAt)

	if err != nil {
		return nil, err
	}
	return conn, nil
}

// GetConnectionByUserAndProvider gets a specific connection from social_logins
func (r *calendarRepository) GetConnectionByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*entity.CalendarConnection, error) {
	// Query from social_logins table where tokens are actually stored during Google login
	query := `
		SELECT sl.user_id, sl.provider_email, sl.access_token, sl.refresh_token, sl.token_expires_at
		FROM social_logins sl
		JOIN oauth_providers op ON sl.provider_id = op.id
		WHERE sl.user_id = $1 
		AND op.name = $2
		AND sl.is_active = true
		AND sl.access_token IS NOT NULL
	`
	var conn entity.CalendarConnection
	var emailPtr, accessTokenPtr, refreshTokenPtr *string
	var expiresAtPtr *time.Time

	err := r.db.QueryRowContext(ctx, query, userID, provider).Scan(
		&conn.UserID, &emailPtr, &accessTokenPtr, &refreshTokenPtr, &expiresAtPtr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	conn.Provider = provider
	conn.IsActive = true
	if emailPtr != nil {
		conn.CalendarEmail = *emailPtr
	}
	if accessTokenPtr != nil {
		conn.AccessToken = *accessTokenPtr
	}
	if refreshTokenPtr != nil {
		conn.RefreshToken = *refreshTokenPtr
	}
	if expiresAtPtr != nil {
		conn.TokenExpiresAt = *expiresAtPtr
	}

	return &conn, nil
}

// GetConnectionsByUserID gets all connections for a user
func (r *calendarRepository) GetConnectionsByUserID(ctx context.Context, userID uuid.UUID) ([]entity.CalendarConnection, error) {
	query := `
		SELECT id, user_id, provider, access_token, refresh_token, token_expires_at, calendar_email, is_active, created_at, updated_at
		FROM calendar_connections
		WHERE user_id = $1 AND is_active = true
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []entity.CalendarConnection
	for rows.Next() {
		var conn entity.CalendarConnection
		if err := rows.Scan(
			&conn.ID, &conn.UserID, &conn.Provider, &conn.AccessToken, &conn.RefreshToken,
			&conn.TokenExpiresAt, &conn.CalendarEmail, &conn.IsActive, &conn.CreatedAt, &conn.UpdatedAt,
		); err != nil {
			return nil, err
		}
		connections = append(connections, conn)
	}
	return connections, nil
}

// UpdateConnection updates a calendar connection
func (r *calendarRepository) UpdateConnection(ctx context.Context, conn *entity.CalendarConnection) error {
	query := `
		UPDATE calendar_connections
		SET access_token = $1, refresh_token = $2, token_expires_at = $3, is_active = $4, updated_at = NOW()
		WHERE id = $5
	`
	return r.db.ExecContext(ctx, query,
		conn.AccessToken, conn.RefreshToken, conn.TokenExpiresAt, conn.IsActive, conn.ID,
	)
}

// DeleteConnection soft deletes a calendar connection
func (r *calendarRepository) DeleteConnection(ctx context.Context, userID uuid.UUID, provider string) error {
	query := `
		UPDATE calendar_connections
		SET is_active = false, updated_at = NOW()
		WHERE user_id = $1 AND provider = $2
	`
	return r.db.ExecContext(ctx, query, userID, provider)
}

// GetConnectionsByUserIDs gets Google calendar connections for multiple users from social_logins
func (r *calendarRepository) GetConnectionsByUserIDs(ctx context.Context, userIDs []uuid.UUID) ([]entity.CalendarConnection, error) {
	if len(userIDs) == 0 {
		return []entity.CalendarConnection{}, nil
	}

	// Convert []uuid.UUID to []string for PostgreSQL array
	userIDStrings := make([]string, len(userIDs))
	for i, id := range userIDs {
		userIDStrings[i] = id.String()
	}

	// Query social_logins table for Google tokens (where auth stores them)
	query := `
		SELECT sl.user_id, sl.provider_email, sl.access_token, sl.refresh_token, sl.token_expires_at
		FROM social_logins sl
		JOIN oauth_providers op ON sl.provider_id = op.id
		WHERE sl.user_id = ANY($1::uuid[]) 
		AND op.name = 'google'
		AND sl.is_active = true
		AND sl.access_token IS NOT NULL
	`
	rows, err := r.db.QueryContext(ctx, query, "{"+joinStrings(userIDStrings, ",")+"}")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []entity.CalendarConnection
	for rows.Next() {
		var conn entity.CalendarConnection
		var emailPtr, accessTokenPtr, refreshTokenPtr *string
		var expiresAtPtr *time.Time

		if err := rows.Scan(&conn.UserID, &emailPtr, &accessTokenPtr, &refreshTokenPtr, &expiresAtPtr); err != nil {
			return nil, err
		}

		conn.Provider = "google"
		conn.IsActive = true
		if emailPtr != nil {
			conn.CalendarEmail = *emailPtr
		}
		if accessTokenPtr != nil {
			conn.AccessToken = *accessTokenPtr
		}
		if refreshTokenPtr != nil {
			conn.RefreshToken = *refreshTokenPtr
		}
		if expiresAtPtr != nil {
			conn.TokenExpiresAt = *expiresAtPtr
		}

		connections = append(connections, conn)
	}
	return connections, nil
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
