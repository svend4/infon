package vocab

import "testing"

func TestResolveLocalized(t *testing.T) {
	v := Default()
	cases := []struct{ loc, alias, want string }{
		{"ru", "гора", "mountain"},
		{"es", "casa", "house"},
		{"ru", "солнце", "sun"},
		{"de", "baum", "tree"},
		{"es", "árbol", "tree"},
		{"es", "CASA", "house"}, // case-insensitive
	}
	for _, c := range cases {
		got, ok := v.Resolve("sigil", c.loc, c.alias)
		if !ok || got != c.want {
			t.Fatalf("Resolve(sigil,%s,%s) = %q ok=%v, want %q", c.loc, c.alias, got, ok, c.want)
		}
	}
}

func TestFallbackAndIdentity(t *testing.T) {
	v := Default()
	if got, ok := v.Resolve("sigil", "ru", "sun"); !ok || got != "sun" {
		t.Fatalf("canonical via default locale: %q ok=%v", got, ok)
	}
	if got, ok := v.Resolve("sigil", "ru", "дракон"); ok || got != "дракон" {
		t.Fatalf("unknown alias should pass through: %q ok=%v", got, ok)
	}
	if got, _ := v.Resolve("sigil", "jp", "sun"); got != "sun" {
		t.Fatalf("unknown locale should fall back: %q", got)
	}
}

func TestLocalizeReverse(t *testing.T) {
	v := Default()
	if a, ok := v.Localize("sigil", "ru", "mountain"); !ok || a != "гора" {
		t.Fatalf("Localize(ru,mountain) = %q ok=%v", a, ok)
	}
	if a, ok := v.Localize("sigil", "es", "house"); !ok || a != "casa" {
		t.Fatalf("Localize(es,house) = %q ok=%v", a, ok)
	}
}

func TestResolveAllScene(t *testing.T) {
	v := Default()
	got := v.ResolveAll("sigil", "ru", []string{"солнце", "гора", "дом"})
	want := []string{"sun", "mountain", "house"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("scene[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if locs := v.Locales("sigil"); len(locs) != 3 {
		t.Fatalf("locales = %v, want 3 (de,es,ru)", locs)
	}
}
