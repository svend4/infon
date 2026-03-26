package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/svend4/infon/internal/export"
	"github.com/svend4/infon/internal/recorder"
)

func runExport() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: tvcp export [options] <input.tvcp> [output.mp4|output.webm]")
		fmt.Fprintln(os.Stderr, "\nOptions:")
		fmt.Fprintln(os.Stderr, "  --format mp4|webm    Output format (default: auto-detect from extension)")
		fmt.Fprintln(os.Stderr, "  --fps <number>       Target FPS (default: 15)")
		fmt.Fprintln(os.Stderr, "  --scale <number>     Pixel scale factor (default: 8)")
		fmt.Fprintln(os.Stderr, "  --quality <0-100>    Quality level (default: 75)")
		fmt.Fprintln(os.Stderr, "  --preset fast|medium|slow  Encoding preset (default: medium)")
		fmt.Fprintln(os.Stderr, "\nExport .tvcp recordings to MP4 or WebM video format.")
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  tvcp export recording.tvcp output.mp4")
		fmt.Fprintln(os.Stderr, "  tvcp export --quality 90 call.tvcp call.webm")
		fmt.Fprintln(os.Stderr, "  tvcp export --fps 30 --scale 10 recording.tvcp hq.mp4")
		fmt.Fprintln(os.Stderr, "\nNote: Requires FFmpeg to be installed")
		os.Exit(1)
	}

	// Parse arguments
	var inputPath, outputPath string
	var format export.ExportFormat
	options := export.ExportOptions{
		FPS:     15,
		Scale:   8,
		Quality: 75,
		Preset:  "medium",
	}

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "--format":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --format requires a value")
				os.Exit(1)
			}
			i++
			formatStr := strings.ToLower(args[i])
			if formatStr == "mp4" {
				format = export.FormatMP4
			} else if formatStr == "webm" {
				format = export.FormatWebM
			} else {
				fmt.Fprintf(os.Stderr, "Error: invalid format %s (must be mp4 or webm)\n", formatStr)
				os.Exit(1)
			}

		case "--fps":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --fps requires a value")
				os.Exit(1)
			}
			i++
			fmt.Sscanf(args[i], "%d", &options.FPS)

		case "--scale":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --scale requires a value")
				os.Exit(1)
			}
			i++
			fmt.Sscanf(args[i], "%d", &options.Scale)

		case "--quality":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --quality requires a value")
				os.Exit(1)
			}
			i++
			fmt.Sscanf(args[i], "%d", &options.Quality)
			if options.Quality < 0 || options.Quality > 100 {
				fmt.Fprintln(os.Stderr, "Error: quality must be 0-100")
				os.Exit(1)
			}

		case "--preset":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --preset requires a value")
				os.Exit(1)
			}
			i++
			options.Preset = args[i]

		default:
			if !strings.HasPrefix(arg, "--") {
				if inputPath == "" {
					inputPath = arg
				} else if outputPath == "" {
					outputPath = arg
				}
			}
		}
	}

	// Validate input
	if inputPath == "" {
		fmt.Fprintln(os.Stderr, "Error: input file required")
		os.Exit(1)
	}

	// Auto-generate output path if not specified
	if outputPath == "" {
		ext := filepath.Ext(inputPath)
		base := strings.TrimSuffix(inputPath, ext)
		outputPath = base + ".mp4"
		fmt.Printf("Output path not specified, using: %s\n", outputPath)
	}

	// Auto-detect format from extension if not specified
	if format == "" {
		ext := strings.ToLower(filepath.Ext(outputPath))
		if ext == ".mp4" {
			format = export.FormatMP4
		} else if ext == ".webm" {
			format = export.FormatWebM
		} else {
			fmt.Fprintf(os.Stderr, "Error: cannot detect format from extension %s\n", ext)
			fmt.Fprintln(os.Stderr, "Please specify --format mp4 or --format webm")
			os.Exit(1)
		}
	}

	options.Format = format
	options.OutputPath = outputPath

	// Load recording
	fmt.Printf("📂 Loading recording: %s\n", inputPath)
	player := recorder.NewPlayer()
	if err := player.Load(inputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading recording: %v\n", err)
		os.Exit(1)
	}

	// Get recording
	recording := player.GetRecording()

	// Create exporter
	exporter := export.NewVideoExporter(recording)

	// Show export info
	info := exporter.GetExportInfo(options)
	fmt.Printf("\n📊 Export Information:\n")
	fmt.Printf("   Duration: %.1fs\n", info.Duration.Seconds())
	fmt.Printf("   Frames: %d\n", info.FrameCount)
	fmt.Printf("   Audio samples: %d\n", info.AudioSamples)
	fmt.Printf("   Resolution: %dx%d\n", info.Width, info.Height)
	fmt.Printf("   FPS: %d\n", info.FPS)
	fmt.Printf("   Format: %s\n", info.Format)
	fmt.Printf("   Quality: %d\n", options.Quality)
	fmt.Printf("   Preset: %s\n\n", options.Preset)

	// Export
	if err := exporter.Export(options); err != nil {
		fmt.Fprintf(os.Stderr, "Error exporting: %v\n", err)
		os.Exit(1)
	}

	// Show output file info
	fileInfo, err := os.Stat(outputPath)
	if err == nil {
		fmt.Printf("   File size: %.2f MB\n", float64(fileInfo.Size())/(1024*1024))
	}
}
