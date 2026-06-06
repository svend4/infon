// Package alttext realizes the project's accessibility isomorphism: because a
// pseudo-image IS a small structured description, every spec can emit its own
// human-readable alt-text. Describe turns any pseudo.Spec into one sentence — the
// same artifact a sighted user sees and a screen reader hears.
package alttext

import (
	"fmt"
	"sort"
	"strings"

	"github.com/svend4/infon/pkg/pseudo"
)

func fmtName(f pseudo.Format) string {
	if f == "" {
		return "grid"
	}
	return string(f)
}

func humanList(items []string) string {
	switch len(items) {
	case 0:
		return "nothing"
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
	}
}

func topColors(rows [][]string, k int) []string {
	count := map[string]int{}
	for _, row := range rows {
		for _, tok := range row {
			if tok != "" {
				count[tok]++
			}
		}
	}
	names := make([]string, 0, len(count))
	for n := range count {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		if count[names[i]] != count[names[j]] {
			return count[names[i]] > count[names[j]]
		}
		return names[i] < names[j]
	})
	if len(names) > k {
		names = names[:k]
	}
	return names
}

func countNonEmpty(rows [][]string) int {
	n := 0
	for _, row := range rows {
		for _, t := range row {
			if t != "" {
				n++
			}
		}
	}
	return n
}

func gridW(rows [][]string) int {
	w := 0
	for _, r := range rows {
		if len(r) > w {
			w = len(r)
		}
	}
	return w
}

// Describe returns a one-sentence alt-text for the spec.
func Describe(s pseudo.Spec) string {
	title := ""
	if strings.TrimSpace(s.Title) != "" {
		title = fmt.Sprintf(" titled %q", s.Title)
	}
	switch s.Format {
	case pseudo.FormatGrid, pseudo.FormatPixels, "":
		g := s.Grid
		if g == nil {
			g = s.Pixels
		}
		if g == nil || len(g.Rows) == 0 {
			return fmt.Sprintf("a %s pseudo-image%s", fmtName(s.Format), title)
		}
		return fmt.Sprintf("a %dx%d color-grid image%s, mostly %s",
			s.Cols, s.Rows, title, humanList(topColors(g.Rows, 3)))
	case pseudo.FormatMarks:
		if s.Marks == nil {
			return "an empty marks image" + title
		}
		return fmt.Sprintf("a marks image%s: %d sub-cell glyphs on a %dx%d field",
			title, countNonEmpty(s.Marks.Rows), gridW(s.Marks.Rows), len(s.Marks.Rows))
	case pseudo.FormatSigils:
		if s.Sigils == nil || len(s.Sigils.Items) == 0 {
			return "an empty sigil scene" + title
		}
		names := make([]string, 0, len(s.Sigils.Items))
		for _, it := range s.Sigils.Items {
			names = append(names, it.Name)
		}
		return fmt.Sprintf("a sigil scene%s with %s", title, humanList(names))
	case pseudo.FormatGlyphs:
		lines := 0
		if s.Glyphs != nil {
			lines = len(s.Glyphs.Rows)
		}
		return fmt.Sprintf("glyph art%s, %d lines", title, lines)
	default:
		return fmt.Sprintf("a %s pseudo-image%s", fmtName(s.Format), title)
	}
}
