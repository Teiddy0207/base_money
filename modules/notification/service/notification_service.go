package service

import (
	"context"
	coreEntity "go-api-starter/core/entity"
	"go-api-starter/core/params"
	"go-api-starter/modules/notification/dto"
	"go-api-starter/modules/notification/entity"
	"go-api-starter/modules/notification/repository"
	"time"

	"github.com/google/uuid"
)

type NotificationService struct {
	repo *repository.NotificationRepository
}

func NewNotificationService(repo *repository.NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

func (s *NotificationService) Create(ctx context.Context, req *dto.CreateNotificationRequest) error {
	notif := &entity.Notification{
		UserID:  req.UserID,
		Title:   req.Title,
		Message: req.Message,
		Type:    req.Type,
		Data:    entity.JSONB(req.Data),
		IsRead:  false,
		BaseEntity: coreEntity.BaseEntity{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	return s.repo.Create(ctx, notif)
}

func (s *NotificationService) GetMyNotifications(ctx context.Context, userID uuid.UUID, queryParams params.QueryParams) (*entity.PaginatedNotificationEntity, error) {
	// Return entity directly for now or map to DTO if strict separation needed
	// Using entity.PaginatedUserEntity pattern
	return s.repo.GetByUserID(ctx, userID, queryParams)
}

func (s *NotificationService) MarkAsRead(ctx context.Context, userID uuid.UUID, ids []string) error {
	return s.repo.MarkAsRead(ctx, userID, ids)
}

func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllAsRead(ctx, userID)
}

func (s *NotificationService) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountUnread(ctx, userID)
}
