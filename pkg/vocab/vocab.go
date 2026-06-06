// Package vocab makes the project's symbol vocabulary PLURAL and localizable.
// The renderer understands a fixed set of CANONICAL tokens (sun, mountain, gold,
// navy...), but "what is sky, what is home" is culture-laden — so vocab maps
// localized aliases (per namespace and locale) onto canonical tokens, with
// fallback. A model or person can speak their own words; rendering stays neutral.
package vocab

import (
	"sort"
	"strings"
)

func norm(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

type triple [3]string // {namespace, locale, term}

// Vocab is a set of namespaced, localized alias<->canonical mappings.
type Vocab struct {
	fwd map[triple]string // {ns, locale, alias} -> canonical
	rev map[triple]string // {ns, locale, canonical} -> alias (first wins)
}

// New returns an empty vocabulary.
func New() *Vocab {
	return &Vocab{fwd: map[triple]string{}, rev: map[triple]string{}}
}

// Define maps alias -> canonical in (namespace, locale). Locale "" is the default
// used as a fallback for every locale.
func (v *Vocab) Define(ns, locale, alias, canonical string) {
	ns, locale, alias, canonical = norm(ns), norm(locale), norm(alias), norm(canonical)
	v.fwd[triple{ns, locale, alias}] = canonical
	if _, ok := v.rev[triple{ns, locale, canonical}]; !ok {
		v.rev[triple{ns, locale, canonical}] = alias
	}
}

// Resolve turns an alias into its canonical token: try (ns, locale), then the
// default locale, else assume the alias is already canonical (ok=false).
func (v *Vocab) Resolve(ns, locale, alias string) (string, bool) {
	ns, locale, alias = norm(ns), norm(locale), norm(alias)
	if c, ok := v.fwd[triple{ns, locale, alias}]; ok {
		return c, true
	}
	if c, ok := v.fwd[triple{ns, "", alias}]; ok {
		return c, true
	}
	return alias, false
}

// Localize turns a canonical token into a locale's alias (for output), falling
// back to the default locale, else the canonical token itself.
func (v *Vocab) Localize(ns, locale, canonical string) (string, bool) {
	ns, locale, canonical = norm(ns), norm(locale), norm(canonical)
	if a, ok := v.rev[triple{ns, locale, canonical}]; ok {
		return a, true
	}
	if a, ok := v.rev[triple{ns, "", canonical}]; ok {
		return a, true
	}
	return canonical, false
}

// ResolveAll resolves a list of aliases (a "scene" in some language) to canonical.
func (v *Vocab) ResolveAll(ns, locale string, aliases []string) []string {
	out := make([]string, len(aliases))
	for i, a := range aliases {
		out[i], _ = v.Resolve(ns, locale, a)
	}
	return out
}

// Locales lists the non-default locales defined in a namespace (sorted).
func (v *Vocab) Locales(ns string) []string {
	ns = norm(ns)
	seen := map[string]bool{}
	for k := range v.fwd {
		if k[0] == ns && k[1] != "" {
			seen[k[1]] = true
		}
	}
	out := make([]string, 0, len(seen))
	for l := range seen {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

// Default returns a vocabulary pre-seeded with English/Russian/Spanish names for
// the sigil and color namespaces — a starting plurality, not the only one.
func Default() *Vocab {
	v := New()
	sigil := map[string][3]string{
		// canonical: {ru, es, de}
		"sun":      {"солнце", "sol", "sonne"},
		"moon":     {"луна", "luna", "mond"},
		"star":     {"звезда", "estrella", "stern"},
		"mountain": {"гора", "montaña", "berg"},
		"house":    {"дом", "casa", "haus"},
		"tree":     {"дерево", "árbol", "baum"},
		"cloud":    {"облако", "nube", "wolke"},
		"wave":     {"волна", "ola", "welle"},
		"boat":     {"лодка", "barco", "boot"},
		"anchor":   {"якорь", "ancla", "anker"},
		"flag":     {"флаг", "bandera", "flagge"},
	}
	locales := []string{"ru", "es", "de"}
	for canon, names := range sigil {
		v.Define("sigil", "", canon, canon) // identity in the default locale
		for i, loc := range locales {
			v.Define("sigil", loc, names[i], canon)
		}
	}
	color := map[string][3]string{
		"gold": {"золотой", "oro", "gold"},
		"navy": {"тёмно-синий", "azul-marino", "marineblau"},
		"teal": {"бирюзовый", "verde-azulado", "blaugrün"},
		"sand": {"песочный", "arena", "sand"},
		"snow": {"снежный", "nieve", "schnee"},
	}
	for canon, names := range color {
		v.Define("color", "", canon, canon)
		for i, loc := range locales {
			v.Define("color", loc, names[i], canon)
		}
	}
	return v
}
