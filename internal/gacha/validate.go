package gacha

import (
	"math"
)


func validateProb(p float64) error {
	if math.IsNaN(p) || math.IsInf(p, 0) {
		return ErrInvalidProb
	}
	if p < 0 || p > 1 {
		return ErrInvalidProb
	}
	return nil
}