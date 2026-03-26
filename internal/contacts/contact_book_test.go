package contacts

import (
	"os"
	"testing"
	"time"
)

func TestNewContactBook(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cb, err := NewContactBook()
	if err != nil {
		t.Fatalf("NewContactBook() failed: %v", err)
	}

	if cb == nil {
		t.Fatal("NewContactBook() returned nil")
	}

	if cb.Count() != 0 {
		t.Errorf("New contact book should have 0 contacts, got %d", cb.Count())
	}
}

func TestContactBook_Add(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cb, _ := NewContactBook()

	contact := &Contact{
		Name:        "Alice",
		DisplayName: "Alice Smith",
		Address:     "192.168.1.100:5000",
		Email:       "alice@example.com",
		Favorite:    true,
	}

	err := cb.Add(contact)
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	if cb.Count() != 1 {
		t.Errorf("After Add(), count = %d, expected 1", cb.Count())
	}

	if contact.ID == "" {
		t.Error("Add() should set ID if empty")
	}

	if contact.CreatedAt.IsZero() {
		t.Error("Add() should set CreatedAt")
	}
}

func TestContactBook_AddInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cb, _ := NewContactBook()

	// Missing name
	err := cb.Add(&Contact{Address: "test"})
	if err == nil {
		t.Error("Add() should fail with empty name")
	}

	// Missing address
	err = cb.Add(&Contact{Name: "test"})
	if err == nil {
		t.Error("Add() should fail with empty address")
	}
}

func TestContactBook_Get(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cb, _ := NewContactBook()

	contact := &Contact{
		ID:      "alice_123",
		Name:    "Alice",
		Address: "192.168.1.100:5000",
	}
	cb.Add(contact)

	retrieved, err := cb.Get("alice_123")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if retrieved.Name != "Alice" {
		t.Errorf("Get() name = %s, expected 'Alice'", retrieved.Name)
	}
}

func TestContactBook_GetByName(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cb, _ := NewContactBook()

	cb.Add(&Contact{Name: "Alice", Address: "addr1"})
	cb.Add(&Contact{Name: "Bob", Address: "addr2"})

	contact, err := cb.GetByName("Alice")
	if err != nil {
		t.Fatalf("GetByName() failed: %v", err)
	}

	if contact.Address != "addr1" {
		t.Errorf("GetByName() address = %s, expected 'addr1'", contact.Address)
	}

	// Case insensitive
	contact, err = cb.GetByName("alice")
	if err != nil {
		t.Error("GetByName() should be case insensitive")
	}
}

func TestContactBook_GetByAddress(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cb, _ := NewContactBook()

	cb.Add(&Contact{Name: "Alice", Address: "192.168.1.100:5000"})

	contact, err := cb.GetByAddress("192.168.1.100:5000")
	if err != nil {
		t.Fatalf("GetByAddress() failed: %v", err)
	}

	if contact.Name != "Alice" {
		t.Errorf("GetByAddress() name = %s, expected 'Alice'", contact.Name)
	}
}

func TestContactBook_GetAll(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cb, _ := NewContactBook()

	cb.Add(&Contact{Name: "Charlie", Address: "addr1"})
	cb.Add(&Contact{Name: "Alice", Address: "addr2"})
	cb.Add(&Contact{Name: "Bob", Address: "addr3"})

	all := cb.GetAll()

	if len(all) != 3 {
		t.Errorf("GetAll() = %d contacts, expected 3", len(all))
	}

	// Verify sorted by name
	if all[0].Name != "Alice" || all[1].Name != "Bob" || all[2].Name != "Charlie" {
		t.Error("GetAll() should return contacts sorted by name")
	}
}

func TestContactBook_GetFavorites(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cb, _ := NewContactBook()

	cb.Add(&Contact{Name: "Alice", Address: "addr1", Favorite: true})
	cb.Add(&Contact{Name: "Bob", Address: "addr2", Favorite: false})
	cb.Add(&Contact{Name: "Charlie", Address: "addr3", Favorite: true})

	favorites := cb.GetFavorites()

	if len(favorites) != 2 {
		t.Errorf("GetFavorites() = %d, expected 2", len(favorites))
	}
}

func TestContactBook_GetByTag(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cb, _ := NewContactBook()

	cb.Add(&Contact{Name: "Alice", Address: "addr1", Tags: []string{"work", "friend"}})
	cb.Add(&Contact{Name: "Bob", Address: "addr2", Tags: []string{"family"}})
	cb.Add(&Contact{Name: "Charlie", Address: "addr3", Tags: []string{"work"}})

	workContacts := cb.GetByTag("work")

	if len(workContacts) != 2 {
		t.Errorf("GetByTag(work) = %d, expected 2", len(workContacts))
	}
}

func TestContactBook_Search(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cb, _ := NewContactBook()

	cb.Add(&Contact{Name: "Alice Smith", Address: "addr1", Email: "alice@example.com"})
	cb.Add(&Contact{Name: "Bob Jones", Address: "addr2", Email: "bob@test.com"})
	cb.Add(&Contact{Name: "Charlie Smith", Address: "addr3", Email: "charlie@example.com"})

	// Search by name
	results := cb.Search("alice")
	if len(results) != 1 {
		t.Errorf("Search(alice) = %d, expected 1", len(results))
	}

	// Search by last name
	results = cb.Search("smith")
	if len(results) != 2 {
		t.Errorf("Search(smith) = %d, expected 2", len(results))
	}

	// Search by email domain
	results = cb.Search("example.com")
	if len(results) != 2 {
		t.Errorf("Search(example.com) = %d, expected 2", len(results))
	}
}

func TestContactBook_Update(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cb, _ := NewContactBook()

	contact := &Contact{
		ID:      "alice_123",
		Name:    "Alice",
		Address: "addr1",
	}
	cb.Add(contact)

	// Update
	contact.Name = "Alice Smith"
	contact.Email = "alice@example.com"

	err := cb.Update(contact)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	updated, _ := cb.Get("alice_123")
	if updated.Name != "Alice Smith" {
		t.Errorf("Updated name = %s, expected 'Alice Smith'", updated.Name)
	}

	if updated.Email != "alice@example.com" {
		t.Errorf("Updated email = %s, expected 'alice@example.com'", updated.Email)
	}
}

func TestContactBook_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cb, _ := NewContactBook()

	contact := &Contact{
		ID:      "alice_123",
		Name:    "Alice",
		Address: "addr1",
	}
	cb.Add(contact)

	if cb.Count() != 1 {
		t.Fatal("Failed to add contact")
	}

	err := cb.Delete("alice_123")
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	if cb.Count() != 0 {
		t.Errorf("After Delete(), count = %d, expected 0", cb.Count())
	}
}

func TestContactBook_UpdateCallStats(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cb, _ := NewContactBook()

	cb.Add(&Contact{Name: "Alice", Address: "192.168.1.100:5000"})

	err := cb.UpdateCallStats("192.168.1.100:5000")
	if err != nil {
		t.Fatalf("UpdateCallStats() failed: %v", err)
	}

	contact, _ := cb.GetByAddress("192.168.1.100:5000")

	if contact.TotalCalls != 1 {
		t.Errorf("TotalCalls = %d, expected 1", contact.TotalCalls)
	}

	if contact.LastCallTime.IsZero() {
		t.Error("LastCallTime should be set")
	}

	// Update again
	time.Sleep(10 * time.Millisecond)
	oldTime := contact.LastCallTime

	cb.UpdateCallStats("192.168.1.100:5000")

	contact, _ = cb.GetByAddress("192.168.1.100:5000")

	if contact.TotalCalls != 2 {
		t.Errorf("TotalCalls = %d, expected 2", contact.TotalCalls)
	}

	if !contact.LastCallTime.After(oldTime) {
		t.Error("LastCallTime should be updated")
	}
}

func TestContactBook_GetRecent(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cb, _ := NewContactBook()

	now := time.Now()

	cb.Add(&Contact{Name: "Alice", Address: "addr1", LastCallTime: now.Add(-3 * time.Hour)})
	cb.Add(&Contact{Name: "Bob", Address: "addr2", LastCallTime: now.Add(-1 * time.Hour)})
	cb.Add(&Contact{Name: "Charlie", Address: "addr3", LastCallTime: now.Add(-2 * time.Hour)})

	recent := cb.GetRecent(2)

	if len(recent) != 2 {
		t.Errorf("GetRecent(2) = %d, expected 2", len(recent))
	}

	// Should be Bob (most recent), then Charlie
	if recent[0].Name != "Bob" {
		t.Errorf("Most recent = %s, expected 'Bob'", recent[0].Name)
	}
}

func TestContactBook_GetFrequent(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cb, _ := NewContactBook()

	cb.Add(&Contact{Name: "Alice", Address: "addr1", TotalCalls: 10})
	cb.Add(&Contact{Name: "Bob", Address: "addr2", TotalCalls: 25})
	cb.Add(&Contact{Name: "Charlie", Address: "addr3", TotalCalls: 15})

	frequent := cb.GetFrequent(2)

	if len(frequent) != 2 {
		t.Errorf("GetFrequent(2) = %d, expected 2", len(frequent))
	}

	// Should be Bob (25 calls), then Charlie (15 calls)
	if frequent[0].Name != "Bob" {
		t.Errorf("Most frequent = %s, expected 'Bob'", frequent[0].Name)
	}
}

func TestContactBook_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cb1, _ := NewContactBook()

	cb1.Add(&Contact{
		ID:      "alice_123",
		Name:    "Alice",
		Address: "192.168.1.100:5000",
		Email:   "alice@example.com",
	})

	// Create new instance and load
	cb2, err := NewContactBook()
	if err != nil {
		t.Fatalf("NewContactBook() failed: %v", err)
	}

	if cb2.Count() != 1 {
		t.Errorf("After load, count = %d, expected 1", cb2.Count())
	}

	contact, _ := cb2.Get("alice_123")
	if contact.Name != "Alice" {
		t.Errorf("Loaded name = %s, expected 'Alice'", contact.Name)
	}

	if contact.Email != "alice@example.com" {
		t.Errorf("Loaded email = %s, expected 'alice@example.com'", contact.Email)
	}
}
