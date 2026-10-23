package contacts

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Export must write valid CSV: commas, quotes and newlines inside a field
// must be quoted, so every row parses back into the 13 header columns with the
// original values.
func TestContactBookExportQuotesFieldsAudit(t *testing.T) {
	dir := t.TempDir()
	cb := &ContactBook{
		contacts: map[string]*Contact{
			"a": {
				ID:          "a",
				Name:        "Doe, Jane",
				DisplayName: `Jane "JD" Doe`,
				Address:     "10.0.0.1:5000",
				Email:       "jane@example.com",
				Notes:       "line one\nline two, with comma",
				Tags:        []string{"work", "a,b"},
			},
			"b": {ID: "b", Name: "Plain", Address: "10.0.0.2:5000"},
		},
		filePath: filepath.Join(dir, "contacts.json"),
	}

	out := filepath.Join(dir, "contacts.csv")
	if err := cb.Export(out); err != nil {
		t.Fatalf("Export: %v", err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("exported file is not valid CSV: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("got %d records, want header + 2 contacts: %q", len(records), records)
	}
	for i, rec := range records {
		if len(rec) != 13 {
			t.Fatalf("record %d has %d fields, want 13: %q", i, len(rec), rec)
		}
	}

	// GetAll sorts by name: "Doe, Jane" < "Plain"
	got := records[1]
	want := cb.contacts["a"]
	if got[0] != want.ID || got[1] != want.Name || got[2] != want.DisplayName ||
		got[3] != want.Address || got[4] != want.Email || got[7] != want.Notes ||
		got[8] != strings.Join(want.Tags, ";") {
		t.Errorf("record = %q, want fields of %+v", got, *want)
	}
	if records[2][1] != "Plain" {
		t.Errorf("second contact name = %q, want Plain", records[2][1])
	}
}
