#!/usr/bin/env python3
"""equipment_brain.py - a tvcp-ai/1 brain that reacts to a machine fleet.

A bridge in the spirit of info150's triage4: it receives a fleet's assessed state
(each unit's severity / level / dominant fault) and authors a rayscene that *shows*
the fleet - small machines whose status light goes green (calm) to red (critical),
the worst ones emitting an alarm halo - plus a short operator cue. So an external
brain can be the "react" stage of the camera-observe -> analyse -> react loop,
exactly as yijing_brain is the director for the hexagram worlds.

    python3 ai/adapters/equipment_brain.py 8096
    BRAIN_URL=http://127.0.0.1:8096/v1/decide go run ./cmd/rayfleet

It speaks the same rayscene as pkg/fleet.SceneFromAssessments, but is written
independently: the Go engine and this brain meet at the rayscene JSON, not in code
(the house rule - see docs/Q6_INTEROP.md).

    python3 ai/adapters/equipment_brain.py --selftest   # validate offline
"""
import json
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer

# object kinds pkg/raydir accepts (validated/clamped on arrival).
_KINDS = {"sphere", "box", "pyramid", "cylinder", "tree", "house", "plane", "mesh", "fractal", "sdf", "water", ""}


def severity_color(s):
    """green (calm) -> amber -> red (critical), matching fleet.severityColor."""
    s = max(0.0, min(1.0, s))
    if s < 0.5:
        t = s / 0.5
        return [0.2 + 0.8 * t, 0.8, 0.2]
    t = (s - 0.5) / 0.5
    return [1.0, 0.8 * (1 - t), 0.15]


def scene_from_fleet(units):
    """Author a rayscene showing the fleet (mirrors fleet.SceneFromAssessments)."""
    objects = [{"kind": "plane", "color": [0.3, 0.31, 0.34]}]
    n = len(units)
    worst = "OK"
    rank = {"OK": 0, "WATCH": 1, "WARN": 2, "CRITICAL": 3}
    for i, u in enumerate(units):
        x = i * 2.2 - (n - 1) * 1.1
        sev = float(u.get("severity", 0.0))
        level = str(u.get("level", "OK"))
        col = severity_color(sev)
        objects.append({"kind": "box", "x": x, "y": 0.6, "z": 2, "s": [0.5, 0.6, 0.5],
                        "color": [0.5, 0.52, 0.55], "metal": 0.6, "rough": 0.3})
        light = {"x": x, "y": 1.5, "z": 2, "r": 0.32, "color": col}
        if rank.get(level, 0) >= rank["WARN"]:
            k = 2.0 + 6.0 * sev
            light["emit"] = [col[0] * k, col[1] * k, col[2] * k]
        objects.append(light)
        if rank.get(level, 0) > rank.get(worst, 0):
            worst = level
    objects.append({"y": 6, "z": -1, "r": 0.7, "emit": [14, 14, 13]})
    return {
        "objects": objects,
        "light": [6, 9, -4],
        "skyTop": [0.16, 0.18, 0.26],
        "skyBottom": [0.4, 0.42, 0.5],
        "name": f"fleet: {n} units, worst {worst}",
    }


def _fleet_from_state(state):
    if isinstance(state, (bytes, bytearray)):
        state = state.decode("utf-8", "ignore")
    if isinstance(state, str):
        try:
            state = json.loads(state)
        except Exception:
            state = {}
    if isinstance(state, dict):
        f = state.get("fleet")
        if isinstance(f, list):
            return f
    return []


def _cue(units):
    crit = sum(1 for u in units if str(u.get("level")) == "CRITICAL")
    warn = sum(1 for u in units if str(u.get("level")) == "WARN")
    return f"equipment: {len(units)} units, {crit} critical, {warn} warning"


def decide(req):
    kind = req.get("kind")
    resp = {"protocol": "tvcp-ai/1", "kind": kind}
    if kind == "move" and req.get("game") == "rayscene":
        units = _fleet_from_state(req.get("state"))
        resp["ray"] = scene_from_fleet(units)
        resp["reasoning"] = _cue(units)
        return resp
    if kind == "move":
        resp["move"] = {"row": 0, "col": 0}
    else:
        resp["cards"] = ["gear"]
    resp["reasoning"] = "equipment:default"
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


def _selftest():
    # colour ramp endpoints and monotonic red
    assert severity_color(0.0)[1] > 0.5 and severity_color(1.0)[0] == 1.0
    assert severity_color(1.0)[1] < severity_color(0.0)[1]
    # a fleet with a critical unit authors a valid, emitting scene
    units = [
        {"unit": "amr-1", "severity": 0.2, "level": "OK", "worst": "thermal"},
        {"unit": "parkbot", "severity": 0.85, "level": "CRITICAL", "worst": "thermal"},
    ]
    sc = scene_from_fleet(units)
    assert sc["objects"] and "worst CRITICAL" in sc["name"]
    emits = 0
    for o in sc["objects"]:
        assert o.get("kind", "") in _KINDS, f"bad kind {o.get('kind')!r}"
        for c in o.get("color", []):
            assert 0 <= c <= 1, f"colour out of range: {c}"
        if o.get("emit"):
            emits += 1
    assert emits >= 2, f"a critical fleet should emit (alarm + sun), got {emits}"
    # the tvcp-ai/1 envelope for a rayscene request
    r = decide({"protocol": "tvcp-ai/1", "kind": "move", "game": "rayscene",
                "state": json.dumps({"prompt": "fleet status", "fleet": units})})
    assert r["protocol"] == "tvcp-ai/1" and isinstance(r["ray"]["objects"], list) and r["ray"]["objects"]
    # empty fleet still authors a valid (ground + sun) scene
    assert len(scene_from_fleet([])["objects"]) >= 2
    print("equipment_brain selftest OK")


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "--selftest":
        _selftest()
        sys.exit(0)
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8096
    print(f"equipment brain (tvcp-ai/1) on 127.0.0.1:{port} /v1/decide")
    HTTPServer(("127.0.0.1", port), H).serve_forever()
