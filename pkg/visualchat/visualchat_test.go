package visualchat

import "testing"

func TestSteerEncodeDecodeRoundTrip(t *testing.T) {
	s := Steer{Prompt: "a calm bay", Style: "watercolor", Strength: 0.6, Seed: 42}
	data, err := s.Encode()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	got, err := DecodeSteer(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Prompt != s.Prompt || got.Style != s.Style || got.Strength != s.Strength || got.Seed != s.Seed {
		t.Errorf("round trip mismatch: %+v vs %+v", got, s)
	}
	if got.Protocol != Protocol || got.Kind != "steer" {
		t.Errorf("encode should set protocol/kind, got %q/%q", got.Protocol, got.Kind)
	}
}

func TestDecodeRejectsWrongKind(t *testing.T) {
	if _, err := DecodeSteer([]byte(`{"kind":"move"}`)); err != ErrNotSteer {
		t.Errorf("expected ErrNotSteer, got %v", err)
	}
	// Missing kind is tolerated (treated as a steer).
	if _, err := DecodeSteer([]byte(`{"prompt":"x"}`)); err != nil {
		t.Errorf("kindless steer should decode, got %v", err)
	}
}

func TestSteerApplyPromptRules(t *testing.T) {
	// Prompt replaces.
	if got := (Steer{Prompt: "new"}).Apply("old"); got != "new" {
		t.Errorf("replace: got %q", got)
	}
	// Append extends.
	if got := (Steer{Append: "at dusk"}).Apply("a harbor"); got != "a harbor, at dusk" {
		t.Errorf("append: got %q", got)
	}
	// Append onto empty.
	if got := (Steer{Append: "start"}).Apply(""); got != "start" {
		t.Errorf("append-empty: got %q", got)
	}
	// Style adds a qualifier once (idempotent).
	once := (Steer{Style: "noir"}).Apply("a street")
	if once != "a street, noir" {
		t.Errorf("style: got %q", once)
	}
	twice := (Steer{Style: "noir"}).Apply(once)
	if twice != once {
		t.Errorf("style should be idempotent: got %q", twice)
	}
}

func TestSteerIsEmpty(t *testing.T) {
	if !(Steer{}).IsEmpty() {
		t.Error("zero steer should be empty")
	}
	if (Steer{Prompt: "x"}).IsEmpty() {
		t.Error("steer with prompt is not empty")
	}
}

// fakeSetter records prompts pushed to it.
type fakeSetter struct{ prompts []string }

func (f *fakeSetter) SetPrompt(p string) { f.prompts = append(f.prompts, p) }

func TestControllerPushesPromptChanges(t *testing.T) {
	target := &fakeSetter{}
	c := NewController("initial", target)
	// Constructor pushes the initial prompt.
	if len(target.prompts) != 1 || target.prompts[0] != "initial" {
		t.Fatalf("constructor should push initial prompt, got %v", target.prompts)
	}

	st := c.Apply(Steer{Prompt: "a calm bay", Strength: 0.5, Seed: 7})
	if st.Prompt != "a calm bay" || st.Strength != 0.5 || st.Seed != 7 {
		t.Errorf("state after steer = %+v", st)
	}
	if target.prompts[len(target.prompts)-1] != "a calm bay" {
		t.Errorf("prompt change should be pushed, got %v", target.prompts)
	}
}

func TestControllerNoPushWhenPromptUnchanged(t *testing.T) {
	target := &fakeSetter{}
	c := NewController("scene", target)
	before := len(target.prompts)
	// A steer that only changes strength must not re-push the (unchanged) prompt.
	c.Apply(Steer{Strength: 0.8})
	if len(target.prompts) != before {
		t.Errorf("unchanged prompt should not be pushed again: %v", target.prompts)
	}
	if c.Snapshot().Strength != 0.8 {
		t.Errorf("strength should still update, got %v", c.Snapshot().Strength)
	}
}

func TestControllerStrengthClamped(t *testing.T) {
	c := NewController("x", nil)
	if got := c.Apply(Steer{Strength: 5}).Strength; got != 1 {
		t.Errorf("strength should clamp to 1, got %v", got)
	}
}

func TestControllerApplyJSON(t *testing.T) {
	c := NewController("base", nil)
	data := []byte(`{"kind":"steer","append":"at night"}`)
	st, err := c.ApplyJSON(data)
	if err != nil {
		t.Fatalf("ApplyJSON: %v", err)
	}
	if st.Prompt != "base, at night" {
		t.Errorf("got %q", st.Prompt)
	}
}
