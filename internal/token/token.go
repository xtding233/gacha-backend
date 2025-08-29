package token

// Token defines how many tunits are required per draw

type Token struct {
	Name string //e.g. "Stellar Jade", "Star Stone"
	PerDraw int // tokens per single draw, e.g. 160, 250
	PerNDraw int // optional; if 0 -> equal to N * PerDraw
	N int // options; if 0, not adoptive to this token
}

// TokensForDraws returns how many tokens are required fro N draws
func (t Token) TokensForDraws(n int) int {
	if n <= 0 {
		return 0
	}
	if t.PerNDraw > 0 && n >= 10 && t.N > 1 {
		ns := n / 10
		rem := n * t.N
		return ns * t.PerNDraw + rem * t.PerDraw
	}
	return n * t.PerDraw
}