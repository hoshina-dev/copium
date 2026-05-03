package idgen_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hoshina-dev/copium/internal/idgen"
)

func TestUUID_NewProducesUnique(t *testing.T) {
	g := idgen.UUID{}
	a := g.New()
	b := g.New()
	if a == uuid.Nil || b == uuid.Nil {
		t.Fatal("expected non-nil uuids")
	}
	if a == b {
		t.Fatal("expected unique uuids")
	}
}

func TestStatic_ReturnsInOrderThenWraps(t *testing.T) {
	id1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	id2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	g := idgen.NewStatic(id1, id2)
	if g.New() != id1 {
		t.Fatal("first must be id1")
	}
	if g.New() != id2 {
		t.Fatal("second must be id2")
	}
	if g.New() != id1 {
		t.Fatal("must wrap around")
	}
}
