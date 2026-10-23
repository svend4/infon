//go:build linux

package contacts

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// A Save that fails part-way (here: the write runs into the file size limit,
// as it would on a full disk) must leave the previous contacts.json intact,
// not a truncated file that NewContactBook then refuses to load.
func TestContactBookSaveFailureKeepsFileAudit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "contacts.json")
	cb := &ContactBook{
		contacts: map[string]*Contact{
			"alice": {ID: "alice", Name: "Alice", Address: "10.0.0.1:5000"},
		},
		filePath: path,
	}
	if err := cb.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	orig, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// The next Save is far larger than the limit set below.
	cb.contacts["bob"] = &Contact{ID: "bob", Name: "Bob", Address: "10.0.0.2:5000", Notes: strings.Repeat("x", 64<<10)}

	var old syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &old); err != nil {
		t.Skipf("Getrlimit: %v", err)
	}
	lim := old
	lim.Cur = uint64(len(orig)) + 4096
	if lim.Cur > old.Max {
		t.Skipf("hard file size limit %d too low", old.Max)
	}
	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &lim); err != nil {
		t.Skipf("Setrlimit: %v", err)
	}
	saveErr := cb.Save()
	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &old); err != nil {
		t.Fatalf("restoring file size limit: %v", err)
	}

	if saveErr == nil {
		t.Fatalf("Save past the file size limit reported success")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("contacts.json after failed Save: %v", err)
	}
	if !bytes.Equal(after, orig) {
		t.Errorf("failed Save left contacts.json as %d bytes, want the previous %d bytes intact", len(after), len(orig))
	}

	reloaded := &ContactBook{contacts: make(map[string]*Contact), filePath: path}
	if err := reloaded.Load(); err != nil {
		t.Errorf("Load after failed Save: %v", err)
	} else if reloaded.Count() != 1 {
		t.Errorf("Load after failed Save: %d contacts, want 1", reloaded.Count())
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("directory holds %v, want only contacts.json", names)
	}
}
