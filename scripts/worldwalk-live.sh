#!/usr/bin/env bash
# worldwalk-live.sh - one command for the live-brain pseudo-3D world walk.
# Starts a tvcp-ai/1 adapter and benchserver -worldbrain together; Ctrl-C stops both.
# Picks a free port automatically if the chosen one is busy.
#
#   scripts/worldwalk-live.sh                 # Anthropic (Haiku) on :8092, server on :8086
#   ADAPTER=openai scripts/worldwalk-live.sh  # use the OpenAI adapter
#   PORT=9000 BRAINPORT=8099 scripts/worldwalk-live.sh
#
# The adapter reads its key from a file/env and never prints it
# (~/.tvcp_anthropic.key or ~/.tvcp_openai.key).
set -euo pipefail
PORT="${PORT:-8086}"
BRAINPORT="${BRAINPORT:-8092}"
ADAPTER="${ADAPTER:-anthropic}"
REPO="$(cd "$(dirname "$0")/.." && pwd)"
PY="ai/adapters/${ADAPTER}_brain.py"
[ -f "$REPO/$PY" ] || { echo "adapter not found: $PY" >&2; exit 1; }

# pick a free port starting at $1 (uses python, which we need anyway)
freeport() {
  python - "$1" <<'PY'
import socket, sys
p = int(sys.argv[1])
for _ in range(60):
    s = socket.socket()
    try:
        s.bind(("127.0.0.1", p)); print(p); s.close(); break
    except OSError:
        p += 1
    finally:
        try: s.close()
        except Exception: pass
else:
    print(p)
PY
}
PORT="$(freeport "$PORT")"
BRAINPORT="$(freeport "$BRAINPORT")"

echo "TVCP world-walk (live brain)"
echo "  adapter : $ADAPTER on http://127.0.0.1:$BRAINPORT/v1/decide"
( cd "$REPO" && python "$PY" "$BRAINPORT" ) &
APID=$!
trap 'echo; echo "stopping adapter..."; kill "$APID" 2>/dev/null || true' EXIT
sleep 5

echo "  open    : http://localhost:$PORT"
echo "            open the live 2.5D world-walk panel, then click the 'live brain' button"
echo "            (the brain renders ~18 frames over ~1 min; the reference world plays until ready)"
echo "  Ctrl-C to stop."
cd "$REPO"
go run ./cmd/benchserver -addr ":$PORT" -worldbrain "http://127.0.0.1:$BRAINPORT/v1/decide"
