package raydir

import "os"

// gallery.go summarises saved worlds (.rwld) and recordings (.rrec) so a collection
// of them can be browsed and shared — each is a tiny file of meaning (region specs
// or timestamped events), so a whole world or a whole walk travels as a few KB.

// WorldInfo summarises a saved world.
type WorldInfo struct {
	Path    string
	Regions int
	Places  []string // region names
	Bytes   int64
}

// SummarizeWorld reads a saved world file and reports its regions and place names.
func SummarizeWorld(path string) (WorldInfo, error) {
	regs, err := LoadWorldFile(path)
	if err != nil {
		return WorldInfo{}, err
	}
	info := WorldInfo{Path: path, Regions: len(regs)}
	if st, err := os.Stat(path); err == nil {
		info.Bytes = st.Size()
	}
	for _, r := range regs {
		info.Places = append(info.Places, regionName(r.Spec, r.Index))
	}
	return info, nil
}

// RecInfo summarises a session recording.
type RecInfo struct {
	Path       string
	Events     int
	DurationMs uint32
	Bytes      int64
}

// SummarizeRecording reads a recording file and reports its length.
func SummarizeRecording(path string) (RecInfo, error) {
	ev, err := LoadRecording(path)
	if err != nil {
		return RecInfo{}, err
	}
	info := RecInfo{Path: path, Events: len(ev)}
	if st, err := os.Stat(path); err == nil {
		info.Bytes = st.Size()
	}
	for _, e := range ev {
		if e.TMs > info.DurationMs {
			info.DurationMs = e.TMs
		}
	}
	return info, nil
}
