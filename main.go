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

type CachedProfile struct {
	Data       interface{}
	Expiration time.Time
}

type ScoreRequest struct {
	Username string `json:"username"`
	Level    int    `json:"level"`
}

var (
	// Cache de Perfis
	cache = make(map[string]CachedProfile)
	mutex sync.RWMutex

	// Ranking (Leaderboard)
	leaderboard = make(map[string]int)
	lbMutex     sync.RWMutex
)

// --- HELPERS ---

// Função para aplicar os headers de CORS em todas as respostas
func setupCORS(w *http.ResponseWriter, r *http.Request) bool {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Se for uma requisição de "preflight" (OPTIONS), paramos por aqui
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

	apiKey := os.Getenv("SKYBLOCK_API_KEY")
	playerID := r.URL.Query().Get("id")

	if playerID == "" {
		http.Error(w, "ID do jogador é obrigatório", http.StatusBadRequest)
		return
	}

	// 1. Verificar Cache
	mutex.RLock()
	cached, found := cache[playerID]
	mutex.RUnlock()

	if found && time.Now().Before(cached.Expiration) {
		fmt.Printf("📦 Servindo %s via Cache\n", playerID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached.Data)
		return
	}

	// 2. Buscar na API Externa
	fmt.Printf("🌐 Buscando %s na API\n", playerID)
	targetURL := fmt.Sprintf("https://skyapi.onrender.com/skyblock/player/profile?id=%s&key=%s", playerID, apiKey)

	resp, err := http.Get(targetURL)
	if err != nil || resp.StatusCode != 200 {
		http.Error(w, "Erro ao buscar dados na API externa", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var apiData interface{}
	if err := json.NewDecoder(resp.Body).Decode(&apiData); err != nil {
		http.Error(w, "Erro ao processar JSON", http.StatusInternalServerError)
		return
	}

	// 3. Salvar no Cache
	mutex.Lock()
	cache[playerID] = CachedProfile{
		Data:       apiData,
		Expiration: time.Now().Add(15 * time.Minute),
	}
	mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiData)
}

func updateTopHandler(w http.ResponseWriter, r *http.Request) {
	if setupCORS(&w, r) {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var req ScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.Username != "" {
		lbMutex.Lock()
		if current, exists := leaderboard[req.Username]; !exists || req.Level > current {
			leaderboard[req.Username] = req.Level
			fmt.Printf("🏆 Ranking atualizado: %s agora é nível %d\n", req.Username, req.Level)
		}
		lbMutex.Unlock()
		w.WriteHeader(http.StatusOK)
	} else {
		http.Error(w, "Dados inválidos", http.StatusBadRequest)
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

	// Ordenar do maior para o menor
	sort.Slice(scores, func(i, j int) bool { return scores[i].Level > scores[j].Level })

	// Limitar ao Top 10
	limit := 10
	if len(scores) < 10 {
		limit = len(scores)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scores[:limit])
}

// --- MAIN ---

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// REGISTRO DAS ROTAS (Deve vir ANTES do ListenAndServe)
	http.HandleFunc("/profile", proxyHandler)
	http.HandleFunc("/update-top", updateTopHandler)
	http.HandleFunc("/top", getTopHandler)

	fmt.Printf("🚀 Servidor iniciado na porta %s\n", port)
	fmt.Printf("📌 Rotas: /profile, /top, /update-top\n")

	// Inicia o servidor (esta função bloqueia a execução)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Printf("❌ Erro ao iniciar servidor: %v\n", err)
	}
}
