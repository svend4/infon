package arena_test

import (
	"testing"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/tangram"
)

func TestSummonCreature(t *testing.T) {
	u, name, ok := arena.Summon(tangram.Cat(), 0, 2, 3)
	if !ok || name != "cat" {
		t.Fatalf("cat should summon: ok=%v name=%s", ok, name)
	}
	if arena.Bestiary[u.Kind].Name != "cat" {
		t.Errorf("summoned kind %q, want cat", arena.Bestiary[u.Kind].Name)
	}
}

func TestSummonPoseInvariant(t *testing.T) {
	if _, name, ok := arena.Summon(tangram.Cat().MirrorX(), 1, 0, 0); !ok || name != "cat" {
		t.Errorf("mirrored cat should still summon a cat: ok=%v name=%s", ok, name)
	}
}

func TestNonCreatureDoesNotSummon(t *testing.T) {
	if _, name, ok := arena.Summon(tangram.House(), 0, 0, 0); ok {
		t.Errorf("house is not a creature, should not summon (got %s)", name)
	}
}
