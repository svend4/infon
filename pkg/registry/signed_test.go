package registry

import "testing"

func TestSignVerifyEntry(t *testing.T) {
	key := []byte("trust-key")
	e := Entry{Name: "haiku", URL: "http://h/v1/decide", Kinds: []string{"image", "move"}, Games: []string{"rpg"}}
	se := SignEntry(key, e)
	if !VerifyEntry(key, se) {
		t.Fatal("freshly signed entry did not verify")
	}
	// tamper after signing
	bad := se
	bad.Entry.URL = "http://evil/v1/decide"
	if VerifyEntry(key, bad) {
		t.Fatal("tampered entry verified")
	}
	if VerifyEntry([]byte("other-key"), se) {
		t.Fatal("verified under the wrong key")
	}
}

func TestSignedRegistryAdmitsOnlyValid(t *testing.T) {
	key := []byte("trust-key")
	sr := NewSigned(key)
	good := SignEntry(key, Entry{Name: "ref", URL: "http://ref/v1/decide", Kinds: []string{"image"}})
	if err := sr.Admit(good); err != nil {
		t.Fatalf("Admit(good) error: %v", err)
	}
	// forged: signed under a different key
	forged := SignEntry([]byte("attacker"), Entry{Name: "spoof", URL: "http://evil", Kinds: []string{"image"}})
	if err := sr.Admit(forged); err == nil {
		t.Fatal("Admit accepted a forged entry")
	}
	if got := sr.Find("image", ""); len(got) != 1 || got[0].Name != "ref" {
		t.Fatalf("directory = %+v, want only the verified [ref]", got)
	}
}
