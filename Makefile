.PHONY: help build run dev clean up down logs backend cli frontend

# Default target
help:
	@echo "Available commands:"
	@echo "  build      - Build Go backend + CLI binaries"
	@echo "  run        - Run backend + frontend locally (no Docker)"
	@echo "  dev        - Backend (go run) + frontend (next dev)"
	@echo "  backend    - Run only the Go backend on :8000"
	@echo "  cli        - Build the Go CLI into cli/cicd"
	@echo "  up         - Start with docker-compose (frontend + backend)"
	@echo "  down       - Stop docker-compose"
	@echo "  logs       - Show container logs"
	@echo "  clean      - Clean build artifacts"

build:
	@echo "Building Go backend..."
	cd backend && go build -o bin/server ./cmd/server
	@echo "Building Go CLI..."
	cd cli && go build -o cicd .

backend:
	cd backend && go run ./cmd/server

cli:
	cd cli && go build -o cicd .

frontend:
	@echo "Open http://localhost:3000 in your browser"

# Run application locally (without Docker)
run dev:
	@echo "Starting backend on :8000..."
	cd backend && go run ./cmd/server &
	@echo "Starting frontend on :3000..."
	cd frontend && BACKEND_URL=http://localhost:8000 npm run dev

clean:
	rm -rf backend/bin cli/cicd
