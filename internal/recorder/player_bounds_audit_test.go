package recorder

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// auditHeader builds a 34-byte recording header with the given entry counts.
func auditHeader(frameCount, audioChunks uint32) []byte {
	h := make([]byte, 34)
	binary.BigEndian.PutUint32(h[0:4], MagicHeader)
	binary.BigEndian.PutUint16(h[4:6], RecordingVersion)
	binary.BigEndian.PutUint32(h[14:18], 1000)
	binary.BigEndian.PutUint16(h[18:20], 40)
	binary.BigEndian.PutUint16(h[20:22], 12)
	binary.BigEndian.PutUint16(h[22:24], 8000)
	binary.BigEndian.PutUint32(h[26:30], frameCount)
	binary.BigEndian.PutUint32(h[30:34], audioChunks)
	return h
}

// auditEntry builds a 12-byte frame or audio entry header.
func auditEntry(timestamp uint64, n uint32) []byte {
	e := make([]byte, 12)
	binary.BigEndian.PutUint64(e[0:8], timestamp)
	binary.BigEndian.PutUint32(e[8:12], n)
	return e
}

func writeAuditFile(t *testing.T, parts ...[]byte) string {
	t.Helper()
	var data []byte
	for _, p := range parts {
		data = append(data, p...)
	}
	path := filepath.Join(t.TempDir(), "forged.tvcp")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// loadMeasured loads path and reports the error and the bytes allocated.
func loadMeasured(t *testing.T, path string) (err error, allocated uint64) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Load panicked on malformed recording: %v", r)
		}
	}()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	err = NewPlayer().Load(path)
	runtime.ReadMemStats(&after)
	return err, after.TotalAlloc - before.TotalAlloc
}

const auditAllocLimit = 4 << 20 // a forged tiny file must not cost more than this

func TestReadRecordingForgedFrameCountAudit(t *testing.T) {
	// 34-byte file whose header claims 16M frames.
	path := writeAuditFile(t, auditHeader(1<<24, 0))
	err, alloc := loadMeasured(t, path)
	if err == nil {
		t.Fatal("Load accepted a header claiming more frames than the file holds")
	}
	if alloc > auditAllocLimit {
		t.Errorf("Load allocated %d bytes for a %d-byte file", alloc, 34)
	}
}

func TestReadRecordingForgedAudioChunksAudit(t *testing.T) {
	path := writeAuditFile(t, auditHeader(0, 1<<24))
	err, alloc := loadMeasured(t, path)
	if err == nil {
		t.Fatal("Load accepted a header claiming more audio chunks than the file holds")
	}
	if alloc > auditAllocLimit {
		t.Errorf("Load allocated %d bytes for a %d-byte file", alloc, 34)
	}
}

func TestReadRecordingForgedFrameLengthAudit(t *testing.T) {
	// One frame entry claiming 256 MiB of data, followed by nothing.
	path := writeAuditFile(t, auditHeader(1, 0), auditEntry(0, 1<<28))
	err, alloc := loadMeasured(t, path)
	if err == nil {
		t.Fatal("Load accepted a frame longer than the file")
	}
	if alloc > auditAllocLimit {
		t.Errorf("Load allocated %d bytes for a %d-byte file", alloc, 34+12)
	}
}

func TestReadRecordingForgedSampleCountAudit(t *testing.T) {
	// One audio entry claiming 2^24 samples, followed by two bytes.
	path := writeAuditFile(t, auditHeader(0, 1), auditEntry(0, 1<<24), []byte{0, 1})
	err, alloc := loadMeasured(t, path)
	if err == nil {
		t.Fatal("Load accepted an audio chunk longer than the file")
	}
	if alloc > auditAllocLimit {
		t.Errorf("Load allocated %d bytes for a %d-byte file", alloc, 34+12+2)
	}
}

func TestReadRecordingSampleCountOverflowAudit(t *testing.T) {
	// sampleCount*2 overflows uint32 to 2, so only two bytes are read but
	// 2^31+1 samples are decoded from them.
	path := writeAuditFile(t, auditHeader(0, 1), auditEntry(0, 0x80000001), []byte{0, 1})
	err, _ := loadMeasured(t, path)
	if err == nil {
		t.Fatal("Load accepted an audio chunk whose byte length overflows uint32")
	}
}
