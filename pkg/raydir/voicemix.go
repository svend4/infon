package raydir

import (
	"encoding/binary"
	"errors"
	"sync"
)

// voicemix.go mixes group voice. When several people talk at once their audio
// must be summed, not played back-to-back — writing each arriving packet straight
// to the speaker serialises the talkers and the lag grows without bound. Here each
// speaker's frame is tagged with its origin id and buffered separately (a short
// jitter buffer); a steady playback pull sums one frame from every active speaker
// into one output frame, with saturation, and prunes silent speakers. Mixing is
// per-frame at the listener, so no clock sync between machines is needed.

var errVoice = errors.New("raydir: malformed voice frame")

// EncodeVoice tags a PCM frame with its origin: [origin u32][nSamples u16][int16…].
func EncodeVoice(origin uint32, samples []int16) []byte {
	buf := make([]byte, 6+2*len(samples))
	binary.BigEndian.PutUint32(buf[0:4], origin)
	binary.BigEndian.PutUint16(buf[4:6], uint16(len(samples)))
	for i, s := range samples {
		binary.BigEndian.PutUint16(buf[6+2*i:], uint16(s))
	}
	return buf
}

// DecodeVoice parses a frame produced by EncodeVoice.
func DecodeVoice(data []byte) (origin uint32, samples []int16, err error) {
	if len(data) < 6 {
		return 0, nil, errVoice
	}
	origin = binary.BigEndian.Uint32(data[0:4])
	n := int(binary.BigEndian.Uint16(data[4:6]))
	if len(data) < 6+2*n {
		return 0, nil, errVoice
	}
	samples = make([]int16, n)
	for i := range samples {
		samples[i] = int16(binary.BigEndian.Uint16(data[6+2*i:]))
	}
	return origin, samples, nil
}

// VoiceMixer sums concurrent speakers into a single real-time output stream. It is
// safe for concurrent use (the receive goroutine adds; a playback goroutine mixes).
type VoiceMixer struct {
	mu     sync.Mutex
	frame  int                // samples per output frame
	maxBuf int                // cap per source (drop oldest past this, to bound lag)
	bufs   map[uint32][]int16 // per-source jitter buffer
}

// NewVoiceMixer returns a mixer producing `frame`-sample output frames, buffering
// up to ~1s per source (frame defaults to 320 = 20ms at 16 kHz).
func NewVoiceMixer(frame int) *VoiceMixer {
	if frame <= 0 {
		frame = 320
	}
	return &VoiceMixer{frame: frame, maxBuf: frame * 50, bufs: map[uint32][]int16{}}
}

// Add appends a speaker's samples to its jitter buffer, dropping the oldest if the
// buffer would exceed the cap (so a fast or stuck sender can't grow latency
// unboundedly).
func (m *VoiceMixer) Add(origin uint32, samples []int16) {
	if len(samples) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	b := append(m.bufs[origin], samples...)
	if len(b) > m.maxBuf {
		b = append([]int16(nil), b[len(b)-m.maxBuf:]...)
	}
	m.bufs[origin] = b
}

// Mix pulls up to one frame from every active speaker, sums them with saturation,
// and returns the mixed frame; it returns nil when no speaker has audio. Drained
// speakers are pruned.
func (m *VoiceMixer) Mix() []int16 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.bufs) == 0 {
		return nil
	}
	out := make([]int16, m.frame)
	any := false
	for id, b := range m.bufs {
		if len(b) == 0 {
			delete(m.bufs, id)
			continue
		}
		any = true
		n := m.frame
		if n > len(b) {
			n = len(b)
		}
		for i := 0; i < n; i++ {
			out[i] = clip16(int(out[i]) + int(b[i]))
		}
		if rest := b[n:]; len(rest) == 0 {
			delete(m.bufs, id)
		} else {
			m.bufs[id] = append([]int16(nil), rest...)
		}
	}
	if !any {
		return nil
	}
	return out
}

// Active reports how many speakers currently have buffered audio.
func (m *VoiceMixer) Active() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.bufs)
}

func clip16(v int) int16 {
	switch {
	case v > 32767:
		return 32767
	case v < -32768:
		return -32768
	default:
		return int16(v)
	}
}
