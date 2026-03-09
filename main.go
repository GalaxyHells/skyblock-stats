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

// --- ESTRUTURAS ---

type CachedData struct {
	Data       interface{}
	Expiration time.Time
}

type ScoreRequest struct {
	Username string `json:"username"`
	Level    int    `json:"level"`
}

const lbFileName = "leaderboard.json"

var (
	// Caches separados para Perfis e Inventários
	profileCache   = make(map[string]CachedData)
	inventoryCache = make(map[string]CachedData)
	cacheMutex     sync.RWMutex

	// Ranking (Leaderboard)
	leaderboard = make(map[string]int)
	lbMutex     sync.RWMutex
)

// --- PERSISTÊNCIA ---

func saveLeaderboard() {
	lbMutex.RLock()
	data, err := json.MarshalIndent(leaderboard, "", "  ")
	lbMutex.RUnlock()

	if err != nil {
		fmt.Println("❌ Erro ao formatar leaderboard:", err)
		return
	}

	_ = os.WriteFile(lbFileName, data, 0644)
}

func loadLeaderboard() {
	data, err := os.ReadFile(lbFileName)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Println("❌ Erro ao ler ranking:", err)
		}
		return
	}

	lbMutex.Lock()
	_ = json.Unmarshal(data, &leaderboard)
	lbMutex.Unlock()
	fmt.Printf("✅ Ranking carregado: %d jogadores\n", len(leaderboard))
}

// --- HELPERS ---

func setupCORS(w *http.ResponseWriter, r *http.Request) bool {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		(*w).WriteHeader(http.StatusOK)
		return true
	}
	return false
}

// --- HANDLERS ---

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	if setupCORS(&w, r) {
		return
	}

	playerID := r.URL.Query().Get("id")
	apiKey := os.Getenv("SKYBLOCK_API_KEY")

	if playerID == "" {
		http.Error(w, "ID obrigatório", http.StatusBadRequest)
		return
	}

	// 1. Check Cache
	cacheMutex.RLock()
	cached, found := profileCache[playerID]
	cacheMutex.RUnlock()

	if found && time.Now().Before(cached.Expiration) {
		fmt.Printf("📦 Profile Cache: %s\n", playerID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached.Data)
		return
	}

	// 2. Fetch API
	fmt.Printf("🌐 Profile API: %s\n", playerID)
	url := fmt.Sprintf("https://skyapi.onrender.com/skyblock/player/profile?id=%s&key=%s", playerID, apiKey)

	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		http.Error(w, "Erro na API externa", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var data interface{}
	json.NewDecoder(resp.Body).Decode(&data)

	// 3. Save Cache
	cacheMutex.Lock()
	profileCache[playerID] = CachedData{Data: data, Expiration: time.Now().Add(15 * time.Minute)}
	cacheMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func handleInventories(w http.ResponseWriter, r *http.Request) {
	if setupCORS(&w, r) {
		return
	}

	playerID := r.URL.Query().Get("id")
	apiKey := os.Getenv("SKYBLOCK_API_KEY")

	if playerID == "" {
		http.Error(w, "ID obrigatório", http.StatusBadRequest)
		return
	}

	// 1. Check Cache
	cacheMutex.RLock()
	cached, found := inventoryCache[playerID]
	cacheMutex.RUnlock()

	if found && time.Now().Before(cached.Expiration) {
		fmt.Printf("📦 Inventory Cache: %s\n", playerID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached.Data)
		return
	}

	// 2. Fetch API
	fmt.Printf("🌐 Inventory API: %s\n", playerID)
	url := fmt.Sprintf("https://skyapi.onrender.com/skyblock/player/inventories?id=%s&key=%s", playerID, apiKey)

	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		http.Error(w, "Erro ao buscar inventários", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var data interface{}
	json.NewDecoder(resp.Body).Decode(&data)

	// 3. Save Cache
	cacheMutex.Lock()
	inventoryCache[playerID] = CachedData{Data: data, Expiration: time.Now().Add(15 * time.Minute)}
	cacheMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func updateTopHandler(w http.ResponseWriter, r *http.Request) {
	if setupCORS(&w, r) {
		return
	}

	var req ScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.Username != "" {
		lbMutex.Lock()
		updated := false
		if current, exists := leaderboard[req.Username]; !exists || req.Level > current {
			leaderboard[req.Username] = req.Level
			updated = true
		}
		lbMutex.Unlock()

		if updated {
			fmt.Printf("🏆 Novo recorde: %s (Nível %d)\n", req.Username, req.Level)
			saveLeaderboard()
		}
		w.WriteHeader(http.StatusOK)
	}
}

func getTopHandler(w http.ResponseWriter, r *http.Request) {
	if setupCORS(&w, r) {
		return
	}

	lbMutex.RLock()
	var scores []ScoreRequest
	for k, v := range leaderboard {
		scores = append(scores, ScoreRequest{Username: k, Level: v})
	}
	lbMutex.RUnlock()

	sort.Slice(scores, func(i, j int) bool { return scores[i].Level > scores[j].Level })

	limit := 10
	if len(scores) < 10 {
		limit = len(scores)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scores[:limit])
}

// --- MAIN ---

func main() {
	loadLeaderboard()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/profile", proxyHandler)
	http.HandleFunc("/inventories", handleInventories)
	http.HandleFunc("/update-top", updateTopHandler)
	http.HandleFunc("/top", getTopHandler)

	fmt.Printf("🚀 API Rodando na porta %s\n", port)
	http.ListenAndServe(":"+port, nil)
}
