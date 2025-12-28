package repository

import (
	"context"
	"database/sql"

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

// GetConnectionByUserAndProvider gets a specific connection
func (r *calendarRepository) GetConnectionByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*entity.CalendarConnection, error) {
	query := `
		SELECT id, user_id, provider, access_token, refresh_token, token_expires_at, calendar_email, is_active, created_at, updated_at
		FROM calendar_connections
		WHERE user_id = $1 AND provider = $2 AND is_active = true
	`
	var conn entity.CalendarConnection
	err := r.db.QueryRowContext(ctx, query, userID, provider).Scan(
		&conn.ID, &conn.UserID, &conn.Provider, &conn.AccessToken, &conn.RefreshToken,
		&conn.TokenExpiresAt, &conn.CalendarEmail, &conn.IsActive, &conn.CreatedAt, &conn.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
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

// GetConnectionsByUserIDs gets connections for multiple users
func (r *calendarRepository) GetConnectionsByUserIDs(ctx context.Context, userIDs []uuid.UUID) ([]entity.CalendarConnection, error) {
	if len(userIDs) == 0 {
		return []entity.CalendarConnection{}, nil
	}

	query := `
		SELECT id, user_id, provider, access_token, refresh_token, token_expires_at, calendar_email, is_active, created_at, updated_at
		FROM calendar_connections
		WHERE user_id = ANY($1) AND is_active = true
	`
	rows, err := r.db.QueryContext(ctx, query, userIDs)
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
