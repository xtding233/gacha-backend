package game

import (
	"fmt"
	"strings"
)

// ValidateRaw checks semantic constraints of a RawConfig.
func ValidateRaw(cfg RawConfig) error {
	var errs []string

	// draw.pity
	if cfg.Draw.Pity != nil && *cfg.Draw.Pity <= 0 {
		errs = append(errs, "draw.pity must be >= 1")
	}
	// draw.p_base
	if cfg.Draw.PBase != nil {
		if *cfg.Draw.PBase <= 0 || *cfg.Draw.PBase >= 1 {
			errs = append(errs, "draw.p_base must be in (0,1)")
		}
	}

	// soft
	if cfg.Draw.Soft != nil {
		mode := cfg.Draw.Soft.Mode
		switch mode {
		case "target_ramp":
			// need start_at or start_pct; need target
			if cfg.Draw.Soft.Target == nil {
				errs = append(errs, "draw.soft.target is required for mode=target_ramp")
			} else if *cfg.Draw.Soft.Target <= 0 || *cfg.Draw.Soft.Target >= 1 {
				errs = append(errs, "draw.soft.target must be in (0,1)")
			}
			if cfg.Draw.Soft.StartAt == nil && cfg.Draw.Soft.StartPct == nil {
				errs = append(errs, "draw.soft.start_at or start_pct is required for mode=target_ramp")
			}
		case "per_draw_increment":
			// need start_at; need increment > 0
			if cfg.Draw.Soft.StartAt == nil {
				errs = append(errs, "draw.soft.start_at is required for mode=per_draw_increment")
			}
			if cfg.Draw.Soft.Increment == nil {
				errs = append(errs, "draw.soft.increment is required for mode=per_draw_increment")
			} else if *cfg.Draw.Soft.Increment <= 0 {
				errs = append(errs, "draw.soft.increment must be > 0 for mode=per_draw_increment")
			}
		case "", "none":
			// treat as no soft pity
		default:
			errs = append(errs, "draw.soft.mode must be one of: target_ramp, per_draw_increment, none")
		}

		// start_at/start_pct bounds if present
		if cfg.Draw.Pity != nil && cfg.Draw.Soft.StartAt != nil {
			if *cfg.Draw.Soft.StartAt < 0 || *cfg.Draw.Soft.StartAt >= *cfg.Draw.Pity {
				errs = append(errs, "draw.soft.start_at must satisfy 0 <= start_at < pity")
			}
		}
		if cfg.Draw.Soft.StartPct != nil {
			if *cfg.Draw.Soft.StartPct < 0 || *cfg.Draw.Soft.StartPct > 1 {
				errs = append(errs, "draw.soft.start_pct must be in [0,1]")
			}
		}
	}

	// banner
	if cfg.Banner != nil {
		if len(cfg.Banner.OffProbs) > 0 {
			for i, p := range cfg.Banner.OffProbs {
				if !(p > 0 && p < 1) {
					errs = append(errs, fmt.Sprintf("banner.off_probs[%d] must be in (0,1)", i))
				}
			}
		}
		if cfg.Banner.MaxOff < 0 {
			errs = append(errs, "banner.max_off must be >= 0 (0 means default to len(off_probs))")
		}
	}

	// tokens (optional)
	if cfg.Tokens != nil {
		if cfg.Tokens.PerDraw != nil && *cfg.Tokens.PerDraw < 0 {
			errs = append(errs, "tokens.per_draw must be >= 0")
		}
		if cfg.Tokens.PerTenDraw != nil && *cfg.Tokens.PerTenDraw < 0 {
			errs = append(errs, "tokens.per_ten_draw must be >= 0")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}
