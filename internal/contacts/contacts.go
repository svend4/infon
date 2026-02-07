package contacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Contact represents a contact entry
type Contact struct {
	Name      string    `json:"name"`
	Address   string    `json:"address"` // IPv6 Yggdrasil address
	PublicKey string    `json:"public_key,omitempty"`
	Alias     string    `json:"alias,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	AddedAt   time.Time `json:"added_at"`
	LastSeen  time.Time `json:"last_seen,omitempty"`
}

// ContactBook manages contacts
type ContactBook struct {
	Contacts []Contact `json:"contacts"`
	filePath string
}

// NewContactBook creates or loads a contact book
func NewContactBook(filePath string) (*ContactBook, error) {
	cb := &ContactBook{
		filePath: filePath,
		Contacts: []Contact{},
	}

	// Create directory if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create contacts directory: %w", err)
	}

	// Load existing contacts
	if _, err := os.Stat(filePath); err == nil {
		if err := cb.Load(); err != nil {
			return nil, err
		}
	}

	return cb, nil
}

// GetDefaultContactBook returns the default contact book path
func GetDefaultContactBook() (*ContactBook, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	contactsPath := filepath.Join(home, ".tvcp", "contacts.json")
	return NewContactBook(contactsPath)
}

// Load loads contacts from file
func (cb *ContactBook) Load() error {
	data, err := os.ReadFile(cb.filePath)
	if err != nil {
		return fmt.Errorf("failed to read contacts: %w", err)
	}

	if err := json.Unmarshal(data, &cb.Contacts); err != nil {
		return fmt.Errorf("failed to parse contacts: %w", err)
	}

	return nil
}

// Save saves contacts to file
func (cb *ContactBook) Save() error {
	data, err := json.MarshalIndent(cb.Contacts, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize contacts: %w", err)
	}

	if err := os.WriteFile(cb.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write contacts: %w", err)
	}

	return nil
}

// Add adds a new contact
func (cb *ContactBook) Add(contact Contact) error {
	// Check if contact already exists
	if cb.FindByName(contact.Name) != nil {
		return fmt.Errorf("contact '%s' already exists", contact.Name)
	}

	if cb.FindByAddress(contact.Address) != nil {
		return fmt.Errorf("contact with address '%s' already exists", contact.Address)
	}

	// Set added time
	if contact.AddedAt.IsZero() {
		contact.AddedAt = time.Now()
	}

	cb.Contacts = append(cb.Contacts, contact)
	return cb.Save()
}

// Remove removes a contact by name
func (cb *ContactBook) Remove(name string) error {
	for i, c := range cb.Contacts {
		if c.Name == name || c.Alias == name {
			cb.Contacts = append(cb.Contacts[:i], cb.Contacts[i+1:]...)
			return cb.Save()
		}
	}
	return fmt.Errorf("contact '%s' not found", name)
}

// FindByName finds a contact by name or alias
func (cb *ContactBook) FindByName(name string) *Contact {
	nameLower := strings.ToLower(name)
	for i := range cb.Contacts {
		if strings.ToLower(cb.Contacts[i].Name) == nameLower ||
			strings.ToLower(cb.Contacts[i].Alias) == nameLower {
			return &cb.Contacts[i]
		}
	}
	return nil
}

// FindByAddress finds a contact by address
func (cb *ContactBook) FindByAddress(address string) *Contact {
	for i := range cb.Contacts {
		if cb.Contacts[i].Address == address {
			return &cb.Contacts[i]
		}
	}
	return nil
}

// List returns all contacts sorted by name
func (cb *ContactBook) List() []Contact {
	contacts := make([]Contact, len(cb.Contacts))
	copy(contacts, cb.Contacts)

	sort.Slice(contacts, func(i, j int) bool {
		return contacts[i].Name < contacts[j].Name
	})

	return contacts
}

// UpdateLastSeen updates the last seen time for a contact
func (cb *ContactBook) UpdateLastSeen(name string) error {
	contact := cb.FindByName(name)
	if contact == nil {
		return fmt.Errorf("contact '%s' not found", name)
	}

	contact.LastSeen = time.Now()
	return cb.Save()
}

// Resolve resolves a name/address to an IPv6 address
// If input is an address, returns it as-is
// If input is a name, looks it up in contacts
func (cb *ContactBook) Resolve(nameOrAddress string) (string, error) {
	// Check if it's already an address (contains colons)
	if strings.Contains(nameOrAddress, ":") {
		return nameOrAddress, nil
	}

	// Try to find in contacts
	contact := cb.FindByName(nameOrAddress)
	if contact == nil {
		return "", fmt.Errorf("contact '%s' not found", nameOrAddress)
	}

	return contact.Address, nil
}
