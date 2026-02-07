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

// Contact represents a saved contact
type Contact struct {
	ID             string    `json:"id"`               // Unique contact ID
	Name           string    `json:"name"`             // Contact name
	DisplayName    string    `json:"display_name"`     // Display name
	Address        string    `json:"address"`          // IP:port or hostname
	Email          string    `json:"email"`            // Email (optional)
	Phone          string    `json:"phone"`            // Phone (optional)
	Avatar         string    `json:"avatar"`           // Path to avatar image (optional)
	Favorite       bool      `json:"favorite"`         // Is favorite contact
	Notes          string    `json:"notes"`            // Optional notes
	Tags           []string  `json:"tags"`             // Tags for categorization
	LastCallTime   time.Time `json:"last_call_time"`   // Last time called
	TotalCalls     int       `json:"total_calls"`      // Total number of calls
	CreatedAt      time.Time `json:"created_at"`       // When contact was added
	UpdatedAt      time.Time `json:"updated_at"`       // Last update time
}

// ContactBook manages contacts
type ContactBook struct {
	contacts map[string]*Contact // Map of ID -> Contact
	filePath string
}

// NewContactBook creates a new contact book
func NewContactBook() (*ContactBook, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	contactsDir := filepath.Join(homeDir, ".tvcp")
	if err := os.MkdirAll(contactsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create contacts directory: %w", err)
	}

	filePath := filepath.Join(contactsDir, "contacts.json")

	cb := &ContactBook{
		contacts: make(map[string]*Contact),
		filePath: filePath,
	}

	// Load existing contacts
	if err := cb.Load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return cb, nil
}

// Add adds a new contact
func (cb *ContactBook) Add(contact *Contact) error {
	if contact.ID == "" {
		contact.ID = generateID(contact.Name)
	}

	if contact.Name == "" {
		return fmt.Errorf("contact name cannot be empty")
	}

	if contact.Address == "" {
		return fmt.Errorf("contact address cannot be empty")
	}

	now := time.Now()
	contact.CreatedAt = now
	contact.UpdatedAt = now

	cb.contacts[contact.ID] = contact
	return cb.Save()
}

// Update updates an existing contact
func (cb *ContactBook) Update(contact *Contact) error {
	if contact.ID == "" {
		return fmt.Errorf("contact ID cannot be empty")
	}

	existing, ok := cb.contacts[contact.ID]
	if !ok {
		return fmt.Errorf("contact not found: %s", contact.ID)
	}

	// Preserve creation time
	contact.CreatedAt = existing.CreatedAt
	contact.UpdatedAt = time.Now()

	cb.contacts[contact.ID] = contact
	return cb.Save()
}

// Delete removes a contact by ID
func (cb *ContactBook) Delete(id string) error {
	if _, ok := cb.contacts[id]; !ok {
		return fmt.Errorf("contact not found: %s", id)
	}

	delete(cb.contacts, id)
	return cb.Save()
}

// Get retrieves a contact by ID
func (cb *ContactBook) Get(id string) (*Contact, error) {
	contact, ok := cb.contacts[id]
	if !ok {
		return nil, fmt.Errorf("contact not found: %s", id)
	}
	return contact, nil
}

// GetByName retrieves a contact by name
func (cb *ContactBook) GetByName(name string) (*Contact, error) {
	for _, contact := range cb.contacts {
		if strings.EqualFold(contact.Name, name) {
			return contact, nil
		}
	}
	return nil, fmt.Errorf("contact not found: %s", name)
}

// GetByAddress retrieves a contact by address
func (cb *ContactBook) GetByAddress(address string) (*Contact, error) {
	for _, contact := range cb.contacts {
		if contact.Address == address {
			return contact, nil
		}
	}
	return nil, fmt.Errorf("contact not found: %s", address)
}

// GetAll returns all contacts, sorted by name
func (cb *ContactBook) GetAll() []*Contact {
	contacts := make([]*Contact, 0, len(cb.contacts))
	for _, contact := range cb.contacts {
		contacts = append(contacts, contact)
	}

	sort.Slice(contacts, func(i, j int) bool {
		return contacts[i].Name < contacts[j].Name
	})

	return contacts
}

// GetFavorites returns all favorite contacts
func (cb *ContactBook) GetFavorites() []*Contact {
	contacts := make([]*Contact, 0)
	for _, contact := range cb.contacts {
		if contact.Favorite {
			contacts = append(contacts, contact)
		}
	}

	sort.Slice(contacts, func(i, j int) bool {
		return contacts[i].Name < contacts[j].Name
	})

	return contacts
}

// GetByTag returns all contacts with a specific tag
func (cb *ContactBook) GetByTag(tag string) []*Contact {
	contacts := make([]*Contact, 0)
	for _, contact := range cb.contacts {
		for _, t := range contact.Tags {
			if strings.EqualFold(t, tag) {
				contacts = append(contacts, contact)
				break
			}
		}
	}

	sort.Slice(contacts, func(i, j int) bool {
		return contacts[i].Name < contacts[j].Name
	})

	return contacts
}

// Search searches contacts by name, address, email, or phone
func (cb *ContactBook) Search(query string) []*Contact {
	query = strings.ToLower(query)
	contacts := make([]*Contact, 0)

	for _, contact := range cb.contacts {
		if strings.Contains(strings.ToLower(contact.Name), query) ||
			strings.Contains(strings.ToLower(contact.DisplayName), query) ||
			strings.Contains(strings.ToLower(contact.Address), query) ||
			strings.Contains(strings.ToLower(contact.Email), query) ||
			strings.Contains(strings.ToLower(contact.Phone), query) {
			contacts = append(contacts, contact)
		}
	}

	sort.Slice(contacts, func(i, j int) bool {
		return contacts[i].Name < contacts[j].Name
	})

	return contacts
}

// GetRecent returns contacts sorted by last call time
func (cb *ContactBook) GetRecent(n int) []*Contact {
	contacts := cb.GetAll()

	// Sort by last call time (most recent first)
	sort.Slice(contacts, func(i, j int) bool {
		return contacts[i].LastCallTime.After(contacts[j].LastCallTime)
	})

	if len(contacts) < n {
		return contacts
	}
	return contacts[:n]
}

// GetFrequent returns contacts sorted by call count
func (cb *ContactBook) GetFrequent(n int) []*Contact {
	contacts := cb.GetAll()

	// Sort by total calls (most frequent first)
	sort.Slice(contacts, func(i, j int) bool {
		return contacts[i].TotalCalls > contacts[j].TotalCalls
	})

	if len(contacts) < n {
		return contacts
	}
	return contacts[:n]
}

// UpdateCallStats updates call statistics for a contact
func (cb *ContactBook) UpdateCallStats(address string) error {
	contact, err := cb.GetByAddress(address)
	if err != nil {
		return err // Not a saved contact, ignore
	}

	contact.LastCallTime = time.Now()
	contact.TotalCalls++
	contact.UpdatedAt = time.Now()

	return cb.Save()
}

// Count returns the number of contacts
func (cb *ContactBook) Count() int {
	return len(cb.contacts)
}

// Clear removes all contacts
func (cb *ContactBook) Clear() error {
	cb.contacts = make(map[string]*Contact)
	return cb.Save()
}

// Save saves contacts to disk
func (cb *ContactBook) Save() error {
	data, err := json.MarshalIndent(cb.contacts, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal contacts: %w", err)
	}

	if err := os.WriteFile(cb.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write contacts: %w", err)
	}

	return nil
}

// Load loads contacts from disk
func (cb *ContactBook) Load() error {
	data, err := os.ReadFile(cb.filePath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &cb.contacts); err != nil {
		return fmt.Errorf("failed to unmarshal contacts: %w", err)
	}

	return nil
}

// Export exports contacts to CSV format
func (cb *ContactBook) Export(path string) error {
	contacts := cb.GetAll()

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer f.Close()

	// Write CSV header
	header := "ID,Name,Display Name,Address,Email,Phone,Favorite,Notes,Tags,Last Call,Total Calls,Created,Updated\n"
	if _, err := f.WriteString(header); err != nil {
		return err
	}

	// Write contacts
	for _, contact := range contacts {
		tags := strings.Join(contact.Tags, ";")

		line := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%t,%s,%s,%s,%d,%s,%s\n",
			contact.ID,
			contact.Name,
			contact.DisplayName,
			contact.Address,
			contact.Email,
			contact.Phone,
			contact.Favorite,
			contact.Notes,
			tags,
			contact.LastCallTime.Format(time.RFC3339),
			contact.TotalCalls,
			contact.CreatedAt.Format(time.RFC3339),
			contact.UpdatedAt.Format(time.RFC3339),
		)

		if _, err := f.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}

// Import imports contacts from CSV format
func (cb *ContactBook) Import(path string) error {
	// TODO: Implement CSV import
	return fmt.Errorf("import not yet implemented")
}

// generateID generates a contact ID from name
func generateID(name string) string {
	// Simple ID generation: lowercase, replace spaces with underscores
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "_")
	id = strings.ReplaceAll(id, ".", "")

	// Add timestamp to ensure uniqueness
	return fmt.Sprintf("%s_%d", id, time.Now().UnixNano())
}
