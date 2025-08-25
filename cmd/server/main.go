package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/xtding233/gacha-backend/internal/gacha"
)

type resp struct {
	Hit bool   `json:"hit"`
	Err string `json:"err,omitempty"`
}

func main() {
	http.HandleFunc("/draw", func(w http.ResponseWriter, r *http.Request) {
		pStr := r.URL.Query().Get("p")
		if pStr == "" {
			http.Error(w, "missing query param p", http.StatusBadRequest)
			return
		}
		p, err := strconv.ParseFloat(pStr, 64)
		if err != nil {
			http.Error(w, "invalid p", http.StatusBadRequest)
			return
		}
		ok, derr := gacha.Draw(p, nil)
		out := resp{Hit: ok}
		if derr != nil {
			out.Err = derr.Error()
			w.WriteHeader(http.StatusBadRequest)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})
	log.Println("listening on :8080 ...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}