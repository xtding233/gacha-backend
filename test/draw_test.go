package test

import (
	"testing"

	"github.com/xtding233/gacha-backend/internal/gacha"
)

func TestDrawBounds(t *testing.T) {
	got, err := gacha.Draw(0, gacha.NewSeededRNG(1))
	if err != nil || got {
		t.Fatalf("p=0 should never hit; got=%v err=%v", got, err)
	}
	got, err = gacha.Draw(1, gacha.NewSeededRNG(1))
	if err != nil || !got {
		t.Fatalf("p=1 should always hit; got=%v err=%v", got, err)
	}
	if _, err := gacha.Draw(-0.1, nil); err == nil {
		t.Fatalf("negative p must error")
	}
	if _, err := gacha.Draw(1.1, nil); err == nil {
		t.Fatalf("p>1 must error")
	}
}

func TestDrawStatApprox(t *testing.T) {
	const p = 0.3
	const n = 100000
	rng := gacha.NewSeededRNG(42)
	hit := 0
	for i := 0; i < n; i++ {
		ok, err := gacha.Draw(p, rng)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			hit++
		}
	}
	freq := float64(hit) / float64(n)
	// should be around 0.3
	if diff := freq - p; diff > 0.01 || diff < -0.01 {
		t.Fatalf("freq=%f not close to p=%f", freq, p)
	}
}