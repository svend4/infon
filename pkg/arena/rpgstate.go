package arena

// rpgstate.go is a RICH game:rpg board — enough for a remote brain to rebuild the
// position and PLAN (HP, kind, terrain, faction), not just the bare positions the
// reference brief carries. MarshalRpgBoard exports it; (RpgBoard).Arena rebuilds.
// This is what lets pkg/planbrain serve the lookahead Planner over tvcp-ai.

// RpgBoardUnit is one unit in the rich board. ID is its index in the unit list.
type RpgBoardUnit struct {
	ID      int   `json:"id"`
	Kind    uint8 `json:"kind"`
	Faction uint8 `json:"faction"`
	X       uint8 `json:"x"`
	Y       uint8 `json:"y"`
	HP      uint8 `json:"hp"`
	Alive   bool  `json:"alive"`
}

// RpgBoard is the full board for a planning brain: terrain, units, and which
// faction is to move (You).
type RpgBoard struct {
	W       int            `json:"w"`
	H       int            `json:"h"`
	Terrain []uint8        `json:"terrain"`
	You     uint8          `json:"you"`
	Units   []RpgBoardUnit `json:"units"`
}

// MarshalRpgBoard captures the arena as a rich board for faction `you`.
func MarshalRpgBoard(a *Arena, you uint8) RpgBoard {
	b := RpgBoard{W: a.W, H: a.H, You: you, Terrain: append([]uint8(nil), a.Terrain...)}
	for i, u := range a.Units {
		b.Units = append(b.Units, RpgBoardUnit{
			ID: i, Kind: u.Kind, Faction: u.Faction, X: u.X, Y: u.Y, HP: u.HP, Alive: u.Alive,
		})
	}
	return b
}

// Arena rebuilds a board. Unit indices follow ID (0..n-1), so a Planner's
// Action.Unit maps straight back to RpgBoardUnit.ID.
func (b RpgBoard) Arena() *Arena {
	a := &Arena{W: b.W, H: b.H, Terrain: append([]uint8(nil), b.Terrain...)}
	a.Units = make([]Unit, len(b.Units))
	for _, ru := range b.Units {
		if ru.ID < 0 || ru.ID >= len(b.Units) {
			continue
		}
		a.Units[ru.ID] = Unit{Kind: ru.Kind, X: ru.X, Y: ru.Y, HP: ru.HP, Faction: ru.Faction, Alive: ru.Alive}
	}
	return a
}
