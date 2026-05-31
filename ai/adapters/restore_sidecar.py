#!/usr/bin/env python3
"""Image restoration / super-resolution sidecar for TVCP (roadmap C1).

Speaks the tiny restoration contract used by internal/aisource.RestoreBackend:

    POST /  JSON {"image": "<base64 png>", "scale": N}
    ->      raw image bytes  (Content-Type: image/png)

Run it next to the Go program; point the backend at it with:

    RESTORE_API_URL=http://127.0.0.1:8094/   tvcp synth neural "..."

Backends (auto-detected, best first):
  * Real-ESRGAN via the `realesrgan` package, if installed (true neural SR);
  * Pillow LANCZOS upscale + light sharpen (always available, model-free);
  * identity (returns the input unchanged) as the final fallback.

This keeps the sidecar runnable with ZERO heavy deps for testing, while leaving
a clear slot to drop in a real model.

    python restore_sidecar.py 8094
"""
import base64, io, json, os, sys
from http.server import BaseHTTPRequestHandler, HTTPServer

# --- backend selection -------------------------------------------------------

_PIL = None
try:
    from PIL import Image, ImageFilter  # type: ignore
    _PIL = Image
except Exception:
    pass

_REAL_ESRGAN = None
try:
    # Optional: a real super-resolution model. Heavy; only used if present.
    from realesrgan import RealESRGAN  # type: ignore
    import torch  # type: ignore
    _dev = "cuda" if torch.cuda.is_available() else "cpu"
    _REAL_ESRGAN = RealESRGAN(_dev, scale=2)
    _REAL_ESRGAN.load_weights("weights/RealESRGAN_x2.pth", download=True)
except Exception:
    _REAL_ESRGAN = None


def restore(png_bytes: bytes, scale: int) -> bytes:
    """Return restored PNG bytes. Falls back gracefully when no model is present."""
    if _PIL is None:
        return png_bytes  # identity: no imaging libs at all

    img = _PIL.open(io.BytesIO(png_bytes)).convert("RGB")

    if _REAL_ESRGAN is not None:
        try:
            img = _REAL_ESRGAN.predict(img)
        except Exception:
            pass  # fall through to classical path
    elif scale and scale > 1:
        img = img.resize((img.width * scale, img.height * scale), _PIL.LANCZOS)
        img = img.filter(ImageFilter.SHARPEN)
    else:
        # scale==1: just denoise/sharpen lightly (the terminal-domain niche).
        img = img.filter(ImageFilter.SHARPEN)

    out = io.BytesIO()
    img.save(out, format="PNG")
    return out.getvalue()


# --- HTTP server -------------------------------------------------------------

class H(BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get("Content-Length", 0))
        req = json.loads(self.rfile.read(n) or b"{}")
        try:
            png_in = base64.b64decode(req.get("image", ""))
            scale = int(req.get("scale", 1) or 1)
            png_out = restore(png_in, scale)
        except Exception as e:
            self.send_response(400)
            self.end_headers()
            self.wfile.write(json.dumps({"error": str(e)}).encode())
            return
        self.send_response(200)
        self.send_header("Content-Type", "image/png")
        self.send_header("Content-Length", str(len(png_out)))
        self.end_headers()
        self.wfile.write(png_out)

    def log_message(self, *a):
        pass


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8094
    backend = "Real-ESRGAN" if _REAL_ESRGAN else ("Pillow LANCZOS+sharpen" if _PIL else "identity")
    print(f"restore sidecar (backend={backend}) on 127.0.0.1:{port}/")
    HTTPServer(("127.0.0.1", port), H).serve_forever()
