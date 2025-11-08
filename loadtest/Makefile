.PHONY: help start stop restart status logs clean loadtest-python loadtest-k6 stats

# Default target
help:
	@echo "Goblet Load Test Harness"
	@echo ""
	@echo "Available targets:"
	@echo "  start          - Start load test environment (HAProxy + 3 Goblet instances + monitoring)"
	@echo "  stop           - Stop all containers"
	@echo "  restart        - Restart all containers"
	@echo "  status         - Show container status"
	@echo "  logs           - Tail logs from all containers"
	@echo "  stats          - Show HAProxy stats"
	@echo "  metrics        - Show Prometheus metrics from Goblet instances"
	@echo "  loadtest-python - Run Python-based load test"
	@echo "  loadtest-k6    - Run k6-based load test"
	@echo "  clean          - Stop and remove all containers and volumes"
	@echo ""
	@echo "Monitoring URLs:"
	@echo "  HAProxy Stats:  http://localhost:8404"
	@echo "  Prometheus:     http://localhost:9090"
	@echo "  Grafana:        http://localhost:3000 (admin/admin)"
	@echo "  Goblet API:     http://localhost:8080"

# Start the load test environment
start:
	@echo "Starting load test environment..."
	docker-compose -f ../docker-compose.loadtest.yml up -d
	@echo ""
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo ""
	@echo "Services started!"
	@echo "  HAProxy:   http://localhost:8080"
	@echo "  Stats:     http://localhost:8404"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Grafana:   http://localhost:3000"

# Stop all containers
stop:
	@echo "Stopping load test environment..."
	docker-compose -f ../docker-compose.loadtest.yml down

# Restart all containers
restart:
	@echo "Restarting load test environment..."
	docker-compose -f ../docker-compose.loadtest.yml restart

# Show container status
status:
	@echo "Container Status:"
	@echo ""
	docker-compose -f ../docker-compose.loadtest.yml ps

# Tail logs from all containers
logs:
	docker-compose -f ../docker-compose.loadtest.yml logs -f

# Show HAProxy stats
stats:
	@echo "HAProxy Statistics:"
	@echo "==================="
	@echo ""
	@curl -s http://localhost:8404 | grep -A 20 "goblet_shards" || echo "HAProxy not responding. Is it running? Try: make start"
	@echo ""
	@echo "Full stats available at: http://localhost:8404"

# Show Prometheus metrics from Goblet instances
metrics:
	@echo "Goblet Instance Metrics:"
	@echo "========================"
	@echo ""
	@echo "Instance 1:"
	@curl -s http://localhost:8080/metrics 2>/dev/null | grep -E "^goblet_" | head -n 10 || echo "Not responding"
	@echo ""
	@echo "Open Prometheus for detailed metrics: http://localhost:9090"

# Run Python-based load test
loadtest-python:
	@echo "Running Python load test..."
	@echo ""
	python3 loadtest.py \
		--url http://localhost:8080 \
		--workers 20 \
		--requests 50 \
		--think-time 100 \
		--repos github.com/kubernetes/kubernetes github.com/golang/go \
		--output results-$$(date +%Y%m%d-%H%M%S).json

# Run k6-based load test
loadtest-k6:
	@echo "Running k6 load test..."
	@echo ""
	docker-compose -f ../docker-compose.loadtest.yml --profile loadtest up k6

# Clean everything
clean:
	@echo "Cleaning up load test environment..."
	@echo "This will remove all containers and volumes!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose -f ../docker-compose.loadtest.yml down -v; \
		echo "Cleanup complete!"; \
	else \
		echo "Cleanup cancelled."; \
	fi
