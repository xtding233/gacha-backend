// types.go
package game

// Raw config loaded from YAML; mirrors your schema.
type RawConfig struct {
	Version string          `yaml:"version"`
	Draw    DrawConfig      `yaml:"draw"`
	Banner  *BannerConfig   `yaml:"banner,omitempty"`
	Tokens  *TokenConfig    `yaml:"tokens,omitempty"`
	Notes   string          `yaml:"notes,omitempty"`
}

type DrawConfig struct {
	PBase *float64 `yaml:"p_base"`
	Pity  *int     `yaml:"pity"`
	Soft  *SoftCfg `yaml:"soft,omitempty"`
}
type SoftCfg struct {
	Mode       string   `yaml:"mode"` // "target_ramp" | "per_draw_increment"
	StartAt    *int     `yaml:"start_at,omitempty"`
	StartPct   *float64 `yaml:"start_pct,omitempty"`
	Target     *float64 `yaml:"target,omitempty"`
	Increment  *float64 `yaml:"increment,omitempty"` // for per_draw_increment
	Easing     string   `yaml:"easing,omitempty"`
}
type BannerConfig struct {
	OffProbs []float64 `yaml:"off_probs"`
	MaxOff   int       `yaml:"max_off"`
	// optional special rules...
}
type TokenConfig struct {
	PerDraw    *int `yaml:"per_draw"`
	PerTenDraw *int `yaml:"per_ten_draw"`
}

// Normalized engine params used by internal/gacha.
type EngineParams struct {
	PBase     float64
	Pity      int
	SoftMode  string
	StartAt   *int
	StartPct  *float64
	Target    *float64
	Increment *float64
	Easing    string
	OffProbs  []float64
	MaxOff    int
	Cushion   int
	Version   string // effective config version for tracing
}
