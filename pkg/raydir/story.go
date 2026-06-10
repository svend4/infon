package raydir

import "github.com/svend4/infon/pkg/raytrace"

// story.go turns a walk into a story. Instead of unrelated regions, the director
// follows a Story: an ordered set of Chapters, each with its own prompt (the seed
// the world is grown from) and a line the guide speaks when you cross into it. A
// chapter spans a few regions; as you walk far enough, the world turns the page —
// a beacon marks the threshold and the narration moves on. The Story is a small,
// pure state machine, so it runs offline and is fully testable; the chapter
// prompts feed the same director, so a live model tells the same arc its own way.

// Chapter is one act of the story.
type Chapter struct {
	Title  string // shown in the HUD / on the page
	Prompt string // the director seed for this chapter's regions
	Line   string // what the guide says on entering the chapter
}

// Story is an ordered arc the director walks you through.
type Story struct {
	Title    string
	Chapters []Chapter
	perCh    int // regions per chapter before the page turns
	cur      int // current chapter index
	grown    int // regions grown so far
}

// NewStory builds a story whose chapters each span perChapter regions.
func NewStory(title string, chapters []Chapter, perChapter int) *Story {
	if perChapter < 1 {
		perChapter = 1
	}
	if len(chapters) == 0 {
		chapters = []Chapter{{Title: "The Walk", Prompt: "a calm world", Line: "Let's see where this leads."}}
	}
	return &Story{Title: title, Chapters: chapters, perCh: perChapter}
}

// DefaultStory is a built-in five-act arc from dawn to summit.
func DefaultStory() *Story {
	return NewStory("A Walk Through Worlds", []Chapter{
		{Title: "The Threshold", Prompt: "a quiet meadow at dawn with trees", Line: "We begin where the grass is still wet with dawn."},
		{Title: "Into the Forest", Prompt: "a dark forest at night with fireflies", Line: "The trees close in — watch for the fireflies."},
		{Title: "The Drowned City", Prompt: "a city by water with ruins and reflections", Line: "Something was here once. The water keeps its reflection."},
		{Title: "The Crystal Heart", Prompt: "a cave of glowing crystals", Line: "Down here the walls themselves are alight."},
		{Title: "The Summit", Prompt: "a vast open summit, grand and golden", Line: "And now — the top of the world, and all of it gold."},
	}, 2)
}

// Chapter returns the current chapter.
func (s *Story) Chapter() Chapter { return s.Chapters[s.cur] }

// Prompt is the director seed for the next region (the current chapter's prompt).
func (s *Story) Prompt() string { return s.Chapters[s.cur].Prompt }

// Advance records that a region was grown and turns the page when the current
// chapter has had its share of regions. It returns true and the new chapter when a
// page turns (so the app can sound a beacon and narrate); otherwise false.
func (s *Story) Advance() (bool, Chapter) {
	s.grown++
	if s.cur < len(s.Chapters)-1 && s.grown >= (s.cur+1)*s.perCh {
		s.cur++
		return true, s.Chapters[s.cur]
	}
	return false, s.Chapters[s.cur]
}

// Progress reports the current chapter number (1-based) and the total.
func (s *Story) Progress() (int, int) { return s.cur + 1, len(s.Chapters) }

// Done reports whether the final chapter has been reached.
func (s *Story) Done() bool { return s.cur >= len(s.Chapters)-1 }

// BeaconObjects builds a glowing beacon — a stacked pillar of emissive orbs — to
// mark a chapter threshold at `at` in the chapter's colour.
func BeaconObjects(at raytrace.Vec3, colour raytrace.Vec3) []raytrace.Object {
	emit := colour.Scale(8)
	var out []raytrace.Object
	for i := 0; i < 5; i++ {
		y := 1.0 + float64(i)*1.6
		out = append(out, raytrace.Sphere{
			Center: raytrace.Vec3{X: at.X, Y: y, Z: at.Z},
			Radius: 0.45 - float64(i)*0.05,
			Mat:    raytrace.Material{Color: colour, Emit: emit},
		})
	}
	return out
}

// ChapterColor gives each chapter a distinct beacon colour.
func ChapterColor(index int) raytrace.Vec3 {
	pal := []raytrace.Vec3{
		{X: 1, Y: 0.85, Z: 0.4}, {X: 0.4, Y: 0.9, Z: 0.6}, {X: 0.4, Y: 0.7, Z: 1},
		{X: 0.8, Y: 0.5, Z: 1}, {X: 1, Y: 0.6, Z: 0.4},
	}
	return pal[index%len(pal)]
}
