#!/usr/bin/env python3
"""tvcp-ai/1 brain adapter backed by Anthropic Claude (e.g. Haiku).

Native Anthropic Messages API (not OpenAI-compatible). The key is read from the
environment and never printed.

    setx ANTHROPIC_API_KEY "sk-ant-..."        # once, in your own terminal
    set ANTHROPIC_MODEL=claude-haiku-4-5-20251001
    python anthropic_brain.py 8092
    # point a client at http://127.0.0.1:8092/v1/decide
"""
import json, os, sys, urllib.request
from http.server import BaseHTTPRequestHandler, HTTPServer

URL   = "https://api.anthropic.com/v1/messages"
KEY   = os.environ.get("ANTHROPIC_API_KEY", "")
# Or read the key from a file OUTSIDE the shared folder (never pasted in chat):
_KF = os.environ.get("ANTHROPIC_KEY_FILE", os.path.expanduser("~/.tvcp_anthropic.key"))
if not KEY and os.path.exists(_KF):
    KEY = open(_KF, encoding="utf-8").read().strip()
MODEL = os.environ.get("ANTHROPIC_MODEL", "claude-haiku-4-5-20251001")
SYS = ("You are a game and art partner speaking the tvcp-ai/1 format. Reply with ONLY "
       "one JSON object, no prose, no code fences. "
       "kind=move (tic-tac-toe) -> {\"move\":{\"row\":R,\"col\":C}} 0-2 on an EMPTY cell. "
       "kind=move with game=wordle -> {\"move\":{\"word\":\"CRANE\"}} one real 5-letter word; "
       "state.guesses lists past words with per-letter marks correct/present/absent, use them. "
       "kind=sketch (PREFERRED for pictures) -> {\"sketch\":{\"sky\":NAME,\"ground\":NAME,"
       "\"shapes\":[{\"kind\":K,\"x\":X,\"y\":Y,\"w\":W,\"h\":H,\"color\":NAME,\"s\":TEXT}]}} "
       "where colors are NAMES like orange/navy/gold/darkgreen/white/purple/teal and shape "
       "kind is one of sun/moon/star/hill/mountain/building/cloud/band/text. "
       "kind=draw -> {\"scene\":{\"width\":W,\"height\":H,\"ops\":[...]}} with [r,g,b] colors. "
       "kind=react -> {\"cards\":[...]} : pick 3 to 4 DIFFERENT cards (no repeats) that "
       "fit the message mood, from: star heart check fire smile sad music sun warning x thumbsup.")

def extract_json(text):
    t = text.strip()
    if t.startswith("```"):
        t = t.strip("`")
        if t[:4].lower() == "json":
            t = t[4:]
    i, j = t.find("{"), t.rfind("}")
    if i != -1 and j != -1 and j > i:
        t = t[i:j + 1]
    return json.loads(t)

def ask(req):
    payload = {"model": MODEL, "max_tokens": 1024, "system": SYS,
               "messages": [{"role": "user", "content": json.dumps(req)},
                            {"role": "assistant", "content": "{"}]}
    r = urllib.request.urlopen(urllib.request.Request(
        URL, json.dumps(payload).encode(),
        {"content-type": "application/json", "x-api-key": KEY,
         "anthropic-version": "2023-06-01"}), timeout=120)
    out = json.loads(r.read().decode())
    if "content" not in out:
        raise RuntimeError("api: " + json.dumps(out)[:300])
    text = "".join(b.get("text", "") for b in out.get("content", []) if b.get("type") == "text")
    text = "{" + text  # we prefilled an opening brace
    try:
        return extract_json(text)
    except Exception:
        try:
            open(os.path.join(os.path.dirname(__file__), "last_raw.txt"), "w", encoding="utf-8").write(text)
        except Exception:
            pass
        raise

def _legal_ok(mv, st):
    try:
        r, c = int(mv["row"]), int(mv["col"])
    except Exception:
        return False
    if not (0 <= r <= 2 and 0 <= c <= 2):
        return False
    legal = st.get("legal")
    if legal is not None:
        return [r, c] in [list(x) for x in legal]
    board = st.get("board")
    if board is not None:
        return board[r][c] == ""
    return True

SAFE_SCENE = {"width": 64, "height": 20, "ops": [
    {"op": "vgradient", "x": 0, "y": 0, "w": 64, "h": 20, "glyph": " ", "from": [30, 30, 60], "to": [200, 120, 80]},
    {"op": "text", "x": 2, "y": 1, "s": "(brain scene unavailable)", "fg": [245, 245, 250]}]}
SAFE_SKETCH = {"sky": "orange", "ground": "navy", "shapes": [
    {"kind": "sun", "x": 48, "y": 6, "w": 3, "color": "gold"},
    {"kind": "mountain", "x": 18, "h": 8, "color": "darkgreen"},
    {"kind": "text", "x": 2, "y": 1, "s": "(sketch unavailable)", "color": "white"}]}

def decide(req):
    kind = req.get("kind")
    resp = {"protocol": "tvcp-ai/1", "kind": kind}
    last = ""
    for attempt in range(3):
        try:
            m = ask(req)
            if kind == "move":
                mv = m.get("move", m)
                if not isinstance(mv, dict):
                    mv = m
                if req.get("game") == "tictactoe":
                    if _legal_ok(mv, req.get("state") or {}):
                        resp["move"] = {"row": int(mv["row"]), "col": int(mv["col"])}
                        resp["reasoning"] = f"move:{MODEL}"
                        return resp
                    last = "illegal move"
                    continue
                if mv.get("word") or ("card_index" in mv) or mv.get("draw"):
                    resp["move"] = mv
                    resp["reasoning"] = f"move:{MODEL}"
                    return resp
                last = "no usable move"
                continue
            if kind == "draw":
                sc = m.get("scene", m)
                if isinstance(sc, dict) and sc.get("ops"):
                    resp["scene"] = sc
                    resp["reasoning"] = f"anthropic:{MODEL}"
                    return resp
                last = "no scene ops"
                continue
            if kind == "sketch":
                sk = m.get("sketch", m)
                if isinstance(sk, dict) and sk.get("shapes"):
                    resp["sketch"] = sk
                    resp["reasoning"] = f"sketch:{MODEL}"
                    return resp
                last = "no sketch shapes"
                continue
            resp["cards"] = m.get("cards", ["★"])
            resp["reasoning"] = f"anthropic:{MODEL}"
            return resp
        except Exception as e:
            last = str(e)
    resp["error"] = last
    if kind == "move":
        legal = (req.get("state") or {}).get("legal") or [[1, 1]]
        resp["move"] = {"row": legal[0][0], "col": legal[0][1]}
    elif kind == "draw":
        resp["scene"] = SAFE_SCENE
    elif kind == "sketch":
        resp["sketch"] = SAFE_SKETCH
    else:
        resp["cards"] = ["★"]
    return resp

class H(BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get("Content-Length", 0))
        req = json.loads(self.rfile.read(n) or b"{}")
        body = json.dumps(decide(req)).encode()
        self.send_response(200); self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body))); self.end_headers()
        self.wfile.write(body)
    def log_message(self, *a): pass

if __name__ == "__main__":
    if not KEY:
        print("warning: ANTHROPIC_API_KEY not set", file=sys.stderr)
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8092
    print(f"anthropic brain (tvcp-ai/1, model={MODEL}) on 127.0.0.1:{port} /v1/decide")
    HTTPServer(("127.0.0.1", port), H).serve_forever()
