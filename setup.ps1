# TVCP Windows Setup Script
# Run this script to build TVCP on Windows

Write-Host "🚀 TVCP Windows Setup" -ForegroundColor Cyan
Write-Host "=====================`n" -ForegroundColor Cyan

# Check Go installation
Write-Host "Checking Go installation..." -ForegroundColor Yellow
if (Get-Command go -ErrorAction SilentlyContinue) {
    $goVersion = go version
    Write-Host "✓ Go found: $goVersion" -ForegroundColor Green
} else {
    Write-Host "✗ Go not found!" -ForegroundColor Red
    Write-Host "Please install Go from: https://go.dev/dl/" -ForegroundColor Yellow
    exit 1
}

# Download dependencies
Write-Host "`nDownloading dependencies..." -ForegroundColor Yellow
go mod download
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ Failed to download dependencies" -ForegroundColor Red
    exit 1
}

go mod tidy
Write-Host "✓ Dependencies ready" -ForegroundColor Green

# Create bin directory
Write-Host "`nCreating bin directory..." -ForegroundColor Yellow
New-Item -ItemType Directory -Force -Path bin | Out-Null
Write-Host "✓ Directory created" -ForegroundColor Green

# Build TVCP
Write-Host "`nBuilding TVCP v0.0.1-alpha..." -ForegroundColor Yellow
$ldflags = "-X main.version=0.0.1-alpha"
go build -ldflags $ldflags -o bin/tvcp.exe ./cmd/tvcp

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Build complete!" -ForegroundColor Green

    # Show file info
    $binary = Get-Item bin\tvcp.exe
    $sizeKB = [math]::Round($binary.Length / 1KB, 2)
    Write-Host "`nBinary info:" -ForegroundColor Cyan
    Write-Host "  Location: $($binary.FullName)" -ForegroundColor White
    Write-Host "  Size: $sizeKB KB" -ForegroundColor White
    Write-Host "  Created: $($binary.LastWriteTime)" -ForegroundColor White

    # Test the binary
    Write-Host "`nTesting binary..." -ForegroundColor Yellow
    .\bin\tvcp.exe version 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ Binary works!" -ForegroundColor Green
    } else {
        Write-Host "⚠ Binary might have issues (this is normal if 'version' command doesn't exist yet)" -ForegroundColor Yellow
    }

    # Success message
    Write-Host "`n🎉 Setup complete!" -ForegroundColor Green
    Write-Host "`n" + "="*50 -ForegroundColor Cyan
    Write-Host "Next steps:" -ForegroundColor Cyan
    Write-Host "="*50 -ForegroundColor Cyan
    Write-Host "`n1. Try the demo:" -ForegroundColor Yellow
    Write-Host "   .\bin\tvcp.exe demo" -ForegroundColor White
    Write-Host "`n2. Test video preview:" -ForegroundColor Yellow
    Write-Host "   .\bin\tvcp.exe preview bounce" -ForegroundColor White
    Write-Host "   .\bin\tvcp.exe preview gradient" -ForegroundColor White
    Write-Host "`n3. Test audio:" -ForegroundColor Yellow
    Write-Host "   .\bin\tvcp.exe audio-test" -ForegroundColor White
    Write-Host "`n4. Make a local call:" -ForegroundColor Yellow
    Write-Host "   Terminal 1: .\bin\tvcp.exe listen 5000" -ForegroundColor White
    Write-Host "   Terminal 2: .\bin\tvcp.exe call localhost:5000" -ForegroundColor White
    Write-Host "`n5. Play games (NEW!):" -ForegroundColor Yellow
    Write-Host "   .\bin\tvcp.exe group --game trivia --port 5000" -ForegroundColor White
    Write-Host "`n" + "="*50 -ForegroundColor Cyan
    Write-Host "`n📚 Documentation:" -ForegroundColor Cyan
    Write-Host "   - BUILD_WINDOWS.md - Windows-specific guide" -ForegroundColor White
    Write-Host "   - GETTING_STARTED.md - Basic usage" -ForegroundColor White
    Write-Host "   - experimental/QUICK_START_GAMES.md - Games & Bots" -ForegroundColor White
    Write-Host "`n🆘 Need help? Check the documentation or create an issue on GitHub" -ForegroundColor Cyan
    Write-Host "`n"

} else {
    Write-Host "✗ Build failed!" -ForegroundColor Red
    Write-Host "`nTroubleshooting:" -ForegroundColor Yellow
    Write-Host "  1. Make sure Go 1.21+ is installed: go version" -ForegroundColor White
    Write-Host "  2. Check you're in the project directory: pwd" -ForegroundColor White
    Write-Host "  3. Try manually: go mod download && go build -o bin/tvcp.exe ./cmd/tvcp" -ForegroundColor White
    Write-Host "`nFor more help, see BUILD_WINDOWS.md" -ForegroundColor Yellow
    exit 1
}
