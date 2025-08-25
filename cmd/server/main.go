package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/xtding233/gacha-backend/internal/gacha"
)

// response for single draw
type singleResp struct {
	Hit   bool   `json:"hit"`
	Count int    `json:"count,omitempty"`
	Err   string `json:"err,omitempty"`
}

// response for ten-draw
type tenResp struct {
	Hits  []bool `json:"hits"`
	Count int    `json:"count,omitempty"`
	Err   string `json:"err,omitempty"`
}

var (
	ps   *gacha.PitySystem
	lock sync.Mutex
)

func main() {
	// single draw (no pity)
	http.HandleFunc("/draw", func(w http.ResponseWriter, r *http.Request) {
		pStr := r.URL.Query().Get("p")
		if pStr == "" {
			http.Error(w, "missing param p", http.StatusBadRequest)
			return
		}
		p, err := strconv.ParseFloat(pStr, 64)
		if err != nil {
			http.Error(w, "invalid p", http.StatusBadRequest)
			return
		}

		hit, derr := gacha.Draw(p, nil)
		out := singleResp{Hit: hit}
		if derr != nil {
			out.Err = derr.Error()
			w.WriteHeader(http.StatusBadRequest)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})

	// single draw with pity
	http.HandleFunc("/draw_pity", func(w http.ResponseWriter, r *http.Request) {
		pStr := r.URL.Query().Get("p")
		pityStr := r.URL.Query().Get("pity")
		if pStr == "" || pityStr == "" {
			http.Error(w, "missing param p or pity", http.StatusBadRequest)
			return
		}
		p, err := strconv.ParseFloat(pStr, 64)
		if err != nil {
			http.Error(w, "invalid p", http.StatusBadRequest)
			return
		}
		pity, err := strconv.Atoi(pityStr)
		if err != nil || pity <= 0 {
			http.Error(w, "invalid pity", http.StatusBadRequest)
			return
		}

		lock.Lock()
		if ps == nil {
			ps = gacha.NewPitySystem(pity, nil)
		}
		hit, derr := ps.Draw(p)
		count := ps.Count
		lock.Unlock()

		out := singleResp{Hit: hit, Count: count}
		if derr != nil {
			out.Err = derr.Error()
			w.WriteHeader(http.StatusBadRequest)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})

	// ten draw (no pity)
	http.HandleFunc("/ten_draw", func(w http.ResponseWriter, r *http.Request) {
		pStr := r.URL.Query().Get("p")
		if pStr == "" {
			http.Error(w, "missing param p", http.StatusBadRequest)
			return
		}
		p, err := strconv.ParseFloat(pStr, 64)
		if err != nil {
			http.Error(w, "invalid p", http.StatusBadRequest)
			return
		}

		hits := make([]bool, 10)
		for i := 0; i < 10; i++ {
			hit, derr := gacha.Draw(p, nil)
			if derr != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(tenResp{Err: derr.Error()})
				return
			}
			hits[i] = hit
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tenResp{Hits: hits})
	})

	// ten draw with pity
	http.HandleFunc("/ten_draw_pity", func(w http.ResponseWriter, r *http.Request) {
		pStr := r.URL.Query().Get("p")
		pityStr := r.URL.Query().Get("pity")
		if pStr == "" || pityStr == "" {
			http.Error(w, "missing param p or pity", http.StatusBadRequest)
			return
		}
		p, err := strconv.ParseFloat(pStr, 64)
		if err != nil {
			http.Error(w, "invalid p", http.StatusBadRequest)
			return
		}
		pity, err := strconv.Atoi(pityStr)
		if err != nil || pity <= 0 {
			http.Error(w, "invalid pity", http.StatusBadRequest)
			return
		}

		lock.Lock()
		if ps == nil {
			ps = gacha.NewPitySystem(pity, nil)
		}
		hits := make([]bool, 10)
		for i := 0; i < 10; i++ {
			hit, derr := ps.Draw(p)
			if derr != nil {
				lock.Unlock()
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(tenResp{Err: derr.Error()})
				return
			}
			hits[i] = hit
		}
		count := ps.Count
		lock.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tenResp{Hits: hits, Count: count})
	})

	log.Println("listening on :8080 ...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
