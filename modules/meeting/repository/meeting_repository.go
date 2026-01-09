package repository

import (
	"context"
	"database/sql"
	"go-api-starter/core/database"
	"go-api-starter/core/logger"
	"go-api-starter/modules/meeting/entity"
	"time"

	"github.com/google/uuid"
)

// MeetingRepository handles event database operations (using events table)
type MeetingRepository struct {
	DB database.Database
}

// NewMeetingRepository creates a new repository instance
func NewMeetingRepository(db database.Database) *MeetingRepository {
	return &MeetingRepository{DB: db}
}

// MeetingRepositoryInterface defines the repository contract
type MeetingRepositoryInterface interface {
	// Event CRUD (using events table)
	CreateEvent(ctx context.Context, event *entity.Event) (*entity.Event, error)
	GetEventByID(ctx context.Context, id uuid.UUID) (*entity.Event, error)
	GetEventsByHostID(ctx context.Context, hostID uuid.UUID) ([]entity.Event, error)
	UpdateEvent(ctx context.Context, event *entity.Event) error
	DeleteEvent(ctx context.Context, id uuid.UUID) error

	// Participants (using user_events table)
	AddParticipant(ctx context.Context, userEvent *entity.UserEvent) error
	GetParticipantsByEventID(ctx context.Context, eventID uuid.UUID) ([]entity.UserEvent, error)
	UpdateParticipantCalendarStatus(ctx context.Context, userID uuid.UUID, eventID uuid.UUID, hasCalendar bool) error
	RemoveParticipant(ctx context.Context, userID uuid.UUID, eventID uuid.UUID) error

	// Slots (using event_slots table)
	SaveSlots(ctx context.Context, slots []entity.EventSlot) error
	GetSlotsByEventID(ctx context.Context, eventID uuid.UUID) ([]entity.EventSlot, error)
	ClearSlotsByEventID(ctx context.Context, eventID uuid.UUID) error
}

// ===================== Event CRUD =====================

func (r *MeetingRepository) CreateEvent(ctx context.Context, event *entity.Event) (*entity.Event, error) {
	query := `
		INSERT INTO events (host_id, title, description, address, duration_minutes, status, timezone, preferences)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, host_id, title, description, address, duration_minutes, status, timezone,
		          start_date, end_date, meeting_link, preferences, created_at, updated_at
	`

	var created entity.Event
	err := r.DB.GetContext(ctx, &created, query,
		event.HostID, event.Title, event.Description, event.Address,
		event.DurationMinutes, event.Status, event.Timezone, event.Preferences)

	if err != nil {
		logger.Error("MeetingRepository:CreateEvent", err)
		return nil, err
	}

	return &created, nil
}

func (r *MeetingRepository) GetEventByID(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
	query := `
		SELECT id, host_id, title, description, address, duration_minutes, status, timezone,
		       start_date, end_date, meeting_link, preferences, created_at, updated_at
		FROM events WHERE id = $1
	`

	var event entity.Event
	err := r.DB.GetContext(ctx, &event, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		logger.Error("MeetingRepository:GetEventByID", err)
		return nil, err
	}

	return &event, nil
}

func (r *MeetingRepository) GetEventsByHostID(ctx context.Context, hostID uuid.UUID) ([]entity.Event, error) {
	query := `
		SELECT id, host_id, title, description, address, duration_minutes, status, timezone,
		       start_date, end_date, meeting_link, preferences, created_at, updated_at
		FROM events 
		WHERE host_id = $1
		ORDER BY created_at DESC
	`

	var events []entity.Event
	err := r.DB.SelectContext(ctx, &events, query, hostID)
	if err != nil {
		logger.Error("MeetingRepository:GetEventsByHostID", err)
		return nil, err
	}

	return events, nil
}

func (r *MeetingRepository) UpdateEvent(ctx context.Context, event *entity.Event) error {
	query := `
		UPDATE events 
		SET title = $2, description = $3, address = $4, duration_minutes = $5, status = $6,
		    start_date = $7, end_date = $8, meeting_link = $9, preferences = $10, updated_at = NOW()
		WHERE id = $1
	`

	// Log time values before saving
	if event.StartDate != nil {
		logger.Info("MeetingRepository:UpdateEvent:BeforeSave",
			"event_id", event.ID.String(),
			"start_date_utc", event.StartDate.UTC().Format(time.RFC3339),
			"start_date_local", event.StartDate.Format(time.RFC3339))
	}
	if event.EndDate != nil {
		logger.Info("MeetingRepository:UpdateEvent:BeforeSave",
			"event_id", event.ID.String(),
			"end_date_utc", event.EndDate.UTC().Format(time.RFC3339),
			"end_date_local", event.EndDate.Format(time.RFC3339))
	}

	err := r.DB.ExecContext(ctx, query,
		event.ID, event.Title, event.Description, event.Address, event.DurationMinutes,
		event.Status, event.StartDate, event.EndDate, event.MeetingLink, event.Preferences)

	if err != nil {
		logger.Error("MeetingRepository:UpdateEvent", err)
		return err
	}

	return nil
}

func (r *MeetingRepository) DeleteEvent(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM events WHERE id = $1`
	err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		logger.Error("MeetingRepository:DeleteEvent", err)
		return err
	}
	return nil
}

// ===================== Participants (user_events) =====================

func (r *MeetingRepository) AddParticipant(ctx context.Context, userEvent *entity.UserEvent) error {
	query := `
		INSERT INTO user_events (user_id, event_id, status, has_calendar_connected)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, event_id) DO UPDATE SET status = $3, has_calendar_connected = $4
	`

	err := r.DB.ExecContext(ctx, query,
		userEvent.UserID, userEvent.EventID, userEvent.Status, userEvent.HasCalendarConnected)
	if err != nil {
		logger.Error("MeetingRepository:AddParticipant", err)
		return err
	}

	return nil
}

func (r *MeetingRepository) GetParticipantsByEventID(ctx context.Context, eventID uuid.UUID) ([]entity.UserEvent, error) {
	query := `
		SELECT user_id, event_id, COALESCE(status, 'pending') as status, 
		       COALESCE(has_calendar_connected, false) as has_calendar_connected, created_at
		FROM user_events
		WHERE event_id = $1
		ORDER BY created_at
	`

	var participants []entity.UserEvent
	err := r.DB.SelectContext(ctx, &participants, query, eventID)
	if err != nil {
		logger.Error("MeetingRepository:GetParticipantsByEventID", err)
		return nil, err
	}

	return participants, nil
}

func (r *MeetingRepository) UpdateParticipantCalendarStatus(ctx context.Context, userID uuid.UUID, eventID uuid.UUID, hasCalendar bool) error {
	query := `UPDATE user_events SET has_calendar_connected = $3 WHERE user_id = $1 AND event_id = $2`
	err := r.DB.ExecContext(ctx, query, userID, eventID, hasCalendar)
	if err != nil {
		logger.Error("MeetingRepository:UpdateParticipantCalendarStatus", err)
		return err
	}
	return nil
}

func (r *MeetingRepository) RemoveParticipant(ctx context.Context, userID uuid.UUID, eventID uuid.UUID) error {
	query := `DELETE FROM user_events WHERE user_id = $1 AND event_id = $2`
	err := r.DB.ExecContext(ctx, query, userID, eventID)
	if err != nil {
		logger.Error("MeetingRepository:RemoveParticipant", err)
		return err
	}
	return nil
}

// ===================== Slots (event_slots) =====================

func (r *MeetingRepository) SaveSlots(ctx context.Context, slots []entity.EventSlot) error {
	query := `
		INSERT INTO event_slots (event_id, start_time, end_time, available_count, total_participants, score)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, slot := range slots {
		err := r.DB.ExecContext(ctx, query,
			slot.EventID, slot.StartTime, slot.EndTime,
			slot.AvailableCount, slot.TotalParticipants, slot.Score)
		if err != nil {
			logger.Error("MeetingRepository:SaveSlots", err)
			return err
		}
	}

	return nil
}

func (r *MeetingRepository) GetSlotsByEventID(ctx context.Context, eventID uuid.UUID) ([]entity.EventSlot, error) {
	query := `
		SELECT id, event_id, start_time, end_time, available_count, total_participants, score, created_at
		FROM event_slots
		WHERE event_id = $1
		ORDER BY score DESC, start_time ASC
	`

	var slots []entity.EventSlot
	err := r.DB.SelectContext(ctx, &slots, query, eventID)
	if err != nil {
		logger.Error("MeetingRepository:GetSlotsByEventID", err)
		return nil, err
	}

	return slots, nil
}

func (r *MeetingRepository) ClearSlotsByEventID(ctx context.Context, eventID uuid.UUID) error {
	query := `DELETE FROM event_slots WHERE event_id = $1`
	err := r.DB.ExecContext(ctx, query, eventID)
	if err != nil {
		logger.Error("MeetingRepository:ClearSlotsByEventID", err)
		return err
	}
	return nil
}
