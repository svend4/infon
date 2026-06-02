package arena

import "github.com/svend4/infon/pkg/tangram"

// summon.go is the crafting bridge: a tangram figure is an address in the
// catalog (pkg/tangram), and naming it summons the matching creature into the
// world. "Build the figure, cast the address, summon the unit." A figure that
// classifies to a non-creature (boat/house/tree) or to gibberish summons
// nothing - like a wrong gate address.

var kindByName = func() map[string]uint8 {
	m := make(map[string]uint8, len(Bestiary))
	for i, k := range Bestiary {
		m[k.Name] = uint8(i)
	}
	return m
}()

// SummonKind returns the bestiary kind index for a creature name.
func SummonKind(name string) (uint8, bool) {
	k, ok := kindByName[name]
	return k, ok
}

// Summon classifies a tangram figure and, if it confidently names a creature in
// the bestiary, returns a unit of that kind. It also returns the classified name
// and whether a creature was summoned.
func Summon(f tangram.Figure, faction, x, y uint8) (Unit, string, bool) {
	m := tangram.Classify(f)
	if m.Score < 0.6 {
		return Unit{}, m.Name, false
	}
	k, ok := SummonKind(m.Name)
	if !ok {
		return Unit{}, m.Name, false
	}
	return Unit{Kind: k, X: x, Y: y, HP: Bestiary[k].HP, Faction: faction, Alive: true}, m.Name, true
}
