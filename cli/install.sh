#!/bin/bash
# Custom CI/CD CLI Installation Script for Linux/macOS

set -e

INSTALL_PATH="${INSTALL_PATH:-/usr/local/bin}"
FORCE="${FORCE:-false}"

echo "🔧 Custom CI/CD CLI Installer"
echo "================================"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed or not in PATH"
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

GO_VERSION=$(go version)
echo "✅ Go found: $GO_VERSION"

# Check if we need sudo for installation path
if [[ "$INSTALL_PATH" == "/usr/local/bin" ]] || [[ "$INSTALL_PATH" == "/usr/bin" ]]; then
    if [[ $EUID -ne 0 ]]; then
        echo "🔐 This installation requires sudo privileges"
        SUDO="sudo"
    else
        SUDO=""
    fi
else
    SUDO=""
    # Create user installation directory if needed
    mkdir -p "$INSTALL_PATH"
fi

# Build the CLI
echo "🔨 Building CLI..."
go build -ldflags "-s -w" -o cicd .

# Install the binary
echo "📦 Installing to $INSTALL_PATH..."
$SUDO mv cicd "$INSTALL_PATH/"
$SUDO chmod +x "$INSTALL_PATH/cicd"

# Test the installation
echo "🧪 Testing installation..."
if command -v cicd &> /dev/null; then
    VERSION=$(cicd version)
    echo "✅ Installation successful!"
    echo "📦 Installed: $VERSION"
else
    echo "⚠️  Binary installed but not in PATH. You may need to:"
    echo "   export PATH=\"$INSTALL_PATH:\$PATH\""
    echo "   Add this to your ~/.bashrc or ~/.zshrc"
fi

echo ""
echo "🎉 Installation complete!"
echo "Usage: cicd --help"
echo "Config: cicd config set api-url http://your-api-server:8000" 
