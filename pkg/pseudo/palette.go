package pseudo

import (
	"strings"

	"github.com/svend4/infon/pkg/color"
)

// Palette is a set of named-color overrides applied on top of the base palette,
// letting a model pick a mood ("neon", "ocean") without re-specifying colors.
type Palette map[string]color.RGB

// resolve looks a token up in the active preset first, then the base palette /
// hex / "r,g,b" via Color.
func resolve(p Palette, tok string, def color.RGB) color.RGB {
	if p != nil {
		if c, ok := p[strings.ToLower(strings.TrimSpace(tok))]; ok {
			return c
		}
	}
	return Color(tok, def)
}

// presetPalette returns a named mood palette remapping the common role names, or
// nil for "" / unknown (callers then fall back to the base palette).
func presetPalette(name string) Palette {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "neon":
		return Palette{
			"navy": mk(12, 6, 36), "blue": mk(40, 60, 255), "skyblue": mk(26, 209, 255),
			"teal": mk(20, 230, 200), "cyan": mk(0, 255, 230), "green": mk(57, 255, 20),
			"gold": mk(255, 61, 240), "amber": mk(255, 120, 200), "orange": mk(255, 80, 160),
			"purple": mk(170, 60, 255), "slate": mk(60, 40, 90), "white": mk(245, 230, 255),
		}
	case "mono":
		return Palette{
			"navy": mk(28, 28, 28), "blue": mk(70, 70, 70), "skyblue": mk(120, 120, 120),
			"teal": mk(150, 150, 150), "green": mk(110, 110, 110), "gold": mk(228, 228, 228),
			"amber": mk(190, 190, 190), "orange": mk(170, 170, 170), "slate": mk(90, 90, 90),
			"purple": mk(80, 80, 80), "white": mk(245, 245, 245), "gray": mk(140, 140, 140),
		}
	case "dusk":
		return Palette{
			"navy": mk(40, 24, 70), "blue": mk(90, 70, 160), "skyblue": mk(150, 120, 200),
			"teal": mk(120, 90, 160), "green": mk(120, 100, 120), "gold": mk(255, 170, 120),
			"amber": mk(230, 130, 110), "orange": mk(240, 110, 90), "slate": mk(80, 60, 100),
			"purple": mk(160, 90, 190), "white": mk(245, 235, 240),
		}
	case "forest":
		return Palette{
			"navy": mk(20, 40, 30), "blue": mk(40, 90, 70), "skyblue": mk(120, 180, 150),
			"teal": mk(60, 150, 110), "green": mk(70, 170, 90), "darkgreen": mk(24, 90, 50),
			"gold": mk(220, 200, 120), "amber": mk(180, 150, 80), "slate": mk(70, 80, 60),
			"brown": mk(110, 80, 50), "white": mk(235, 240, 230),
		}
	case "ocean":
		return Palette{
			"navy": mk(10, 30, 70), "blue": mk(30, 90, 180), "skyblue": mk(110, 180, 230),
			"teal": mk(40, 170, 190), "cyan": mk(120, 230, 235), "green": mk(60, 180, 170),
			"gold": mk(240, 225, 170), "amber": mk(220, 200, 140), "slate": mk(50, 80, 110),
			"white": mk(235, 245, 250),
		}
	case "dawn":
		return Palette{
			"navy": mk(40, 40, 90), "blue": mk(110, 120, 210), "skyblue": mk(180, 190, 240),
			"teal": mk(120, 200, 200), "gold": mk(255, 215, 130), "amber": mk(245, 175, 110),
			"orange": mk(250, 150, 100), "purple": mk(170, 130, 210), "white": mk(250, 245, 240),
		}
	default:
		return nil
	}
}
