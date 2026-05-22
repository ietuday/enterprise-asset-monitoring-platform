package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"rule-service/internal/db"
	"rule-service/internal/handlers"
	"rule-service/internal/repository"

	"github.com/go-chi/chi/v5"
)

func main() {
	ctx := context.Background()

	pool, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer pool.Close()

	if err := db.Init(ctx, pool); err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	ruleRepo := repository.NewRuleRepository(pool)
	ruleHandler := handlers.NewRuleHandler(ruleRepo)

	r := chi.NewRouter()

	r.Get("/health", ruleHandler.Health)

	r.Post("/rules", ruleHandler.CreateRule)
	r.Get("/rules", ruleHandler.ListRules)
	r.Get("/rules/enabled", ruleHandler.ListEnabledRules)
	r.Get("/rules/history", ruleHandler.ListRuleAuditLogs)
	r.Get("/rules/{id}/history", ruleHandler.ListRuleAuditLogsByRuleID)
	r.Get("/rules/{id}", ruleHandler.GetRuleByID)
	r.Put("/rules/{id}", ruleHandler.UpdateRule)
	r.Delete("/rules/{id}", ruleHandler.DeleteRule)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5004"
	}

	log.Printf("rule-service running on port %s", port)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("failed to start rule-service: %v", err)
	}
}
