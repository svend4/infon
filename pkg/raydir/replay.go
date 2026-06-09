package raydir

import (
	"encoding/binary"
	"errors"
	"os"
	"sort"
	"sync"
)

// replay.go records a shared-world session — the regions that appear, the walkers'
// poses, the chat and the time of day, each stamped with when it happened — and
// plays it back. A recording is just those timestamped events (meaning, not
// pixels), so a whole walk is a small file you can re-watch or share. The Player
// rebuilds the world and everyone's avatars at any moment by applying every event
// up to that time.

// Recording event types.
const (
	RecRegion uint8 = 1 // a Region (authored or placed)
	RecPoses  uint8 = 2 // a PoseSet snapshot
	RecChat   uint8 = 3 // a batch of ChatMsg
	RecEnv    uint8 = 4 // a time-of-day update
)

// RecEvent is one timestamped event (TMs = milliseconds since the session start).
type RecEvent struct {
	TMs  uint32
	Type uint8
	Data []byte
}

var recMagic = [4]byte{'R', 'R', 'E', 'C'}
var errRecording = errors.New("raydir: malformed recording")

// Recorder collects timestamped session events; it is safe for concurrent use.
type Recorder struct {
	mu sync.Mutex
	ev []RecEvent
}

// NewRecorder returns an empty recorder.
func NewRecorder() *Recorder { return &Recorder{} }

// Add appends a raw event at time tMs.
func (r *Recorder) Add(tMs uint32, typ uint8, data []byte) {
	r.mu.Lock()
	r.ev = append(r.ev, RecEvent{TMs: tMs, Type: typ, Data: append([]byte(nil), data...)})
	r.mu.Unlock()
}

// Region/Poses/Chat/Env record the corresponding event, reusing the wire encoders.
func (r *Recorder) Region(tMs uint32, reg Region)   { r.Add(tMs, RecRegion, reg.Encode()) }
func (r *Recorder) Poses(tMs uint32, ps PoseSet)    { r.Add(tMs, RecPoses, ps.Encode()) }
func (r *Recorder) Chat(tMs uint32, msgs []ChatMsg) { r.Add(tMs, RecChat, EncodeChatMsgs(msgs)) }
func (r *Recorder) Env(tMs uint32, t float64)       { r.Add(tMs, RecEnv, EncodeEnv(t)) }

// Events returns a time-sorted copy of the recorded events.
func (r *Recorder) Events() []RecEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := append([]RecEvent(nil), r.ev...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].TMs < out[j].TMs })
	return out
}

// Save writes the recording to a file (atomically).
func (r *Recorder) Save(path string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, EncodeRecording(r.Events()), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// EncodeRecording serialises events: magic + count + per-event
// [tMs u32][type u8][len u32][data].
func EncodeRecording(ev []RecEvent) []byte {
	out := append([]byte(nil), recMagic[:]...)
	var u [4]byte
	binary.BigEndian.PutUint32(u[:], uint32(len(ev)))
	out = append(out, u[:]...)
	for _, e := range ev {
		binary.BigEndian.PutUint32(u[:], e.TMs)
		out = append(out, u[:]...)
		out = append(out, e.Type)
		binary.BigEndian.PutUint32(u[:], uint32(len(e.Data)))
		out = append(out, u[:]...)
		out = append(out, e.Data...)
	}
	return out
}

// DecodeRecording parses bytes produced by EncodeRecording.
func DecodeRecording(data []byte) ([]RecEvent, error) {
	if len(data) < 8 || [4]byte{data[0], data[1], data[2], data[3]} != recMagic {
		return nil, errRecording
	}
	n := int(binary.BigEndian.Uint32(data[4:8]))
	off := 8
	out := make([]RecEvent, 0, n)
	for i := 0; i < n; i++ {
		if off+9 > len(data) {
			return nil, errRecording
		}
		e := RecEvent{TMs: binary.BigEndian.Uint32(data[off : off+4]), Type: data[off+4]}
		l := int(binary.BigEndian.Uint32(data[off+5 : off+9]))
		off += 9
		if l < 0 || off+l > len(data) {
			return nil, errRecording
		}
		e.Data = append([]byte(nil), data[off:off+l]...)
		off += l
		out = append(out, e)
	}
	return out, nil
}

// LoadRecording reads a recording from a file.
func LoadRecording(path string) ([]RecEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return DecodeRecording(data)
}

// Player replays a recording: Advance applies every event up to a time, rebuilding
// the world, the current poses, the chat log and the time of day.
type Player struct {
	ev    []RecEvent
	i     int
	world *World
	poses PoseSet
	chat  *ChatLog
}

// NewPlayer prepares to replay the (time-sorted) events.
func NewPlayer(ev []RecEvent) *Player {
	out := append([]RecEvent(nil), ev...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].TMs < out[j].TMs })
	return &Player{ev: out, world: NewWorld(), poses: PoseSet{}, chat: NewChatLog(8)}
}

// Advance applies all not-yet-applied events with TMs <= tMs.
func (p *Player) Advance(tMs uint32) {
	for p.i < len(p.ev) && p.ev[p.i].TMs <= tMs {
		e := p.ev[p.i]
		p.i++
		switch e.Type {
		case RecRegion:
			if reg, err := DecodeRegion(e.Data); err == nil {
				p.world.AddRegion(reg)
			}
		case RecPoses:
			if ps, err := DecodePoseSet(e.Data); err == nil {
				p.poses = ps
			}
		case RecChat:
			if msgs, err := DecodeChatMsgs(e.Data); err == nil {
				for _, m := range msgs {
					p.chat.Add(m.Sender, m.Text)
				}
			}
		case RecEnv:
			if t, err := DecodeEnv(e.Data); err == nil {
				p.world.SetTime(t)
			}
		}
	}
}

// World, Poses and Chat expose the current replay state.
func (p *Player) World() *World  { return p.world }
func (p *Player) Poses() PoseSet { return p.poses }
func (p *Player) Chat() *ChatLog { return p.chat }

// Duration is the timestamp of the last event (ms), i.e. the recording length.
func (p *Player) Duration() uint32 {
	if len(p.ev) == 0 {
		return 0
	}
	return p.ev[len(p.ev)-1].TMs
}
