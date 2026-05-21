package main

import (
	"context"
	"log"
	"net/http"
	"os"

	alertdb "alert-service/internal/db"
	"alert-service/internal/handlers"
	"alert-service/internal/metrics"
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

	alertRepo := repository.NewAlertRepository(dbpool)
	alertHandler := handlers.NewAlertHandler(alertRepo)

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
