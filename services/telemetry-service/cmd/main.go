package main

import (
	"context"
	"log"
	"net/http"
	"os"

	telemetrydb "telemetry-service/internal/db"
	"telemetry-service/internal/handlers"
	"telemetry-service/internal/metrics"
	"telemetry-service/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	dbURL := getEnv(
		"DATABASE_URL",
		"postgres://monitoring_user:monitoring_pass@localhost:5435/monitoring_db?sslmode=disable",
	)

	ctx := context.Background()

	dbpool, err := telemetrydb.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer dbpool.Close()

	if err := telemetrydb.CreateTelemetryTable(ctx, dbpool); err != nil {
		log.Fatalf("failed to create telemetry table: %v", err)
	}

	telemetryRepo := repository.NewTelemetryRepository(dbpool)
	telemetryHandler := handlers.NewTelemetryHandler(telemetryRepo)

	metrics.Register()

	r := chi.NewRouter()

	r.Get("/health", telemetryHandler.Health)
	r.Handle("/metrics", promhttp.Handler())
	r.Post("/telemetry", telemetryHandler.CreateTelemetry)
	r.Get("/telemetry/latest/{assetId}", telemetryHandler.GetLatestTelemetry)

	port := getEnv("PORT", "5002")

	log.Printf("telemetry-service running on port %s", port)

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
