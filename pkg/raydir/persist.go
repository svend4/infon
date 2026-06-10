package raydir

import (
	"encoding/binary"
	"errors"
	"os"
)

// persist.go saves and reloads an authored world. A world here is just its list of
// Regions — each a position plus the director's scene description (meaning, not
// pixels) — so a whole explored world is a tiny file: a host can stop and resume,
// a late joiner can be handed the world deterministically, and a world becomes
// something you can copy and share. The on-disk form is the same Region.Encode the
// network uses: a magic+count header then length-prefixed region blobs.

var (
	worldMagic   = [4]byte{'R', 'W', 'L', 'D'}
	errWorldFile = errors.New("raydir: malformed world file")
)

// EncodeWorld serialises regions to the world-file byte form.
func EncodeWorld(regions []Region) []byte {
	out := make([]byte, 0, 8+len(regions)*64)
	out = append(out, worldMagic[:]...)
	var u32 [4]byte
	binary.BigEndian.PutUint32(u32[:], uint32(len(regions)))
	out = append(out, u32[:]...)
	for _, r := range regions {
		blob := r.Encode()
		binary.BigEndian.PutUint32(u32[:], uint32(len(blob)))
		out = append(out, u32[:]...)
		out = append(out, blob...)
	}
	return out
}

// DecodeWorld parses bytes produced by EncodeWorld.
func DecodeWorld(data []byte) ([]Region, error) {
	if len(data) < 8 || [4]byte{data[0], data[1], data[2], data[3]} != worldMagic {
		return nil, errWorldFile
	}
	n := int(binary.BigEndian.Uint32(data[4:8]))
	off := 8
	out := make([]Region, 0, n)
	for i := 0; i < n; i++ {
		if off+4 > len(data) {
			return nil, errWorldFile
		}
		l := int(binary.BigEndian.Uint32(data[off : off+4]))
		off += 4
		if l < 0 || off+l > len(data) {
			return nil, errWorldFile
		}
		r, err := DecodeRegion(data[off : off+l])
		if err != nil {
			return nil, err
		}
		out = append(out, r)
		off += l
	}
	return out, nil
}

// SaveWorld writes regions to path atomically (temp file + rename), so a crash
// mid-write can't leave a half-written world.
func SaveWorld(path string, regions []Region) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, EncodeWorld(regions), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadWorldFile reads regions previously saved with SaveWorld.
func LoadWorldFile(path string) ([]Region, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return DecodeWorld(data)
}
