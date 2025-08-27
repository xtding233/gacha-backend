package gacha

import (
	"math"
	"sort"
)

// TrialGoal selects what the simulation measures per trial.
type TrialGoal string

const (
	// Draws until the first high-rarity Hit (ignores UP/off layer).
	GoalFirstHit TrialGoal = "first_hit"
	// Draws until the first UP (respects multi-off/guarantee banner rules).
	GoalFirstUP  TrialGoal = "first_up"
	// Given a fixed budget N, count number of Hits or UPs (depending on Banner!=nil).
	GoalFixedBudget TrialGoal = "fixed_budget"
)

// SimParams describes the mechanics for one simulation run.
type SimParams struct {
	// Base probability far from pity (the "p" you pass elsewhere).
	PBase float64

	// Soft/Hard pity configuration.
	Pity       int
	StartAt    *int     // optional: start index for ramp; ignored if nil
	StartPct   *float64 // optional: start percentage (0..1); unused if StartAt is provided
	TargetProb *float64 // optional: target probability at (Pity-1); if nil -> no soft ramp
	Easing     string   // "linear", "easeOutQuad", "easeInOutCubic"; default linear
	Cushion    int      // carry-over draws since last Hit when entering this pool

	// Banner multi-off configuration. If OffProbs is empty, banner is disabled.
	OffProbs []float64 // e.g., [0.5] or [0.5,0.4,0.3]
	MaxOff   int       // <=0 means defaults to len(OffProbs)
}

// SimBudget controls the number of draws used in GoalFixedBudget.
type SimBudget struct {
	NumDraws int // number of draws in one trial
}

// Stats summarizes simulation results.
type Stats struct {
	Mean   float64
	Var    float64
	StdDev float64
	P50    float64
	P90    float64
	P99    float64
	// Optional: raw samples if caller needs histograms/exports
	Samples []int `json:"-"`
}

// calcStats computes mean/variance/percentiles for integer samples.
func calcStats(xs []int) Stats {
	n := len(xs)
	if n == 0 {
		return Stats{}
	}
	// mean
	var sum float64
	for _, v := range xs {
		sum += float64(v)
	}
	mean := sum / float64(n)

	// variance (population)
	var acc float64
	for _, v := range xs {
		d := float64(v) - mean
		acc += d * d
	}
	variance := acc / float64(n)
	stddev := math.Sqrt(variance)

	// percentiles
	cp := append([]int(nil), xs...)
	sort.Ints(cp)
	percentile := func(p float64) float64 {
		if n == 1 {
			return float64(cp[0])
		}
		if p <= 0 {
			return float64(cp[0])
		}
		if p >= 1 {
			return float64(cp[n-1])
		}
		pos := p * float64(n-1)
		i := int(math.Floor(pos))
		f := pos - float64(i)
		if i+1 >= n {
			return float64(cp[i])
		}
		return float64(cp[i])*(1-f) + float64(cp[i+1])*f
	}

	return Stats{
		Mean:    mean,
		Var:     variance,
		StdDev:  stddev,
		P50:     percentile(0.50),
		P90:     percentile(0.90),
		P99:     percentile(0.99),
		Samples: xs,
	}
}

// newSoft constructs a fresh SoftPitySystem using SimParams.
func newSoft(p SimParams) (*SoftPitySystem, error) {
	var cfg *SoftPityConfig
	if p.TargetProb != nil && (p.StartAt != nil || p.StartPct != nil) {
		startAt := 0
		if p.StartAt != nil {
			startAt = *p.StartAt
		} else {
			sp := *p.StartPct
			if sp < 0 {
				sp = 0
			}
			if sp > 1 {
				sp = 1
			}
			startAt = int(math.Ceil(sp * float64(p.Pity)))
			if startAt >= p.Pity {
				startAt = p.Pity - 1
			}
		}
		easing := Easing(p.Easing)
		if easing == "" {
			easing = EaseLinear
		}
		cfg = &SoftPityConfig{
			Pity:       p.Pity,
			StartAt:    startAt,
			TargetProb: *p.TargetProb,
			Easing:     easing,
		}
	}
	sp, err := NewSoftPitySystem(p.Pity, cfg, nil)
	if err != nil {
		return nil, err
	}
	// apply cushion as initial count
	c := p.Cushion
	if c < 0 {
		c = 0
	}
	if c >= p.Pity {
		c = p.Pity - 1
	}
	sp.Count = c
	return sp, nil
}

// newBanner wraps a fresh BannerSystem if OffProbs provided; else returns nil.
func newBanner(sp *SoftPitySystem, p SimParams) *BannerSystem {
	if len(p.OffProbs) == 0 {
		return nil
	}
	return NewBannerSystem(sp, p.OffProbs, p.MaxOff)
}

// simulateOne returns the primary metric for one trial depending on the goal.
// - GoalFirstHit: number of draws until first Hit
// - GoalFirstUP:  number of draws until first UP
// - GoalFixedBudget: number of Hits (if banner==nil) or UPs (if banner!=nil) within budget.NumDraws
func simulateOne(p SimParams, goal TrialGoal, budget *SimBudget) (int, error) {
	sp, err := newSoft(p)
	if err != nil {
		return 0, err
	}
	banner := newBanner(sp, p)

	switch goal {
	case GoalFirstHit:
		draws := 0
		for {
			draws++
			hit, err := sp.Draw(p.PBase)
			if err != nil {
				return 0, err
			}
			if hit {
				return draws, nil
			}
		}

	case GoalFirstUP:
		if banner == nil {
			// If banner layer is not configured, fall back to first Hit.
			draws := 0
			for {
				draws++
				hit, err := sp.Draw(p.PBase)
				if err != nil {
					return 0, err
				}
				if hit {
					return draws, nil
				}
			}
		}
		draws := 0
		for {
			draws++
			out, err := banner.Draw(p.PBase)
			if err != nil {
				return 0, err
			}
			if out.Hit && out.IsUp {
				return draws, nil
			}
		}

	case GoalFixedBudget:
		if budget == nil || budget.NumDraws <= 0 {
			return 0, nil
		}
		count := 0
		for i := 0; i < budget.NumDraws; i++ {
			if banner == nil {
				hit, err := sp.Draw(p.PBase)
				if err != nil {
					return 0, err
				}
				if hit {
					count++
				}
			} else {
				out, err := banner.Draw(p.PBase)
				if err != nil {
					return 0, err
				}
				if out.Hit && out.IsUp {
					count++
				}
			}
		}
		return count, nil
	}

	return 0, nil
}

// RunMonteCarlo repeats trials and returns summary stats.
// goal determines what metric is recorded per trial.
func RunMonteCarlo(p SimParams, goal TrialGoal, trials int, budget *SimBudget) (Stats, error) {
	if trials <= 0 {
		return Stats{}, nil
	}
	samples := make([]int, trials)
	for i := 0; i < trials; i++ {
		v, err := simulateOne(p, goal, budget)
		if err != nil {
			return Stats{}, err
		}
		samples[i] = v
	}
	return calcStats(samples), nil
}
