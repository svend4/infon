# 🪟 Building TVCP on Windows

**Quick guide for building and running TVCP on Windows without make**

---

## 📋 Prerequisites

1. **Go 1.21+**
   ```powershell
   # Check Go version
   go version
   ```
   
   If not installed: https://go.dev/dl/

2. **Git** (already installed ✅)

---

## 🚀 Quick Build (PowerShell)

### Step 1: Navigate to project

```powershell
cd C:\Users\stefan\infon
```

### Step 2: Download dependencies

```powershell
go mod download
go mod tidy
```

### Step 3: Build the binary

```powershell
# Create bin directory
New-Item -ItemType Directory -Force -Path bin

# Build tvcp.exe
go build -ldflags "-X main.version=0.0.1-alpha" -o bin/tvcp.exe ./cmd/tvcp
```

### Step 4: Verify installation

```powershell
# Check the binary was created
.\bin\tvcp.exe version
```

---

## 🎯 Running TVCP on Windows

### Demo Mode (static image)

```powershell
# Create a test pattern
.\bin\tvcp.exe demo test_pattern.png
```

### Live Preview

```powershell
# Test with bounce animation
.\bin\tvcp.exe preview bounce

# Or gradient animation
.\bin\tvcp.exe preview gradient
```

### Audio Test

```powershell
.\bin\tvcp.exe audio-test
```

### Making a Call

**Terminal 1 (Receiver):**
```powershell
.\bin\tvcp.exe listen 5000
```

**Terminal 2 (Caller):**
```powershell
.\bin\tvcp.exe call localhost:5000
```

---

## 🎮 Games & Bots (NEW!)

### Trivia Game

**Terminal 1:**
```powershell
.\bin\tvcp.exe group --game trivia --port 5000 localhost:5001
```

**Terminal 2:**
```powershell
.\bin\tvcp.exe group --game trivia --port 5001 localhost:5000
```

### With AI Bot

```powershell
# Set OpenAI API key
$env:OPENAI_API_KEY = "sk-..."

# Start game with bot
.\bin\tvcp.exe group --game trivia --bots 1 --port 5000 localhost:5001
```

---

## 🔧 Common Commands

### Build

```powershell
# Clean build
Remove-Item -Recurse -Force bin -ErrorAction SilentlyContinue
go clean
go build -ldflags "-X main.version=0.0.1-alpha" -o bin/tvcp.exe ./cmd/tvcp
```

### Run Tests

```powershell
go test -v ./...
```

### Clean

```powershell
go clean
Remove-Item -Recurse -Force bin -ErrorAction SilentlyContinue
```

---

## 📝 Build Script (Optional)

Create `build.ps1` for easier building:

```powershell
# build.ps1
param(
    [string]$Version = "0.0.1-alpha"
)

Write-Host "Building TVCP v$Version..." -ForegroundColor Green

# Create bin directory
New-Item -ItemType Directory -Force -Path bin | Out-Null

# Build
$ldflags = "-X main.version=$Version"
go build -ldflags $ldflags -o bin/tvcp.exe ./cmd/tvcp

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Build complete: bin\tvcp.exe" -ForegroundColor Green
    
    # Show file info
    Get-Item bin\tvcp.exe | Format-List Name, Length, LastWriteTime
} else {
    Write-Host "✗ Build failed" -ForegroundColor Red
    exit 1
}
```

**Usage:**
```powershell
# Run the build script
.\build.ps1

# Or with custom version
.\build.ps1 -Version "0.0.2-dev"
```

---

## ⚠️ Windows-Specific Notes

### 1. Camera Access

Windows cameras require DirectShow or Media Foundation:

```powershell
# List available cameras
.\bin\tvcp.exe list-cameras
```

TVCP uses V4L2 on Linux. On Windows, you may need to:
- Use WSL2 for Linux compatibility
- Wait for Windows camera support (planned)
- Use virtual camera (OBS Virtual Camera)

### 2. Audio

Windows uses WASAPI for audio. TVCP has experimental WASAPI support:

```powershell
# Test audio capture
.\bin\tvcp.exe audio-test --backend wasapi
```

### 3. Yggdrasil Network

For peer-to-peer calls over Yggdrasil mesh network:

1. **Install Yggdrasil:**
   - Download from: https://yggdrasil-network.github.io/
   - Run installer
   - Start service

2. **Get your Yggdrasil IPv6:**
   ```powershell
   ipconfig | Select-String "200:"
   ```

3. **Make call over Yggdrasil:**
   ```powershell
   .\bin\tvcp.exe call [200:1234::5678]:5000
   ```

---

## 🐛 Troubleshooting

### "go: command not found"

**Solution:** Install Go from https://go.dev/dl/

### "cannot find package"

**Solution:**
```powershell
go mod download
go mod tidy
```

### "Access denied" when building

**Solution:** Run PowerShell as Administrator or build in your user directory

### Camera not working

**Options:**
1. Use WSL2 with Linux V4L2 cameras
2. Wait for Windows DirectShow support
3. Use test animations: `bounce`, `gradient`, `noise`

### Audio not working

**Solution:**
```powershell
# Check WASAPI backend
.\bin\tvcp.exe audio-test --backend wasapi --list-devices

# Use specific device
.\bin\tvcp.exe call localhost:5000 --audio-input "Microphone (USB)"
```

---

## 📦 Cross-Compilation (Advanced)

### Build for Linux on Windows

```powershell
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o bin/tvcp-linux ./cmd/tvcp
```

### Build for macOS on Windows

```powershell
$env:GOOS = "darwin"
$env:GOARCH = "amd64"
go build -o bin/tvcp-macos ./cmd/tvcp
```

### Reset to Windows

```powershell
Remove-Item Env:\GOOS
Remove-Item Env:\GOARCH
```

---

## 🚀 Quick Setup Script

Save as `setup.ps1`:

```powershell
# setup.ps1 - One-click setup for TVCP on Windows

Write-Host "🚀 TVCP Windows Setup" -ForegroundColor Cyan
Write-Host "=====================`n" -ForegroundColor Cyan

# Check Go
Write-Host "Checking Go installation..." -ForegroundColor Yellow
if (Get-Command go -ErrorAction SilentlyContinue) {
    $goVersion = go version
    Write-Host "✓ Go found: $goVersion" -ForegroundColor Green
} else {
    Write-Host "✗ Go not found. Please install from https://go.dev/dl/" -ForegroundColor Red
    exit 1
}

# Download dependencies
Write-Host "`nDownloading dependencies..." -ForegroundColor Yellow
go mod download
go mod tidy
Write-Host "✓ Dependencies downloaded" -ForegroundColor Green

# Build
Write-Host "`nBuilding TVCP..." -ForegroundColor Yellow
New-Item -ItemType Directory -Force -Path bin | Out-Null
go build -ldflags "-X main.version=0.0.1-alpha" -o bin/tvcp.exe ./cmd/tvcp

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Build complete!" -ForegroundColor Green
    
    # Test
    Write-Host "`nTesting installation..." -ForegroundColor Yellow
    .\bin\tvcp.exe version
    
    Write-Host "`n🎉 Setup complete!" -ForegroundColor Green
    Write-Host "`nNext steps:" -ForegroundColor Cyan
    Write-Host "  1. Try demo: .\bin\tvcp.exe demo" -ForegroundColor White
    Write-Host "  2. Try preview: .\bin\tvcp.exe preview bounce" -ForegroundColor White
    Write-Host "  3. Make a call: .\bin\tvcp.exe listen 5000" -ForegroundColor White
    Write-Host "`nFor more info: Get-Help .\bin\tvcp.exe" -ForegroundColor White
} else {
    Write-Host "✗ Build failed" -ForegroundColor Red
    exit 1
}
```

**Run it:**
```powershell
.\setup.ps1
```

---

## 📚 Next Steps

1. **Read the documentation:**
   - [GETTING_STARTED.md](./GETTING_STARTED.md) - Basic usage
   - [QUICK_START_GAMES.md](./experimental/QUICK_START_GAMES.md) - Games & Bots
   - [FEATURE_STATUS.md](./FEATURE_STATUS.md) - Available features

2. **Try the demos:**
   ```powershell
   .\bin\tvcp.exe demo
   .\bin\tvcp.exe preview gradient
   .\bin\tvcp.exe audio-test
   ```

3. **Make your first call:**
   ```powershell
   # Terminal 1
   .\bin\tvcp.exe listen 5000
   
   # Terminal 2
   .\bin\tvcp.exe call localhost:5000
   ```

4. **Play games:**
   ```powershell
   # Terminal 1
   .\bin\tvcp.exe group --game trivia --port 5000
   ```

---

## 🆘 Getting Help

- **Documentation:** Check the `.md` files in this repository
- **Issues:** https://github.com/svend4/infon/issues
- **Community:** Join our Discord (see README.md)

---

**Happy calling! 📞**
