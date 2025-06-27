# Custom CI/CD CLI

A modern, fast CLI tool for managing CI/CD pipelines and runs. Built with Go for performance and easy distribution.

## Features

- 🚀 **Fast & Lightweight**: Single binary with no runtime dependencies
- 📦 **Pipeline Management**: Create, run, list, and manage pipelines
- 🏃 **Run Monitoring**: Real-time monitoring of pipeline runs
- 🎨 **Rich Output**: Colorized output with emojis for better UX
- ⚙️ **Configuration**: Persistent configuration management
- 🔌 **API Integration**: Seamless integration with CI/CD backend API

## Installation

### Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/your-org/custom-cicd-cli/releases).

#### Windows
```powershell
# Download and extract to your PATH
Invoke-WebRequest -Uri "https://github.com/your-org/custom-cicd-cli/releases/latest/download/cicd-windows-amd64.exe" -OutFile "cicd.exe"
```

#### macOS
```bash
# Intel Macs
curl -L "https://github.com/your-org/custom-cicd-cli/releases/latest/download/cicd-darwin-amd64.tar.gz" | tar -xz
sudo mv cicd /usr/local/bin/

# Apple Silicon Macs
curl -L "https://github.com/your-org/custom-cicd-cli/releases/latest/download/cicd-darwin-arm64.tar.gz" | tar -xz
sudo mv cicd /usr/local/bin/
```

#### Linux
```bash
# x64
curl -L "https://github.com/your-org/custom-cicd-cli/releases/latest/download/cicd-linux-amd64.tar.gz" | tar -xz
sudo mv cicd /usr/local/bin/

# ARM64
curl -L "https://github.com/your-org/custom-cicd-cli/releases/latest/download/cicd-linux-arm64.tar.gz" | tar -xz
sudo mv cicd /usr/local/bin/
```

### Build from Source

```bash
git clone https://github.com/your-org/custom-cicd-cli.git
cd custom-cicd-cli/cli
make build
```

## Quick Start

1. **Configure the API URL** (if different from default):
   ```bash
   cicd config set api-url http://your-api-server:8000
   ```

2. **Check API health**:
   ```bash
   cicd health
   ```

3. **Create a pipeline**:
   ```bash
   cicd pipeline create pipeline.json
   ```

4. **List pipelines**:
   ```bash
   cicd pipeline list
   ```

5. **Run a pipeline**:
   ```bash
   cicd pipeline run <pipeline-id>
   ```

6. **Monitor in real-time**:
   ```bash
   cicd monitor <pipeline-id>
   ```

## Usage

### Pipeline Management

```bash
# Create a new pipeline
cicd pipeline create pipeline.json
cat pipeline.json | cicd pipeline create -

# List all pipelines
cicd pipeline list

# Get pipeline status
cicd pipeline status <pipeline-id>

# Run a pipeline
cicd pipeline run <pipeline-id>

# Create and run in one step
cicd pipeline create-and-run pipeline.json

# Cancel a running pipeline
cicd pipeline cancel <pipeline-id>

# Delete a pipeline
cicd pipeline delete <pipeline-id>

# Monitor a pipeline in real-time
cicd pipeline monitor <pipeline-id>
```

### Run Management

```bash
# List all runs
cicd run list

# List runs for a specific pipeline
cicd run list <pipeline-id>

# Get run status
cicd run status <run-id>

# Cancel a running pipeline run
cicd run cancel <run-id>

# Delete a run
cicd run delete <run-id>
```

### Monitoring

```bash
# Monitor any pipeline or run (auto-detects type)
cicd monitor <id>

# Monitor with custom refresh interval
cicd monitor <id> --interval 5
```

### Configuration

```bash
# View current configuration
cicd config view

# Set API URL
cicd config set api-url http://localhost:8000

# Reset to defaults
cicd config reset
```

### Health Check

```bash
# Check API health
cicd health
```

## Pipeline Configuration

Pipeline configurations are JSON files with the following structure:

```json
{
  "name": "My Pipeline",
  "steps": [
    {
      "name": "Build",
      "command": "npm install && npm run build"
    },
    {
      "name": "Test",
      "command": "npm test"
    },
    {
      "name": "Deploy",
      "command": "npm run deploy"
    }
  ]
}
```

## Configuration

The CLI stores configuration in `~/.custom-cicd/config.yaml`. You can override settings with:

- Environment variables: `CICD_API_URL`
- Command line flags: `--api-url`
- Configuration commands: `cicd config set api-url <url>`

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `CICD_API_URL` | CI/CD API URL | `http://localhost:8000` |

## Development

### Prerequisites

- Go 1.21 or later
- Make

### Building

```bash
# Install dependencies
make deps

# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Install locally
make install
```

### Project Structure

```
cli/
├── cmd/              # Cobra command definitions
├── internal/
│   ├── client/       # API client
│   ├── config/       # Configuration management
│   └── display/      # Output formatting
├── build/            # Build artifacts
├── main.go           # Entry point
├── go.mod            # Go module
├── Makefile          # Build automation
└── README.md         # This file
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.

## Comparison with Original Python CLI

This Go CLI is a complete rewrite that provides:

- **Better Performance**: Faster startup and execution
- **Easy Distribution**: Single binary with no dependencies
- **Modern UX**: Rich terminal output with better formatting
- **Cross-platform**: Native binaries for Windows, macOS, and Linux
- **Maintainability**: Clean, modular architecture with Go

All functionality from the original Python CLI has been preserved and enhanced. 
