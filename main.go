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

// --- SISTEMA DE LEADERBOARD ---
var (
	leaderboard = make(map[string]int) // Guarda "username" -> "nivel"
	lbMutex     sync.RWMutex           // Protege contra leitura/escrita simultânea
)

// Rota para o Frontend avisar o nível do jogador pesquisado
func updateTopHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var data struct {
		Username string `json:"username"`
		Level    int    `json:"level"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err == nil && data.Username != "" {
		lbMutex.Lock()
		// Só atualiza se o nível novo for maior que o antigo (ou se for novo)
		if currentLevel, exists := leaderboard[data.Username]; !exists || data.Level > currentLevel {
			leaderboard[data.Username] = data.Level
		}
		lbMutex.Unlock()
	}
	w.WriteHeader(http.StatusOK)
}

// Rota para entregar o Top 10 para quem acessar o site
func getTopHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Estrutura temporária para ordenar
	type PlayerScore struct {
		Username string `json:"username"`
		Level    int    `json:"level"`
	}

	lbMutex.RLock()
	var scores []PlayerScore
	for k, v := range leaderboard {
		scores = append(scores, PlayerScore{Username: k, Level: v})
	}
	lbMutex.RUnlock()

	// Ordena do maior nível para o menor
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Level > scores[j].Level
	})

	// Pega no máximo os Top 10
	limit := 10
	if len(scores) < 10 {
		limit = len(scores)
	}

	json.NewEncoder(w).Encode(scores[:limit])
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/profile", proxyHandler)
	fmt.Printf("🚀 Proxy com Cache (15min) na porta %s\n", port)
	http.ListenAndServe(":"+port, nil)
	http.HandleFunc("/profile", proxyHandler)
	http.HandleFunc("/update-top", updateTopHandler) // Rota nova
	http.HandleFunc("/top", getTopHandler)           // Rota nova
}
