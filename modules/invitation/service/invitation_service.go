package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go-api-starter/core/logger"
	authRepo "go-api-starter/modules/auth/repository"
	"go-api-starter/modules/invitation/dto"
	"go-api-starter/modules/invitation/entity"
	"go-api-starter/modules/invitation/repository"
	notifDto "go-api-starter/modules/notification/dto"
	notifService "go-api-starter/modules/notification/service"

	"github.com/google/uuid"
)

type InvitationService struct {
	repo         *repository.InvitationRepository
	notifService *notifService.NotificationService
	authRepo     authRepo.AuthRepositoryInterface
}

func NewInvitationService(repo *repository.InvitationRepository, notifService *notifService.NotificationService, authRepo authRepo.AuthRepositoryInterface) *InvitationService {
	return &InvitationService{
		repo:         repo,
		notifService: notifService,
		authRepo:     authRepo,
	}
}

// CreateInvitations creates invitations for all invitees and sends notifications
func (s *InvitationService) CreateInvitations(ctx context.Context, req *dto.CreateInvitationRequest) error {
	for _, inviteeID := range req.InviteeIDs {
		// Skip if invitee is the creator
		if inviteeID == req.CreatorID {
			continue
		}

		invitation := &entity.EventInvitation{
			EventGoogleID: req.EventGoogleID,
			CreatorID:     req.CreatorID,
			InviteeID:     inviteeID,
			Status:        entity.InvitationStatusPending,
			EventData: entity.EventData{
				Title:       req.EventData.Title,
				Description: req.EventData.Description,
				StartTime:   req.EventData.StartTime,
				EndTime:     req.EventData.EndTime,
				Location:    req.EventData.Location,
				MeetingLink: req.EventData.MeetingLink,
			},
		}

		if err := s.repo.Create(ctx, invitation); err != nil {
			logger.Error("InvitationService:CreateInvitations:Create:Error:", err)
			continue // Don't fail entire operation for one invitee
		}

		// Create notification for invitee
		notification := &notifDto.CreateNotificationRequest{
			UserID:  inviteeID,
			Title:   "Lời mời sự kiện mới",
			Message: fmt.Sprintf("Bạn được mời tham gia sự kiện: %s", req.EventData.Title),
			Type:    "invitation",
			Data: map[string]interface{}{
				"invitation_id": invitation.ID.String(),
				"event_id":      req.EventGoogleID,
			},
		}

		if err := s.notifService.Create(ctx, notification); err != nil {
			logger.Error("InvitationService:CreateInvitations:Notify:Error:", err)
		}
	}

	return nil
}

// GetPendingInvitations returns pending invitations for a user
func (s *InvitationService) GetPendingInvitations(ctx context.Context, userID uuid.UUID) (*dto.PendingInvitationsResponse, error) {
	logger.Info("InvitationService:GetPendingInvitations:Start", "user_id", userID)
	invitations, err := s.repo.GetPendingByInviteeID(ctx, userID)
	if err != nil {
		logger.Error("InvitationService:GetPendingInvitations:RepoError", "error", err)
		return nil, err
	}

	logger.Info("InvitationService:GetPendingInvitations:GotInvitations", "count", len(invitations))

	var dtos []dto.InvitationResponse
	for _, inv := range invitations {
		// Convert entity.EventData to dto.EventDataDTO
		eventData := dto.EventDataDTO{
			Title:       inv.EventData.Title,
			Description: inv.EventData.Description,
			StartTime:   inv.EventData.StartTime,
			EndTime:     inv.EventData.EndTime,
			Location:    inv.EventData.Location,
			MeetingLink: inv.EventData.MeetingLink,
			Timezone:    inv.EventData.Timezone,
		}

		dtos = append(dtos, dto.InvitationResponse{
			ID:            inv.ID,
			EventGoogleID: inv.EventGoogleID,
			CreatorID:     inv.CreatorID,
			Status:        string(inv.Status),
			EventData:     eventData,
			CreatedAt:     inv.CreatedAt,
		})
	}

	return &dto.PendingInvitationsResponse{
		Invitations: dtos,
		Total:       len(dtos),
	}, nil
}

// CountPending counts pending invitations for a user
func (s *InvitationService) CountPending(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountPendingByInviteeID(ctx, userID)
}

// AcceptInvitation accepts an invitation
func (s *InvitationService) AcceptInvitation(ctx context.Context, invitationID uuid.UUID, userID uuid.UUID) (*entity.EventInvitation, error) {
	invitation, err := s.repo.GetByID(ctx, invitationID)
	if err != nil {
		return nil, err
	}

	if invitation.InviteeID != userID {
		return nil, fmt.Errorf("unauthorized: not the invitee")
	}

	if invitation.Status != entity.InvitationStatusPending {
		return nil, fmt.Errorf("invitation already responded")
	}

	if err := s.repo.UpdateStatus(ctx, invitationID, string(entity.InvitationStatusAccepted)); err != nil {
		return nil, err
	}

	invitation.Status = entity.InvitationStatusAccepted

	// Sync to Google Calendar
	go func() {
		// Use a background context or a new context with timeout
		bgCtx := context.Background()
		if err := s.updateGoogleEventStatus(bgCtx, userID, invitation.EventGoogleID, "accepted"); err != nil {
			logger.Error("AcceptInvitation:GoogleSync:Error", "error", err, "event_id", invitation.EventGoogleID)
		}
	}()

	return invitation, nil
}

// DeclineInvitation declines an invitation
func (s *InvitationService) DeclineInvitation(ctx context.Context, invitationID uuid.UUID, userID uuid.UUID) error {
	invitation, err := s.repo.GetByID(ctx, invitationID)
	if err != nil {
		return err
	}

	if invitation.InviteeID != userID {
		return fmt.Errorf("unauthorized: not the invitee")
	}

	if invitation.Status != entity.InvitationStatusPending {
		return fmt.Errorf("invitation already responded")
	}

	if err := s.repo.UpdateStatus(ctx, invitationID, string(entity.InvitationStatusDeclined)); err != nil {
		return err
	}

	// Sync to Google Calendar
	go func() {
		bgCtx := context.Background()
		if err := s.updateGoogleEventStatus(bgCtx, userID, invitation.EventGoogleID, "declined"); err != nil {
			logger.Error("DeclineInvitation:GoogleSync:Error", "error", err, "event_id", invitation.EventGoogleID)
		}
	}()

	return nil
}

// updateGoogleEventStatus updates the attendee response status on Google Calendar
func (s *InvitationService) updateGoogleEventStatus(ctx context.Context, userID uuid.UUID, eventGoogleID string, status string) error {
	logger.Info("updateGoogleEventStatus:Start", "user_id", userID, "event_id", eventGoogleID, "status", status)

	// 1. Get Google provider and social login data (contains email and token)
	provider, err := s.authRepo.GetOAuthProviderByName(ctx, "google")
	if err != nil || provider == nil {
		return fmt.Errorf("google provider not found")
	}

	socialData, err := s.authRepo.GetSocialLoginByUserIDAndProvider(ctx, userID, provider.ID)
	if err != nil || socialData == nil {
		return fmt.Errorf("google social login not found")
	}

	if socialData.AccessToken == nil {
		return fmt.Errorf("google token not found")
	}
	token := *socialData.AccessToken

	// Get email from social login
	if socialData.ProviderEmail == nil {
		return fmt.Errorf("google email not found in social login")
	}
	email := *socialData.ProviderEmail
	logger.Info("updateGoogleEventStatus:UserEmail", "email", email)

	// 3. GET current event to retrieve all attendees
	eventURL := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/primary/events/%s", eventGoogleID)

	getReq, _ := http.NewRequest("GET", eventURL, nil)
	getReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	getResp, err := client.Do(getReq)
	if err != nil {
		logger.Error("updateGoogleEventStatus:GET:Error", "error", err)
		return err
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != 200 {
		body, _ := io.ReadAll(getResp.Body)
		logger.Error("updateGoogleEventStatus:GET:APIError", "status", getResp.StatusCode, "body", string(body))
		return fmt.Errorf("google api GET returned status: %d", getResp.StatusCode)
	}

	var eventData map[string]interface{}
	if err := json.NewDecoder(getResp.Body).Decode(&eventData); err != nil {
		return err
	}

	// 4. Update the specific attendee's responseStatus
	attendees, ok := eventData["attendees"].([]interface{})
	if !ok {
		logger.Warn("updateGoogleEventStatus:NoAttendees")
		return fmt.Errorf("no attendees in event")
	}

	found := false
	for _, att := range attendees {
		attendee, ok := att.(map[string]interface{})
		if !ok {
			continue
		}
		if attendee["email"] == email {
			attendee["responseStatus"] = status
			found = true
			logger.Info("updateGoogleEventStatus:FoundAttendee", "email", email, "new_status", status)
			break
		}
	}

	if !found {
		logger.Warn("updateGoogleEventStatus:AttendeeNotFound", "email", email)
		return fmt.Errorf("attendee %s not found in event", email)
	}

	// 5. PATCH event with updated attendees
	patchPayload := map[string]interface{}{
		"attendees": attendees,
	}

	jsonBody, _ := json.Marshal(patchPayload)
	patchReq, _ := http.NewRequest("PATCH", eventURL, bytes.NewBuffer(jsonBody))
	patchReq.Header.Set("Authorization", "Bearer "+token)
	patchReq.Header.Set("Content-Type", "application/json")

	patchResp, err := client.Do(patchReq)
	if err != nil {
		logger.Error("updateGoogleEventStatus:PATCH:Error", "error", err)
		return err
	}
	defer patchResp.Body.Close()

	if patchResp.StatusCode != 200 {
		body, _ := io.ReadAll(patchResp.Body)
		logger.Error("updateGoogleEventStatus:PATCH:APIError", "status", patchResp.StatusCode, "body", string(body))
		return fmt.Errorf("google api PATCH returned status: %d", patchResp.StatusCode)
	}

	logger.Info("updateGoogleEventStatus:Success", "event_id", eventGoogleID, "email", email, "status", status)
	return nil
}
