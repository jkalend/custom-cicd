# CI/CD Agent

A simple containerized CI/CD agent that runs custom pipelines with web and CLI interfaces.

## What it does

- Runs custom pipelines defined in JSON format
- Provides a web dashboard at http://localhost:8080
- Includes a command-line interface for automation
- Supports Docker commands and variable substitution
- Monitors pipeline execution in real-time

## Quick Start

1. **Start the agent:**
   ```bash
   cd custom-cicd/core
   docker-compose up -d
   ```

2. **Use the web interface:**
   - Open http://localhost:8080
   - Paste a pipeline JSON and click "Create & Run"

3. **Use the CLI:**
   ```bash
   # List pipelines
   docker-compose exec cicd-agent python cli.py list
   
   # Run a pipeline
   docker-compose exec cicd-agent python cli.py create-run pipeline.json
   ```

## Pipeline Example

```json
{
    "name": "Build and Test",
    "variables": {
        "APP_NAME": "my-app"
    },
    "steps": [
        {
            "name": "Install Dependencies",
            "command": "npm install"
        },
        {
            "name": "Build",
            "command": "npm run build"
        },
        {
            "name": "Test",
            "command": "npm test"
        }
    ]
}
```

## Management

```bash
# View logs
docker-compose logs -f

# Stop the agent
docker-compose down

# Restart
docker-compose restart
``` 
or through the provided Makefile
