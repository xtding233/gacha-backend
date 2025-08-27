package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/xtding233/gacha-backend/internal/gacha"
)

//
// ---------- Response DTOs ----------
//

// Plain pity/no-pity N-draw response (no UP/off logic)
type nResp struct {
	Hits  []bool `json:"hits"`
	Count int    `json:"count,omitempty"`
	Err   string `json:"err,omitempty"`
}

// Per-draw banner result (hit + whether it's UP)
type bannerItem struct {
	Hit  bool `json:"hit"`
	IsUp bool `json:"isUp"`
}

// Banner N-draw response (soft/hard pity + multi-off logic)
type bannerResp struct {
	Results        []bannerItem `json:"results"`
	Count          int          `json:"count,omitempty"`
	GuaranteedNext bool         `json:"guaranteedNext"`
	Err            string       `json:"err,omitempty"`
}

//
// ---------- Global state (demo-grade) ----------
//
// NOTE: In production, avoid globals; keep per-user/session state instead.

var (
	mu        sync.Mutex
	softSys   *gacha.SoftPitySystem // shared soft/hard pity system
	bannerSys *gacha.BannerSystem   // shared banner system (wraps softSys)
)

//
// ---------- Helpers: query parsing ----------
//

func qFloat(r *http.Request, key string) (float64, bool, string) {
	s := r.URL.Query().Get(key)
	if s == "" {
		return 0, false, ""
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false, "invalid " + key
	}
	return v, true, ""
}

func qInt(r *http.Request, key string) (int, bool, string) {
	s := r.URL.Query().Get(key)
	if s == "" {
		return 0, false, ""
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, false, "invalid " + key
	}
	return v, true, ""
}

// parse "off_probs=0.5,0.4,0.3" -> []float64{0.5,0.4,0.3}
func parseOffProbs(s string) ([]float64, string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ""
	}
	parts := strings.Split(s, ",")
	out := make([]float64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return nil, "invalid off_probs element: " + p
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil, "off_probs empty after parsing"
	}
	return out, ""
}

//
// ---------- Soft-pity resolver ----------
//
// Given pity + optional soft params (start/start_pct + target + easing),
// build or reuse a SoftPitySystem. If soft params are absent, it degrades to hard pity only.

func ensureSoftPity(pity int, startOpt *int, startPctOpt *float64, targetOpt *float64, easing string) (*gacha.SoftPitySystem, string) {
	// No soft ramp -> pure hard pity (Soft=nil)
	if targetOpt == nil || (startOpt == nil && startPctOpt == nil) {
		if softSys == nil || softSys.Pity != pity || softSys.Soft != nil {
			sp, _ := gacha.NewSoftPitySystem(pity, nil, nil)
			softSys = sp
		}
		return softSys, ""
	}

	// Compute StartAt from start or start_pct
	startAt := 0
	if startOpt != nil {
		startAt = *startOpt
	} else {
		sp := *startPctOpt
		if sp < 0 {
			sp = 0
		}
		if sp > 1 {
			sp = 1
		}
		startAt = int(math.Ceil(sp * float64(pity)))
		if startAt >= pity {
			startAt = pity - 1
		}
	}

	target := *targetOpt
	cfg := &gacha.SoftPityConfig{
		Pity:       pity,
		StartAt:    startAt,
		TargetProb: target,
		Easing:     gacha.Easing(easing),
	}

	need := false
	if softSys == nil || softSys.Pity != pity {
		need = true
	} else if softSys.Soft == nil {
		need = true
	} else if softSys.Soft.StartAt != startAt || softSys.Soft.TargetProb != target || string(softSys.Soft.Easing) != easing {
		need = true
	}

	if need {
		sp, err := gacha.NewSoftPitySystem(pity, cfg, nil)
		if err != nil {
			return nil, err.Error()
		}
		softSys = sp
	}
	return softSys, ""
}

//
// ---------- Handlers ----------
//

// /draw_n?p=0.006&n=10
// Plain Bernoulli N draws, no pity, no banner.
func handleDrawN(w http.ResponseWriter, r *http.Request) {
	p, ok, msg := qFloat(r, "p")
	if !ok {
		http.Error(w, "missing param p", http.StatusBadRequest)
		return
	}
	if msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	n, ok, msg := qInt(r, "n")
	if !ok || n <= 0 {
		http.Error(w, "missing/invalid param n", http.StatusBadRequest)
		return
	}

	hits := make([]bool, n)
	for i := 0; i < n; i++ {
		h, derr := gacha.Draw(p, nil)
		if derr != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(nResp{Err: derr.Error()})
			return
		}
		hits[i] = h
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(nResp{Hits: hits})
}

// /draw_n_pity?p=0.006&pity=90&n=10[&start=74|&start_pct=0.9][&target=0.5][&easing=linear]
// Soft/hard pity N draws (no UP/off decision layer).
func handleDrawNPity(w http.ResponseWriter, r *http.Request) {
	p, ok, msg := qFloat(r, "p")
	if !ok {
		http.Error(w, "missing param p", http.StatusBadRequest)
		return
	}
	if msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	pity, ok, msg := qInt(r, "pity")
	if !ok || pity <= 0 {
		http.Error(w, "missing/invalid param pity", http.StatusBadRequest)
		return
	}
	n, ok, msg := qInt(r, "n")
	if !ok || n <= 0 {
		http.Error(w, "missing/invalid param n", http.StatusBadRequest)
		return
	}

	// optional soft params
	var startOpt *int
	if v, has, _ := qInt(r, "start"); has {
		startOpt = &v
	}
	var startPctOpt *float64
	if v, has, _ := qFloat(r, "start_pct"); has {
		startPctOpt = &v
	}
	var targetOpt *float64
	if v, has, _ := qFloat(r, "target"); has {
		targetOpt = &v
	}
	easing := r.URL.Query().Get("easing")

	mu.Lock()
	sp, errStr := ensureSoftPity(pity, startOpt, startPctOpt, targetOpt, easing)
	if errStr != "" {
		mu.Unlock()
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(nResp{Err: errStr})
		return
	}

	hits := make([]bool, n)
	for i := 0; i < n; i++ {
		h, derr := sp.Draw(p)
		if derr != nil {
			mu.Unlock()
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(nResp{Err: derr.Error()})
			return
		}
		hits[i] = h
	}
	count := sp.Count
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(nResp{Hits: hits, Count: count})
}

// /draw_n_banner?p=0.006&pity=90&n=10
//   [&start=74 | &start_pct=0.9][&target=0.5][&easing=linear]
//   [&off_probs=0.5,0.5,0.4][&max_off=3]
// Soft/hard pity + banner multi-off logic for N draws.
func handleDrawNBanner(w http.ResponseWriter, r *http.Request) {
	p, ok, msg := qFloat(r, "p")
	if !ok {
		http.Error(w, "missing param p", http.StatusBadRequest)
		return
	}
	if msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	pity, ok, msg := qInt(r, "pity")
	if !ok || pity <= 0 {
		http.Error(w, "missing/invalid param pity", http.StatusBadRequest)
		return
	}
	n, ok, msg := qInt(r, "n")
	if !ok || n <= 0 {
		http.Error(w, "missing/invalid param n", http.StatusBadRequest)
		return
	}

	// optional soft params
	var startOpt *int
	if v, has, _ := qInt(r, "start"); has {
		startOpt = &v
	}
	var startPctOpt *float64
	if v, has, _ := qFloat(r, "start_pct"); has {
		startPctOpt = &v
	}
	var targetOpt *float64
	if v, has, _ := qFloat(r, "target"); has {
		targetOpt = &v
	}
	easing := r.URL.Query().Get("easing")

	// multi-off params
	rawOff := r.URL.Query().Get("off_probs")
	offProbs, perr := parseOffProbs(rawOff)
	if perr != "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(bannerResp{Err: perr})
		return
	}
	maxOff, hasMax, _ := qInt(r, "max_off")

	mu.Lock()
	defer mu.Unlock()

	// ensure soft pity
	sp, errStr := ensureSoftPity(pity, startOpt, startPctOpt, targetOpt, easing)
	if errStr != "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(bannerResp{Err: errStr})
		return
	}

	// ensure banner system (rebuild if config changed)
	rebuild := false
	if bannerSys == nil || bannerSys.SoftPity != sp {
		rebuild = true
	} else {
		// compare off probs
		if len(offProbs) > 0 {
			if len(offProbs) != len(bannerSys.OffProbs) {
				rebuild = true
			} else {
				for i := range offProbs {
					if offProbs[i] != bannerSys.OffProbs[i] {
						rebuild = true
						break
					}
				}
			}
		}
		// compare max_off
		if hasMax && !rebuild && bannerSys.MaxOff != maxOff {
			rebuild = true
		}
	}
	if rebuild {
		useMax := maxOff
		if !hasMax {
			useMax = 0 // let constructor default to len(offProbs)
		}
		bannerSys = gacha.NewBannerSystem(sp, offProbs, useMax)
	}

	// perform N draws
	results := make([]bannerItem, n)
	for i := 0; i < n; i++ {
		out, derr := bannerSys.Draw(p)
		if derr != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(bannerResp{Err: derr.Error()})
			return
		}
		results[i] = bannerItem{Hit: out.Hit, IsUp: out.IsUp}
	}

	resp := bannerResp{
		Results:        results,
		Count:          bannerSys.SoftPity.Count,
		GuaranteedNext: bannerSys.GuaranteedNext,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// Simulation response DTO
type simResp struct {
	Mean   float64 `json:"mean"`
	Var    float64 `json:"var"`
	StdDev float64 `json:"stdDev"`
	P50    float64 `json:"p50"`
	P90    float64 `json:"p90"`
	P99    float64 `json:"p99"`
	Err    string  `json:"err,omitempty"`
}

// /simulate?goal=first_up&trials=200000
//   &p=0.006&pity=90[&start=74|&start_pct=0.9][&target=0.8][&easing=easeOutQuad][&cushion=73]
//   [&off_probs=0.5,0.4,0.3][&max_off=3]
//   [&budget_n=90]  // only for goal=fixed_budget
func handleSimulate(w http.ResponseWriter, r *http.Request) {
	goalStr := r.URL.Query().Get("goal")
	if goalStr == "" {
		goalStr = string(gacha.GoalFirstUP) // sensible default
	}
	var goal gacha.TrialGoal
	switch goalStr {
	case string(gacha.GoalFirstHit):
		goal = gacha.GoalFirstHit
	case string(gacha.GoalFirstUP):
		goal = gacha.GoalFirstUP
	case string(gacha.GoalFixedBudget):
		goal = gacha.GoalFixedBudget
	default:
		http.Error(w, "invalid goal", http.StatusBadRequest)
		return
	}

	p, ok, msg := qFloat(r, "p")
	if !ok {
		http.Error(w, "missing param p", http.StatusBadRequest)
		return
	}
	if msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	pity, ok, msg := qInt(r, "pity")
	if !ok || pity <= 0 {
		http.Error(w, "missing/invalid param pity", http.StatusBadRequest)
		return
	}
	trials, ok, msg := qInt(r, "trials")
	if !ok || trials <= 0 {
		http.Error(w, "missing/invalid param trials", http.StatusBadRequest)
		return
	}

	// soft pity optional params
	var startOpt *int
	if v, has, _ := qInt(r, "start"); has {
		startOpt = &v
	}
	var startPctOpt *float64
	if v, has, _ := qFloat(r, "start_pct"); has {
		startPctOpt = &v
	}
	var targetOpt *float64
	if v, has, _ := qFloat(r, "target"); has {
		targetOpt = &v
	}
	easing := r.URL.Query().Get("easing")
	cushion := 0
	if v, has, _ := qInt(r, "cushion"); has {
		cushion = v
	}

	// banner multi-off
	rawOff := r.URL.Query().Get("off_probs")
	offProbs, perr := parseOffProbs(rawOff)
	if perr != "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(simResp{Err: perr})
		return
	}
	maxOff, hasMax, _ := qInt(r, "max_off")
	if !hasMax {
		maxOff = 0
	}

	// fixed budget (optional; only used for goal=fixed_budget)
	var budget *gacha.SimBudget
	if goal == gacha.GoalFixedBudget {
		if v, has, _ := qInt(r, "budget_n"); has && v > 0 {
			budget = &gacha.SimBudget{NumDraws: v}
		} else {
			http.Error(w, "missing/invalid budget_n for fixed_budget", http.StatusBadRequest)
			return
		}
	}

	params := gacha.SimParams{
		PBase:     p,
		Pity:      pity,
		StartAt:   startOpt,
		StartPct:  startPctOpt,
		TargetProb: targetOpt,
		Easing:    easing,
		Cushion:   cushion,
		OffProbs:  offProbs,
		MaxOff:    maxOff,
	}

	stats, err := gacha.RunMonteCarlo(params, goal, trials, budget)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(simResp{Err: err.Error()})
		return
	}

	resp := simResp{
		Mean:   stats.Mean,
		Var:    stats.Var,
		StdDev: stats.StdDev,
		P50:    stats.P50,
		P90:    stats.P90,
		P99:    stats.P99,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}


func main() {
	http.HandleFunc("/draw_n", handleDrawN)            // plain N draws (no pity, no banner)
	http.HandleFunc("/draw_n_pity", handleDrawNPity)   // soft/hard pity N draws
	http.HandleFunc("/draw_n_banner", handleDrawNBanner) // soft/hard pity + multi-off banner N draws
	http.HandleFunc("/simulate", handleSimulate)
	log.Println("listening on :8080 ...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
