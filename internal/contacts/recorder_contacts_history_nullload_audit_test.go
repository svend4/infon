package contacts

import (
	"os"
	"path/filepath"
	"testing"
)

// A contacts.json holding JSON null (or a null contact) must load into a
// usable contact book: adding, updating and looking up contacts afterwards
// must not panic on a nil map or a nil *Contact.
func TestContactBookLoadNullAudit(t *testing.T) {
	cases := map[string]string{
		"null book":    "null",
		"null contact": `{"ghost": null, "bob": {"id": "bob", "name": "Bob", "address": "10.0.0.2:5000"}}`,
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
			dir := filepath.Join(home, ".tvcp")
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "contacts.json"), []byte(data), 0644); err != nil {
				t.Fatal(err)
			}

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("contact book loaded from %q panicked: %v", data, r)
				}
			}()

			cb, err := NewContactBook()
			if err != nil {
				t.Fatalf("NewContactBook: %v", err)
			}
			if err := cb.Add(&Contact{ID: "alice", Name: "Alice", Address: "10.0.0.1:5000"}); err != nil {
				t.Fatalf("Add: %v", err)
			}
			if err := cb.Update(&Contact{ID: "alice", Name: "Alice", Address: "10.0.0.9:5000"}); err != nil {
				t.Fatalf("Update: %v", err)
			}
			if addr, err := cb.Resolve("alice"); err != nil || addr != "10.0.0.9:5000" {
				t.Fatalf("Resolve(alice) = %q, %v", addr, err)
			}
			_, _ = cb.GetByAddress("nowhere")
			_ = cb.GetAll()
			_ = cb.GetFavorites()
			_ = cb.GetByTag("x")
			_ = cb.Search("a")
			_ = cb.GetRecent(5)
			_ = cb.GetFrequent(5)
			for _, c := range cb.GetAll() {
				if c == nil {
					t.Fatalf("GetAll returned a nil contact")
				}
			}
			if err := cb.Export(filepath.Join(home, "out.csv")); err != nil {
				t.Fatalf("Export: %v", err)
			}
		})
	}
}
