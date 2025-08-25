package test

import (
	"testing"

	"github.com/xtding233/gacha-backend/internal/gacha"
)

func TestPitySystem(t *testing.T) {
	rng := gacha.NewSeededRNG(42)
	ps := gacha.NewPitySystem(10, rng)

	// first 9 draws with p=0 should not hit
	for i := 0; i < 9; i++ {
		hit, err := ps.Draw(0.0)
		if err != nil {
			t.Fatal(err)
		}
		if hit {
			t.Fatalf("should not hit before pity, i=%d", i)
		}
	}
	// the 10th draw should be guaranteed by pity
	hit, err := ps.Draw(0.0)
	if err != nil {
		t.Fatal(err)
	}
	if !hit {
		t.Fatalf("expected pity hit at 10th draw")
	}
	if ps.Count != 0 {
		t.Fatalf("count should reset after pity hit; got %d", ps.Count)
	}

}