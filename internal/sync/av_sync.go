package sync

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// MediaType represents audio or video
type MediaType string

const (
	MediaTypeAudio MediaType = "audio"
	MediaTypeVideo MediaType = "video"
)

// clock is the time source for arrival timestamps and playout readiness; tests
// replace it to step time instead of sleeping.
var clock = time.Now

// Frame represents a media frame with timestamp
type Frame struct {
	Type      MediaType
	Data      []byte
	Timestamp time.Time
	PTS       int64 // Presentation timestamp in microseconds
	Sequence  uint32
}

// AVSynchronizer handles audio/video synchronization
type AVSynchronizer struct {
	mu sync.RWMutex

	// Jitter buffer
	audioBuffer *JitterBuffer
	videoBuffer *JitterBuffer

	// Clock synchronization
	clockOffset time.Duration // Offset between local and remote clock
	lastSync    time.Time

	// Sync settings
	maxSyncOffset time.Duration // Maximum allowed A/V offset
	maxWaitOffset time.Duration // Larger offsets are treated as a bogus PTS
	targetDelay   time.Duration // Target playout delay

	// Statistics
	audioFrames     uint64
	videoFrames     uint64
	droppedAudio    uint64
	droppedVideo    uint64
	syncCorrections uint64

	// Drift compensation
	driftEstimate  time.Duration // Clock offset change per second
	lastDriftCalc  time.Time
	driftRefOffset time.Duration // Clock offset at lastDriftCalc
	driftRefValid  bool
}

// NewAVSynchronizer creates a new A/V synchronizer
func NewAVSynchronizer() *AVSynchronizer {
	return &AVSynchronizer{
		audioBuffer:   NewJitterBuffer(100, 50*time.Millisecond),  // 100 frames, 50ms target
		videoBuffer:   NewJitterBuffer(30, 100*time.Millisecond),  // 30 frames, 100ms target
		maxSyncOffset: 100 * time.Millisecond,
		maxWaitOffset: 5 * time.Second,
		targetDelay:   50 * time.Millisecond,
		lastSync:      clock(),
		lastDriftCalc: clock(),
	}
}

// AddAudioFrame adds an audio frame to the buffer. data is copied, so the
// caller may reuse its buffer.
func (avs *AVSynchronizer) AddAudioFrame(data []byte, pts int64, seq uint32) error {
	avs.mu.Lock()
	defer avs.mu.Unlock()

	frame := &Frame{
		Type:      MediaTypeAudio,
		Data:      append([]byte(nil), data...),
		Timestamp: clock(),
		PTS:       pts,
		Sequence:  seq,
	}

	if err := avs.audioBuffer.Add(frame); err != nil {
		avs.droppedAudio++
		return fmt.Errorf("audio frame dropped: %w", err)
	}

	avs.audioFrames++
	return nil
}

// AddVideoFrame adds a video frame to the buffer. data is copied, so the
// caller may reuse its buffer.
func (avs *AVSynchronizer) AddVideoFrame(data []byte, pts int64, seq uint32) error {
	avs.mu.Lock()
	defer avs.mu.Unlock()

	frame := &Frame{
		Type:      MediaTypeVideo,
		Data:      append([]byte(nil), data...),
		Timestamp: clock(),
		PTS:       pts,
		Sequence:  seq,
	}

	if err := avs.videoBuffer.Add(frame); err != nil {
		avs.droppedVideo++
		return fmt.Errorf("video frame dropped: %w", err)
	}

	avs.videoFrames++
	return nil
}

// syncHead returns the head of the other stream's buffer to sync against, or
// nil if there is none. A head frame that has been ready for playout well
// past the point a running consumer would have taken it means that stream is
// not being played, so it must not hold back this one forever.
func (avs *AVSynchronizer) syncHead(other *JitterBuffer) *Frame {
	f, _ := other.Peek()
	if f == nil || clock().Sub(f.Timestamp) > other.targetDelay+2*avs.maxSyncOffset {
		return nil
	}
	return f
}

// GetNextAudioFrame gets the next audio frame if ready for playout
func (avs *AVSynchronizer) GetNextAudioFrame() (*Frame, error) {
	avs.mu.Lock()
	defer avs.mu.Unlock()

	frame, err := avs.audioBuffer.ready()
	if err != nil {
		return nil, err
	}

	// Check sync with video
	if videoFrame := avs.syncHead(avs.videoBuffer); videoFrame != nil {
		offset := time.Duration(frame.PTS-videoFrame.PTS) * time.Microsecond

		// If audio is implausibly far ahead, discard it: leaving it at the
		// head would make every video frame look late.
		if offset > avs.maxWaitOffset {
			avs.audioBuffer.discard()
			avs.droppedAudio++
			avs.syncCorrections++
			return nil, fmt.Errorf("audio ahead by %v, discarding", offset)
		}

		// If audio is too far ahead, leave it at the head and wait
		if offset > avs.maxSyncOffset {
			return nil, fmt.Errorf("audio ahead by %v, waiting", offset)
		}

		// If audio is too far behind, skip frames
		if offset < -avs.maxSyncOffset {
			_, _ = avs.audioBuffer.Get()
			avs.droppedAudio++
			avs.syncCorrections++
			return nil, fmt.Errorf("audio behind by %v, skipping", offset)
		}
	}

	return avs.audioBuffer.Get()
}

// GetNextVideoFrame gets the next video frame if ready for playout
func (avs *AVSynchronizer) GetNextVideoFrame() (*Frame, error) {
	avs.mu.Lock()
	defer avs.mu.Unlock()

	frame, err := avs.videoBuffer.ready()
	if err != nil {
		return nil, err
	}

	// Check sync with audio
	if audioFrame := avs.syncHead(avs.audioBuffer); audioFrame != nil {
		offset := time.Duration(frame.PTS-audioFrame.PTS) * time.Microsecond

		// If video is implausibly far ahead, discard it: leaving it at the
		// head would make every audio frame look late.
		if offset > avs.maxWaitOffset {
			avs.videoBuffer.discard()
			avs.droppedVideo++
			avs.syncCorrections++
			return nil, fmt.Errorf("video ahead by %v, discarding", offset)
		}

		// If video is too far ahead, leave it at the head and wait
		if offset > avs.maxSyncOffset {
			return nil, fmt.Errorf("video ahead by %v, waiting", offset)
		}

		// If video is too far behind, skip frames
		if offset < -avs.maxSyncOffset {
			_, _ = avs.videoBuffer.Get()
			avs.droppedVideo++
			avs.syncCorrections++
			return nil, fmt.Errorf("video behind by %v, skipping", offset)
		}
	}

	return avs.videoBuffer.Get()
}

// GetSyncOffset returns the current A/V sync offset
func (avs *AVSynchronizer) GetSyncOffset() time.Duration {
	avs.mu.RLock()
	defer avs.mu.RUnlock()

	audioFrame, _ := avs.audioBuffer.Peek()
	videoFrame, _ := avs.videoBuffer.Peek()

	if audioFrame == nil || videoFrame == nil {
		return 0
	}

	return time.Duration(audioFrame.PTS-videoFrame.PTS) * time.Microsecond
}

// UpdateClockOffset updates the clock offset based on RTCP sender reports
func (avs *AVSynchronizer) UpdateClockOffset(offset time.Duration) {
	avs.mu.Lock()
	defer avs.mu.Unlock()

	avs.clockOffset = offset
	avs.lastSync = clock()

	// Calculate drift: how much the offset changed per second since the
	// last calculation.
	if !avs.driftRefValid {
		avs.driftRefOffset = offset
		avs.driftRefValid = true
		avs.lastDriftCalc = clock()
	} else if elapsed := clock().Sub(avs.lastDriftCalc); elapsed > 10*time.Second {
		avs.driftEstimate = time.Duration(float64(offset-avs.driftRefOffset) / elapsed.Seconds())
		avs.driftRefOffset = offset
		avs.lastDriftCalc = clock()
	}
}

// GetStatistics returns synchronization statistics
func (avs *AVSynchronizer) GetStatistics() Statistics {
	avs.mu.RLock()
	defer avs.mu.RUnlock()

	return Statistics{
		AudioFrames:     avs.audioFrames,
		VideoFrames:     avs.videoFrames,
		DroppedAudio:    avs.droppedAudio,
		DroppedVideo:    avs.droppedVideo,
		SyncCorrections: avs.syncCorrections,
		AudioBufferSize: avs.audioBuffer.Size(),
		VideoBufferSize: avs.videoBuffer.Size(),
		ClockOffset:     avs.clockOffset,
		DriftEstimate:   avs.driftEstimate,
		SyncOffset:      avs.GetSyncOffsetUnsafe(),
	}
}

// GetSyncOffsetUnsafe returns sync offset without locking (for use when already locked)
func (avs *AVSynchronizer) GetSyncOffsetUnsafe() time.Duration {
	audioFrame, _ := avs.audioBuffer.Peek()
	videoFrame, _ := avs.videoBuffer.Peek()

	if audioFrame == nil || videoFrame == nil {
		return 0
	}

	return time.Duration(audioFrame.PTS-videoFrame.PTS) * time.Microsecond
}

// Statistics represents synchronization statistics
type Statistics struct {
	AudioFrames     uint64
	VideoFrames     uint64
	DroppedAudio    uint64
	DroppedVideo    uint64
	SyncCorrections uint64
	AudioBufferSize int
	VideoBufferSize int
	ClockOffset     time.Duration
	DriftEstimate   time.Duration // ClockOffset change per second
	SyncOffset      time.Duration
}

// Reset resets all statistics and buffers
func (avs *AVSynchronizer) Reset() {
	avs.mu.Lock()
	defer avs.mu.Unlock()

	avs.audioBuffer.Clear()
	avs.videoBuffer.Clear()
	avs.audioFrames = 0
	avs.videoFrames = 0
	avs.droppedAudio = 0
	avs.droppedVideo = 0
	avs.syncCorrections = 0
	avs.clockOffset = 0
	avs.driftEstimate = 0
	avs.driftRefValid = false
	avs.lastSync = clock()
	avs.lastDriftCalc = clock()
}

// JitterBuffer is a buffer that handles jitter and reordering
type JitterBuffer struct {
	mu           sync.RWMutex
	frames       []*Frame
	maxSize      int
	targetDelay  time.Duration
	lastPTS      int64  // PTS of the last frame returned by Get
	lastSeq      uint32 // and its Sequence
	lastPTSValid bool
	lateRun      int // Frames rejected as late in a row
}

// Errors returned by JitterBuffer.Add for frames that must not be played.
var (
	ErrLateFrame      = errors.New("late frame")
	ErrDuplicateFrame = errors.New("duplicate frame")
)

// restartRun is how many frames in a row must land behind the playout point
// before the buffer takes it that the sender's clock restarted and resyncs. A
// single stray frame (a late retransmission) is only dropped.
const restartRun = 8

// NewJitterBuffer creates a new jitter buffer
func NewJitterBuffer(maxSize int, targetDelay time.Duration) *JitterBuffer {
	return &JitterBuffer{
		frames:      make([]*Frame, 0, maxSize),
		maxSize:     maxSize,
		targetDelay: targetDelay,
	}
}

// Add adds a frame to the buffer. It returns ErrLateFrame for a frame at or
// behind the last one played and ErrDuplicateFrame for one already buffered.
func (jb *JitterBuffer) Add(frame *Frame) error {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	if len(jb.frames) >= jb.maxSize {
		return fmt.Errorf("buffer full")
	}

	// A frame at or behind one already played arrived too late: playing it now
	// would be out of order and hold back the frames that are ready. Two cases
	// look the same by PTS but are not late:
	//   - the sender numbers its frames and this one comes after the last one
	//     played: that frame carried a PTS that jumped ahead, and the stream must
	//     not be dropped until it catches up;
	//   - restartRun frames in a row land behind: the sender's clock restarted,
	//     and the frames still buffered from before the restart are stale too.
	if jb.lastPTSValid && frame.PTS <= jb.lastPTS {
		switch {
		case int32(frame.Sequence-jb.lastSeq) > 0: // newer, wrap-aware
		case jb.lateRun+1 >= restartRun:
			jb.frames = jb.frames[:0]
		default:
			jb.lateRun++
			return ErrLateFrame
		}
		jb.lastPTSValid = false
	}
	jb.lateRun = 0

	// Insert frame in order by PTS
	insertIdx := len(jb.frames)
	for i, f := range jb.frames {
		if frame.PTS == f.PTS {
			return ErrDuplicateFrame
		}
		if frame.PTS < f.PTS {
			insertIdx = i
			break
		}
	}

	// Insert at position
	jb.frames = append(jb.frames, nil)
	copy(jb.frames[insertIdx+1:], jb.frames[insertIdx:])
	jb.frames[insertIdx] = frame

	return nil
}

// Get gets the next frame if ready for playout
func (jb *JitterBuffer) Get() (*Frame, error) {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	frame, err := jb.readyLocked()
	if err != nil {
		return nil, err
	}

	// Remove and return
	jb.frames = jb.frames[1:]
	jb.lastPTS = frame.PTS
	jb.lastSeq = frame.Sequence
	jb.lastPTSValid = true

	return frame, nil
}

// ready returns the next frame if it is ready for playout, without removing it.
func (jb *JitterBuffer) ready() (*Frame, error) {
	jb.mu.RLock()
	defer jb.mu.RUnlock()

	return jb.readyLocked()
}

func (jb *JitterBuffer) readyLocked() (*Frame, error) {
	if len(jb.frames) == 0 {
		return nil, fmt.Errorf("buffer empty")
	}

	// Check if oldest frame is ready (past target delay)
	oldest := jb.frames[0]
	if clock().Sub(oldest.Timestamp) < jb.targetDelay {
		return nil, fmt.Errorf("not ready yet")
	}

	return oldest, nil
}

// discard removes the next frame without treating it as played, so a bogus
// PTS does not become the point later frames are judged late against.
func (jb *JitterBuffer) discard() {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	if len(jb.frames) > 0 {
		jb.frames = jb.frames[1:]
	}
}

// Peek looks at the next frame without removing it
func (jb *JitterBuffer) Peek() (*Frame, error) {
	jb.mu.RLock()
	defer jb.mu.RUnlock()

	if len(jb.frames) == 0 {
		return nil, fmt.Errorf("buffer empty")
	}

	return jb.frames[0], nil
}

// Size returns the number of frames in the buffer
func (jb *JitterBuffer) Size() int {
	jb.mu.RLock()
	defer jb.mu.RUnlock()

	return len(jb.frames)
}

// Clear clears the buffer
func (jb *JitterBuffer) Clear() {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	jb.frames = make([]*Frame, 0, jb.maxSize)
	jb.lastPTS = 0
	jb.lastSeq = 0
	jb.lastPTSValid = false
	jb.lateRun = 0
}
