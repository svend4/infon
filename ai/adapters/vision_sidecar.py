#!/usr/bin/env python3
"""Vision-analysis sidecar for TVCP (roadmap C3, external-model tier).

Speaks the contract used by internal/vision.HTTPAnalyzer:

    POST /  JSON {"image": "<base64 png>"}
    ->      JSON {"detections": [
                {"label","confidence","x","y","w","h","points":[[x,y],...]}]}

Coordinates are NORMALIZED 0..1 (resolution-independent), so the Go side maps
them onto the terminal grid.

Backends (auto-detected, best first):
  * ultralytics YOLO (object detection), if installed;
  * MediaPipe face detection, if installed;
  * a deterministic "center box" stub (model-free) so the sidecar runs and the
    overlay path is testable with zero heavy deps.

    python vision_sidecar.py 8096
"""
import base64, io, json, sys
from http.server import BaseHTTPRequestHandler, HTTPServer

_PIL = None
try:
    from PIL import Image  # type: ignore
    _PIL = Image
except Exception:
    pass

_YOLO = None
try:
    from ultralytics import YOLO  # type: ignore
    _YOLO = YOLO("yolov8n.pt")  # small, fast
except Exception:
    _YOLO = None

_MP = None
try:
    import mediapipe as mp  # type: ignore
    _MP = mp.solutions.face_detection.FaceDetection(min_detection_confidence=0.5)
except Exception:
    _MP = None


def analyze(png_bytes: bytes):
    """Return a list of normalized detections."""
    if _PIL is None:
        # No imaging at all: emit a single deterministic center box so the Go
        # overlay path can be exercised end to end.
        return [{"label": "stub", "confidence": 1.0, "x": 0.25, "y": 0.25, "w": 0.5, "h": 0.5}]

    img = _PIL.open(io.BytesIO(png_bytes)).convert("RGB")
    W, H = img.size
    dets = []

    if _YOLO is not None:
        try:
            res = _YOLO(img, verbose=False)[0]
            for b in res.boxes:
                x1, y1, x2, y2 = b.xyxy[0].tolist()
                cls = int(b.cls[0])
                dets.append({
                    "label": res.names.get(cls, str(cls)),
                    "confidence": float(b.conf[0]),
                    "x": x1 / W, "y": y1 / H, "w": (x2 - x1) / W, "h": (y2 - y1) / H,
                })
            return dets
        except Exception:
            pass

    if _MP is not None:
        try:
            import numpy as np  # type: ignore
            res = _MP.process(np.asarray(img))
            if res.detections:
                for d in res.detections:
                    bb = d.location_data.relative_bounding_box
                    dets.append({
                        "label": "face",
                        "confidence": float(d.score[0]),
                        "x": bb.xmin, "y": bb.ymin, "w": bb.width, "h": bb.height,
                    })
            return dets
        except Exception:
            pass

    # Model-free fallback: a centered box (keeps the path runnable for tests).
    return [{"label": "frame", "confidence": 1.0, "x": 0.2, "y": 0.2, "w": 0.6, "h": 0.6}]


class H(BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get("Content-Length", 0))
        req = json.loads(self.rfile.read(n) or b"{}")
        try:
            dets = analyze(base64.b64decode(req.get("image", "")))
            body = json.dumps({"detections": dets}).encode()
        except Exception as e:
            self.send_response(400)
            self.end_headers()
            self.wfile.write(json.dumps({"error": str(e)}).encode())
            return
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *a):
        pass


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8096
    backend = "YOLO" if _YOLO else ("MediaPipe" if _MP else ("center-box stub" if _PIL else "no-PIL stub"))
    print(f"vision sidecar (backend={backend}) on 127.0.0.1:{port}/")
    HTTPServer(("127.0.0.1", port), H).serve_forever()
