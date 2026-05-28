package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"notification-service/internal/models"
	"notification-service/internal/sender"

	"github.com/jackc/pgx/v5"
)

func TestCreateChannelValidation(t *testing.T) {
	repo := newFakeChannelRepository()
	history := newFakeHistoryRepository()
	service := NewNotificationService(repo, history, sender.NewRegistry(sender.NewEmailSender(sender.EmailConfig{}), sender.NewWebhookSender(time.Second)))

	channel := models.NotificationChannel{
		Name:    "Ops Webhook",
		Type:    "WEBHOOK",
		Target:  "https://example.com/hooks/alerts",
		Enabled: true,
	}
	if err := service.CreateChannel(context.Background(), &channel); err != nil {
		t.Fatalf("expected valid channel to succeed: %v", err)
	}
	if channel.ID == 0 {
		t.Fatal("expected channel ID to be assigned")
	}

	assertInvalidChannel(t, service, models.NotificationChannel{Name: "Bad", Type: "SMS", Target: "x", Enabled: true})
	assertInvalidChannel(t, service, models.NotificationChannel{Name: "Bad", Type: "WEBHOOK", Target: "ftp://example.com", Enabled: true})
	assertInvalidChannel(t, service, models.NotificationChannel{Name: "Bad", Type: "EMAIL", Target: "not-an-email", Enabled: true})
}

func TestListEnabledChannels(t *testing.T) {
	repo := newFakeChannelRepository()
	ctx := context.Background()

	enabled := models.NotificationChannel{Name: "Enabled", Type: models.ChannelTypeWebhook, Target: "https://example.com/enabled", Enabled: true}
	disabled := models.NotificationChannel{Name: "Disabled", Type: models.ChannelTypeWebhook, Target: "https://example.com/disabled", Enabled: false}
	if err := repo.CreateChannel(ctx, &enabled); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateChannel(ctx, &disabled); err != nil {
		t.Fatal(err)
	}

	channels, err := repo.ListEnabledChannels(ctx)
	if err != nil {
		t.Fatalf("expected list to succeed: %v", err)
	}
	if len(channels) != 1 || channels[0].Name != "Enabled" {
		t.Fatalf("expected only enabled channel, got %+v", channels)
	}
}

func TestSendWebhookNotificationCreatesSentHistory(t *testing.T) {
	webhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer webhook.Close()

	service, _, history := testServiceWithWebhook(t, webhook.URL)

	summary, err := service.Send(context.Background(), models.SendNotificationRequest{
		EventType: models.EventIncidentCreated,
		Subject:   "Incident created",
		Message:   "Incident #1 created",
	})
	if err != nil {
		t.Fatalf("expected send to succeed: %v", err)
	}
	if summary.Total != 1 || summary.Sent != 1 || summary.Failed != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if len(history.items) != 1 || history.items[1].Status != models.NotificationStatusSent {
		t.Fatalf("expected SENT history, got %+v", history.items)
	}
}

func TestSendWebhookNotificationCreatesFailedHistory(t *testing.T) {
	webhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer webhook.Close()

	service, _, history := testServiceWithWebhook(t, webhook.URL)

	summary, err := service.Send(context.Background(), models.SendNotificationRequest{
		EventType: models.EventIncidentCreated,
		Subject:   "Incident created",
		Message:   "Incident #1 created",
	})
	if err != nil {
		t.Fatalf("expected send to return summary, got error: %v", err)
	}
	if summary.Total != 1 || summary.Sent != 0 || summary.Failed != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if len(history.items) != 1 || history.items[1].Status != models.NotificationStatusFailed || history.items[1].ErrorMessage == nil {
		t.Fatalf("expected FAILED history with error, got %+v", history.items)
	}
}

func TestRetryFailedNotificationIncrementsRetryCount(t *testing.T) {
	webhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer webhook.Close()

	service, channels, history := testServiceWithWebhook(t, webhook.URL)
	channel := channels.channels[1]
	channelID := channel.ID
	history.items[1] = models.NotificationHistory{
		ID:          1,
		EventType:   models.EventIncidentResolved,
		ChannelID:   &channelID,
		ChannelName: channel.Name,
		ChannelType: channel.Type,
		Recipient:   channel.Target,
		Subject:     "Incident resolved",
		Message:     "Incident #1 resolved",
		Status:      models.NotificationStatusFailed,
	}

	result, err := service.Retry(context.Background(), "1")
	if err != nil {
		t.Fatalf("expected retry to succeed: %v", err)
	}
	if result.Status != models.NotificationStatusSent {
		t.Fatalf("expected retry SENT, got %+v", result)
	}
	if history.items[1].RetryCount != 1 {
		t.Fatalf("expected retry count 1, got %d", history.items[1].RetryCount)
	}
}

func TestHistoryFilters(t *testing.T) {
	history := newFakeHistoryRepository()
	history.items[1] = models.NotificationHistory{ID: 1, Status: models.NotificationStatusFailed, ChannelType: models.ChannelTypeWebhook, EventType: models.EventIncidentCreated}
	history.items[2] = models.NotificationHistory{ID: 2, Status: models.NotificationStatusSent, ChannelType: models.ChannelTypeEmail, EventType: models.EventIncidentClosed}
	service := NewNotificationService(newFakeChannelRepository(), history, fakeSender{})

	items, err := service.ListHistory(context.Background(), models.HistoryFilters{
		Status:      "FAILED",
		ChannelType: "WEBHOOK",
		EventType:   models.EventIncidentCreated,
	})
	if err != nil {
		t.Fatalf("expected filters to succeed: %v", err)
	}
	if len(items) != 1 || items[0].ID != 1 {
		t.Fatalf("expected failed webhook incident-created history, got %+v", items)
	}
}

func assertInvalidChannel(t *testing.T, service *NotificationService, channel models.NotificationChannel) {
	t.Helper()
	if err := service.CreateChannel(context.Background(), &channel); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for %+v, got %v", channel, err)
	}
}

func testServiceWithWebhook(t *testing.T, target string) (*NotificationService, *fakeChannelRepository, *fakeHistoryRepository) {
	t.Helper()
	channels := newFakeChannelRepository()
	history := newFakeHistoryRepository()
	channel := models.NotificationChannel{
		Name:    "Ops Webhook",
		Type:    models.ChannelTypeWebhook,
		Target:  target,
		Enabled: true,
	}
	if err := channels.CreateChannel(context.Background(), &channel); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	service := NewNotificationService(
		channels,
		history,
		sender.NewRegistry(sender.NewEmailSender(sender.EmailConfig{}), sender.NewWebhookSender(time.Second)),
	)
	return service, channels, history
}

type fakeSender struct{}

func (fakeSender) Send(context.Context, models.NotificationChannel, models.SendNotificationRequest) error {
	return nil
}

type fakeChannelRepository struct {
	nextID   int64
	channels map[int64]models.NotificationChannel
}

func newFakeChannelRepository() *fakeChannelRepository {
	return &fakeChannelRepository{
		nextID:   1,
		channels: make(map[int64]models.NotificationChannel),
	}
}

func (r *fakeChannelRepository) CreateChannel(ctx context.Context, channel *models.NotificationChannel) error {
	channel.ID = r.nextID
	r.nextID++
	channel.CreatedAt = time.Now()
	channel.UpdatedAt = channel.CreatedAt
	r.channels[channel.ID] = *channel
	return nil
}

func (r *fakeChannelRepository) ListChannels(ctx context.Context) ([]models.NotificationChannel, error) {
	items := make([]models.NotificationChannel, 0, len(r.channels))
	for _, channel := range r.channels {
		items = append(items, channel)
	}
	return items, nil
}

func (r *fakeChannelRepository) GetChannelByID(ctx context.Context, id string) (*models.NotificationChannel, error) {
	parsedID, _ := strconv.ParseInt(id, 10, 64)
	channel, ok := r.channels[parsedID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return &channel, nil
}

func (r *fakeChannelRepository) UpdateChannel(ctx context.Context, channel *models.NotificationChannel) error {
	if _, ok := r.channels[channel.ID]; !ok {
		return pgx.ErrNoRows
	}
	r.channels[channel.ID] = *channel
	return nil
}

func (r *fakeChannelRepository) DeleteChannel(ctx context.Context, id string) error {
	parsedID, _ := strconv.ParseInt(id, 10, 64)
	if _, ok := r.channels[parsedID]; !ok {
		return pgx.ErrNoRows
	}
	delete(r.channels, parsedID)
	return nil
}

func (r *fakeChannelRepository) SetChannelEnabled(ctx context.Context, id string, enabled bool) (*models.NotificationChannel, error) {
	parsedID, _ := strconv.ParseInt(id, 10, 64)
	channel, ok := r.channels[parsedID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	channel.Enabled = enabled
	r.channels[parsedID] = channel
	return &channel, nil
}

func (r *fakeChannelRepository) ListEnabledChannels(ctx context.Context) ([]models.NotificationChannel, error) {
	items := make([]models.NotificationChannel, 0)
	for _, channel := range r.channels {
		if channel.Enabled {
			items = append(items, channel)
		}
	}
	return items, nil
}

type fakeHistoryRepository struct {
	nextID int64
	items  map[int64]models.NotificationHistory
}

func newFakeHistoryRepository() *fakeHistoryRepository {
	return &fakeHistoryRepository{
		nextID: 1,
		items:  make(map[int64]models.NotificationHistory),
	}
}

func (r *fakeHistoryRepository) CreateHistory(ctx context.Context, history *models.NotificationHistory) error {
	history.ID = r.nextID
	r.nextID++
	history.CreatedAt = time.Now()
	r.items[history.ID] = *history
	return nil
}

func (r *fakeHistoryRepository) ListHistory(ctx context.Context, filters models.HistoryFilters) ([]models.NotificationHistory, error) {
	items := make([]models.NotificationHistory, 0)
	for _, item := range r.items {
		if filters.Status != "" && item.Status != filters.Status {
			continue
		}
		if filters.ChannelType != "" && item.ChannelType != filters.ChannelType {
			continue
		}
		if filters.EventType != "" && item.EventType != filters.EventType {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *fakeHistoryRepository) GetHistoryByID(ctx context.Context, id string) (*models.NotificationHistory, error) {
	parsedID, _ := strconv.ParseInt(id, 10, 64)
	item, ok := r.items[parsedID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return &item, nil
}

func (r *fakeHistoryRepository) UpdateHistoryStatus(ctx context.Context, id int64, status string, errorMessage *string) error {
	item, ok := r.items[id]
	if !ok {
		return pgx.ErrNoRows
	}
	item.Status = status
	item.ErrorMessage = errorMessage
	if status == models.NotificationStatusSent {
		now := time.Now()
		item.SentAt = &now
	}
	r.items[id] = item
	return nil
}

func (r *fakeHistoryRepository) IncrementRetryCount(ctx context.Context, id int64) error {
	item, ok := r.items[id]
	if !ok {
		return pgx.ErrNoRows
	}
	item.RetryCount++
	r.items[id] = item
	return nil
}

func (r *fakeHistoryRepository) ListFailedHistory(ctx context.Context) ([]models.NotificationHistory, error) {
	return r.ListHistory(ctx, models.HistoryFilters{Status: models.NotificationStatusFailed})
}
