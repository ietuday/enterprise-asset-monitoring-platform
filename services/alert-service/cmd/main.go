package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	alertdb "alert-service/internal/db"
	"alert-service/internal/handlers"
	"alert-service/internal/metrics"
	"alert-service/internal/notification"
	"alert-service/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	dbURL := getEnv(
		"DATABASE_URL",
		"postgres://monitoring_user:monitoring_pass@localhost:5435/monitoring_db?sslmode=disable",
	)

	ctx := context.Background()

	dbpool, err := alertdb.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer dbpool.Close()

	if err := alertdb.CreateAlertsTable(ctx, dbpool); err != nil {
		log.Fatalf("failed to create alerts table: %v", err)
	}
	if err := alertdb.CreateIncidentTables(ctx, dbpool); err != nil {
		log.Fatalf("failed to create incident tables: %v", err)
	}

	alertRepo := repository.NewAlertRepository(dbpool)
	notificationClient := notification.NewClient(os.Getenv("NOTIFICATION_SERVICE_URL"), 3*time.Second)
	alertHandler := handlers.NewAlertHandler(alertRepo, notificationClient)

	metrics.Register()

	r := chi.NewRouter()

	r.Get("/health", alertHandler.Health)
	r.Handle("/metrics", promhttp.Handler())

	r.Post("/alerts/webhook", alertHandler.AlertmanagerWebhook)

	r.Post("/alerts", alertHandler.CreateAlert)
	r.Get("/alerts", alertHandler.ListAlerts)
	r.Get("/alerts/{id}", alertHandler.GetAlertByID)
	r.Put("/alerts/{id}/acknowledge", alertHandler.AcknowledgeAlert)
	r.Put("/alerts/{id}/resolve", alertHandler.ResolveAlert)
	r.Put("/alerts/resolve-active", alertHandler.ResolveActiveAlert)

	r.Post("/incidents", alertHandler.CreateIncident)
	r.Get("/incidents", alertHandler.ListIncidents)
	r.Get("/incidents/{id}", alertHandler.GetIncidentByID)
	r.Put("/incidents/{id}/assign", alertHandler.AssignIncident)
	r.Put("/incidents/{id}/acknowledge", alertHandler.AcknowledgeIncident)
	r.Put("/incidents/{id}/resolve", alertHandler.ResolveIncident)
	r.Put("/incidents/{id}/close", alertHandler.CloseIncident)
	r.Get("/incidents/{id}/history", alertHandler.GetIncidentHistory)

	port := getEnv("PORT", "5003")

	log.Printf("alert-service running on port %s", port)

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
