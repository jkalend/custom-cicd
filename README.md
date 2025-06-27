# Custom CI/CD Pipeline Manager

A simple CI/CD pipeline management system with a web interface and Python backend.

## Quick Start

### Prerequisites
- Docker
- Docker Compose

### Run the Application

1. Clone and start:
```bash
git clone <repository-url>
cd custom-cicd
docker-compose up -d
```

2. Open your browser:
- **Frontend**: http://localhost

That's it! 🎉

## Usage

### Create a Pipeline
1. Open the web interface
2. Paste your pipeline JSON in the text area
3. Click "Create & Run"

### Example Pipeline
```json
{
  "name": "Simple Test",
  "version": "1.0.0",
  "description": "A basic test pipeline",
  "variables": {
    "PROJECT_NAME": "my-app"
  },
  "steps": [
    {
      "name": "Say Hello",
      "command": "echo 'Hello from ${PROJECT_NAME}!'",
      "timeout": 30
    }
  ]
}
```

## Features

- 🚀 Web dashboard for pipeline management
- 📊 Real-time pipeline monitoring
- 🏃‍♂️ Background job execution
- 📝 Live log streaming
- 🔄 Auto-refresh dashboard

## Stop the Application

```bash
docker-compose down
```

## License

MIT License
