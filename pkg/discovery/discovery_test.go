package discovery

import "testing"

func TestNameAndHandle(t *testing.T) {
	if got := Name("alice", "example.com"); got != "_tvcp.alice.example.com" {
		t.Fatalf("Name = %q", got)
	}
	u, d, err := ParseHandle("alice@example.com")
	if err != nil || u != "alice" || d != "example.com" {
		t.Fatalf("ParseHandle = %q,%q,%v", u, d, err)
	}
	for _, bad := range []string{"alice", "@example.com", "alice@", "a@b@c", ""} {
		if _, _, err := ParseHandle(bad); err == nil {
			t.Fatalf("ParseHandle(%q) should error", bad)
		}
	}
}

func TestFormatParseRoundTrip(t *testing.T) {
	for _, r := range []Record{
		{Ygg: "200:abc:def::1:5000", Version: "1", PubKey: "deadbeef"},
		{Ygg: "201:1:2:3::9:6000", Version: "1"},
	} {
		got, err := Parse(Format(r))
		if err != nil {
			t.Fatalf("Parse(Format(%+v)) error: %v", r, err)
		}
		if got != r {
			t.Fatalf("round-trip: got %+v, want %+v", got, r)
		}
	}
}

func TestParseRejectsNoYgg(t *testing.T) {
	if _, err := Parse("v=1 pubkey=abc"); err == nil {
		t.Fatal("a record with no ygg= should be rejected")
	}
	// unknown keys are tolerated
	r, err := Parse("ygg=200::1:5000 v=2 region=eu")
	if err != nil || r.Ygg != "200::1:5000" || r.Version != "2" {
		t.Fatalf("Parse tolerant = %+v, %v", r, err)
	}
}

func TestLookup(t *testing.T) {
	m := MapResolver{
		Name("alice", "example.com"): {"ygg=200:abc::1:5000 v=1 pubkey=ff"},
		Name("bob", "example.com"):   {"not a tvcp record", "ygg=201::2:6000 v=1"},
	}
	rec, err := Lookup(m, "alice@example.com")
	if err != nil || rec.Ygg != "200:abc::1:5000" || rec.PubKey != "ff" {
		t.Fatalf("Lookup alice = %+v, %v", rec, err)
	}
	// skips the junk TXT and finds the valid one
	if rec, err := Lookup(m, "bob@example.com"); err != nil || rec.Ygg != "201::2:6000" {
		t.Fatalf("Lookup bob = %+v, %v", rec, err)
	}
	if _, err := Lookup(m, "carol@example.com"); err == nil {
		t.Fatal("Lookup of unknown handle should error")
	}
	if _, err := Lookup(m, "bogus"); err == nil {
		t.Fatal("Lookup of bad handle should error")
	}
}
