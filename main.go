package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

type CachedProfile struct {
	Data       interface{}
	Expiration time.Time
}

type PlayerScore struct {
	Username string `json:"username"`
	Level    int    `json:"level"`
}

var (
	cache       = make(map[string]CachedProfile)
	leaderboard = make(map[string]int)
	globalMutex sync.RWMutex // Um único mutex para simplificar
)

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	apiKey := os.Getenv("SKYBLOCK_API_KEY")
	playerID := r.URL.Query().Get("id")

	if playerID == "" {
		http.Error(w, `{"error": "ID obrigatório"}`, http.StatusBadRequest)
		return
	}

	globalMutex.RLock()
	cached, found := cache[playerID]
	globalMutex.RUnlock()

	if found && time.Now().Before(cached.Expiration) {
		json.NewEncoder(w).Encode(cached.Data)
		return
	}

	targetURL := fmt.Sprintf("https://skyapi.onrender.com/skyblock/player/profile?id=%s&key=%s", playerID, apiKey)
	resp, err := http.Get(targetURL)
	if err != nil || resp.StatusCode != 200 {
		http.Error(w, `{"error": "Erro na API externa"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var apiData interface{}
	json.NewDecoder(resp.Body).Decode(&apiData)

	globalMutex.Lock()
	cache[playerID] = CachedProfile{Data: apiData, Expiration: time.Now().Add(15 * time.Minute)}
	globalMutex.Unlock()

	json.NewEncoder(w).Encode(apiData)
}

func updateTopHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	var data PlayerScore
	if err := json.NewDecoder(r.Body).Decode(&data); err == nil && data.Username != "" {
		globalMutex.Lock()
		if current, exists := leaderboard[data.Username]; !exists || data.Level > current {
			leaderboard[data.Username] = data.Level
		}
		globalMutex.Unlock()
	}
	w.WriteHeader(http.StatusOK)
}

func getTopHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	globalMutex.RLock()
	var scores []PlayerScore
	for k, v := range leaderboard {
		scores = append(scores, PlayerScore{Username: k, Level: v})
	}
	globalMutex.RUnlock()

	sort.Slice(scores, func(i, j int) bool { return scores[i].Level > scores[j].Level })

	limit := 10
	if len(scores) < 10 {
		limit = len(scores)
	}
	json.NewEncoder(w).Encode(scores[:limit])
}

func main() {
	http.HandleFunc("/profile", proxyHandler)
	http.HandleFunc("/top", getTopHandler)
	http.HandleFunc("/update-top", updateTopHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Servidor online na porta %s\n", port)
	http.ListenAndServe(":"+port, nil)
}
