// Command vocabdemo shows the plural, localizable vocabulary (pkg/vocab): one
// canonical scene spoken in several languages. A Russian scene resolves to
// canonical tokens the renderer understands, and those re-localize to Spanish and
// German — the picture is the same, the words are anyone's.
//
//	go run ./cmd/vocabdemo
package main

import (
	"fmt"

	"github.com/svend4/infon/pkg/vocab"
)

func main() {
	v := vocab.Default()
	ru := []string{"солнце", "гора", "дом", "дерево", "лодка"}
	canon := v.ResolveAll("sigil", "ru", ru)

	localize := func(loc string, canon []string) []string {
		out := make([]string, len(canon))
		for i, c := range canon {
			out[i], _ = v.Localize("sigil", loc, c)
		}
		return out
	}
	es := localize("es", canon)
	de := localize("de", canon)

	fmt.Println("one scene, many languages (sigil namespace):")
	fmt.Printf("%-12s %-10s %-12s %-10s\n", "ru", "canonical", "es", "de")
	for i := range canon {
		fmt.Printf("%-12s %-10s %-12s %-10s\n", ru[i], canon[i], es[i], de[i])
	}
	fmt.Printf("\ndefined locales for 'sigil': %v\n", v.Locales("sigil"))
	fmt.Println("the renderer only ever sees the canonical column; the rest is plurality.")
}
