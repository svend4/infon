//go:build windows

package audio

import (
	"testing"
	"time"
)

func TestWASAPIRingBuffer(t *testing.T) {
	rb := newWASAPIRingBuffer(1000)

	if rb == nil {
		t.Fatal("newWASAPIRingBuffer returned nil")
	}

	if rb.size != 1000 {
		t.Errorf("Ring buffer size = %d, expected 1000", rb.size)
	}

	// Test write
	samples := []int16{100, 200, 300, 400, 500}
	written := rb.Write(samples)

	if written != len(samples) {
		t.Errorf("Written = %d, expected %d", written, len(samples))
	}

	if rb.Available() != len(samples) {
		t.Errorf("Available = %d, expected %d", rb.Available(), len(samples))
	}

	// Test read
	readBuf := make([]int16, 3)
	read := rb.Read(readBuf)

	if read != 3 {
		t.Errorf("Read = %d, expected 3", read)
	}

	if readBuf[0] != 100 || readBuf[1] != 200 || readBuf[2] != 300 {
		t.Error("Read data doesn't match written data")
	}

	if rb.Available() != 2 {
		t.Errorf("Available after read = %d, expected 2", rb.Available())
	}
}

func TestWASAPIRingBufferWrap(t *testing.T) {
	rb := newWASAPIRingBuffer(10)

	// Fill buffer almost completely
	samples1 := []int16{1, 2, 3, 4, 5, 6, 7, 8}
	rb.Write(samples1)

	// Read some
	readBuf := make([]int16, 5)
	rb.Read(readBuf)

	// Write more (should wrap around)
	samples2 := []int16{9, 10, 11, 12, 13}
	written := rb.Write(samples2)

	if written != 5 {
		t.Errorf("Written after wrap = %d, expected 5", written)
	}

	// Verify total available
	expected := 8 - 5 + 5 // 8 written - 5 read + 5 written
	if rb.Available() != expected {
		t.Errorf("Available = %d, expected %d", rb.Available(), expected)
	}
}

func TestWASAPIRingBufferFull(t *testing.T) {
	rb := newWASAPIRingBuffer(5)

	// Try to write more than capacity
	samples := []int16{1, 2, 3, 4, 5, 6, 7, 8}
	written := rb.Write(samples)

	if written != 5 {
		t.Errorf("Written to full buffer = %d, expected 5", written)
	}

	if rb.Available() != 5 {
		t.Error("Buffer should be full")
	}
}

func TestWASAPIRingBufferEmpty(t *testing.T) {
	rb := newWASAPIRingBuffer(100)

	// Try to read from empty buffer
	readBuf := make([]int16, 10)
	read := rb.Read(readBuf)

	if read != 0 {
		t.Errorf("Read from empty buffer = %d, expected 0", read)
	}
}

func TestWASAPIRingBufferConcurrent(t *testing.T) {
	rb := newWASAPIRingBuffer(1000)

	done := make(chan bool, 2)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			samples := []int16{int16(i), int16(i + 1), int16(i + 2)}
			rb.Write(samples)
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		totalRead := 0
		for totalRead < 300 { // 100 iterations * 3 samples
			readBuf := make([]int16, 10)
			read := rb.Read(readBuf)
			totalRead += read
			if read == 0 {
				time.Sleep(time.Microsecond)
			}
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	t.Log("Concurrent read/write test passed")
}

func TestInitCOM(t *testing.T) {
	err := initCOM()
	if err != nil {
		t.Fatalf("initCOM failed: %v", err)
	}

	// Should be idempotent
	err = initCOM()
	if err != nil {
		t.Errorf("Second initCOM failed: %v", err)
	}
}

func TestListCaptureDevices(t *testing.T) {
	devices, err := listCaptureDevicesImpl()
	if err != nil {
		t.Fatalf("listCaptureDevicesImpl failed: %v", err)
	}

	if len(devices) == 0 {
		t.Error("No capture devices found")
	}

	// Check default device
	if devices[0].Name != "Default Capture Device" {
		t.Errorf("First device name = %s, expected 'Default Capture Device'", devices[0].Name)
	}

	if !devices[0].IsDefault {
		t.Error("First device should be default")
	}

	if devices[0].Type != "capture" {
		t.Errorf("Device type = %s, expected 'capture'", devices[0].Type)
	}
}

func TestListPlaybackDevices(t *testing.T) {
	devices, err := listPlaybackDevicesImpl()
	if err != nil {
		t.Fatalf("listPlaybackDevicesImpl failed: %v", err)
	}

	if len(devices) == 0 {
		t.Error("No playback devices found")
	}

	if devices[0].Name != "Default Playback Device" {
		t.Errorf("First device name = %s, expected 'Default Playback Device'", devices[0].Name)
	}

	if !devices[0].IsDefault {
		t.Error("First device should be default")
	}

	if devices[0].Type != "playback" {
		t.Errorf("Device type = %s, expected 'playback'", devices[0].Type)
	}
}

func TestNewCaptureImpl(t *testing.T) {
	format := DefaultFormat()
	capture, err := newCaptureImpl(0, format)

	if err != nil {
		t.Fatalf("newCaptureImpl failed: %v", err)
	}

	if capture == nil {
		t.Fatal("newCaptureImpl returned nil")
	}

	wasapi, ok := capture.(*WASAPICapture)
	if !ok {
		t.Fatal("Capture is not *WASAPICapture")
	}

	if wasapi.format.SampleRate != format.SampleRate {
		t.Errorf("Sample rate = %d, expected %d", wasapi.format.SampleRate, format.SampleRate)
	}

	if wasapi.IsOpen() {
		t.Error("Capture should not be open initially")
	}
}

func TestNewPlaybackImpl(t *testing.T) {
	format := DefaultFormat()
	playback, err := newPlaybackImpl(0, format)

	if err != nil {
		t.Fatalf("newPlaybackImpl failed: %v", err)
	}

	if playback == nil {
		t.Fatal("newPlaybackImpl returned nil")
	}

	wasapi, ok := playback.(*WASAPIPlayback)
	if !ok {
		t.Fatal("Playback is not *WASAPIPlayback")
	}

	if wasapi.format.SampleRate != format.SampleRate {
		t.Errorf("Sample rate = %d, expected %d", wasapi.format.SampleRate, format.SampleRate)
	}

	if wasapi.IsOpen() {
		t.Error("Playback should not be open initially")
	}
}

func TestWASAPICaptureOpenClose(t *testing.T) {
	capture, err := newDefaultCaptureImpl()
	if err != nil {
		t.Fatalf("newDefaultCaptureImpl failed: %v", err)
	}

	// Test open
	err = capture.Open()
	if err != nil {
		t.Logf("Open failed (may be expected if no audio device): %v", err)
		t.Skip("Skipping test - no audio device available")
		return
	}

	if !capture.IsOpen() {
		t.Error("Capture should be open")
	}

	// Test read (should not error even if no data)
	buffer := make([]int16, 320)
	_, err = capture.Read(buffer)
	if err != nil {
		t.Errorf("Read failed: %v", err)
	}

	// Test close
	err = capture.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	if capture.IsOpen() {
		t.Error("Capture should be closed")
	}
}

func TestWASAPIPlaybackOpenClose(t *testing.T) {
	playback, err := newDefaultPlaybackImpl()
	if err != nil {
		t.Fatalf("newDefaultPlaybackImpl failed: %v", err)
	}

	// Test open
	err = playback.Open()
	if err != nil {
		t.Logf("Open failed (may be expected if no audio device): %v", err)
		t.Skip("Skipping test - no audio device available")
		return
	}

	if !playback.IsOpen() {
		t.Error("Playback should be open")
	}

	// Test write
	buffer := make([]int16, 320)
	for i := range buffer {
		buffer[i] = int16(i * 100)
	}

	_, err = playback.Write(buffer)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}

	// Test close
	err = playback.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	if playback.IsOpen() {
		t.Error("Playback should be closed")
	}
}

func TestWASAPICaptureReadBeforeOpen(t *testing.T) {
	capture, _ := newDefaultCaptureImpl()

	buffer := make([]int16, 320)
	_, err := capture.Read(buffer)

	if err == nil {
		t.Error("Read should fail when device not open")
	}

	if err != ErrDeviceNotOpen {
		t.Errorf("Error = %v, expected ErrDeviceNotOpen", err)
	}
}

func TestWASAPIPlaybackWriteBeforeOpen(t *testing.T) {
	playback, _ := newDefaultPlaybackImpl()

	buffer := make([]int16, 320)
	_, err := playback.Write(buffer)

	if err == nil {
		t.Error("Write should fail when device not open")
	}

	if err != ErrDeviceNotOpen {
		t.Errorf("Error = %v, expected ErrDeviceNotOpen", err)
	}
}

func TestWASAPIGetFormat(t *testing.T) {
	format := AudioFormat{
		SampleRate: 16000,
		Channels:   1,
		BitDepth:   16,
	}

	capture, _ := newCaptureImpl(0, format)

	gotFormat := capture.GetFormat()

	if gotFormat.SampleRate != format.SampleRate {
		t.Errorf("SampleRate = %d, expected %d", gotFormat.SampleRate, format.SampleRate)
	}

	if gotFormat.Channels != format.Channels {
		t.Errorf("Channels = %d, expected %d", gotFormat.Channels, format.Channels)
	}

	if gotFormat.BitDepth != format.BitDepth {
		t.Errorf("BitDepth = %d, expected %d", gotFormat.BitDepth, format.BitDepth)
	}
}

// Benchmarks

func BenchmarkWASAPIRingBufferWrite(b *testing.B) {
	rb := newWASAPIRingBuffer(10000)
	samples := make([]int16, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Write(samples)
	}
}

func BenchmarkWASAPIRingBufferRead(b *testing.B) {
	rb := newWASAPIRingBuffer(10000)

	// Fill buffer
	samples := make([]int16, 10000)
	rb.Write(samples)

	readBuf := make([]int16, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Read(readBuf)
	}
}
