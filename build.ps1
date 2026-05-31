# Quick build script for TVCP on Windows
# Usage: .\build.ps1 [-Version "0.0.2-dev"] [-Clean]

param(
    [string]$Version = "0.0.1-alpha",
    [switch]$Clean = $false
)

Write-Host "Building TVCP v$Version..." -ForegroundColor Green

# Clean if requested
if ($Clean) {
    Write-Host "Cleaning..." -ForegroundColor Yellow
    go clean
    Remove-Item -Recurse -Force bin -ErrorAction SilentlyContinue
    Write-Host "✓ Clean complete" -ForegroundColor Green
}

# Create bin directory
New-Item -ItemType Directory -Force -Path bin | Out-Null

# Build
$ldflags = "-X main.version=$Version"
go build -ldflags $ldflags -o bin/tvcp.exe ./cmd/tvcp

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Build complete: bin\tvcp.exe" -ForegroundColor Green

    # Show file info
    $binary = Get-Item bin\tvcp.exe
    $sizeKB = [math]::Round($binary.Length / 1KB, 2)
    Write-Host "  Size: $sizeKB KB" -ForegroundColor Cyan
} else {
    Write-Host "✗ Build failed" -ForegroundColor Red
    exit 1
}
