package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	maintenancedb "maintenance-service/internal/db"
	"maintenance-service/internal/handlers"
	maintenanceservice "maintenance-service/internal/service"

	"github.com/go-chi/chi/v5"
)

func main() {
	ctx := context.Background()
	dbpool, err := maintenancedb.Connect(ctx, databaseURL())
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer dbpool.Close()

	if err := maintenancedb.CreateMaintenanceTables(ctx, dbpool); err != nil {
		log.Fatalf("failed to create maintenance tables: %v", err)
	}

	handler := handlers.NewMaintenanceHandler(maintenanceservice.NewMaintenanceService(dbpool))

	r := chi.NewRouter()
	r.Get("/health", handler.Health)
	r.Get("/maintenance/tasks", handler.ListTasks)
	r.Post("/maintenance/tasks", handler.CreateTask)
	r.Get("/maintenance/tasks/{id}", handler.GetTask)
	r.Put("/maintenance/tasks/{id}", handler.UpdateTask)
	r.Patch("/maintenance/tasks/{id}/status", handler.ChangeStatus)
	r.Post("/maintenance/tasks/{id}/complete", handler.CompleteTask)
	r.Post("/maintenance/tasks/{id}/cancel", handler.CancelTask)
	r.Get("/maintenance/assets/{assetId}/tasks", handler.ListAssetTasks)
	r.Get("/maintenance/overdue", handler.ListOverdueTasks)
	r.Get("/maintenance/history/{taskId}", handler.ListHistory)

	port := getEnv("PORT", "8087")
	log.Printf("maintenance-service running on port %s", port)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func databaseURL() string {
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value
	}

	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5435")
	user := getEnv("DB_USER", "monitoring_user")
	password := getEnv("DB_PASSWORD", "monitoring_pass")
	name := getEnv("DB_NAME", "monitoring_db")

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, name)
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
