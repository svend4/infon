package main

import (
	"fmt"
	"os"
)

const (
	version = "0.0.1-alpha"
	appName = "TVCP - Terminal Video Communication Platform"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "version", "-v", "--version":
		printVersion()
	case "help", "-h", "--help":
		printHelp()
	case "daemon":
		runDaemon()
	case "call":
		runCall()
	case "test":
		runTest()
	case "demo":
		runDemo()
	case "generate":
		runGenerate()
	case "preview":
		runPreview()
	case "send":
		runSend()
	case "receive", "recv":
		runReceive()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printVersion() {
	fmt.Printf("%s v%s\n", appName, version)
	fmt.Println("Copyright (c) 2026 Stefan Engel (svend4)")
	fmt.Println("License: MIT")
}

func printUsage() {
	fmt.Println("Usage: tvcp <command> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  daemon              Start TVCP daemon")
	fmt.Println("  call <address>      Make a video call")
	fmt.Println("  test                Run video/audio test")
	fmt.Println("  demo <image>        Display image in terminal (proof-of-concept)")
	fmt.Println("  preview [pattern]   Live camera preview (animated test patterns)")
	fmt.Println("  send <host:port>    Stream video to remote host")
	fmt.Println("  receive [port]      Receive video stream (default port: 5000)")
	fmt.Println("  generate <file>     Generate a test image")
	fmt.Println("  version             Show version information")
	fmt.Println("  help                Show this help message")
	fmt.Println("\nFor more information, visit: https://github.com/svend4/infon")
}

func printHelp() {
	printVersion()
	fmt.Println()
	printUsage()
	fmt.Println("\nExamples:")
	fmt.Println("  tvcp daemon")
	fmt.Println("  tvcp call 200:abc:def::1")
	fmt.Println("  tvcp call \"Alice/laptop\"")
	fmt.Println("  tvcp test")
}

func runDaemon() {
	fmt.Println("🚧 TVCP Daemon - Coming Soon")
	fmt.Println("\nThis is a pre-alpha version. Daemon functionality is not yet implemented.")
	fmt.Println("For updates, check: https://github.com/svend4/infon")
	os.Exit(0)
}

func runCall() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Error: Missing call address")
		fmt.Fprintln(os.Stderr, "Usage: tvcp call <address>")
		os.Exit(1)
	}

	address := os.Args[2]
	fmt.Printf("🚧 Making call to: %s - Coming Soon\n", address)
	fmt.Println("\nThis is a pre-alpha version. Call functionality is not yet implemented.")
	fmt.Println("For updates, check: https://github.com/svend4/infon")
	os.Exit(0)
}

func runTest() {
	fmt.Println("🚧 TVCP Test Mode - Coming Soon")
	fmt.Println("\nThis is a pre-alpha version. Test functionality is not yet implemented.")
	fmt.Println("\nPlanned test features:")
	fmt.Println("  - Camera detection and preview")
	fmt.Println("  - Microphone test")
	fmt.Println("  - Terminal capability detection")
	fmt.Println("  - Network connectivity test")
	fmt.Println("\nFor updates, check: https://github.com/svend4/infon")
	os.Exit(0)
}
