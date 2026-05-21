package main

import (
	"context"
	"log"
	"net/http"
	"os"

	assetdb "asset-service/internal/db"
	"asset-service/internal/handlers"
	"asset-service/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	dbURL := getEnv(
		"DATABASE_URL",
		"postgres://monitoring_user:monitoring_pass@localhost:5435/monitoring_db?sslmode=disable",
	)

	ctx := context.Background()

	dbpool, err := assetdb.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer dbpool.Close()

	if err := assetdb.CreateAssetsTable(ctx, dbpool); err != nil {
		log.Fatalf("failed to create assets table: %v", err)
	}

	assetRepo := repository.NewAssetRepository(dbpool)
	assetHandler := handlers.NewAssetHandler(assetRepo)

	r := chi.NewRouter()

	r.Get("/health", assetHandler.Health)
	r.Handle("/metrics", promhttp.Handler())
	r.Post("/assets", assetHandler.CreateAsset)
	r.Get("/assets", assetHandler.ListAssets)
	r.Get("/assets/{id}", assetHandler.GetAssetByID)
	r.Put("/assets/{id}", assetHandler.UpdateAsset)
	r.Delete("/assets/{id}", assetHandler.DeleteAsset)

	port := getEnv("PORT", "5001")

	log.Printf("asset-service running on port %s", port)

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
