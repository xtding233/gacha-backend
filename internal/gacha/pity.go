package gacha

// PitySystem handles a "hard pity": after reaching the threshold, the next draw is guaranteed

type PitySystem struct {
	Pity int // threshold count before guaranteed hit
	Count int // number of draws since last hit
	RNG RandomSource // random source for probability count 
}

// NewPitySystem creates a new hard pity system with given threshold and RNG
func NewPitySystem(pity int, rng RandomSource) *PitySystem {
	if rng == nil {
		rng = DefaultRNG()
	}
	return &PitySystem{Pity: pity, RNG: rng}
}

// Draw performs one draw with proability p
// - If the next draw reaches the pity threshold, it is guaranted to hit
// - Otherwise, it uses probability p.
// - On hit, Count resets to 0; otherwise, Count increments
func (ps *PitySystem) Draw(p float64) (bool, error) {
	if ps.Pity <= 0 {
		// invalid pity threshold -> fallback to normal Draw
		return Draw(p, ps.RNG)
	}

	// check if this draw will trigger pity
	if ps.Count+1 >= ps.Pity {
		ps.Count = 0
		return true, nil
	}

	hit, err := Draw(p, ps.RNG)
	if err != nil {
		return false, err
	}

	if hit {
		ps.Count = 0
	} else {
		ps.Count ++ 
	}
	return hit, nil
}