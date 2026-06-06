package registry

import "testing"

func sample() *Registry {
	r := New()
	r.Register(Entry{Name: "haiku", URL: "http://h/v1/decide", Kinds: []string{"move", "draw", "image"}, Games: []string{"tangram", "rpg"}})
	r.Register(Entry{Name: "reference", URL: "http://ref/v1/decide", Kinds: []string{"move", "image", "react"}, Games: []string{"tictactoe", "rpg"}})
	return r
}

func TestFindByCapability(t *testing.T) {
	r := sample()
	if got := r.Find("draw", ""); len(got) != 1 || got[0].Name != "haiku" {
		t.Fatalf("Find(draw) = %+v, want [haiku]", got)
	}
	img := r.Find("image", "")
	if len(img) != 2 {
		t.Fatalf("Find(image) = %d entries, want 2", len(img))
	}
	tg := r.Find("move", "tangram")
	if len(tg) != 1 || tg[0].Name != "haiku" {
		t.Fatalf("Find(move,tangram) = %+v, want [haiku]", tg)
	}
	if got := r.Find("move", "wordle"); len(got) != 0 {
		t.Fatalf("Find(move,wordle) = %+v, want none", got)
	}
}

func TestRegisterReplacesByName(t *testing.T) {
	r := sample()
	r.Register(Entry{Name: "haiku", URL: "http://new/v1/decide", Kinds: []string{"image"}})
	if l := r.List(); len(l) != 2 {
		t.Fatalf("expected 2 entries after replace, got %d", len(l))
	}
	if got := r.Find("draw", ""); len(got) != 0 {
		t.Fatal("replaced entry should no longer advertise draw")
	}
}
