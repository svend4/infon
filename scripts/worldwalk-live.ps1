# worldwalk-live.ps1 - one command for the live-brain pseudo-3D world walk.
# Starts a tvcp-ai/1 adapter and benchserver -worldbrain together, then you open
# the browser. Ctrl-C stops both. Picks a free port automatically if the chosen
# one is busy.
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
try { [Console]::OutputEncoding = [System.Text.Encoding]::UTF8 } catch {}
$repo = Split-Path $PSScriptRoot -Parent
$py = "ai/adapters/$($adapter)_brain.py"
if (-not (Test-Path (Join-Path $repo $py))) { Write-Error "adapter not found: $py"; exit 1 }

function Find-FreePort([int]$p) {
  # Query the TCP table for a listener; binding-to-probe is unreliable on Windows
  # because Go's server and the probe can both bind the same port (SO_REUSEADDR).
  for ($i = 0; $i -lt 60; $i++) {
    $busy = $null
    try { $busy = Get-NetTCPConnection -LocalPort $p -State Listen -ErrorAction SilentlyContinue } catch {}
    if (-not $busy) { return $p }
    $p++
  }
  return $p
}
$port = Find-FreePort $port
$brainport = Find-FreePort $brainport

Write-Host "TVCP world-walk (live brain)" -ForegroundColor Cyan
Write-Host ("  adapter : {0} on http://127.0.0.1:{1}/v1/decide" -f $adapter, $brainport)
$a = Start-Process -FilePath 'python' -ArgumentList "$py $brainport" -WorkingDirectory $repo -PassThru
Start-Sleep -Seconds 5

Write-Host ("  open    : http://localhost:{0}" -f $port) -ForegroundColor Green
Write-Host "            open the live 2.5D world-walk panel, then click the 'live brain' button"
Write-Host "            (the brain renders ~18 frames over ~1 min; the reference world plays until ready)"
Write-Host "  Ctrl-C to stop."
try {
  Push-Location $repo
  & go run ./cmd/benchserver -addr (":{0}" -f $port) -worldbrain ("http://127.0.0.1:{0}/v1/decide" -f $brainport)
}
finally {
  Pop-Location
  Write-Host "stopping adapter..."
  Stop-Process -Id $a.Id -Force -ErrorAction SilentlyContinue
}
