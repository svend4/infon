#!/usr/bin/env python3
"""Face-generation sidecar — the RECEIVER side of neural avatars (roadmap C5).

Reconstructs a face image from received keypoints + a one-time reference image,
the heavy half of C5. Contract (matches internal/avatar.Reconstructor):

    POST /  JSON {"reference":"<base64 png>", "points":[[x,y],...],
                  "yaw":F,"pitch":F,"roll":F}
    ->      raw image bytes (the reconstructed face)

Backends (auto, best first):
  * a real talking-head model (LivePortrait / Thin-Plate-Spline / face-vid2vid)
    if you wire one in at the marked slot — needs a GPU;
  * Pillow: warp/redraw the reference toward the keypoints (crude, model-free);
  * a procedural face drawn from the keypoints (no deps), so the path runs.

    python avatar_generate_sidecar.py 8098
"""
import base64, io, json, sys
from http.server import BaseHTTPRequestHandler, HTTPServer

_PIL = None
try:
    from PIL import Image, ImageDraw  # type: ignore
    _PIL = Image
    _Draw = ImageDraw
except Exception:
    pass

# ---- slot for a real model -------------------------------------------------
# Drop a talking-head model here (LivePortrait / TPS-Motion / face-vid2vid):
#   from liveportrait import LivePortrait
#   _MODEL = LivePortrait(device="cuda")
_MODEL = None


def generate(reference_png: bytes, points, pose) -> bytes:
    if _MODEL is not None:
        try:
            return _MODEL.animate(reference_png, points, pose)  # -> png bytes
        except Exception:
            pass

    # Model-free: draw the keypoints as a face on a canvas. Crude but proves the
    # reconstruction path; a real model replaces this with a photorealistic warp.
    W = H = 128
    if _PIL is not None:
        img = _PIL.new("RGB", (W, H), (20, 20, 30))
        d = _Draw.Draw(img)
        if reference_png:
            try:
                ref = _PIL.open(io.BytesIO(reference_png)).convert("RGB").resize((W, H))
                img.paste(ref, (0, 0))
                d = _Draw.Draw(img)
            except Exception:
                pass
        for (x, y) in points:
            px, py = int(x * W), int(y * H)
            d.ellipse([px - 1, py - 1, px + 1, py + 1], fill=(240, 220, 200))
        out = io.BytesIO(); img.save(out, format="PNG")
        return out.getvalue()

    # No PIL at all: return a minimal valid PNG built by hand.
    return _tiny_png(points)


def _tiny_png(points):
    import struct, zlib
    W = H = 32
    grid = [[(20, 20, 30)] * W for _ in range(H)]
    for (x, y) in points:
        px = min(W - 1, max(0, int(x * W)))
        py = min(H - 1, max(0, int(y * H)))
        grid[py][px] = (240, 220, 200)
    raw = b""
    for row in grid:
        raw += b"\x00" + b"".join(struct.pack("BBB", *c) for c in row)

    def chunk(t, d):
        c = t + d
        return struct.pack(">I", len(d)) + c + struct.pack(">I", zlib.crc32(c) & 0xffffffff)

    png = b"\x89PNG\r\n\x1a\n"
    png += chunk(b"IHDR", struct.pack(">IIBBBBB", W, H, 8, 2, 0, 0, 0))
    png += chunk(b"IDAT", zlib.compress(raw))
    png += chunk(b"IEND", b"")
    return png


class H(BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get("Content-Length", 0))
        req = json.loads(self.rfile.read(n) or b"{}")
        try:
            ref = base64.b64decode(req.get("reference", "") or "")
            pts = req.get("points", [])
            pose = (req.get("yaw", 0), req.get("pitch", 0), req.get("roll", 0))
            out = generate(ref, pts, pose)
        except Exception as e:
            self.send_response(400); self.end_headers()
            self.wfile.write(json.dumps({"error": str(e)}).encode()); return
        self.send_response(200)
        self.send_header("Content-Type", "image/png")
        self.send_header("Content-Length", str(len(out)))
        self.end_headers()
        self.wfile.write(out)

    def log_message(self, *a):
        pass


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8098
    backend = "talking-head model" if _MODEL else ("Pillow draw" if _PIL else "handmade PNG")
    print(f"avatar generate sidecar (backend={backend}) on 127.0.0.1:{port}/")
    HTTPServer(("127.0.0.1", port), H).serve_forever()
