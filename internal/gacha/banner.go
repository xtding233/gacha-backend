package gacha

import "math"

// BannerOutcome reports one draw's result under banner rules.

type BannerOutcome struct {
	Hit bool // true if high-rarity occurred this draw
	IsUp bool // true if featured (UP) when Hit == true; false if off-banner
	Count int // draws since last Hit after this draw (from SoftPitySystem)
	GuaranteedNext bool // whether the next Hit is forced UP
	OffStreak int // consecutive off-banner streak after this draw
}

// BannerSystem composes soft/hard pity with multi-off logic
// - SoftPity decides whether a Hit occurs (soft ramp + hard pity).
// - On Hit, if GuaranteedNext is true => force UP and clear it, OffStreak = 0.
// - Otherwise, pick an off probability based on current OffStreak index:
// idx = min(OffStreak, len(OffProbs) - 1)
// off ~ Bernoulli(OffProbs[idx])
// if off: OffStreak++, and if OffStreak >= MaxOff => GuaranteedNext = true
// else: IsUp = true, OffStreak = 0
// Notes:
// - A Hit (UP of off) always resets SoftPity.Count to 0; misses increment Count.
// - MaxOff controls how many consecutive offs are allowed before forcing UP on the next Hit.
// If MaxOff <= 0, it defaults to len(OffProbs).

type BannerSystem struct {
	SoftPity *SoftPitySystem
	OffProbs []float64 // per-off probabilities; last value repeats if OffStreak exceeds len-1
	MaxOff int // threshold for consecutive offs before guarantee flips
	GuaranteedNext bool
	OffStreak int
}

// NewBannerSystem initializes a BannerSystem.
// If maxOff <= 0 => maxOff = len(offProbs). If offProbs empty => default to [0.5]

func NewBannerSystem(soft *SoftPitySystem, offProbs []float64, maxOff int) *BannerSystem {
	if len(offProbs) == 0 {
		offProbs = []float64{0.5}
	}

	// clamp probs to (0,1)
	clamped := make([]float64, len(offProbs))
	for i, p := range offProbs {
		if !(p > 0 && p < 1) {
			p = 0.5
		}
		clamped[i] = p
	}
	if maxOff <= 0 {
		maxOff = len(clamped)
	}
	return &BannerSystem{
		SoftPity: soft,
		OffProbs: clamped,
		MaxOff: maxOff,
		GuaranteedNext: false,
		OffStreak: 0,
	}
}

// currentOffProb returns the probability of going off-banner at the current streak
func (b *BannerSystem) currentOffProb() float64 {
	if len(b.OffProbs) == 0 {
		return 0.5
	}
	idx := b.OffStreak
	if idx < 0 {
		idx = 0
	}
	if idx >= len(b.OffProbs) {
		idx = len(b.OffProbs) - 1 // repeat the last value
	}
	// keep strictly within (0,1) to avoid degenracy
	p := b.OffProbs[idx]
	if p <= 0 {
		p = math.SmallestNonzeroFloat64
	}
	if p >= 1 {
		p = 1 - 1e-12
	}
	return p
}

// Draw performs one banner draw using base probability pBase for early segment.
// The SoftPity system determines Hit; upon Hit, this function decides UP vs off-banner.
func (b *BannerSystem) Draw(pBase float64) (BannerOutcome, error) {
	// 1) decide hit via soft/hard pity
	hit, err := b.SoftPity.Draw(pBase)
	if err != nil {
		return BannerOutcome{}, err
	}

	// miss: nothing to do besides forwarding states
	if !hit {
		return BannerOutcome{
			Hit: false,
			IsUp: false,
			Count: b.SoftPity.Count,
			GuaranteedNext: b.GuaranteedNext,
			OffStreak: b.OffStreak,
		}, nil
	}

	// 2) on hit: guarantee or 50/50-like decision chain
	if b.GuaranteedNext {
		b.GuaranteedNext = false
		b.OffStreak = 0
		return BannerOutcome{
			Hit: true,
			IsUp: true,
			Count: b.SoftPity.Count, // 0
			GuaranteedNext: b.GuaranteedNext,
			OffStreak: b.OffStreak,
		}, nil
	}

	// decide off vs up using per-streak probability
	offProbs := b.currentOffProb()
	off, derr := Draw(offProbs, b.SoftPity.RNG)
	if derr != nil {
		return BannerOutcome{}, derr
	}
	if off {
		b.OffStreak++
		// after reaching MaxOff consecutive offs, flip guarantee for the next hit
		if b.OffStreak > b.MaxOff {
			b.GuaranteedNext = true
		}
		return BannerOutcome{
			Hit: true,
			IsUp: false,
			Count: b.SoftPity.Count,
			GuaranteedNext: b.GuaranteedNext,
			OffStreak: b.OffStreak,
		}, nil
	}

	// UP
	b.OffStreak = 0
	b.GuaranteedNext = false
	return BannerOutcome{
		Hit: true,
		IsUp: true,
		Count: b.SoftPity.Count, //0
		GuaranteedNext: b.GuaranteedNext,
		OffStreak: b.OffStreak,
	}, nil
}