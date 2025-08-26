package gacha

import "errors"

// Easing specifies how the probability ramps up as we approach pity.
type Easing string

const (
	EaseLinear        Easing = "linear"
	EaseOutQuad       Easing = "easeOutQuad"
	EaseInOutCubic    Easing = "easeInOutCubic"
)

var ErrSoftPityConfig = errors.New("invalid soft pity config")

// SoftPityConfig defines the ramp behavior before the hard pity.
// Example: Pity=90, StartAt=74, Target=0.5 → from draw #74 up to #89, p ramps to 0.5
type SoftPityConfig struct {
	Pity       int     // hard pity threshold (same as PitySystem.Pity)
	StartAt    int     // start draw index (since last hit) to begin ramp, e.g., 74
	TargetProb float64 // probability at draw (Pity-1), must be in (0,1)
	Easing     Easing  // easing function
}

// normalize validates and adjusts StartAt; returns error if invalid.
func (c *SoftPityConfig) normalize() error {
	if c.Pity <= 1 {
		return ErrSoftPityConfig
	}
	if c.TargetProb <= 0 || c.TargetProb >= 1 {
		return ErrSoftPityConfig
	}
	if c.StartAt < 0 {
		c.StartAt = 0
	}
	// Ramp ends at (Pity-1). StartAt must be < (Pity-1) to have room to ramp.
	if c.StartAt >= c.Pity-1 {
		return ErrSoftPityConfig
	}
	if c.Easing == "" {
		c.Easing = EaseLinear
	}
	return nil
}

// SoftPitySystem extends PitySystem with a soft ramp before hard pity.
type SoftPitySystem struct {
	PitySystem
	Soft *SoftPityConfig
}

// NewSoftPitySystem creates a pity system with an optional soft ramp.
// If soft is nil → behaves like plain hard pity.
func NewSoftPitySystem(pity int, soft *SoftPityConfig, rng RandomSource) (*SoftPitySystem, error) {
	if rng == nil {
		rng = DefaultRNG()
	}
	base := PitySystem{Pity: pity, RNG: rng}
	if soft != nil {
		soft.Pity = pity
		if err := soft.normalize(); err != nil {
			return nil, err
		}
	}
	return &SoftPitySystem{PitySystem: base, Soft: soft}, nil
}

// effectiveProb computes the actual probability this draw should use:
// - If Count+1 >= Pity: return 1 (hard pity).
// - Else if soft ramp is configured and Count >= StartAt: ramp p toward TargetProb at (Pity-1).
// - Else: return base p.
func (s *SoftPitySystem) effectiveProb(pBase float64) float64 {
	// hard pity
	if s.Count+1 >= s.Pity {
		return 1.0
	}
	// no soft pity configured → pure base probability
	if s.Soft == nil {
		return pBase
	}
	// ramp only from StartAt up to (Pity-1)
	if s.Count < s.Soft.StartAt {
		return pBase
	}
	end := s.Pity - 1
	// progress t in [0,1], inclusive at end (Count == end)
	// Example: StartAt=74, end=89 → length = 16 draws (74..89)
	length := float64(end - s.Soft.StartAt)
	if length <= 0 {
		return pBase
	}
	t := float64(s.Count - s.Soft.StartAt) / length
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	// easing
	switch s.Soft.Easing {
	case EaseOutQuad:
		// f(t) = 1 - (1 - t)^2
		t = 1 - (1-t)*(1-t)
	case EaseInOutCubic:
		// smoother curve: accelerate then decelerate
		if t < 0.5 {
			t = 4 * t * t * t
		} else {
			t = 1 - (-2*t+2)*(-2*t+2)*(-2*t+2)/2
		}
	default:
		// linear
	}
	// interpolate from base to target
	p := pBase + (s.Soft.TargetProb-pBase)*t
	// clamp to [0,1)
	if p < 0 {
		p = 0
	}
	if p > 0.999999999999 { // keep < 1 to avoid pre-hard-pity guarantee
		p = 0.999999999999
	}
	return p
}

// Draw performs one draw using the soft/hard pity rules.
// On hit → Count resets; else → Count++.
func (s *SoftPitySystem) Draw(pBase float64) (bool, error) {
	// if hard pity triggers this draw, short-circuit
	if s.Count+1 >= s.Pity {
		s.Count = 0
		return true, nil
	}
	// compute effective probability with soft ramp
	pEff := s.effectiveProb(pBase)
	hit, err := Draw(pEff, s.RNG)
	if err != nil {
		return false, err
	}
	if hit {
		s.Count = 0
	} else {
		s.Count++
	}
	return hit, nil
}
