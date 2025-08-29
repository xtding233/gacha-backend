package game

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Paths helper for default/game/pool files.
type Paths struct {
	BaseDir string // base directory, e.g., /opt/app/config
}

func (p Paths) DefaultPath() string {
	return filepath.Join(p.BaseDir, "games", "default.yaml")
}
func (p Paths) GamePath(game string) string {
	return filepath.Join(p.BaseDir, "games", game+".yaml")
}
func (p Paths) PoolPath(game, pool string) string {
	return filepath.Join(p.BaseDir, "games", game, "pools", pool+".yaml")
}

// Loader reads YAML configs and merges default → game → pool.
type Loader struct {
	paths Paths

	mu    sync.RWMutex
	cache map[string]RawConfig // key: "game" or "game/pool" or "$default"
}

// NewLoader creates a config loader with the given base directory.
func NewLoader(baseDir string) *Loader {
	return &Loader{
		paths: Paths{BaseDir: baseDir},
		cache: make(map[string]RawConfig),
	}
}

// LoadMerged loads and merges default → game → pool (pool optional).
// It returns the merged RawConfig (without normalization).
func (l *Loader) LoadMerged(game, pool string) (RawConfig, error) {
	l.mu.RLock()
	if pool != "" {
		if cfg, ok := l.cache[game+"/"+pool]; ok {
			l.mu.RUnlock()
			return cfg, nil
		}
	}
	if cfg, ok := l.cache["$default"]; ok && pool == "" {
		// allow returning just default if caller explicitly wants default only
		_ = cfg
	}
	l.mu.RUnlock()

	// Read files from disk
	defCfg, err := readYAML(l.paths.DefaultPath())
	if err != nil {
		return RawConfig{}, fmt.Errorf("read default: %w", err)
	}
	gameCfg, _ := readYAML(l.paths.GamePath(game)) // game file may not exist
	var poolCfg RawConfig
	if pool != "" {
		poolCfg, _ = readYAML(l.paths.PoolPath(game, pool)) // pool file optional
	}

	// Merge: default <- game <- pool
	merged := defCfg
	merged = mergeRaw(merged, gameCfg)
	merged = mergeRaw(merged, poolCfg)

	// Cache
	l.mu.Lock()
	// cache game-level merged too (handy if no pool next time)
	l.cache[game] = mergeRaw(defCfg, gameCfg)
	if pool != "" {
		l.cache[game+"/"+pool] = merged
	}
	// keep a copy of default (optional)
	l.cache["$default"] = defCfg
	l.mu.Unlock()

	return merged, nil
}

// Invalidate clears loader's cache. Call after hot-reload detects changes.
func (l *Loader) Invalidate() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cache = make(map[string]RawConfig)
}

// readYAML loads a YAML file into RawConfig. Missing files return zero cfg, no error.
func readYAML(path string) (RawConfig, error) {
	var cfg RawConfig
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RawConfig{}, nil
		}
		return RawConfig{}, err
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return RawConfig{}, err
	}
	return cfg, nil
}

// mergeRaw performs a deep merge: 'b' overrides 'a' where non-zero/non-nil.
// For slices (e.g., OffProbs), 'b' replaces 'a' if provided.
func mergeRaw(a, b RawConfig) RawConfig {
	out := a

	// top-level scalars
	if b.Version != "" {
		out.Version = b.Version
	}
	if b.Notes != "" {
		out.Notes = b.Notes
	}

	// draw
	if out.Draw.PBase == nil && b.Draw.PBase != nil {
		out.Draw.PBase = b.Draw.PBase
	}
	if out.Draw.Pity == nil && b.Draw.Pity != nil {
		out.Draw.Pity = b.Draw.Pity
	}
	// soft
	switch {
	case out.Draw.Soft == nil && b.Draw.Soft != nil:
		softCopy := *b.Draw.Soft
		out.Draw.Soft = &softCopy
	case out.Draw.Soft != nil && b.Draw.Soft != nil:
		if b.Draw.Soft.Mode != "" {
			out.Draw.Soft.Mode = b.Draw.Soft.Mode
		}
		if out.Draw.Soft.StartAt == nil && b.Draw.Soft.StartAt != nil {
			out.Draw.Soft.StartAt = b.Draw.Soft.StartAt
		}
		if out.Draw.Soft.StartPct == nil && b.Draw.Soft.StartPct != nil {
			out.Draw.Soft.StartPct = b.Draw.Soft.StartPct
		}
		if out.Draw.Soft.Target == nil && b.Draw.Soft.Target != nil {
			out.Draw.Soft.Target = b.Draw.Soft.Target
		}
		if out.Draw.Soft.Increment == nil && b.Draw.Soft.Increment != nil {
			out.Draw.Soft.Increment = b.Draw.Soft.Increment
		}
		if b.Draw.Soft.Easing != "" {
			out.Draw.Soft.Easing = b.Draw.Soft.Easing
		}
	}

	// banner
	switch {
	case out.Banner == nil && b.Banner != nil:
		c := *b.Banner
		out.Banner = &c
	case out.Banner != nil && b.Banner != nil:
		if len(b.Banner.OffProbs) > 0 {
			out.Banner.OffProbs = append([]float64(nil), b.Banner.OffProbs...)
		}
		if b.Banner.MaxOff != 0 {
			out.Banner.MaxOff = b.Banner.MaxOff
		}
		// special windows left as-is; extend if you add them to schema
	}

	// tokens
	switch {
	case out.Tokens == nil && b.Tokens != nil:
		c := *b.Tokens
		out.Tokens = &c
	case out.Tokens != nil && b.Tokens != nil:
		if out.Tokens.PerDraw == nil && b.Tokens.PerDraw != nil {
			out.Tokens.PerDraw = b.Tokens.PerDraw
		}
		if out.Tokens.PerTenDraw == nil && b.Tokens.PerTenDraw != nil {
			out.Tokens.PerTenDraw = b.Tokens.PerTenDraw
		}
	}

	return out
}
