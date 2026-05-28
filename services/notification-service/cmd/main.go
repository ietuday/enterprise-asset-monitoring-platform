package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	notificationdb "notification-service/internal/db"
	"notification-service/internal/handlers"
	"notification-service/internal/repository"
	"notification-service/internal/sender"
	notificationservice "notification-service/internal/service"

	"github.com/go-chi/chi/v5"
)

func main() {
	dbURL := getEnv(
		"DATABASE_URL",
		"postgres://monitoring_user:monitoring_pass@localhost:5435/monitoring_db?sslmode=disable",
	)

	ctx := context.Background()

	dbpool, err := notificationdb.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer dbpool.Close()

	if err := notificationdb.CreateNotificationTables(ctx, dbpool); err != nil {
		log.Fatalf("failed to create notification tables: %v", err)
	}

	channelRepo := repository.NewChannelRepository(dbpool)
	historyRepo := repository.NewHistoryRepository(dbpool)
	emailSender := sender.NewEmailSender(sender.EmailConfig{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     os.Getenv("SMTP_PORT"),
		User:     os.Getenv("SMTP_USER"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     os.Getenv("SMTP_FROM"),
	})
	webhookSender := sender.NewWebhookSender(5 * time.Second)
	registry := sender.NewRegistry(emailSender, webhookSender)
	service := notificationservice.NewNotificationService(channelRepo, historyRepo, registry)
	handler := handlers.NewHandler(service)

	r := chi.NewRouter()

	r.Get("/health", handler.Health)

	r.Post("/notification-channels", handler.CreateChannel)
	r.Get("/notification-channels", handler.ListChannels)
	r.Get("/notification-channels/{id}", handler.GetChannelByID)
	r.Put("/notification-channels/{id}", handler.UpdateChannel)
	r.Delete("/notification-channels/{id}", handler.DeleteChannel)
	r.Patch("/notification-channels/{id}/enable", handler.EnableChannel)
	r.Patch("/notification-channels/{id}/disable", handler.DisableChannel)

	r.Post("/notifications/send", handler.Send)
	r.Post("/notifications/test", handler.Test)
	r.Get("/notifications/history", handler.ListHistory)
	r.Get("/notifications/history/{id}", handler.GetHistoryByID)
	r.Post("/notifications/{id}/retry", handler.Retry)

	port := getEnv("PORT", getEnv("NOTIFICATION_SERVICE_PORT", "8090"))
	log.Printf("notification-service running on port %s", port)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
