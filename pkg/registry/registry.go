// Package registry is a tiny "DNS for brains": a directory of tvcp-ai/1 backends
// with capability negotiation. A client asks for the kind (and optional game) it
// needs and gets the brains that advertise that capability, so an agent can find
// a partner without hard-coding a URL. It is in-memory and deterministic; a
// federated, signed directory layers pkg/sign over the same entries.
package registry

import "sort"

// Entry advertises a brain and what it can do.
type Entry struct {
	Name  string   `json:"name"`
	URL   string   `json:"url"`
	Kinds []string `json:"kinds"`           // e.g. move, draw, image, react
	Games []string `json:"games,omitempty"` // e.g. tictactoe, tangram, world, rpg
}

// Registry is a directory of brains.
type Registry struct {
	entries []Entry
}

// New returns an empty registry.
func New() *Registry { return &Registry{} }

func has(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// Register adds a brain, replacing any existing entry with the same Name. Entries
// are kept sorted by Name for deterministic lookups.
func (r *Registry) Register(e Entry) {
	for i := range r.entries {
		if r.entries[i].Name == e.Name {
			r.entries[i] = e
			return
		}
	}
	r.entries = append(r.entries, e)
	sort.Slice(r.entries, func(i, j int) bool { return r.entries[i].Name < r.entries[j].Name })
}

// Find returns the brains supporting kind (and game, if non-empty), by name.
func (r *Registry) Find(kind, game string) []Entry {
	var out []Entry
	for _, e := range r.entries {
		if kind != "" && !has(e.Kinds, kind) {
			continue
		}
		if game != "" && !has(e.Games, game) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// List returns all entries (a copy), sorted by name.
func (r *Registry) List() []Entry {
	out := make([]Entry, len(r.entries))
	copy(out, r.entries)
	return out
}
