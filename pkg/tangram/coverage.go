package tangram

import "github.com/svend4/infon/pkg/glyphset"

// glyphsetSub and coverageAt isolate the glyphset mask seam used by the
// validator, keeping the tiling logic in one place.
func glyphsetSub() int { return glyphset.Sub }

func coverageAt(token string, x, y, S int) bool { return glyphset.Coverage(token, x, y, S) }
