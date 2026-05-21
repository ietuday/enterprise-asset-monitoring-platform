.PHONY: help up down restart build config logs ps health clean seed test e2e \
        logs-gateway logs-auth logs-asset logs-telemetry logs-alert logs-rule logs-report \
        logs-prometheus logs-alertmanager logs-grafana \
        test-api test-metrics db-shell rules reload-prometheus

help:
	@echo "Enterprise Asset Monitoring Platform"
	@echo ""
	@echo "Available commands:"
	@echo "  make up                  Start all services"
	@echo "  make down                Stop all services"
	@echo "  make restart             Restart all services"
	@echo "  make build               Build all services"
	@echo "  make config              Validate Docker Compose config"
	@echo "  make logs                Show logs for all services"
	@echo "  make ps                  Show running containers"
	@echo "  make health              Run service health checks"
	@echo "  make test                Run Go service tests"
	@echo "  make e2e                 Run end-to-end smoke test"
	@echo "  make test-api            Run basic API checks"
	@echo "  make test-metrics        Check Prometheus metrics endpoints"
	@echo "  make seed                Seed default users and sample assets"
	@echo "  make rules               List dynamic monitoring rules"
	@echo "  make reload-prometheus   Reload Prometheus rules"
	@echo "  make db-shell            Open PostgreSQL shell"
	@echo "  make clean               Stop services and remove local volumes"
	@echo ""
	@echo "Service logs:"
	@echo "  make logs-gateway"
	@echo "  make logs-auth"
	@echo "  make logs-asset"
	@echo "  make logs-telemetry"
	@echo "  make logs-alert"
	@echo "  make logs-rule"
	@echo "  make logs-report"
	@echo "  make logs-prometheus"
	@echo "  make logs-alertmanager"
	@echo "  make logs-grafana"

up:
	docker compose up -d

down:
	docker compose down

restart:
	docker compose restart

build:
	docker compose up -d --build

config:
	docker compose config

logs:
	docker compose logs -f

ps:
	docker ps

health:
	@echo "API Gateway:"
	@curl -s http://localhost:4000/health || true
	@echo "\nAuth Service:"
	@curl -s http://localhost:4001/health || true
	@echo "\nAsset Service:"
	@curl -s http://localhost:5001/health || true
	@echo "\nTelemetry Service:"
	@curl -s http://localhost:5002/health || true
	@echo "\nAlert Service:"
	@curl -s http://localhost:5003/health || true
	@echo "\nRule Service:"
	@curl -s http://localhost:5004/health || true
	@echo "\nReport Service:"
	@curl -s http://localhost:8000/health || true
	@echo "\nPrometheus:"
	@curl -s http://localhost:9090/-/healthy || true
	@echo "\nAlertmanager:"
	@curl -s http://localhost:9093/-/healthy || true
	@echo ""

logs-gateway:
	docker logs -f monitoring-api-gateway

logs-auth:
	docker logs -f monitoring-auth-service

logs-asset:
	docker logs -f monitoring-asset-service

logs-telemetry:
	docker logs -f monitoring-telemetry-service

logs-alert:
	docker logs -f monitoring-alert-service

logs-rule:
	docker logs -f monitoring-rule-service

logs-report:
	docker logs -f monitoring-report-service

logs-prometheus:
	docker logs -f monitoring-prometheus

logs-alertmanager:
	docker logs -f monitoring-alertmanager

logs-grafana:
	docker logs -f monitoring-grafana

test:
	@echo "Running Asset Service tests..."
	cd services/asset-service && go test ./...
	@echo "Running Telemetry Service tests..."
	cd services/telemetry-service && go test ./...
	@echo "Running Alert Service tests..."
	cd services/alert-service && go test ./...
	@echo "Running Rule Service tests..."
	cd services/rule-service && go test ./...

e2e:
	./scripts/e2e-smoke-test.sh

test-api:
	@echo "Testing API Gateway..."
	@curl -s http://localhost:4000/health || true
	@echo "\nTesting Auth Service..."
	@curl -s http://localhost:4001/health || true
	@echo "\nTesting Asset Service..."
	@curl -s http://localhost:5001/health || true
	@echo "\nTesting Telemetry Service..."
	@curl -s http://localhost:5002/health || true
	@echo "\nTesting Alert Service..."
	@curl -s http://localhost:5003/health || true
	@echo "\nTesting Rule Service..."
	@curl -s http://localhost:5004/health || true
	@echo "\nTesting Report Service..."
	@curl -s http://localhost:8000/health || true
	@echo ""

test-metrics:
	@echo "Asset Service metrics:"
	@curl -s http://localhost:5001/metrics | grep -m 1 "go_goroutines" || true
	@echo "Telemetry Service telemetry counter:"
	@curl -s http://localhost:5002/metrics | grep -m 1 "telemetry_received_total" || true
	@echo "Telemetry Service temperature gauge:"
	@curl -s http://localhost:5002/metrics | grep -m 1 "asset_temperature_celsius" || true
	@echo "Telemetry Service CPU gauge:"
	@curl -s http://localhost:5002/metrics | grep -m 1 "asset_cpu_usage_percent" || true
	@echo "Telemetry Service memory gauge:"
	@curl -s http://localhost:5002/metrics | grep -m 1 "asset_memory_usage_percent" || true
	@echo "Alert Service created alerts metric:"
	@curl -s http://localhost:5003/metrics | grep -m 1 "alerts_created_total" || true
	@echo "Alert Service resolved alerts metric:"
	@curl -s http://localhost:5003/metrics | grep -m 1 "alerts_resolved_total" || true
	@echo "Alertmanager metrics:"
	@curl -s http://localhost:9093/metrics | grep -m 1 "alertmanager" || true

seed:
	./scripts/seed.sh

rules:
	curl http://localhost:4000/api/rules \
	  -H "Authorization: Bearer $$TOKEN"

reload-prometheus:
	curl -X POST http://localhost:9090/-/reload

db-shell:
	docker exec -it monitoring-postgres psql -U monitoring_user -d monitoring_db

clean:
	docker compose down -v