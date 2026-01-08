package repository

import (
	"context"
	"go-api-starter/core/database"
	"go-api-starter/core/logger"
	"go-api-starter/core/params"
	"go-api-starter/modules/notification/entity"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type NotificationRepository struct {
	db database.Database
}

func NewNotificationRepository(db database.Database) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, notification *entity.Notification) error {
	query := `
		INSERT INTO notifications (title, message, type, data, user_id, is_read, created_at, updated_at)
		VALUES (:title, :message, :type, :data, :user_id, :is_read, :created_at, :updated_at)
		RETURNING id
	`
	rows, err := r.db.NamedQueryContext(ctx, query, notification)
	if err != nil {
		logger.Error("NotificationRepository:Create:Error:", err)
		return err
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&notification.ID)
	}
	return nil
}

func (r *NotificationRepository) GetByUserID(ctx context.Context, userID uuid.UUID, params params.QueryParams) (*entity.PaginatedNotificationEntity, error) {
	offset := (params.PageNumber - 1) * params.PageSize

	// Base query
	baseQuery := `FROM notifications WHERE user_id = $1`

	// Count query
	var totalItems int
	err := r.db.GetContext(ctx, &totalItems, "SELECT COUNT(*) "+baseQuery, userID)
	if err != nil {
		logger.Error("NotificationRepository:GetByUserID:Count:Error:", err)
		return nil, err
	}

	// Data query
	query := `
		SELECT * ` + baseQuery + `
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var notifications []entity.Notification
	err = r.db.SelectContext(ctx, &notifications, query, userID, params.PageSize, offset)
	if err != nil {
		logger.Error("NotificationRepository:GetByUserID:Select:Error:", err)
		return nil, err
	}

	return &entity.PaginatedNotificationEntity{
		Items:      notifications,
		TotalItems: totalItems,
		PageNumber: params.PageNumber,
		PageSize:   params.PageSize,
	}, nil
}

func (r *NotificationRepository) MarkAsRead(ctx context.Context, userID uuid.UUID, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	query, args, err := sqlx.In(`UPDATE notifications SET is_read = true WHERE user_id = ? AND id IN (?)`, userID, ids)
	if err != nil {
		return err
	}

	query = r.db.SQLx().Rebind(query)
	err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		logger.Error("NotificationRepository:MarkAsRead:Error:", err)
		return err
	}
	return nil
}

func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE notifications SET is_read = true WHERE user_id = $1`
	err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		logger.Error("NotificationRepository:MarkAllAsRead:Error:", err)
		return err
	}
	return nil
}

func (r *NotificationRepository) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`
	err := r.db.GetContext(ctx, &count, query, userID)
	if err != nil {
		logger.Error("NotificationRepository:CountUnread:Error:", err)
		return 0, err
	}
	return count, nil
}
