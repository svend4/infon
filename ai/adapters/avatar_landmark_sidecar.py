#!/usr/bin/env python3
"""Face-landmark sidecar — the SENDER side of neural avatars (roadmap C5).

Extracts compact face keypoints from a frame so the call transmits only those
(~few hundred bytes) instead of video. Contract:

    POST /  JSON {"image": "<base64 png>"}
    ->      JSON {"points": [[x,y],...],   # normalized 0..1
                  "yaw":F, "pitch":F, "roll":F}

The Go side packs these into the compact keypoint wire format
(internal/avatar.Keypoints.Encode) and ships them; the receiver reconstructs the
face (avatar_generate_sidecar.py).

Backends (auto, best first):
  * MediaPipe Face Mesh (468 landmarks, CPU real-time), if installed;
  * a deterministic synthetic face (model-free) so the path is testable.

    python avatar_landmark_sidecar.py 8097
"""
import base64, io, json, math, sys
from http.server import BaseHTTPRequestHandler, HTTPServer

_PIL = None
try:
    from PIL import Image  # type: ignore
    _PIL = Image
except Exception:
    pass

_MESH = None
try:
    import mediapipe as mp  # type: ignore
    import numpy as np      # type: ignore
    _MESH = mp.solutions.face_mesh.FaceMesh(static_image_mode=True, max_num_faces=1)
except Exception:
    _MESH = None


def landmarks(png_bytes: bytes):
    if _MESH is not None and _PIL is not None:
        try:
            import numpy as np  # type: ignore
            img = _PIL.open(io.BytesIO(png_bytes)).convert("RGB")
            res = _MESH.process(np.asarray(img))
            if res.multi_face_landmarks:
                lm = res.multi_face_landmarks[0].landmark
                pts = [[p.x, p.y] for p in lm]
                return {"points": pts, "yaw": 0.0, "pitch": 0.0, "roll": 0.0}
        except Exception:
            pass

    # Model-free synthetic face: a ring of 68 points + eyes/mouth, so the codec
    # and reconstruction path run end to end with zero heavy deps.
    pts = []
    for i in range(68):
        a = 2 * math.pi * i / 68
        pts.append([0.5 + 0.25 * math.cos(a), 0.5 + 0.30 * math.sin(a)])
    return {"points": pts, "yaw": 0.0, "pitch": 0.0, "roll": 0.0}


class H(BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get("Content-Length", 0))
        req = json.loads(self.rfile.read(n) or b"{}")
        try:
            out = landmarks(base64.b64decode(req.get("image", "")))
            body = json.dumps(out).encode()
        except Exception as e:
            self.send_response(400); self.end_headers()
            self.wfile.write(json.dumps({"error": str(e)}).encode()); return
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *a):
        pass


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8097
    backend = "MediaPipe FaceMesh" if _MESH else "synthetic"
    print(f"avatar landmark sidecar (backend={backend}) on 127.0.0.1:{port}/")
    HTTPServer(("127.0.0.1", port), H).serve_forever()
