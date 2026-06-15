# worldwalk-live.ps1 - one command for the live-brain pseudo-3D world walk.
# Starts a tvcp-ai/1 adapter and benchserver -worldbrain together, then you open
# the browser. Ctrl-C stops both.
#
#   powershell -ExecutionPolicy Bypass -File scripts\worldwalk-live.ps1
#   ... -adapter openai      # use the OpenAI adapter instead of Anthropic
#   ... -port 9000 -brainport 8099
#
# The adapter reads its key from a file/env and never prints it
# (~/.tvcp_anthropic.key or ~/.tvcp_openai.key).
param(
  [int]$port = 8086,
  [int]$brainport = 8092,
  [string]$adapter = 'anthropic'   # anthropic | openai | ollama
)
$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
$py = "ai/adapters/$($adapter)_brain.py"
if (-not (Test-Path (Join-Path $repo $py))) { Write-Error "adapter not found: $py"; exit 1 }

Write-Host "TVCP world-walk (live brain)" -ForegroundColor Cyan
Write-Host "  adapter : $adapter on http://127.0.0.1:$brainport/v1/decide"
$a = Start-Process -FilePath 'python' -ArgumentList "$py $brainport" -WorkingDirectory $repo -PassThru
Start-Sleep -Seconds 5

Write-Host "  open    : http://localhost:$port" -ForegroundColor Green
Write-Host "            panel 'прогулка по живому 2.5D-миру' -> button 'живой мозг'"
Write-Host "            (the brain renders ~18 frames over ~1 min; reference world plays until ready)"
Write-Host "  Ctrl-C to stop."
try {
  Push-Location $repo
  & go run ./cmd/benchserver -addr ":$port" -worldbrain "http://127.0.0.1:$brainport/v1/decide"
}
finally {
  Pop-Location
  Write-Host "stopping adapter..."
  Stop-Process -Id $a.Id -Force -ErrorAction SilentlyContinue
}
