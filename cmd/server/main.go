package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"

	"github.com/xtding233/gacha-backend/internal/gacha"
)

type singleResp struct {
	Hit   bool   `json:"hit"`
	Count int    `json:"count,omitempty"`
	Err   string `json:"err,omitempty"`
}

type tenResp struct {
	Hits  []bool `json:"hits"`
	Count int    `json:"count,omitempty"`
	Err   string `json:"err,omitempty"`
}

var (
	psHard *gacha.PitySystem
	psSoft *gacha.SoftPitySystem
	lock   sync.Mutex
)

func parseFloat(r *http.Request, key string) (float64, bool, string) {
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

func parseInt(r *http.Request, key string) (int, bool, string) {
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

// no pity, single draw
func handleDraw(w http.ResponseWriter, r *http.Request) {
	p, ok, msg := parseFloat(r, "p")
	if !ok {
		http.Error(w, "missing param p", http.StatusBadRequest)
		return
	}
	if msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	hit, derr := gacha.Draw(p, nil)
	resp := singleResp{Hit: hit}
	if derr != nil {
		resp.Err = derr.Error()
		w.WriteHeader(http.StatusBadRequest)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// pity, single draw
func handleDrawPity(w http.ResponseWriter, r *http.Request) {
	p, ok, msg := parseFloat(r, "p")
	if !ok {
		http.Error(w, "missing param p", http.StatusBadRequest)
		return
	}
	if msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	pity, ok, msg := parseInt(r, "pity")
	if !ok || pity <= 0 {
		http.Error(w, "missing/invalid param pity", http.StatusBadRequest)
		return
	}
	// soft params (optional)
	start, hasStart, _ := parseInt(r, "start")
	startPct, hasStartPct, _ := parseFloat(r, "start_pct")
	target, hasTarget, _ := parseFloat(r, "target")
	easingStr := r.URL.Query().Get("easing") // linear, easeOutQuad, easeInOutCubic

	lock.Lock()
	defer lock.Unlock()

	// decide whether to build soft or hard pity system
	if hasTarget && (hasStart || hasStartPct) {
		// compute StartAt
		startAt := start
		if !hasStart && hasStartPct {
			if startPct < 0 {
				startPct = 0
			}
			if startPct > 1 {
				startPct = 1
			}
			startAt = int(math.Ceil(startPct * float64(pity)))
			if startAt >= pity {
				startAt = pity - 1
			}
		}
		soft := &gacha.SoftPityConfig{
			Pity:       pity,
			StartAt:    startAt,
			TargetProb: target,
			Easing:     gacha.Easing(easingStr),
		}
		if psSoft == nil || psSoft.Pity != pity || psSoft.Soft == nil ||
			psSoft.Soft.StartAt != startAt || psSoft.Soft.TargetProb != target || string(psSoft.Soft.Easing) != easingStr {
			sp, err := gacha.NewSoftPitySystem(pity, soft, nil)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(singleResp{Err: err.Error()})
				return
			}
			psSoft = sp
		}
		hit, derr := psSoft.Draw(p)
		resp := singleResp{Hit: hit, Count: psSoft.Count}
		if derr != nil {
			resp.Err = derr.Error()
			w.WriteHeader(http.StatusBadRequest)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// pure hard pity fallback
	if psHard == nil || psHard.Pity != pity {
		psHard = gacha.NewPitySystem(pity, nil)
	}
	hit, derr := psHard.Draw(p)
	resp := singleResp{Hit: hit, Count: psHard.Count}
	if derr != nil {
		resp.Err = derr.Error()
		w.WriteHeader(http.StatusBadRequest)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// no pity, 10 draw
func handleTenDraw(w http.ResponseWriter, r *http.Request) {
	p, ok, msg := parseFloat(r, "p")
	if !ok {
		http.Error(w, "missing param p", http.StatusBadRequest)
		return
	}
	if msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	hits := make([]bool, 10)
	for i := 0; i < 10; i++ {
		h, derr := gacha.Draw(p, nil)
		if derr != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(tenResp{Err: derr.Error()})
			return
		}
		hits[i] = h
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tenResp{Hits: hits})
}

// pity, 10 draw
func handleTenDrawPity(w http.ResponseWriter, r *http.Request) {
	p, ok, msg := parseFloat(r, "p")
	if !ok {
		http.Error(w, "missing param p", http.StatusBadRequest)
		return
	}
	if msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	pity, ok, msg := parseInt(r, "pity")
	if !ok || pity <= 0 {
		http.Error(w, "missing/invalid param pity", http.StatusBadRequest)
		return
	}
	start, hasStart, _ := parseInt(r, "start")
	startPct, hasStartPct, _ := parseFloat(r, "start_pct")
	target, hasTarget, _ := parseFloat(r, "target")
	easingStr := r.URL.Query().Get("easing")

	lock.Lock()
	defer lock.Unlock()

	// ensure psSoft/psHard built as in single handler
	useSoft := hasTarget && (hasStart || hasStartPct)
	if useSoft {
		startAt := start
		if !hasStart && hasStartPct {
			if startPct < 0 {
				startPct = 0
			}
			if startPct > 1 {
				startPct = 1
			}
			startAt = int(math.Ceil(startPct * float64(pity)))
			if startAt >= pity {
				startAt = pity - 1
			}
		}
		soft := &gacha.SoftPityConfig{
			Pity:       pity,
			StartAt:    startAt,
			TargetProb: target,
			Easing:     gacha.Easing(easingStr),
		}
		if psSoft == nil || psSoft.Pity != pity || psSoft.Soft == nil ||
			psSoft.Soft.StartAt != startAt || psSoft.Soft.TargetProb != target || string(psSoft.Soft.Easing) != easingStr {
			sp, err := gacha.NewSoftPitySystem(pity, soft, nil)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(tenResp{Err: err.Error()})
				return
			}
			psSoft = sp
		}
		hits := make([]bool, 10)
		for i := 0; i < 10; i++ {
			h, derr := psSoft.Draw(p)
			if derr != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(tenResp{Err: derr.Error()})
				return
			}
			hits[i] = h
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tenResp{Hits: hits, Count: psSoft.Count})
		return
	}

	// hard pity only
	if psHard == nil || psHard.Pity != pity {
		psHard = gacha.NewPitySystem(pity, nil)
	}
	hits := make([]bool, 10)
	for i := 0; i < 10; i++ {
		h, derr := psHard.Draw(p)
		if derr != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(tenResp{Err: derr.Error()})
			return
		}
		hits[i] = h
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tenResp{Hits: hits, Count: psHard.Count})
}

func main() {
	http.HandleFunc("/draw", handleDraw)
	http.HandleFunc("/draw_pity", handleDrawPity)
	http.HandleFunc("/ten_draw", handleTenDraw)
	http.HandleFunc("/ten_draw_pity", handleTenDrawPity)

	log.Println("listening on :8080 ...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
