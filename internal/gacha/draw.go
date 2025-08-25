package gacha

import "errors"

var ErrInvalidProb = errors.New("invalid probability p; must be 0..1")

// Draw under p, return if it is hit
// p <=0 => no hit. p>= 1 => must hit. otherwise, rng.Float64() < p

func Draw(p float64, rng RandomSource) (bool, error) {
	if err := validateProb(p); err != nil {
		return false, err
	}
	if p <= 0 {
		return false, nil
	}
	if p >= 1{
		return true, nil
	}
	if rng == nil {
		rng = DefaultRNG()
	}
	return rng.Float64() < p, nil
}