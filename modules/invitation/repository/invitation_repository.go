package repository

import (
	"context"
	"time"

	"go-api-starter/core/database"
	"go-api-starter/core/logger"
	"go-api-starter/modules/invitation/entity"

	"github.com/google/uuid"
)

type InvitationRepository struct {
	db database.Database
}

func NewInvitationRepository(db database.Database) *InvitationRepository {
	return &InvitationRepository{db: db}
}

// Create creates a new invitation
func (r *InvitationRepository) Create(ctx context.Context, invitation *entity.EventInvitation) error {
	query := `
		INSERT INTO event_invitations (event_google_id, creator_id, invitee_id, status, event_data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	now := time.Now()
	invitation.CreatedAt = now
	invitation.UpdatedAt = now
	if invitation.Status == "" {
		invitation.Status = entity.InvitationStatusPending
	}

	eventDataValue, err := invitation.EventData.Value()
	if err != nil {
		logger.Error("InvitationRepository:Create:EventDataValue:Error:", err)
		return err
	}

	row := r.db.QueryRowContext(ctx, query,
		invitation.EventGoogleID,
		invitation.CreatorID,
		invitation.InviteeID,
		invitation.Status,
		eventDataValue,
		invitation.CreatedAt,
		invitation.UpdatedAt,
	)
	return row.Scan(&invitation.ID)
}

// GetByID gets an invitation by ID
func (r *InvitationRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.EventInvitation, error) {
	query := `
		SELECT id, event_google_id, creator_id, invitee_id, status, event_data, responded_at, created_at, updated_at
		FROM event_invitations
		WHERE id = $1
	`
	var inv entity.EventInvitation
	err := r.db.GetContext(ctx, &inv, query, id)
	if err != nil {
		logger.Error("InvitationRepository:GetByID:Error:", err)
		return nil, err
	}
	return &inv, nil
}

// GetPendingByInviteeID gets all pending invitations for a user
func (r *InvitationRepository) GetPendingByInviteeID(ctx context.Context, inviteeID uuid.UUID) ([]entity.EventInvitation, error) {
	query := `
		SELECT id, event_google_id, creator_id, invitee_id, status, event_data, responded_at, created_at, updated_at
		FROM event_invitations
		WHERE invitee_id = $1 AND status = 'pending'
		ORDER BY created_at DESC
	`
	var invitations []entity.EventInvitation
	err := r.db.SelectContext(ctx, &invitations, query, inviteeID)
	if err != nil {
		logger.Error("InvitationRepository:GetPendingByInviteeID:Error:", err)
		return nil, err
	}
	return invitations, nil
}

// UpdateStatus updates the invitation status
func (r *InvitationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE event_invitations 
		SET status = $1, responded_at = $2, updated_at = $3
		WHERE id = $4
	`
	now := time.Now()
	err := r.db.ExecContext(ctx, query, status, now, now, id)
	if err != nil {
		logger.Error("InvitationRepository:UpdateStatus:Error:", err)
		return err
	}
	return nil
}

// CountPendingByInviteeID counts pending invitations for a user
func (r *InvitationRepository) CountPendingByInviteeID(ctx context.Context, inviteeID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM event_invitations WHERE invitee_id = $1 AND status = 'pending'`
	err := r.db.GetContext(ctx, &count, query, inviteeID)
	if err != nil {
		logger.Error("InvitationRepository:CountPendingByInviteeID:Error:", err)
		return 0, err
	}
	return count, nil
}
