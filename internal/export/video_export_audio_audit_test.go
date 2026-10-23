package export

import (
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/svend4/infon/internal/recorder"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

func auditRecording(frames int, audioSamples int) *recorder.Recording {
	rec := &recorder.Recording{
		Metadata: recorder.RecordingMetadata{
			Duration:    1000,
			FrameWidth:  4,
			FrameHeight: 4,
			AudioRate:   8000,
			FrameCount:  uint32(frames),
		},
	}
	for i := 0; i < frames; i++ {
		f := terminal.NewFrame(4, 4)
		f.SetBlock(i%4, i%4, '#', color.RGB{R: 255, G: 255, B: 255}, color.RGB{R: uint8(i * 10)})
		rec.Frames = append(rec.Frames, recorder.RecordedFrame{Timestamp: uint64(i * 66), Frame: f})
	}
	if audioSamples > 0 {
		samples := make([]int16, audioSamples)
		for i := range samples {
			samples[i] = int16((i % 64) * 256)
		}
		rec.Audio = []recorder.RecordedAudio{{Timestamp: 0, Samples: samples}}
		rec.Metadata.AudioChunks = 1
	}
	return rec
}

// probeStreams returns the number of decoded video frames and whether the
// file has an audio stream.
func probeStreams(t *testing.T, path string) (videoFrames int, hasAudio bool) {
	t.Helper()
	out, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-count_frames",
		"-show_entries", "stream=nb_read_frames", "-of", "default=nokey=1:noprint_wrappers=1", path).Output()
	if err != nil {
		t.Fatalf("ffprobe video: %v", err)
	}
	if s := strings.TrimSpace(string(out)); s != "" && s != "N/A" {
		videoFrames, _ = strconv.Atoi(s)
	}
	out, err = exec.Command("ffprobe", "-v", "error", "-select_streams", "a",
		"-show_entries", "stream=index", "-of", "csv=p=0", path).Output()
	if err != nil {
		t.Fatalf("ffprobe audio: %v", err)
	}
	return videoFrames, strings.TrimSpace(string(out)) != ""
}

func requireFFmpeg(t *testing.T) {
	t.Helper()
	for _, tool := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not installed", tool)
		}
	}
}

// TestExportWithoutAudioAudit: a recording with no audio must still produce
// every video frame.
func TestExportWithoutAudioAudit(t *testing.T) {
	requireFFmpeg(t)
	const frames = 15
	out := filepath.Join(t.TempDir(), "silent.mp4")
	ve := NewVideoExporter(auditRecording(frames, 0))
	if err := ve.Export(ExportOptions{Format: FormatMP4, OutputPath: out, Preset: "ultrafast"}); err != nil {
		t.Fatalf("Export: %v", err)
	}
	n, hasAudio := probeStreams(t, out)
	if n != frames {
		t.Errorf("exported video has %d frames, want %d", n, frames)
	}
	if hasAudio {
		t.Errorf("exported video has an audio stream for a recording without audio")
	}
}

func auditHasArg(args []string, arg string) bool {
	for _, a := range args {
		if a == arg {
			return true
		}
	}
	return false
}

// TestFFmpegArgsAudioAudit pins the argument construction without needing
// FFmpeg: no audio input, -c:a or -shortest when there is no audio.
func TestFFmpegArgsAudioAudit(t *testing.T) {
	opts := ExportOptions{Format: FormatMP4, OutputPath: "out.mp4", VideoCodec: "libx264",
		AudioCodec: "aac", Preset: "fast", Quality: 75}

	silent := NewVideoExporter(auditRecording(2, 0))
	if silent.hasAudio() {
		t.Fatal("hasAudio = true for a recording without samples")
	}
	args := silent.ffmpegArgs(opts, "")
	for _, a := range []string{"-c:a", "-shortest", ""} {
		if auditHasArg(args, a) {
			t.Errorf("args without audio contain %q: %q", a, args)
		}
	}
	if n := strings.Count(strings.Join(args, " "), "-i "); n != 1 {
		t.Errorf("args without audio have %d inputs, want 1: %q", n, args)
	}

	voiced := NewVideoExporter(auditRecording(2, 10))
	if !voiced.hasAudio() {
		t.Fatal("hasAudio = false for a recording with samples")
	}
	args = voiced.ffmpegArgs(opts, "audio.wav")
	for _, a := range []string{"audio.wav", "-c:a", "aac", "-shortest"} {
		if !auditHasArg(args, a) {
			t.Errorf("args with audio miss %q: %q", a, args)
		}
	}
	if args[len(args)-1] != "out.mp4" {
		t.Errorf("last arg = %q, want output path", args[len(args)-1])
	}
}

// TestExportWithAudioAudit: audio is still muxed when the recording has it.
func TestExportWithAudioAudit(t *testing.T) {
	requireFFmpeg(t)
	out := filepath.Join(t.TempDir(), "voiced.mp4")
	ve := NewVideoExporter(auditRecording(15, 8000))
	if err := ve.Export(ExportOptions{Format: FormatMP4, OutputPath: out, Preset: "ultrafast"}); err != nil {
		t.Fatalf("Export: %v", err)
	}
	n, hasAudio := probeStreams(t, out)
	if n == 0 {
		t.Errorf("exported video has no frames")
	}
	if !hasAudio {
		t.Errorf("exported video has no audio stream")
	}
}
