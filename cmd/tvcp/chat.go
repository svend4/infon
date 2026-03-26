package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/svend4/infon/internal/network"
)

func runChat() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: tvcp chat <host:port>")
		fmt.Fprintln(os.Stderr, "\nThis starts a text chat session.")
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  tvcp chat localhost:5000")
		fmt.Fprintln(os.Stderr, "  tvcp chat [200:abc::1]:5000")
		os.Exit(1)
	}

	remoteAddr := os.Args[2]
	localPort := "5000"

	// Add default port if not specified
	if !strings.Contains(remoteAddr, ":") {
		remoteAddr = remoteAddr + ":5000"
	}

	fmt.Println("💬 TVCP Text Chat")
	fmt.Printf("Remote: %s\n", remoteAddr)
	fmt.Printf("Local port: %s\n\n", localPort)

	// Resolve remote address
	udpAddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving address: %v\n", err)
		os.Exit(1)
	}

	// Create UDP transport
	transport, err := network.NewTransport(":" + localPort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating transport: %v\n", err)
		os.Exit(1)
	}
	defer transport.Close()

	fmt.Println("Type your messages and press Enter to send.")
	fmt.Println("Press Ctrl+C to exit.")
	fmt.Println()

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Username
	username := "User"
	if hostname, err := os.Hostname(); err == nil {
		username = hostname
	}

	// Goroutine to receive messages
	go func() {
		for {
			packet, _, err := transport.ReceivePacket()
			if err != nil {
				continue
			}

			if packet.Type == network.PacketTypeTextChat {
				textMsg, err := network.DecodeTextMessage(packet.Payload)
				if err != nil {
					continue
				}

				sender := textMsg.Sender
				if sender == "" {
					sender = "Remote"
				}
				fmt.Printf("\r💬 [%s] %s: %s\n> ", textMsg.FormatTime(), sender, textMsg.Message)
			}
		}
	}()

	// Goroutine to send messages
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for {
			fmt.Print("> ")
			if !scanner.Scan() {
				break
			}

			message := strings.TrimSpace(scanner.Text())
			if message == "" {
				continue
			}

			// Create text message
			textMsg := network.NewTextMessage(username, message)
			payload, err := network.EncodeTextMessage(textMsg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding message: %v\n", err)
				continue
			}

			// Send packet
			packet := &network.Packet{
				Type:      network.PacketTypeTextChat,
				Sequence:  transport.NextSequence(),
				Timestamp: uint64(time.Now().UnixMilli()),
				Payload:   payload,
			}

			if err := transport.SendPacket(packet, udpAddr); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending message: %v\n", err)
			}
		}
	}()

	// Wait for interrupt
	<-sigChan
	fmt.Println("\n\n👋 Chat ended")
}
