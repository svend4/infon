# Heraldry — blazon → pseudo-image

`pkg/blazon` compiles a (simplified) **heraldic blazon** — the medieval language for
describing a coat of arms — straight into a `pkg/pseudo` spec that renders in the
terminal. Heraldry was an **ultra-low-bandwidth visual-identity system**: a knight
recognizable at a distance from a few symbols on a colored field. That is exactly
TVCP's problem, a thousand years early — so a blazon *is* a pseudo-image.

```go
spec := blazon.Parse("Gules, three lions Or")   // red field, three gold lions
frame, _ := spec.Frame()                        // -> terminal.Frame
img, _   := spec.Image(384, 192)                // -> image.Image
```

```bash
go run ./cmd/blazondemo ./_blazon   # renders example arms (England, France, ...) to PNG
```

### Grammar (forgiving keyword scan)
- **Field**: the first *tincture* in the blazon colors the whole shield.
- **Tinctures**: `or`(gold) `argent`(white) `gules`(red) `azure`(blue) `vert`(green)
  `sable`(black) `purpure`(purple) `tenne`(orange) `sanguine` `murrey` — plus plain
  color names. In heraldry the tincture *follows* the noun: "a chief **Argent**".
- **Ordinaries** (painted onto the field): `chief base fess pale bend cross saltire
  bordure chevron`.
- **Charges** (placed as sigils, singular or plural): `sun moon/crescent star/mullet
  cross heart rose/flower fleur anchor key sword crown tower/castle lion ship/boat
  tree mountain fish shield flag ...`.
- **Number**: `a/an/one … two three four five six`.
- **Position**: `in chief | base | dexter | sinister | fess | honour | nombril`.

Example: `Gules, on a chief Argent three mullets Sable` → red field, white band across
the top, three black stars on it. In the terminal the charges are **real glyphs**
(`⚜ ♜ ★ ⚓ ⚔ ♌`); the PNG preview shows them as colored cells.

This is Tier 0 of the project's "ladder of ideas": an ancient compression codec
(symbols for identity) compiled into the modern open protocol.
