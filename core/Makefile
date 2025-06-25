.PHONY: help build run stop clean logs shell test-pipeline cli api

# Default target
help:
	@echo "CI/CD Agent Commands:"
	@echo "  build          - Build the agent Docker image"
	@echo "  run            - Run the agent (API + CLI available)"
	@echo "  stop           - Stop the agent"
	@echo "  logs           - View agent logs"
	@echo "  shell          - Open shell in agent container"
	@echo "  clean          - Clean up containers and images"
	@echo "  test-pipeline  - Run the simple test pipeline (recommended)"
	@echo "  test-simple    - Run simple test pipeline (no dependencies)"
	@echo "  test-docker    - Run self-contained Docker pipeline"
	@echo "  cli            - Access CLI (requires agent to be running)"
	@echo "  create-stdin   - Create pipeline from stdin (pipe JSON to make)"
	@echo "  create-run-stdin - Create and run pipeline from stdin (pipe JSON to make)"
# @echo "  api            - Open web dashboard"

# Build the agent Docker image
build:
	@echo "🔨 Building CI/CD Agent..."
	docker-compose build cicd-agent

# Run the agent
run:
	@echo "🚀 Starting CI/CD Agent..."
	docker-compose up -d cicd-agent
	@echo "✅ Agent started!"
	@echo "🌐 Web Dashboard: http://localhost:8080"
	@echo "🖥️  CLI Access: make cli"

# Stop the agent
stop:
	@echo "🛑 Stopping CI/CD Agent..."
	docker-compose down

# View logs
logs:
	@echo "📋 Agent logs:"
	docker-compose logs -f cicd-agent

# Open shell in agent container
shell:
	@echo "🐚 Opening shell in agent container..."
	docker-compose exec cicd-agent bash

# Clean up
clean:
	@echo "🧹 Cleaning up..."
	docker-compose down -v
	docker system prune -f
	@echo "✅ Cleanup complete!"

# Run example pipeline
test-pipeline:
	@echo "🧪 Running example pipeline..."
	docker-compose exec cicd-agent python cli.py create-run pipeline_simple_example.json

# Run simple test pipeline (no external dependencies)
test-simple:
	@echo "🧪 Running simple test pipeline..."
	docker-compose exec cicd-agent python cli.py create-run pipeline_simple_example.json

# Run self-contained Docker pipeline
test-docker:
	@echo "🧪 Running self-contained Docker pipeline..."
	docker-compose exec cicd-agent python cli.py create-run pipeline_self_contained_example.json

# Create pipeline from stdin (pipe JSON to make)
create-stdin:
	@echo "📥 Creating pipeline from stdin..."
	@docker-compose exec -T cicd-agent python cli.py create -

# Create and run pipeline from stdin (pipe JSON to make)
create-run-stdin:
	@echo "📥 Creating and running pipeline from stdin..."
	@docker-compose exec -T cicd-agent python cli.py create-run -

# Access CLI
cli:
ifeq ($(strip $(CMD)),)
	@echo "🖥️  CI/CD Agent CLI"
	@echo "Available commands:"
	@echo "  - create <pipeline.json>     : Create a pipeline from a JSON file (use '-' for stdin)"
	@echo "  - run <pipeline_id>          : Run a pipeline by its ID"
	@echo "  - create-run <pipeline.json> : Create and run a pipeline from a JSON file (use '-' for stdin)"
	@echo "  - list                       : List all pipelines"
	@echo "  - status <pipeline_id>       : Get the status of a pipeline"
	@echo "  - monitor <pipeline_id>      : Monitor a pipeline in real-time"
	@echo "  - delete <pipeline_id>       : Delete a pipeline"
	@echo "  - list-runs                  : List all runs"
	@echo "  - run-status <run_id>        : Get the status of a specific run"
	@echo "  - cancel-run <run_id>        : Cancel a running pipeline run"
	@echo "  - delete-run <run_id>        : Delete a specific run"
	@echo ""
	@echo "Usage: make cli CMD=\"<command> [args]\""
	@echo "Example: make cli CMD=\"list\""
	@echo "Example: make cli CMD=\"create pipeline_simple_example.json\""
	@echo "Example: make cli CMD=\"create -\" (read from stdin)"
	@echo "Example: make cli CMD=\"list-runs --pipeline-id <id>\""
	@echo "Example: make cli CMD=\"run-status <run_id>\""
else
	@echo "🖥️  Executing: $(CMD)"
	@docker-compose exec cicd-agent python cli.py $(CMD)
endif

# Open web dashboard
# api:
# 	@echo "🌐 Opening web dashboard..."
# 	@echo "Dashboard URL: http://localhost:8080"
# 	@command -v open >/dev/null 2>&1 && open http://localhost:8080 || \
# 	command -v xdg-open >/dev/null 2>&1 && xdg-open http://localhost:8080 || \
# 	echo "Please open http://localhost:8080 in your browser"

# Check agent status
status:
	@echo "📊 Agent Status:"
	@docker-compose ps cicd-agent
	@echo ""
	@echo "🏥 Health Check:"
	@curl -s http://localhost:8080/health | python -m json.tool || echo "Agent not responding"

# Follow logs in real-time
follow-logs:
	@echo "📋 Following agent logs (Ctrl+C to stop)..."
	docker-compose logs -f cicd-agent

# Restart the agent
restart:
	@echo "🔄 Restarting CI/CD Agent..."
	docker-compose restart cicd-agent
	@echo "✅ Agent restarted!"

# Update and rebuild
update:
	@echo "🔄 Updating and rebuilding agent..."
	docker-compose down
	docker-compose build --no-cache cicd-agent
	docker-compose up -d cicd-agent
	@echo "✅ Agent updated and running!" 
