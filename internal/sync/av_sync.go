package sync

import (
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
	targetDelay   time.Duration // Target playout delay

	// Statistics
	audioFrames     uint64
	videoFrames     uint64
	droppedAudio    uint64
	droppedVideo    uint64
	syncCorrections uint64

	// Drift compensation
	driftEstimate time.Duration
	lastDriftCalc time.Time
}

// NewAVSynchronizer creates a new A/V synchronizer
func NewAVSynchronizer() *AVSynchronizer {
	return &AVSynchronizer{
		audioBuffer:   NewJitterBuffer(100, 50*time.Millisecond),  // 100 frames, 50ms target
		videoBuffer:   NewJitterBuffer(30, 100*time.Millisecond),  // 30 frames, 100ms target
		maxSyncOffset: 100 * time.Millisecond,
		targetDelay:   50 * time.Millisecond,
		lastSync:      time.Now(),
		lastDriftCalc: time.Now(),
	}
}

// AddAudioFrame adds an audio frame to the buffer
func (avs *AVSynchronizer) AddAudioFrame(data []byte, pts int64, seq uint32) error {
	avs.mu.Lock()
	defer avs.mu.Unlock()

	frame := &Frame{
		Type:      MediaTypeAudio,
		Data:      data,
		Timestamp: time.Now(),
		PTS:       pts,
		Sequence:  seq,
	}

	if err := avs.audioBuffer.Add(frame); err != nil {
		avs.droppedAudio++
		return fmt.Errorf("audio buffer full: %w", err)
	}

	avs.audioFrames++
	return nil
}

// AddVideoFrame adds a video frame to the buffer
func (avs *AVSynchronizer) AddVideoFrame(data []byte, pts int64, seq uint32) error {
	avs.mu.Lock()
	defer avs.mu.Unlock()

	frame := &Frame{
		Type:      MediaTypeVideo,
		Data:      data,
		Timestamp: time.Now(),
		PTS:       pts,
		Sequence:  seq,
	}

	if err := avs.videoBuffer.Add(frame); err != nil {
		avs.droppedVideo++
		return fmt.Errorf("video buffer full: %w", err)
	}

	avs.videoFrames++
	return nil
}

// GetNextAudioFrame gets the next audio frame if ready for playout
func (avs *AVSynchronizer) GetNextAudioFrame() (*Frame, error) {
	avs.mu.Lock()
	defer avs.mu.Unlock()

	frame, err := avs.audioBuffer.Get()
	if err != nil {
		return nil, err
	}

	// Check sync with video
	if avs.videoBuffer.Size() > 0 {
		videoFrame, _ := avs.videoBuffer.Peek()
		if videoFrame != nil {
			offset := time.Duration(frame.PTS-videoFrame.PTS) * time.Microsecond

			// If audio is too far ahead, wait
			if offset > avs.maxSyncOffset {
				avs.audioBuffer.Add(frame) // Put it back
				return nil, fmt.Errorf("audio ahead by %v, waiting", offset)
			}

			// If audio is too far behind, skip frames
			if offset < -avs.maxSyncOffset {
				avs.droppedAudio++
				avs.syncCorrections++
				return nil, fmt.Errorf("audio behind by %v, skipping", offset)
			}
		}
	}

	return frame, nil
}

// GetNextVideoFrame gets the next video frame if ready for playout
func (avs *AVSynchronizer) GetNextVideoFrame() (*Frame, error) {
	avs.mu.Lock()
	defer avs.mu.Unlock()

	frame, err := avs.videoBuffer.Get()
	if err != nil {
		return nil, err
	}

	// Check sync with audio
	if avs.audioBuffer.Size() > 0 {
		audioFrame, _ := avs.audioBuffer.Peek()
		if audioFrame != nil {
			offset := time.Duration(frame.PTS-audioFrame.PTS) * time.Microsecond

			// If video is too far ahead, wait
			if offset > avs.maxSyncOffset {
				avs.videoBuffer.Add(frame) // Put it back
				return nil, fmt.Errorf("video ahead by %v, waiting", offset)
			}

			// If video is too far behind, skip frames
			if offset < -avs.maxSyncOffset {
				avs.droppedVideo++
				avs.syncCorrections++
				return nil, fmt.Errorf("video behind by %v, skipping", offset)
			}
		}
	}

	return frame, nil
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
	avs.lastSync = time.Now()

	// Calculate drift
	if time.Since(avs.lastDriftCalc) > 10*time.Second {
		avs.driftEstimate = offset / time.Since(avs.lastDriftCalc)
		avs.lastDriftCalc = time.Now()
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
	DriftEstimate   time.Duration
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
	avs.lastSync = time.Now()
	avs.lastDriftCalc = time.Now()
}

// JitterBuffer is a buffer that handles jitter and reordering
type JitterBuffer struct {
	mu            sync.RWMutex
	frames        []*Frame
	maxSize       int
	targetDelay   time.Duration
	lastSequence  uint32
	sequenceValid bool
}

// NewJitterBuffer creates a new jitter buffer
func NewJitterBuffer(maxSize int, targetDelay time.Duration) *JitterBuffer {
	return &JitterBuffer{
		frames:      make([]*Frame, 0, maxSize),
		maxSize:     maxSize,
		targetDelay: targetDelay,
	}
}

// Add adds a frame to the buffer
func (jb *JitterBuffer) Add(frame *Frame) error {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	if len(jb.frames) >= jb.maxSize {
		return fmt.Errorf("buffer full")
	}

	// Insert frame in order by PTS
	insertIdx := len(jb.frames)
	for i, f := range jb.frames {
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

	if len(jb.frames) == 0 {
		return nil, fmt.Errorf("buffer empty")
	}

	// Check if oldest frame is ready (past target delay)
	oldest := jb.frames[0]
	if time.Since(oldest.Timestamp) < jb.targetDelay {
		return nil, fmt.Errorf("not ready yet")
	}

	// Remove and return
	frame := jb.frames[0]
	jb.frames = jb.frames[1:]

	return frame, nil
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
	jb.lastSequence = 0
	jb.sequenceValid = false
}
