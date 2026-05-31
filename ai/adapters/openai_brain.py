#!/usr/bin/env python3
"""tvcp-ai/1 brain adapter backed by any OpenAI-COMPATIBLE endpoint
(OpenAI, Azure, llama.cpp --api, vLLM, Anthropic-compatible gateways).

    export OPENAI_API_KEY=sk-...
    export OPENAI_BASE=https://api.openai.com/v1   # or your gateway
    OPENAI_MODEL=gpt-4o-mini python openai_brain.py 8091
"""
import json, os, sys, urllib.request
from http.server import BaseHTTPRequestHandler, HTTPServer

BASE  = os.environ.get("OPENAI_BASE", "https://api.openai.com/v1").rstrip("/")
KEY   = os.environ.get("OPENAI_API_KEY", "")
MODEL = os.environ.get("OPENAI_MODEL", "gpt-4o-mini")
TAG = "openai"


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

IMG_SYS = (
    'You paint pseudo-images for a terminal. Reply with ONLY one JSON object: no prose, '
    'no markdown, no extra keys. '
    'format=grid (default): {"grid":{"rows":[[C,C,C],[C,C,C]]}} a grid about 10-12 columns '
    'by 7-8 rows; each C is a COLOR NAME. Compose top-to-bottom (sky, subject, water/ground). '
    'Colors: navy blue skyblue teal green darkgreen gold amber orange purple slate white gray sand brown pink. '
    'format=glyphs: {"glyphs":{"bg":NAME,"palette":{"#":NAME,"~":NAME},"rows":["text art lines"]}}. '
    'format=sigils: {"sigils":{"sky":NAME,"ground":NAME,"items":[{"name":sun,"x":0.7,"y":0.2,"color":gold}]}} '
    'names: sun moon star mountain cloud wave boat anchor tree house flag. '
    'Output ONLY the JSON for the requested format.')


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
SAFE_IMAGE = {"format": "grid", "grid": {"rows": [
    ["navy", "navy", "purple", "amber", "gold", "amber", "purple", "navy"],
    ["navy", "purple", "amber", "gold", "white", "gold", "amber", "purple"],
    ["slate", "amber", "gold", "amber", "teal", "teal", "slate", "slate"],
    ["teal", "skyblue", "teal", "blue", "teal", "skyblue", "teal", "teal"]]}}


def _norm_image(spec, req):
    if not isinstance(spec, dict):
        return None
    if "format" not in spec:
        if spec.get("grid"):
            spec["format"] = "grid"
        elif spec.get("glyphs"):
            spec["format"] = "glyphs"
        elif spec.get("sigils"):
            spec["format"] = "sigils"
        elif spec.get("mixed"):
            spec["format"] = "mixed"
        elif spec.get("rows"):
            spec = {"format": "grid", "grid": {"rows": spec["rows"]}}
        else:
            return None
    if req.get("palette") and not spec.get("palette"):
        spec["palette"] = req["palette"]
    return spec


def decide(req):
    kind = req.get("kind")
    resp = {"protocol": "tvcp-ai/1", "kind": kind}
    last = ""
    for attempt in range(3):
        try:
            m = ask(req, IMG_SYS if kind == "image" else SYS)
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
                    resp["reasoning"] = f"{TAG}:{MODEL}"
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
            if kind == "image":
                spec = _norm_image(m.get("image", m), req)
                if spec is not None:
                    resp["image"] = spec
                    resp["reasoning"] = f"image:{MODEL}"
                    return resp
                last = "no usable image"
                continue
            resp["cards"] = m.get("cards", ["star"])
            resp["reasoning"] = f"{TAG}:{MODEL}"
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
    elif kind == "image":
        resp["image"] = SAFE_IMAGE
    else:
        resp["cards"] = ["star"]
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


def ask(req, system=SYS):
    payload = {"model": MODEL, "temperature": 0.2,
               "response_format": {"type": "json_object"},
               "messages": [{"role": "system", "content": system},
                            {"role": "user", "content": json.dumps(req)}]}
    r = urllib.request.urlopen(urllib.request.Request(
        BASE + "/chat/completions", json.dumps(payload).encode(),
        {"Content-Type": "application/json", "Authorization": "Bearer " + KEY}), timeout=120)
    out = json.loads(r.read().decode())
    return json.loads(out["choices"][0]["message"]["content"])


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8091
    print(f"openai-compatible brain (tvcp-ai/1, model={MODEL}) on 127.0.0.1:{port} /v1/decide")
    HTTPServer(("127.0.0.1", port), H).serve_forever()
