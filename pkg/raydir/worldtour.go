package raydir

// worldtour.go turns the Gray-code grand tour of the 64 hexagrams (Hexagram.GrayWalk)
// into cinema. The tour visits every hexagram once, changing a single line at each
// step; TourMorph expands that walk into a smooth SceneVector path, inserting
// interpolation steps between consecutive worlds so the world changes one trait
// GRADUALLY — a film that morphs through every possible world, one axis at a time.

// TourMorph expands a hexagram walk into a SceneVector path with perStep interpolation
// steps between each consecutive pair (perStep<=1 returns just the corner vectors, the
// canonical VectorFromHexagram worlds). Because consecutive hexagrams differ by one
// line, each segment slides a single Q6 axis smoothly from one side to the other.
func TourMorph(walk []Hexagram, perStep int) []SceneVector {
	if len(walk) == 0 {
		return nil
	}
	if perStep < 1 {
		perStep = 1
	}
	out := []SceneVector{VectorFromHexagram(walk[0])}
	for i := 1; i < len(walk); i++ {
		a := VectorFromHexagram(walk[i-1])
		b := VectorFromHexagram(walk[i])
		for s := 1; s <= perStep; s++ {
			t := float64(s) / float64(perStep)
			var v SceneVector
			for k := 0; k < 6; k++ {
				v[k] = a[k]*(1-t) + b[k]*t
			}
			out = append(out, v)
		}
	}
	return out
}
