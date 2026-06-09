package raydir

import (
	"image"
	_ "image/jpeg" // register JPEG decoder for LoadTextureDir
	_ "image/png"  // register PNG decoder for LoadTextureDir
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/svend4/infon/pkg/raytrace"
)

// texlib.go is the named-texture library the director draws from. A scene object
// can name a texture (ObjSpec.Tex); the bridge looks it up here and sets it as the
// surface's Material.Tex, so the renderer samples it by UV/position. Built-ins are
// procedural (no assets, reconstructed from a name — meaning, not pixels);
// LoadTextureDir adds image files from disk, so a deployment can ship real image
// textures and still send only a name over the wire.

var (
	texMu  sync.RWMutex
	texReg = map[string]raytrace.Texture{}
)

// RegisterTexture adds (or replaces) a named texture.
func RegisterTexture(name string, t raytrace.Texture) {
	if t == nil {
		return
	}
	texMu.Lock()
	texReg[strings.ToLower(name)] = t
	texMu.Unlock()
}

// TextureFor returns the named texture, or nil if unknown.
func TextureFor(name string) raytrace.Texture {
	texMu.RLock()
	defer texMu.RUnlock()
	return texReg[strings.ToLower(name)]
}

var bumpReg = map[string]raytrace.Texture{}

// RegisterBump adds (or replaces) a named bump/normal map (assigned to
// Material.Bump). Like textures, bumps are procedural and reconstructed from a
// name — surface detail for nothing on the wire.
func RegisterBump(name string, t raytrace.Texture) {
	if t == nil {
		return
	}
	texMu.Lock()
	bumpReg[strings.ToLower(name)] = t
	texMu.Unlock()
}

// BumpFor returns the named bump map, or nil if unknown.
func BumpFor(name string) raytrace.Texture {
	texMu.RLock()
	defer texMu.RUnlock()
	return bumpReg[strings.ToLower(name)]
}

// LoadTextureDir registers every .png/.jpg/.jpeg in dir under its base name as a
// UV-sampled (wrapping) image texture, so custom textures need no code change.
func LoadTextureDir(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if e.IsDir() || (ext != ".png" && ext != ".jpg" && ext != ".jpeg") {
			continue
		}
		f, err := os.Open(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		img, _, err := image.Decode(f)
		_ = f.Close()
		if err != nil {
			continue
		}
		RegisterTexture(strings.TrimSuffix(strings.ToLower(e.Name()), ext), raytrace.ImageTex{Img: img, Wrap: true})
		n++
	}
	return n, nil
}

func init() {
	RegisterTexture("checker", raytrace.CheckerTex{
		A: raytrace.Vec3{X: 0.9, Y: 0.9, Z: 0.92}, B: raytrace.Vec3{X: 0.12, Y: 0.12, Z: 0.16}, Scale: 0.5,
	})
	RegisterTexture("marble", raytrace.MarbleTex{
		A: raytrace.Vec3{X: 0.86, Y: 0.86, Z: 0.88}, B: raytrace.Vec3{X: 0.24, Y: 0.22, Z: 0.2}, Scale: 1.2, Turb: 1.5,
	})
	RegisterTexture("wood", raytrace.MarbleTex{
		A: raytrace.Vec3{X: 0.45, Y: 0.28, Z: 0.13}, B: raytrace.Vec3{X: 0.72, Y: 0.52, Z: 0.3}, Scale: 0.4, Turb: 3,
	})
	RegisterTexture("stone", raytrace.NoiseTex{
		A: raytrace.Vec3{X: 0.33, Y: 0.32, Z: 0.31}, B: raytrace.Vec3{X: 0.62, Y: 0.6, Z: 0.57}, Scale: 0.6, Octaves: 5,
	})
	RegisterTexture("clouds", raytrace.NoiseTex{
		A: raytrace.Vec3{X: 0.45, Y: 0.55, Z: 0.8}, B: raytrace.Vec3{X: 1, Y: 1, Z: 1}, Scale: 1.6, Octaves: 6,
	})
	// procedural bump/normal maps (Material.Bump)
	RegisterBump("ripple", raytrace.WaveNormalTex{Freq: 6, Amp: 0.25})
	RegisterBump("waves", raytrace.WaveNormalTex{Freq: 4, Amp: 0.3})
	RegisterBump("bumps", raytrace.WaveNormalTex{Freq: 14, Amp: 0.18})
}
