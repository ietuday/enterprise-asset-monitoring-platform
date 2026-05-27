package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strconv"
	"strings"

	"notification-service/internal/models"
	"notification-service/internal/sender"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrNotFound          = errors.New("resource not found")
	ErrRetryNotAllowed   = errors.New("only FAILED notifications can be retried")
	ErrUnsupportedStatus = errors.New("invalid notification status")
)

type ChannelRepository interface {
	CreateChannel(ctx context.Context, channel *models.NotificationChannel) error
	ListChannels(ctx context.Context) ([]models.NotificationChannel, error)
	GetChannelByID(ctx context.Context, id string) (*models.NotificationChannel, error)
	UpdateChannel(ctx context.Context, channel *models.NotificationChannel) error
	DeleteChannel(ctx context.Context, id string) error
	SetChannelEnabled(ctx context.Context, id string, enabled bool) (*models.NotificationChannel, error)
	ListEnabledChannels(ctx context.Context) ([]models.NotificationChannel, error)
}

type HistoryRepository interface {
	CreateHistory(ctx context.Context, history *models.NotificationHistory) error
	ListHistory(ctx context.Context, filters models.HistoryFilters) ([]models.NotificationHistory, error)
	GetHistoryByID(ctx context.Context, id string) (*models.NotificationHistory, error)
	UpdateHistoryStatus(ctx context.Context, id int64, status string, errorMessage *string) error
	IncrementRetryCount(ctx context.Context, id int64) error
	ListFailedHistory(ctx context.Context) ([]models.NotificationHistory, error)
}

type NotificationService struct {
	channels ChannelRepository
	history  HistoryRepository
	sender   sender.Sender
}

func NewNotificationService(channels ChannelRepository, history HistoryRepository, sender sender.Sender) *NotificationService {
	return &NotificationService{
		channels: channels,
		history:  history,
		sender:   sender,
	}
}

func (s *NotificationService) CreateChannel(ctx context.Context, channel *models.NotificationChannel) error {
	if err := ValidateChannel(channel); err != nil {
		return err
	}

	return mapNoRows(s.channels.CreateChannel(ctx, channel))
}

func (s *NotificationService) ListChannels(ctx context.Context) ([]models.NotificationChannel, error) {
	return s.channels.ListChannels(ctx)
}

func (s *NotificationService) GetChannelByID(ctx context.Context, id string) (*models.NotificationChannel, error) {
	channel, err := s.channels.GetChannelByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}

	return channel, err
}

func (s *NotificationService) UpdateChannel(ctx context.Context, channel *models.NotificationChannel) error {
	if err := ValidateChannel(channel); err != nil {
		return err
	}

	return mapNoRows(s.channels.UpdateChannel(ctx, channel))
}

func (s *NotificationService) DeleteChannel(ctx context.Context, id string) error {
	return mapNoRows(s.channels.DeleteChannel(ctx, id))
}

func (s *NotificationService) SetChannelEnabled(ctx context.Context, id string, enabled bool) (*models.NotificationChannel, error) {
	channel, err := s.channels.SetChannelEnabled(ctx, id, enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}

	return channel, err
}

func (s *NotificationService) Send(ctx context.Context, req models.SendNotificationRequest) (models.SendSummary, error) {
	if err := ValidateSendRequest(req); err != nil {
		return models.SendSummary{}, err
	}

	channels, err := s.channels.ListEnabledChannels(ctx)
	if err != nil {
		return models.SendSummary{}, err
	}

	summary := models.SendSummary{
		Total:   len(channels),
		Results: make([]models.SendResult, 0, len(channels)),
	}

	for _, channel := range channels {
		result := s.sendToChannel(ctx, channel, req, 0)
		if result.Status == models.NotificationStatusSent {
			summary.Sent++
		} else {
			summary.Failed++
		}
		summary.Results = append(summary.Results, result)
	}

	return summary, nil
}

func (s *NotificationService) Test(ctx context.Context, req models.TestNotificationRequest) (models.SendResult, error) {
	if req.ChannelID == 0 || strings.TrimSpace(req.Subject) == "" || strings.TrimSpace(req.Message) == "" {
		return models.SendResult{}, fmt.Errorf("%w: channel_id, subject and message are required", ErrInvalidInput)
	}

	channel, err := s.GetChannelByID(ctx, strconv.FormatInt(req.ChannelID, 10))
	if err != nil {
		return models.SendResult{}, err
	}
	if !channel.Enabled {
		return models.SendResult{}, fmt.Errorf("%w: channel is disabled", ErrInvalidInput)
	}

	sendReq := models.SendNotificationRequest{
		EventType: models.EventTestNotification,
		Subject:   req.Subject,
		Message:   req.Message,
	}

	return s.sendToChannel(ctx, *channel, sendReq, 0), nil
}

func (s *NotificationService) ListHistory(ctx context.Context, filters models.HistoryFilters) ([]models.NotificationHistory, error) {
	normalizeHistoryFilters(&filters)
	if filters.Status != "" && !isValidStatus(filters.Status) {
		return nil, ErrUnsupportedStatus
	}
	if filters.ChannelType != "" && !isValidChannelType(filters.ChannelType) {
		return nil, fmt.Errorf("%w: invalid channel_type", ErrInvalidInput)
	}

	return s.history.ListHistory(ctx, filters)
}

func (s *NotificationService) GetHistoryByID(ctx context.Context, id string) (*models.NotificationHistory, error) {
	item, err := s.history.GetHistoryByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}

	return item, err
}

func (s *NotificationService) Retry(ctx context.Context, id string) (models.SendResult, error) {
	item, err := s.GetHistoryByID(ctx, id)
	if err != nil {
		return models.SendResult{}, err
	}
	if item.Status != models.NotificationStatusFailed {
		return models.SendResult{}, ErrRetryNotAllowed
	}

	if err := s.history.IncrementRetryCount(ctx, item.ID); err != nil {
		return models.SendResult{}, mapNoRows(err)
	}

	channel := models.NotificationChannel{
		ID:      0,
		Name:    item.ChannelName,
		Type:    item.ChannelType,
		Target:  item.Recipient,
		Enabled: true,
	}
	if item.ChannelID != nil {
		channel.ID = *item.ChannelID
	}

	req := models.SendNotificationRequest{
		EventType: item.EventType,
		Subject:   item.Subject,
		Message:   item.Message,
		Payload:   item.Payload,
	}

	result := s.attemptSend(ctx, channel, req, item.ID)
	result.HistoryID = item.ID
	return result, nil
}

func (s *NotificationService) sendToChannel(ctx context.Context, channel models.NotificationChannel, req models.SendNotificationRequest, retryCount int) models.SendResult {
	channelID := channel.ID
	history := models.NotificationHistory{
		EventType:   req.EventType,
		ChannelID:   &channelID,
		ChannelName: channel.Name,
		ChannelType: channel.Type,
		Recipient:   channel.Target,
		Subject:     req.Subject,
		Message:     req.Message,
		Payload:     req.Payload,
		Status:      models.NotificationStatusPending,
		RetryCount:  retryCount,
	}

	if err := s.history.CreateHistory(ctx, &history); err != nil {
		errMsg := err.Error()
		return models.SendResult{
			ChannelID:    channel.ID,
			ChannelName:  channel.Name,
			ChannelType:  channel.Type,
			Recipient:    channel.Target,
			Status:       models.NotificationStatusFailed,
			ErrorMessage: errMsg,
		}
	}

	return s.attemptSend(ctx, channel, req, history.ID)
}

func (s *NotificationService) attemptSend(ctx context.Context, channel models.NotificationChannel, req models.SendNotificationRequest, historyID int64) models.SendResult {
	result := models.SendResult{
		ChannelID:   channel.ID,
		ChannelName: channel.Name,
		ChannelType: channel.Type,
		Recipient:   channel.Target,
		HistoryID:   historyID,
	}

	if err := s.sender.Send(ctx, channel, req); err != nil {
		errMsg := err.Error()
		if updateErr := s.history.UpdateHistoryStatus(ctx, historyID, models.NotificationStatusFailed, &errMsg); updateErr != nil {
			errMsg = errMsg + "; failed to update history: " + updateErr.Error()
		}
		result.Status = models.NotificationStatusFailed
		result.ErrorMessage = errMsg
		return result
	}

	if err := s.history.UpdateHistoryStatus(ctx, historyID, models.NotificationStatusSent, nil); err != nil {
		errMsg := err.Error()
		result.Status = models.NotificationStatusFailed
		result.ErrorMessage = errMsg
		return result
	}

	result.Status = models.NotificationStatusSent
	return result
}

func ValidateChannel(channel *models.NotificationChannel) error {
	channel.Name = strings.TrimSpace(channel.Name)
	channel.Type = strings.ToUpper(strings.TrimSpace(channel.Type))
	channel.Target = strings.TrimSpace(channel.Target)

	if channel.Name == "" || channel.Type == "" || channel.Target == "" {
		return fmt.Errorf("%w: name, type and target are required", ErrInvalidInput)
	}
	if !isValidChannelType(channel.Type) {
		return fmt.Errorf("%w: type must be EMAIL or WEBHOOK", ErrInvalidInput)
	}
	if channel.Type == models.ChannelTypeEmail {
		if _, err := mail.ParseAddress(channel.Target); err != nil {
			return fmt.Errorf("%w: EMAIL target must be a valid email address", ErrInvalidInput)
		}
	}
	if channel.Type == models.ChannelTypeWebhook {
		parsed, err := url.ParseRequestURI(channel.Target)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return fmt.Errorf("%w: WEBHOOK target must be a valid http:// or https:// URL", ErrInvalidInput)
		}
	}

	return nil
}

func ValidateSendRequest(req models.SendNotificationRequest) error {
	if strings.TrimSpace(req.EventType) == "" || strings.TrimSpace(req.Subject) == "" || strings.TrimSpace(req.Message) == "" {
		return fmt.Errorf("%w: event_type, subject and message are required", ErrInvalidInput)
	}

	if len(req.Payload) > 0 && !json.Valid(req.Payload) {
		return fmt.Errorf("%w: payload must be valid JSON", ErrInvalidInput)
	}

	return nil
}

func normalizeHistoryFilters(filters *models.HistoryFilters) {
	filters.Status = strings.ToUpper(strings.TrimSpace(filters.Status))
	filters.ChannelType = strings.ToUpper(strings.TrimSpace(filters.ChannelType))
	filters.EventType = strings.ToUpper(strings.TrimSpace(filters.EventType))
}

func isValidChannelType(value string) bool {
	return value == models.ChannelTypeEmail || value == models.ChannelTypeWebhook
}

func isValidStatus(value string) bool {
	return value == models.NotificationStatusPending ||
		value == models.NotificationStatusSent ||
		value == models.NotificationStatusFailed
}

func mapNoRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	return err
}
