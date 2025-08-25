package gacha

import (
	cryptoRand "crypto/rand"
	"encoding/binary"
	"math/rand/v2"
)

// RandomSource abstract

type RandomSource interface {
	Float64() float64 // [0, 1] 
}

// crypto random : default generation method
type cryptoRNG struct{}

func (cryptoRNG) Float64() float64 {
	// Read 53bit random => [0, 1]
	var buf [8]byte
	if _, err := cryptoRand.Read(buf[:]); err != nil {
		// backto math / rand/ v2
		return rand.Float64()
	}

	// max 53
	u := binary.BigEndian.Uint64(buf[:]) >> 11 // 53 bits
	return float64(u) / (1 << 53)
}


func DefaultRNG() RandomSource { return cryptoRNG{} }

// Replicable RNG (e.g. Monte Carlo)
type seededRNG struct { r *rand.Rand }

func NewSeededRNG(seed uint64) RandomSource {
	return &seededRNG{r: rand.New(rand.NewPCG(seed, 0))}
}

func (s *seededRNG) Float64() float64 { return s.r.Float64()}