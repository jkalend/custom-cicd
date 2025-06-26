.PHONY: help build test deploy clean run pipeline docker-compose-up docker-compose-down

# Default target
help:
	@echo "Available commands:"
	@echo "  build      - Build Docker images (frontend + backend)"
	@echo "  test       - Run tests"
	@echo "  deploy     - Deploy application"
	@echo "  pipeline   - Run full CI/CD pipeline"
	@echo "  run        - Run application locally (frontend + backend)"
	@echo "  clean      - Clean up containers and images"
	@echo "  up         - Start with docker-compose (frontend + backend)"
	@echo "  down       - Stop docker-compose"
	@echo "  logs       - Show container logs"
	@echo "  frontend   - Open frontend in browser"

# Build the Docker images
build:
	@echo "Building Docker images..."
	docker-compose build --parallel

# Run tests
test:
	@echo "Running tests..."
	@chmod +x scripts/test.sh
	@./scripts/test.sh

# Deploy the application
deploy:
	@echo "Deploying application..."
	@chmod +x scripts/deploy.sh
	@./scripts/deploy.sh

# Run the full pipeline
pipeline:
	@echo "Running CI/CD pipeline..."
	@chmod +x scripts/pipeline.sh
	@./scripts/pipeline.sh

# Run application locally (without Docker)
run:
	@echo "Running application locally..."
	@echo "Starting backend..."
	cd core && python backend_api.py &
	@echo "Starting frontend..."
	cd core/frontend && npm run dev

# Clean up containers and images
clean:
	@echo "Cleaning up..."
	docker stop custom-cicd-app || true
	docker rm custom-cicd-app || true
	docker rmi custom-cicd-app:latest || true
	docker system prune -f

# Start with docker-compose (frontend + backend)
up:
	@echo "Starting frontend and backend containers..."
	docker-compose up -d
	@echo "✅ Application started!"
	@echo "🌐 Frontend: http://localhost:3000"
	@echo "📊 Backend: Internal (proxied through Next.js API routes)"

# Stop docker-compose
down:
	@echo "Stopping all containers..."
	docker-compose down

# Show container logs
logs:
	@echo "Container logs:"
	docker-compose logs -f

# Open frontend in browser
frontend:
	@echo "Opening frontend..."
	@command -v open >/dev/null 2>&1 && open http://localhost:3000 || \
	command -v xdg-open >/dev/null 2>&1 && xdg-open http://localhost:3000 || \
	echo "Please open http://localhost:3000 in your browser"

# Development mode (with auto-reload)
dev:
	@echo "Starting development environment..."
	@echo "Backend: http://localhost:8000"
	@echo "Frontend: http://localhost:3000"
	cd core && python backend_api.py &
	cd core/frontend && npm run dev 
