# Custom CI/CD CLI Installation Script for Windows
# This script builds and installs the CLI to the user's PATH

param(
    [string]$InstallPath = "$env:USERPROFILE\bin",
    [switch]$Force
)

Write-Host "🔧 Custom CI/CD CLI Installer" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan

# Check if Go is installed
try {
    $goVersion = go version
    Write-Host "✅ Go found: $goVersion" -ForegroundColor Green
} catch {
    Write-Host "❌ Go is not installed or not in PATH" -ForegroundColor Red
    Write-Host "Please install Go from https://golang.org/dl/" -ForegroundColor Yellow
    exit 1
}

# Create installation directory
if (!(Test-Path $InstallPath)) {
    Write-Host "📁 Creating installation directory: $InstallPath" -ForegroundColor Yellow
    New-Item -ItemType Directory -Path $InstallPath -Force | Out-Null
}

# Build the CLI
Write-Host "🔨 Building CLI..." -ForegroundColor Yellow
try {
    & go build -ldflags "-s -w" -o "$InstallPath\cicd.exe" .
    if ($LASTEXITCODE -ne 0) {
        throw "Build failed"
    }
} catch {
    Write-Host "❌ Build failed: $_" -ForegroundColor Red
    exit 1
}

Write-Host "✅ CLI built successfully" -ForegroundColor Green

# Check if installation path is in PATH
$currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($currentPath -notlike "*$InstallPath*") {
    Write-Host "⚠️  Installation directory is not in your PATH" -ForegroundColor Yellow
    Write-Host "Adding $InstallPath to your PATH..." -ForegroundColor Yellow
    
    $newPath = $currentPath + ";" + $InstallPath
    [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
    
    Write-Host "✅ PATH updated. Please restart your terminal." -ForegroundColor Green
} else {
    Write-Host "✅ Installation directory is already in PATH" -ForegroundColor Green
}

# Test the installation
Write-Host "🧪 Testing installation..." -ForegroundColor Yellow
try {
    $version = & "$InstallPath\cicd.exe" version
    Write-Host "✅ Installation successful!" -ForegroundColor Green
    Write-Host "📦 Installed: $version" -ForegroundColor Cyan
} catch {
    Write-Host "⚠️  Installation may have issues. Try restarting your terminal." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "🎉 Installation complete!" -ForegroundColor Green
Write-Host "Usage: cicd --help" -ForegroundColor Cyan
Write-Host "Config: cicd config set api-url http://your-api-server:8000" -ForegroundColor Cyan 
