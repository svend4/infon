package yggdrasil

import (
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// Info contains Yggdrasil network information
type Info struct {
	Address   string   `json:"address"`   // IPv6 address (200:xxxx::/7)
	Subnet    string   `json:"subnet"`    // IPv6 subnet
	PublicKey string   `json:"public_key"`
	Peers     []string `json:"peers"`
	IsRunning bool     `json:"is_running"`
}

// IsYggdrasilRunning checks if Yggdrasil daemon is running
func IsYggdrasilRunning() bool {
	// Check if yggdrasil process is running
	cmd := exec.Command("pgrep", "-x", "yggdrasil")
	err := cmd.Run()
	return err == nil
}

// GetYggdrasilInfo retrieves Yggdrasil network information
func GetYggdrasilInfo() (*Info, error) {
	if !IsYggdrasilRunning() {
		return nil, fmt.Errorf("yggdrasil daemon is not running")
	}

	// Try yggdrasilctl first
	cmd := exec.Command("yggdrasilctl", "getSelf")
	output, err := cmd.Output()
	if err != nil {
		// Fallback: try to get address from network interfaces
		return getInfoFromInterfaces()
	}

	// Parse JSON output
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return getInfoFromInterfaces()
	}

	info := &Info{
		IsRunning: true,
	}

	// Extract address
	if addr, ok := result["address"].(string); ok {
		info.Address = addr
	}

	// Extract subnet
	if subnet, ok := result["subnet"].(string); ok {
		info.Subnet = subnet
	}

	// Extract public key
	if pubkey, ok := result["key"].(string); ok {
		info.PublicKey = pubkey
	}

	return info, nil
}

// getInfoFromInterfaces tries to find Yggdrasil address from network interfaces
func getInfoFromInterfaces() (*Info, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, iface := range interfaces {
		// Look for tun0 or ygg0 interface (common Yggdrasil interface names)
		if iface.Name != "tun0" && iface.Name != "ygg0" && !strings.HasPrefix(iface.Name, "ygg") {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP
			// Yggdrasil addresses are in 200::/7 range
			if ip.To16() != nil && len(ip) == 16 {
				// Check if it's in 200::/7 or 300::/7 range
				if ip[0] == 0x02 || ip[0] == 0x03 {
					return &Info{
						Address:   ip.String(),
						IsRunning: true,
					}, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no yggdrasil address found")
}

// GetPeers retrieves list of connected peers
func GetPeers() ([]string, error) {
	if !IsYggdrasilRunning() {
		return nil, fmt.Errorf("yggdrasil daemon is not running")
	}

	cmd := exec.Command("yggdrasilctl", "getPeers")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get peers: %w", err)
	}

	// Parse JSON output
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse peers: %w", err)
	}

	var peers []string
	if peersMap, ok := result["peers"].(map[string]interface{}); ok {
		for peer := range peersMap {
			peers = append(peers, peer)
		}
	}

	return peers, nil
}

// IsYggdrasilAddress checks if an address is a valid Yggdrasil IPv6 address
func IsYggdrasilAddress(addr string) bool {
	// Remove port if present (IPv6 format is [addr]:port)
	host := addr
	if strings.HasPrefix(addr, "[") {
		// Format: [ipv6]:port
		if idx := strings.Index(addr, "]:"); idx != -1 {
			host = addr[1:idx]
		} else if strings.HasSuffix(addr, "]") {
			// Format: [ipv6]
			host = addr[1 : len(addr)-1]
		}
	}

	// Parse IP
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	// Check if it's IPv6
	if ip.To4() != nil {
		return false
	}

	// Yggdrasil addresses are in 200::/7 or 300::/7 range
	return ip[0] == 0x02 || ip[0] == 0x03
}

// FormatAddress formats an IPv6 address for display
func FormatAddress(addr string) string {
	ip := net.ParseIP(addr)
	if ip == nil {
		return addr
	}
	return ip.String()
}

// GetSelfAddress returns the local Yggdrasil IPv6 address
func GetSelfAddress() (string, error) {
	info, err := GetYggdrasilInfo()
	if err != nil {
		return "", err
	}
	return info.Address, nil
}
