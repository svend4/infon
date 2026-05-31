package video

// EncodedSize returns the serialized byte size of an encoded frame. It lets a
// sender measure and compare I-frame vs P-frame cost without actually
// transmitting — useful for adaptive decisions and for proving the P-frame
// bandwidth win on slowly-changing (e.g. AI-generated) video.
func EncodedSize(encoded *EncodedFrame) int {
	if encoded == nil {
		return 0
	}
	return len(SerializeEncodedFrame(encoded))
}

// StreamStats accumulates I/P frame counts and byte totals across a stream, so a
// caller can report the realized compression ratio of the P-frame path.
type StreamStats struct {
	IFrames int
	PFrames int
	IBytes  int
	PBytes  int
}

// Observe records one encoded frame.
func (s *StreamStats) Observe(encoded *EncodedFrame) {
	n := EncodedSize(encoded)
	if encoded.Type == FrameTypeI {
		s.IFrames++
		s.IBytes += n
	} else {
		s.PFrames++
		s.PBytes += n
	}
}

// TotalBytes is the total serialized bytes observed.
func (s *StreamStats) TotalBytes() int { return s.IBytes + s.PBytes }

// CompressionRatio returns (bytes if every frame were an I-frame) / (actual
// bytes). A ratio > 1 means P-frames saved bandwidth. avgIFrameBytes is used as
// the hypothetical per-frame I-frame cost; pass the mean observed I-frame size.
func (s *StreamStats) CompressionRatio(avgIFrameBytes int) float64 {
	total := s.IFrames + s.PFrames
	if total == 0 || s.TotalBytes() == 0 {
		return 1
	}
	allIBytes := avgIFrameBytes * total
	return float64(allIBytes) / float64(s.TotalBytes())
}

// AverageIFrameBytes returns the mean serialized size of observed I-frames.
func (s *StreamStats) AverageIFrameBytes() int {
	if s.IFrames == 0 {
		return 0
	}
	return s.IBytes / s.IFrames
}
