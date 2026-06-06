package sign

import "testing"

func TestSignVerify(t *testing.T) {
	key := []byte("shared-secret")
	msg := []byte(`{"kind":"move","move":{"row":1,"col":1}}`)
	s := Sign(Version, key, msg)
	if !Verify(key, msg, s, Version) {
		t.Fatal("valid stamp did not verify")
	}
}

func TestTamperFails(t *testing.T) {
	key := []byte("k")
	msg := []byte("hello world")
	s := Sign(Version, key, msg)
	if Verify(key, []byte("hello worlD"), s, Version) {
		t.Fatal("tampered payload verified")
	}
	if Verify([]byte("other-key"), msg, s, Version) {
		t.Fatal("wrong key verified")
	}
	if Verify(key, msg, s, "tvcp-ai/2") {
		t.Fatal("wrong version verified")
	}
	bad := s
	bad.Sig = "zz" + bad.Sig[2:]
	if Verify(key, msg, bad, Version) {
		t.Fatal("corrupt signature verified")
	}
}
