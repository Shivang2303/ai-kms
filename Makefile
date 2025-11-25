.PHONY: build run test clean docker-up docker-down

# Build the server
build:
	@echo "Building server..."
	@go build -o bin/server ./cmd/server
	@echo "✓ Build complete: bin/server"

# Run the server
run:
	@go run cmd/server/main.go

# Run tests
test:
	@go test -v ./...

# Clean build artifacts
clean:
	@rm -rf bin/
	@echo "✓ Clean complete"

# Start PostgreSQL with Docker
docker-up:
	@echo "Starting PostgreSQL with pgvector..."
	@docker run -d \
		--name ai-kms-postgres \
		-e POSTGRES_PASSWORD=postgres \
		-e POSTGRES_DB=ai_kms \
		-p 5432:5432 \
		ankane/pgvector
	@echo "✓ PostgreSQL running on localhost:5432"

# Stop PostgreSQL Docker container
docker-down:
	@docker stop ai-kms-postgres
	@docker rm ai-kms-postgres
	@echo "✓ PostgreSQL container removed"

# Start Jaeger for distributed tracing
jaeger-up:
	@echo "Starting Jaeger all-in-one..."
	@docker run -d --name jaeger \
		-p 16686:16686 \
		-p 14268:14268 \
		jaegertracing/all-in-one:latest
	@echo "✓ Jaeger running"
	@echo "  UI: http://localhost:16686"
	@echo "  Collector: http://localhost:14268"

# Stop Jaeger
jaeger-down:
	@docker stop jaeger
	@docker rm jaeger
	@echo "✓ Jaeger container removed"

# Start all infrastructure (Postgres + Jaeger)
infra-up: docker-up jaeger-up
	@echo "✓ All infrastructure running"

# Stop all infrastructure
infra-down: docker-down jaeger-down
	@echo "✓ All infrastructure stopped"

# Install development tools
dev-tools:
	@echo "Installing development tools..."
	@go install github.com/air-verse/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "✓ Development tools installed"

# Format code
fmt:
	@go fmt ./...
	@echo "✓ Code formatted"

# Lint code
lint:
	@golangci-lint run

# Docker commands

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t ai-kms:latest .
	@echo "✓ Docker image built: ai-kms:latest"

# Run with Docker Compose
docker-up:
	@echo "Starting services with Docker Compose..."
	@docker-compose up -d
	@echo "✓ Services started"
	@echo "  AI-KMS: http://localhost:8080"
	@echo "  Jaeger UI: http://localhost:16686"
	@echo "  PostgreSQL: localhost:5432"

# Stop Docker Compose
docker-down:
	@docker-compose down
	@echo "✓ Services stopped"

# Stop and remove volumes
docker-clean:
	@docker-compose down -v
	@echo "✓ Services stopped and volumes removed"

# View Docker logs
docker-logs:
	@docker-compose logs -f ai-kms

# Kubernetes commands

# Deploy to Kubernetes
k8s-deploy:
	@echo "Deploying to Kubernetes..."
	@kubectl apply -f k8s/namespace.yaml
	@kubectl apply -f k8s/configmap.yaml
	@kubectl apply -f k8s/postgres.yaml
	@kubectl apply -f k8s/jaeger.yaml
	@kubectl apply -f k8s/ai-kms.yaml
	@echo "✓ Deployed to Kubernetes"

# Delete from Kubernetes
k8s-delete:
	@kubectl delete namespace ai-kms
	@echo "✓ Removed from Kubernetes"

# View Kubernetes status
k8s-status:
	@kubectl get all -n ai-kms
