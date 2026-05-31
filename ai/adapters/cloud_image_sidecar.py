#!/usr/bin/env python3
"""Cloud image-generation proxy sidecar for TVCP (no local GPU needed).

Exposes the raster contract used by internal/aisource.ImageBackend:

    POST /  JSON {"prompt": "...", "width": W, "height": H}
    ->      raw PNG bytes

…and forwards it to a hosted text-to-image provider, so `tvcp synth neural`
works with zero local model. Point the Go side at this sidecar:

    IMAGE_API_URL=http://127.0.0.1:8099/   tvcp synth neural "a calm bay"

Provider chosen by CLOUD_PROVIDER (default: replicate):

  replicate   needs REPLICATE_API_TOKEN; model via REPLICATE_MODEL
              (default: black-forest-labs/flux-schnell — fast & cheap)
  fal         needs FAL_KEY; model via FAL_MODEL (default: fal-ai/flux/schnell)
  openai      needs OPENAI_API_KEY; uses the images API (gpt-image-1/dall-e-3)

Keys are read from the environment and never logged. Falls back to a clear HTTP
error if the provider call fails, so the Go side keeps showing the last frame.

    python cloud_image_sidecar.py 8099
"""
import base64, io, json, os, sys, time, urllib.request, urllib.error
from http.server import BaseHTTPRequestHandler, HTTPServer

PROVIDER = os.environ.get("CLOUD_PROVIDER", "replicate").lower()


def _http_json(url, payload, headers, timeout=120):
    req = urllib.request.Request(url, json.dumps(payload).encode(), headers)
    with urllib.request.urlopen(req, timeout=timeout) as r:
        return json.loads(r.read().decode())


def _fetch_bytes(url, timeout=120):
    with urllib.request.urlopen(url, timeout=timeout) as r:
        return r.read()


# --- providers: each returns PNG/JPEG bytes for (prompt,w,h) -----------------

def gen_replicate(prompt, w, h):
    token = os.environ["REPLICATE_API_TOKEN"]
    model = os.environ.get("REPLICATE_MODEL", "black-forest-labs/flux-schnell")
    hdr = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}
    # Create a prediction, then poll until it completes.
    out = _http_json(
        f"https://api.replicate.com/v1/models/{model}/predictions",
        {"input": {"prompt": prompt, "width": w, "height": h}},
        {**hdr, "Prefer": "wait"},  # 'Prefer: wait' returns synchronously when possible
    )
    for _ in range(60):
        status = out.get("status")
        if status == "succeeded":
            url = out["output"][0] if isinstance(out.get("output"), list) else out.get("output")
            return _fetch_bytes(url)
        if status in ("failed", "canceled"):
            raise RuntimeError("replicate: " + json.dumps(out.get("error"))[:200])
        time.sleep(1)
        out = _http_get(out["urls"]["get"], hdr)
    raise RuntimeError("replicate: timed out")


def _http_get(url, headers):
    req = urllib.request.Request(url, headers=headers)
    with urllib.request.urlopen(req, timeout=120) as r:
        return json.loads(r.read().decode())


def gen_fal(prompt, w, h):
    key = os.environ["FAL_KEY"]
    model = os.environ.get("FAL_MODEL", "fal-ai/flux/schnell")
    out = _http_json(
        f"https://fal.run/{model}",
        {"prompt": prompt, "image_size": {"width": w, "height": h}},
        {"Authorization": f"Key {key}", "Content-Type": "application/json"},
    )
    imgs = out.get("images") or []
    if not imgs:
        raise RuntimeError("fal: no images")
    return _fetch_bytes(imgs[0]["url"])


def gen_openai(prompt, w, h):
    key = os.environ["OPENAI_API_KEY"]
    model = os.environ.get("OPENAI_IMAGE_MODEL", "gpt-image-1")
    size = "1024x1024"  # OpenAI fixed sizes; the terminal downsamples anyway
    out = _http_json(
        "https://api.openai.com/v1/images/generations",
        {"model": model, "prompt": prompt, "size": size, "n": 1},
        {"Authorization": f"Bearer {key}", "Content-Type": "application/json"},
    )
    d = out["data"][0]
    if d.get("b64_json"):
        return base64.b64decode(d["b64_json"])
    return _fetch_bytes(d["url"])


PROVIDERS = {"replicate": gen_replicate, "fal": gen_fal, "openai": gen_openai}


def generate(prompt, w, h):
    fn = PROVIDERS.get(PROVIDER)
    if fn is None:
        raise RuntimeError(f"unknown CLOUD_PROVIDER={PROVIDER}")
    return fn(prompt, w, h)


class H(BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get("Content-Length", 0))
        req = json.loads(self.rfile.read(n) or b"{}")
        prompt = req.get("prompt", "")
        w = int(req.get("width", 512) or 512)
        h = int(req.get("height", 512) or 512)
        try:
            img = generate(prompt, w, h)
        except Exception as e:
            self.send_response(502)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps({"error": str(e)[:300]}).encode())
            return
        self.send_response(200)
        self.send_header("Content-Type", "image/png")
        self.send_header("Content-Length", str(len(img)))
        self.end_headers()
        self.wfile.write(img)

    def log_message(self, *a):
        pass


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8099
    print(f"cloud image sidecar (provider={PROVIDER}) on 127.0.0.1:{port}/")
    HTTPServer(("127.0.0.1", port), H).serve_forever()
