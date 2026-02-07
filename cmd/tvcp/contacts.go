package main

import (
	"fmt"
	"os"
	"time"

	"github.com/svend4/infon/internal/contacts"
	"github.com/svend4/infon/internal/yggdrasil"
)

func runContacts() {
	if len(os.Args) < 3 {
		printContactsUsage()
		return
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "list", "ls":
		runContactsList()
	case "add":
		runContactsAdd()
	case "remove", "rm":
		runContactsRemove()
	case "show":
		runContactsShow()
	default:
		fmt.Fprintf(os.Stderr, "Unknown contacts subcommand: %s\n", subcommand)
		printContactsUsage()
		os.Exit(1)
	}
}

func printContactsUsage() {
	fmt.Println("Usage: tvcp contacts <subcommand> [options]")
	fmt.Println("\nSubcommands:")
	fmt.Println("  list              List all contacts")
	fmt.Println("  add <name> <addr> Add a new contact")
	fmt.Println("  remove <name>     Remove a contact")
	fmt.Println("  show <name>       Show contact details")
	fmt.Println("\nExamples:")
	fmt.Println("  tvcp contacts list")
	fmt.Println("  tvcp contacts add alice 200:1234:5678::1")
	fmt.Println("  tvcp contacts remove alice")
}

func runContactsList() {
	cb, err := contacts.GetDefaultContactBook()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading contacts: %v\n", err)
		os.Exit(1)
	}

	contactsList := cb.List()

	if len(contactsList) == 0 {
		fmt.Println("📭 No contacts yet.")
		fmt.Println("\nAdd a contact:")
		fmt.Println("  tvcp contacts add <name> <yggdrasil-address>")
		fmt.Println("\nExample:")
		fmt.Println("  tvcp contacts add alice 200:1234:5678:90ab:cdef::1")
		return
	}

	fmt.Printf("📇 Contacts (%d)\n\n", len(contactsList))

	for _, contact := range contactsList {
		fmt.Printf("  %s\n", contact.Name)
		if contact.Alias != "" {
			fmt.Printf("    Alias: %s\n", contact.Alias)
		}
		fmt.Printf("    Address: %s\n", contact.Address)
		if !contact.LastSeen.IsZero() {
			fmt.Printf("    Last seen: %s\n", formatTime(contact.LastSeen))
		}
		if contact.Notes != "" {
			fmt.Printf("    Notes: %s\n", contact.Notes)
		}
		fmt.Println()
	}

	fmt.Println("Usage:")
	fmt.Println("  To call a contact:")
	fmt.Println("  tvcp call <name>")
}

func runContactsAdd() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "Usage: tvcp contacts add <name> <address>")
		fmt.Fprintln(os.Stderr, "\nExample:")
		fmt.Fprintln(os.Stderr, "  tvcp contacts add alice 200:1234:5678::1")
		os.Exit(1)
	}

	name := os.Args[3]
	address := os.Args[4]

	// Validate address
	if !yggdrasil.IsYggdrasilAddress(address) {
		fmt.Fprintf(os.Stderr, "Error: '%s' does not appear to be a valid Yggdrasil address\n", address)
		fmt.Fprintln(os.Stderr, "\nYggdrasil addresses:")
		fmt.Fprintln(os.Stderr, "  - Are IPv6 addresses")
		fmt.Fprintln(os.Stderr, "  - Start with 200: or 300:")
		fmt.Fprintln(os.Stderr, "  - Example: 200:1234:5678:90ab:cdef::1")
		os.Exit(1)
	}

	cb, err := contacts.GetDefaultContactBook()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading contacts: %v\n", err)
		os.Exit(1)
	}

	contact := contacts.Contact{
		Name:    name,
		Address: address,
		AddedAt: time.Now(),
	}

	if err := cb.Add(contact); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding contact: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Contact '%s' added\n", name)
	fmt.Printf("  Address: %s\n", address)
	fmt.Println("\nYou can now call them:")
	fmt.Printf("  tvcp call %s\n", name)
}

func runContactsRemove() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Usage: tvcp contacts remove <name>")
		os.Exit(1)
	}

	name := os.Args[3]

	cb, err := contacts.GetDefaultContactBook()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading contacts: %v\n", err)
		os.Exit(1)
	}

	if err := cb.Remove(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error removing contact: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Contact '%s' removed\n", name)
}

func runContactsShow() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Usage: tvcp contacts show <name>")
		os.Exit(1)
	}

	name := os.Args[3]

	cb, err := contacts.GetDefaultContactBook()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading contacts: %v\n", err)
		os.Exit(1)
	}

	contact := cb.FindByName(name)
	if contact == nil {
		fmt.Fprintf(os.Stderr, "Contact '%s' not found\n", name)
		os.Exit(1)
	}

	fmt.Printf("📇 Contact: %s\n\n", contact.Name)
	if contact.Alias != "" {
		fmt.Printf("  Alias: %s\n", contact.Alias)
	}
	fmt.Printf("  Address: %s\n", contact.Address)
	if contact.PublicKey != "" {
		fmt.Printf("  Public Key: %s\n", contact.PublicKey)
	}
	fmt.Printf("  Added: %s\n", formatTime(contact.AddedAt))
	if !contact.LastSeen.IsZero() {
		fmt.Printf("  Last Seen: %s\n", formatTime(contact.LastSeen))
	}
	if contact.Notes != "" {
		fmt.Printf("  Notes: %s\n", contact.Notes)
	}
	fmt.Println()
}

func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		mins := int(diff.Minutes())
		return fmt.Sprintf("%d minute%s ago", mins, plural(mins))
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%d hour%s ago", hours, plural(hours))
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d day%s ago", days, plural(days))
	}

	return t.Format("2006-01-02")
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func runYggdrasil() {
	fmt.Println("🌐 Yggdrasil Network Status")
	fmt.Println()

	if !yggdrasil.IsYggdrasilRunning() {
		fmt.Println("❌ Yggdrasil daemon is not running")
		fmt.Println("\nYggdrasil is required for P2P video calls.")
		fmt.Println("\nTo install and start Yggdrasil:")
		fmt.Println("  1. Install: https://yggdrasil-network.github.io/installation.html")
		fmt.Println("  2. Start daemon: sudo systemctl start yggdrasil")
		fmt.Println("  3. Enable on boot: sudo systemctl enable yggdrasil")
		fmt.Println("\nFor now, you can still use local calls:")
		fmt.Println("  tvcp call localhost:5001")
		os.Exit(1)
	}

	fmt.Println("✓ Yggdrasil daemon is running")
	fmt.Println()

	info, err := yggdrasil.GetYggdrasilInfo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting Yggdrasil info: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Your Yggdrasil Address:\n")
	fmt.Printf("  %s\n", info.Address)
	if info.Subnet != "" {
		fmt.Printf("\nYour Subnet:\n")
		fmt.Printf("  %s\n", info.Subnet)
	}
	if info.PublicKey != "" {
		fmt.Printf("\nPublic Key:\n")
		fmt.Printf("  %s\n", info.PublicKey)
	}

	fmt.Println()

	peers, err := yggdrasil.GetPeers()
	if err == nil && len(peers) > 0 {
		fmt.Printf("Connected Peers: %d\n", len(peers))
		for _, peer := range peers {
			fmt.Printf("  • %s\n", peer)
		}
	} else {
		fmt.Println("No connected peers")
		fmt.Println("\nTo connect to peers:")
		fmt.Println("  1. Edit /etc/yggdrasil/yggdrasil.conf")
		fmt.Println("  2. Add peer addresses in the Peers section")
		fmt.Println("  3. Restart: sudo systemctl restart yggdrasil")
	}

	fmt.Println()
	fmt.Println("Share your address with others:")
	fmt.Printf("  %s\n", info.Address)
	fmt.Println()
	fmt.Println("To call someone:")
	fmt.Println("  tvcp call <their-yggdrasil-address>")
	fmt.Println("  tvcp call 200:1234:5678::1")
}
