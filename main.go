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

// Estrutura para guardar o perfil e quando ele expira
type CachedProfile struct {
	Data       interface{}
	Expiration time.Time
}

var (
	// Nosso "banco de dados" temporário
	cache = make(map[string]CachedProfile)
	// Mutex para evitar que duas requisições tentem escrever no mapa ao mesmo tempo
	mutex sync.RWMutex
)

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := os.Getenv("SKYBLOCK_API_KEY")
	playerID := r.URL.Query().Get("id")

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if playerID == "" {
		http.Error(w, "ID do jogador é obrigatório", http.StatusBadRequest)
		return
	}

	// 1. Verificar se temos o perfil no cache e se ainda é válido (15 min)
	mutex.RLock()
	cached, found := cache[playerID]
	mutex.RUnlock()

	if found && time.Now().Before(cached.Expiration) {
		fmt.Printf("📦 Servindo %s via Cache\n", playerID)
		json.NewEncoder(w).Encode(cached.Data)
		return
	}

	// 2. Se não estiver no cache ou expirou, buscar na API
	fmt.Printf("🌐 Buscando %s na API (Cache Expirado/Inexistente)\n", playerID)
	targetURL := fmt.Sprintf("https://skyapi.onrender.com/skyblock/player/profile?id=%s&key=%s", playerID, apiKey)

	resp, err := http.Get(targetURL)
	if err != nil || resp.StatusCode != 200 {
		http.Error(w, "Erro ao buscar dados", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var apiData interface{}
	if err := json.NewDecoder(resp.Body).Decode(&apiData); err != nil {
		http.Error(w, "Erro ao processar JSON", http.StatusInternalServerError)
		return
	}

	// 3. Salvar no Cache com validade de 15 minutos
	mutex.Lock()
	cache[playerID] = CachedProfile{
		Data:       apiData,
		Expiration: time.Now().Add(15 * time.Minute),
	}
	mutex.Unlock()

	json.NewEncoder(w).Encode(apiData)
}

// No seu main.go, adicione estas estruturas e handlers
var (
	leaderboard = make(map[string]int) // Guarda Nome -> Nível
	lbMutex     sync.RWMutex
)

type ScoreRequest struct {
	Username string `json:"username"`
	Level    int    `json:"level"`
}

func updateTopHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method != http.MethodPost {
		return
	}

	var req ScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.Username != "" {
		lbMutex.Lock()
		// Só atualiza se o nível novo for maior
		if current, exists := leaderboard[req.Username]; !exists || req.Level > current {
			leaderboard[req.Username] = req.Level
		}
		lbMutex.Unlock()
	}
}

func getTopHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	lbMutex.RLock()
	var scores []ScoreRequest
	for k, v := range leaderboard {
		scores = append(scores, ScoreRequest{Username: k, Level: v})
	}
	lbMutex.RUnlock()

	// Ordenar do maior para o menor
	sort.Slice(scores, func(i, j int) bool { return scores[i].Level > scores[j].Level })

	// Retorna top 10
	limit := 10
	if len(scores) < 10 {
		limit = len(scores)
	}
	json.NewEncoder(w).Encode(scores[:limit])
}

// No func main(), registre as rotas:
// http.HandleFunc("/update-top", updateTopHandler)
// http.HandleFunc("/top", getTopHandler)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/profile", proxyHandler)
	fmt.Printf("🚀 Proxy com Cache (15min) na porta %s\n", port)
	http.ListenAndServe(":"+port, nil)
	http.HandleFunc("/profile", proxyHandler)
}
