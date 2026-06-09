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
# Or read the key from a file OUTSIDE the repo (never pasted in chat):
_KF = os.environ.get("OPENAI_KEY_FILE", os.path.expanduser("~/.tvcp_openai.key"))
if not KEY and os.path.exists(_KF):
    KEY = open(_KF, encoding="utf-8").read().strip()
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
       "fit the message mood, from: star heart check fire smile sad music sun warning x thumbsup. "
       "kind=move with game=world -> reply a JSON object with integer keys fold, rise, "
       "spin, camera, each -1, 0 or 1 (fold opens the body, rise raises the terrain, spin orbits "
       "the scene, camera pans); state gives fold_pct, relief_pct, camera_deg to react to. "
       "kind=move with game=ray -> reply a JSON object with key ray = a list of {id,dx,dy,dz}, "
       "moving each sphere in state.spheres (by id); dx,dy,dz each -1,0,1. "
       "kind=move with game=rayscene -> reply a JSON object with key ray = a scene graph "
       "{objects:[{kind,name,tex,x,y,z,r,s:[hx,hy,hz],color:[r,g,b],emit:[r,g,b],glass,metal,reflect,rough}],light:[x,y,z],"
       "skyTop:[r,g,b],skyBottom:[r,g,b]}; kind is sphere, box/pyramid/cylinder (s=half-extents), tree, house, mesh, or plane; "
       "for kind=mesh set name to a model (crystal, rock); tex names a surface texture (checker/marble/wood/stone/clouds); author the world that "
       "state.prompt describes (use emit for lights, glass~1.5, metal/reflect for shiny). "
       "kind=move with game=rpg -> reply a JSON object with key rpg = a list of {id,dx,dy}, where "
       "for each of your units (state.units, by id) dx,dy are -1,0,1 stepping toward the nearest "
       "enemy (state.enemies); combat is automatic on contact.")

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

TANGRAM_SYS = ("You assemble tangram figures from a FIXED set of sub-cell pieces. Reply with ONLY "
               "one JSON object, no prose, no code fences. The request state has h, w, palette "
               "(allowed glyph tokens) and target: a grid of glyph tokens where \"\" is an empty "
               "cell. Reproduce the shape exactly: for EVERY non-empty target cell, emit one piece "
               "with the SAME glyph token at the SAME y (row 0..h-1) and x (column 0..w-1). Use ONLY "
               "tokens that appear in palette. Reply "
               "{\"tangram\":{\"h\":H,\"w\":W,\"pieces\":[{\"glyph\":TOK,\"y\":Y,\"x\":X}]}}. "
               "Never put two pieces in one cell. Output ONLY that JSON.")


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
            sysmsg = IMG_SYS if kind == "image" else SYS
            if kind == "move" and req.get("game") == "tangram":
                sysmsg = TANGRAM_SYS
            m = ask(req, sysmsg)
            if kind == "move":
                if req.get("game") == "tangram":
                    fig = m.get("tangram", m)
                    if isinstance(fig, dict) and fig.get("pieces"):
                        st = req.get("state") or {}
                        fig.setdefault("h", st.get("h"))
                        fig.setdefault("w", st.get("w"))
                        resp["tangram"] = fig
                        resp["reasoning"] = f"tangram:{MODEL}"
                        return resp
                    last = "no tangram pieces"
                    continue
                if req.get("game") == "world":
                    d = m.get("world", m)
                    if not isinstance(d, dict):
                        d = m
                    def _clip(v):
                        try:
                            return max(-1, min(1, int(v)))
                        except Exception:
                            return 0
                    resp["world"] = {"fold": _clip(d.get("fold", 0)), "rise": _clip(d.get("rise", 0)),
                                     "spin": _clip(d.get("spin", 0)), "camera": _clip(d.get("camera", 0))}
                    resp["reasoning"] = f"world:{MODEL}"
                    return resp
                if req.get("game") == "ray":
                    mv = m.get("ray", m)
                    if isinstance(mv, list):
                        resp["ray"] = mv
                        resp["reasoning"] = "ray"
                        return resp
                    last = "no ray moves"
                    continue
                if req.get("game") == "rayscene":
                    sc = m.get("ray", m)
                    if isinstance(sc, dict) and sc.get("objects"):
                        resp["ray"] = sc
                        resp["reasoning"] = "rayscene"
                        return resp
                    last = "no rayscene"
                    continue
                if req.get("game") == "rpg":
                    mv = m.get("rpg", m)
                    if isinstance(mv, list):
                        resp["rpg"] = mv
                        resp["reasoning"] = f"rpg:{MODEL}"
                        return resp
                    last = "no rpg moves"
                    continue
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
        if req.get("game") == "world":
            resp["world"] = {"fold": 1, "rise": 1, "spin": 1, "camera": 1}
        elif req.get("game") == "rpg":
            resp["rpg"] = []
        elif req.get("game") == "ray":
            resp["ray"] = []
        elif req.get("game") == "rayscene":
            resp["ray"] = {
                "objects": [
                    {"kind": "plane", "color": [0.8, 0.8, 0.8]},
                    {"x": 0, "y": 1, "z": 0, "r": 1, "color": [0.8, 0.4, 0.4]},
                    {"x": 0, "y": 6, "z": -1, "r": 0.7, "emit": [18, 18, 17]},
                ],
                "light": [6, 9, -4], "skyTop": [0.4, 0.55, 0.85], "skyBottom": [0.85, 0.88, 0.95],
            }
        elif req.get("game") == "tangram":
            st = req.get("state") or {}
            resp["tangram"] = {"h": st.get("h", 1), "w": st.get("w", 1), "pieces": []}
        else:
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
