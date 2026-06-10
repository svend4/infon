#!/usr/bin/env python3
"""yijing_brain.py - a tvcp-ai/1 brain that "thinks in hexagrams".

A bridge from the YiJing-Transformer (github.com/svend4/pro2) to the infon world
director. It reads the prompt, decides a single I-Ching hexagram for the region
(six lines = two trigrams), and authors a rayscene scene graph from the two
trigrams' themes - so a transformer that thinks in hexagrams literally directs the
hexagram worlds of `rayexplore` / `raymeet`.

    python3 ai/adapters/yijing_brain.py 8095
    BRAIN_URL=http://127.0.0.1:8095/v1/decide go run ./cmd/rayexplore -hexagram ...

How the hexagram is chosen:
  - If pro2 is importable and a checkpoint is given (env YIJING_CKPT), the model
    embeds the prompt into Q6 = {-1,+1}^6 and the sign of each coordinate is a line
    (the model's own "reading").
  - Otherwise a deterministic hash of the prompt picks the hexagram, so the bridge
    always runs as a reference sidecar with no heavy dependencies.

Either way the hexagram -> world mapping is the same, mirroring pkg/raydir's
Hexagram (G4): bit 0 = bottom line, lines 0..2 = lower trigram, 3..5 = upper.

    python3 ai/adapters/yijing_brain.py --selftest   # validate offline
"""
import hashlib
import json
import os
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer

# The eight trigrams (3-bit value -> name, sky tint, and scene objects). The
# object kinds are exactly those pkg/raydir accepts (validated/clamped on arrival).
TRIGRAMS = [
    # idx, name, sky_top, sky_bottom, objects(builder takes a base x offset)
    ("Earth", [0.55, 0.5, 0.45], [0.7, 0.65, 0.6],
     lambda x: [{"kind": "mesh", "name": "rock", "x": x, "y": 0.5, "z": 2, "r": 1, "color": [0.42, 0.4, 0.38], "rough": 0.7}]),
    ("Mountain", [0.45, 0.55, 0.7], [0.8, 0.82, 0.85],
     lambda x: [{"kind": "mesh", "name": "crystal", "x": x, "y": 1, "z": 1, "r": 1.2, "color": [0.4, 0.7, 0.95], "reflect": 0.2},
                {"kind": "mesh", "name": "rock", "x": x - 2, "y": 0.5, "z": 3, "r": 1}]),
    ("Water", [0.2, 0.35, 0.55], [0.5, 0.6, 0.7],
     lambda x: [{"kind": "water", "y": 0.1, "color": [0.1, 0.3, 0.45], "reflect": 0.55}]),
    ("Wind", [0.5, 0.7, 0.8], [0.85, 0.9, 0.85],
     lambda x: [{"kind": "tree", "x": x, "z": 1, "r": 1.2}, {"kind": "tree", "x": x - 2.5, "z": 3, "r": 1.0}]),
    ("Thunder", [0.25, 0.22, 0.35], [0.6, 0.45, 0.4],
     lambda x: [{"kind": "tree", "x": x, "z": 1, "r": 1.3},
                {"x": x + 1.5, "y": 2, "z": 2, "r": 0.5, "emit": [5, 4, 6]}]),
    ("Fire", [0.5, 0.3, 0.2], [0.95, 0.6, 0.35],
     lambda x: [{"x": x, "y": 2, "z": 2, "r": 0.7, "emit": [7, 5, 3]}]),
    ("Lake", [0.45, 0.6, 0.85], [0.9, 0.9, 0.95],
     lambda x: [{"kind": "water", "y": 0.05, "color": [0.12, 0.32, 0.48], "reflect": 0.6}]),
    ("Heaven", [0.45, 0.62, 0.95], [0.95, 0.97, 1.0],
     lambda x: [{"x": x, "y": 4, "z": 3, "r": 1, "color": [0.9, 0.9, 0.95], "reflect": 0.3}]),
]


def hexagram_from_prompt(prompt):
    """Return a six-bit hexagram (0..63) for the prompt, bottom line = bit 0."""
    h = _pro2_hexagram(prompt)
    if h is not None:
        return h & 63
    # deterministic fallback: first byte of sha256
    return hashlib.sha256((prompt or "").encode("utf-8")).digest()[0] & 63


def _pro2_hexagram(prompt):
    """Try the YiJing-Transformer: embed the prompt into Q6 and read the lines.
    Returns None if pro2 / torch / a checkpoint are unavailable (the common case
    for a dependency-free sidecar)."""
    ckpt = os.environ.get("YIJING_CKPT")
    if not ckpt:
        return None
    try:  # pragma: no cover - exercised only when pro2 is installed
        import torch  # noqa: F401
        from yijing_transformer.inference import embed_to_hexagram  # type: ignore
        return int(embed_to_hexagram(prompt, ckpt)) & 63
    except Exception:
        return None


def trigram(idx):
    return TRIGRAMS[idx & 7]


def scene_from_hexagram(h):
    """Author a rayscene scene graph from a hexagram's two trigrams."""
    lower = h & 7
    upper = (h >> 3) & 7
    lo_name, _, _, lo_objs = trigram(lower)
    up_name, up_top, up_bot, up_objs = trigram(upper)
    objects = [{"kind": "plane", "color": [0.55, 0.55, 0.58]}]
    objects += lo_objs(-2.5)        # lower trigram on the near/left
    objects += up_objs(2.5)         # upper trigram on the far/right
    objects.append({"x": 0, "y": 6, "z": -1, "r": 0.7, "emit": [16, 16, 15]})  # a sun
    return {
        "objects": objects,
        "light": [6, 9, -4],
        "skyTop": up_top,
        "skyBottom": up_bot,
        "name": f"{up_name} over {lo_name}",
    }


def decide(req):
    kind = req.get("kind")
    resp = {"protocol": "tvcp-ai/1", "kind": kind}
    if kind == "move" and req.get("game") == "rayscene":
        state = req.get("state")
        prompt = ""
        if isinstance(state, (bytes, bytearray)):
            state = state.decode("utf-8", "ignore")
        if isinstance(state, str):
            try:
                state = json.loads(state)
            except Exception:
                state = {"prompt": state}
        if isinstance(state, dict):
            prompt = str(state.get("prompt", ""))
        h = hexagram_from_prompt(prompt)
        resp["ray"] = scene_from_hexagram(h)
        resp["reasoning"] = f"yijing:hexagram-{h}"
        return resp
    # other kinds: a minimal, valid default so the bridge never breaks a client.
    if kind == "move":
        resp["move"] = {"row": 0, "col": 0}
    else:
        resp["cards"] = ["star"]
    resp["reasoning"] = "yijing:default"
    return resp


class H(BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get("Content-Length", 0))
        req = json.loads(self.rfile.read(n) or b"{}")
        body = json.dumps(decide(req)).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *a):
        pass


_KINDS = {"sphere", "box", "pyramid", "cylinder", "tree", "house", "plane", "mesh", "fractal", "sdf", "water", ""}


def _selftest():
    # determinism
    assert hexagram_from_prompt("a forest") == hexagram_from_prompt("a forest")
    # variety across prompts
    seen = {hexagram_from_prompt(f"world {i}") for i in range(40)}
    assert len(seen) > 5, f"too few distinct hexagrams: {len(seen)}"
    # every hexagram authors a valid rayscene
    for h in range(64):
        sc = scene_from_hexagram(h)
        assert isinstance(sc["objects"], list) and sc["objects"], "empty scene"
        assert sc["name"] and " over " in sc["name"]
        for o in sc["objects"]:
            assert o.get("kind", "") in _KINDS, f"bad kind {o.get('kind')!r}"
            for c in o.get("color", []):
                assert 0 <= c <= 1, f"colour out of range: {c}"
    # the tvcp-ai/1 envelope for a rayscene request
    r = decide({"protocol": "tvcp-ai/1", "kind": "move", "game": "rayscene",
                "state": json.dumps({"prompt": "a forest at night"})})
    assert r["protocol"] == "tvcp-ai/1" and r["kind"] == "move"
    assert isinstance(r["ray"]["objects"], list) and r["ray"]["objects"]
    print("yijing_brain selftest OK")


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "--selftest":
        _selftest()
        sys.exit(0)
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8095
    print(f"yijing brain (tvcp-ai/1) on 127.0.0.1:{port} /v1/decide")
    HTTPServer(("127.0.0.1", port), H).serve_forever()
