package service

import (
	"context"
	"encoding/json"
	"go-api-starter/core/errors"
	"go-api-starter/modules/meeting/dto"
	"go-api-starter/modules/meeting/entity"
	"go-api-starter/modules/meeting/repository"
	"time"

	"github.com/google/uuid"
)

// MeetingService handles event business logic
type MeetingService struct {
	repo       repository.MeetingRepositoryInterface
	slotFinder *SlotFinder
}

// MeetingServiceInterface defines the service contract
type MeetingServiceInterface interface {
	CreateEvent(ctx context.Context, hostID uuid.UUID, req *dto.CreateEventRequest) (*dto.EventResponse, *errors.AppError)
	GetEventByID(ctx context.Context, id uuid.UUID) (*dto.EventResponse, *errors.AppError)
	GetMyEvents(ctx context.Context, hostID uuid.UUID) ([]dto.EventResponse, *errors.AppError)
	UpdateEvent(ctx context.Context, eventID uuid.UUID, hostID uuid.UUID, req *dto.UpdateEventRequest) (*dto.EventResponse, *errors.AppError)
	DeleteEvent(ctx context.Context, eventID uuid.UUID, hostID uuid.UUID) *errors.AppError
	FindSlots(ctx context.Context, eventID uuid.UUID, req *dto.FindSlotsRequest) (*dto.FindSlotsResponse, *errors.AppError)
	SelectSlot(ctx context.Context, eventID uuid.UUID, hostID uuid.UUID, req *dto.SelectSlotRequest) (*dto.EventResponse, *errors.AppError)
}

// NewMeetingService creates a new meeting service
func NewMeetingService(repo repository.MeetingRepositoryInterface) MeetingServiceInterface {
	return &MeetingService{
		repo:       repo,
		slotFinder: NewSlotFinder(),
	}
}

// CreateEvent creates a new event with participants
func (s *MeetingService) CreateEvent(ctx context.Context, hostID uuid.UUID, req *dto.CreateEventRequest) (*dto.EventResponse, *errors.AppError) {
	// Prepare preferences JSON
	var preferencesJSON *string
	if req.Preferences != nil {
		jsonBytes, _ := json.Marshal(req.Preferences)
		jsonStr := string(jsonBytes)
		preferencesJSON = &jsonStr
	}

	// Create event entity
	event := &entity.Event{
		HostID:          &hostID,
		Title:           req.Title,
		DurationMinutes: req.DurationMinutes,
		Status:          entity.EventStatusPending,
		Timezone:        "Asia/Ho_Chi_Minh",
		Preferences:     preferencesJSON,
	}

	if req.Description != "" {
		event.Description = &req.Description
	}
	if req.Address != "" {
		event.Address = &req.Address
	}

	// Save event
	created, err := s.repo.CreateEvent(ctx, event)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInternalServer, "Failed to create event", err)
	}

	// Add participants
	participants := make([]entity.UserEvent, 0)
	for _, userIDStr := range req.Participants {
		userID, parseErr := uuid.Parse(userIDStr)
		if parseErr != nil {
			continue
		}

		participant := &entity.UserEvent{
			UserID:               userID,
			EventID:              created.ID,
			Status:               entity.ParticipantStatusPending,
			HasCalendarConnected: false,
		}

		err := s.repo.AddParticipant(ctx, participant)
		if err != nil {
			continue
		}
		participants = append(participants, *participant)
	}

	return dto.ToEventResponse(created, participants), nil
}

// GetEventByID retrieves an event by ID
func (s *MeetingService) GetEventByID(ctx context.Context, id uuid.UUID) (*dto.EventResponse, *errors.AppError) {
	event, err := s.repo.GetEventByID(ctx, id)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInternalServer, "Failed to get event", err)
	}
	if event == nil {
		return nil, errors.NewAppError(errors.ErrNotFound, "Event not found", nil)
	}

	participants, _ := s.repo.GetParticipantsByEventID(ctx, id)
	return dto.ToEventResponse(event, participants), nil
}

// GetMyEvents retrieves all events for a host
func (s *MeetingService) GetMyEvents(ctx context.Context, hostID uuid.UUID) ([]dto.EventResponse, *errors.AppError) {
	events, err := s.repo.GetEventsByHostID(ctx, hostID)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInternalServer, "Failed to get events", err)
	}

	result := make([]dto.EventResponse, 0, len(events))
	for _, e := range events {
		participants, _ := s.repo.GetParticipantsByEventID(ctx, e.ID)
		result = append(result, *dto.ToEventResponse(&e, participants))
	}

	return result, nil
}

// UpdateEvent updates event details
func (s *MeetingService) UpdateEvent(ctx context.Context, eventID uuid.UUID, hostID uuid.UUID, req *dto.UpdateEventRequest) (*dto.EventResponse, *errors.AppError) {
	event, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil || event == nil {
		return nil, errors.NewAppError(errors.ErrNotFound, "Event not found", err)
	}

	if event.HostID == nil || *event.HostID != hostID {
		return nil, errors.NewAppError(errors.ErrForbidden, "Not authorized", nil)
	}

	// Update fields
	if req.Title != "" {
		event.Title = req.Title
	}
	if req.Description != "" {
		event.Description = &req.Description
	}
	if req.Address != "" {
		event.Address = &req.Address
	}
	if req.DurationMinutes > 0 {
		event.DurationMinutes = req.DurationMinutes
	}

	err = s.repo.UpdateEvent(ctx, event)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInternalServer, "Failed to update event", err)
	}

	return s.GetEventByID(ctx, eventID)
}

// DeleteEvent deletes an event
func (s *MeetingService) DeleteEvent(ctx context.Context, eventID uuid.UUID, hostID uuid.UUID) *errors.AppError {
	event, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil || event == nil {
		return errors.NewAppError(errors.ErrNotFound, "Event not found", err)
	}

	if event.HostID == nil || *event.HostID != hostID {
		return errors.NewAppError(errors.ErrForbidden, "Not authorized", nil)
	}

	err = s.repo.DeleteEvent(ctx, eventID)
	if err != nil {
		return errors.NewAppError(errors.ErrInternalServer, "Failed to delete event", err)
	}

	return nil
}

// FindSlots finds available time slots for an event
func (s *MeetingService) FindSlots(ctx context.Context, eventID uuid.UUID, req *dto.FindSlotsRequest) (*dto.FindSlotsResponse, *errors.AppError) {
	event, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil || event == nil {
		return nil, errors.NewAppError(errors.ErrNotFound, "Event not found", err)
	}

	participants, _ := s.repo.GetParticipantsByEventID(ctx, eventID)

	// Determine search range
	var searchStart, searchEnd time.Time
	now := time.Now()

	if req.SearchStartDate != "" {
		searchStart, _ = time.Parse("2006-01-02", req.SearchStartDate)
	} else {
		searchStart = now
	}

	if req.SearchEndDate != "" {
		searchEnd, _ = time.Parse("2006-01-02", req.SearchEndDate)
	} else if req.SearchDays > 0 {
		searchEnd = now.AddDate(0, 0, req.SearchDays)
	} else {
		searchEnd = now.AddDate(0, 0, 7)
	}

	// TODO: Get actual busy times from Calendar API
	busyTimes := []entity.TimeSlot{}

	// Parse preferences
	var preferences *entity.EventPreferences
	if event.Preferences != nil {
		preferences = &entity.EventPreferences{}
		_ = json.Unmarshal([]byte(*event.Preferences), preferences)
	}

	// Find slots using algorithm
	slots := s.slotFinder.FindAvailableSlots(
		event.DurationMinutes,
		searchStart,
		searchEnd,
		busyTimes,
		preferences,
		len(participants),
	)

	// Clear old slots and save new ones
	_ = s.repo.ClearSlotsByEventID(ctx, eventID)

	for i := range slots {
		slots[i].EventID = eventID
	}
	_ = s.repo.SaveSlots(ctx, slots)

	// Build response
	response := &dto.FindSlotsResponse{
		EventID:      eventID.String(),
		Slots:        make([]dto.SuggestedSlotDTO, 0, len(slots)),
		Participants: make([]dto.ParticipantResponse, 0, len(participants)),
	}

	for _, slot := range slots {
		response.Slots = append(response.Slots, *dto.ToSlotDTO(&slot))
	}

	for _, p := range participants {
		response.Participants = append(response.Participants, dto.ParticipantResponse{
			UserID:               p.UserID.String(),
			EventID:              p.EventID.String(),
			Status:               string(p.Status),
			HasCalendarConnected: p.HasCalendarConnected,
		})
	}

	return response, nil
}

// SelectSlot confirms a slot for the event
func (s *MeetingService) SelectSlot(ctx context.Context, eventID uuid.UUID, hostID uuid.UUID, req *dto.SelectSlotRequest) (*dto.EventResponse, *errors.AppError) {
	event, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil || event == nil {
		return nil, errors.NewAppError(errors.ErrNotFound, "Event not found", err)
	}

	if event.HostID == nil || *event.HostID != hostID {
		return nil, errors.NewAppError(errors.ErrForbidden, "Not authorized", nil)
	}

	// Parse slot time
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInvalidInput, "Invalid start time format", err)
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInvalidInput, "Invalid end time format", err)
	}

	// Update event
	event.StartDate = &startTime
	event.EndDate = &endTime
	event.Status = entity.EventStatusScheduled

	err = s.repo.UpdateEvent(ctx, event)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInternalServer, "Failed to schedule event", err)
	}

	return s.GetEventByID(ctx, eventID)
}
