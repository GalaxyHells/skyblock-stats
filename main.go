package main

import (
	"encoding/json"
	"fmt"
	"io"
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

const lbFileName = "leaderboard.json"

var (
	// Cache de Perfis
	cache = make(map[string]CachedProfile)
	mutex sync.RWMutex

	// Ranking (Leaderboard)
	leaderboard = make(map[string]int)
	lbMutex     sync.RWMutex
)

// --- PERSISTÊNCIA (SALVAR/CARREGAR) ---

// Salva o ranking em um arquivo JSON local
func saveLeaderboard() {
	lbMutex.RLock()
	data, err := json.MarshalIndent(leaderboard, "", "  ")
	lbMutex.RUnlock()

	if err != nil {
		fmt.Println("❌ Erro ao formatar leaderboard:", err)
		return
	}

	err = os.WriteFile(lbFileName, data, 0644)
	if err != nil {
		fmt.Println("❌ Erro ao salvar arquivo de ranking:", err)
	}
}

// Carrega o ranking do arquivo ao iniciar o servidor
func loadLeaderboard() {
	data, err := os.ReadFile(lbFileName)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("ℹ️ Arquivo de ranking não encontrado. Iniciando novo.")
			return
		}
		fmt.Println("❌ Erro ao ler arquivo de ranking:", err)
		return
	}

	lbMutex.Lock()
	err = json.Unmarshal(data, &leaderboard)
	lbMutex.Unlock()

	if err != nil {
		fmt.Println("❌ Erro ao processar JSON do ranking:", err)
	} else {
		fmt.Printf("✅ %d jogadores carregados no ranking.\n", len(leaderboard))
	}
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

	apiKey := os.Getenv("SKYBLOCK_API_KEY")
	playerID := r.URL.Query().Get("id")

	if playerID == "" {
		http.Error(w, "ID do jogador é obrigatório", http.StatusBadRequest)
		return
	}

	mutex.RLock()
	cached, found := cache[playerID]
	mutex.RUnlock()

	if found && time.Now().Before(cached.Expiration) {
		fmt.Printf("📦 Servindo %s via Cache\n", playerID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached.Data)
		return
	}

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
		updated := false
		if current, exists := leaderboard[req.Username]; !exists || req.Level > current {
			leaderboard[req.Username] = req.Level
			updated = true
			fmt.Printf("🏆 Ranking atualizado: %s agora é nível %d\n", req.Username, req.Level)
		}
		lbMutex.Unlock()

		// Só salva no disco se houve uma mudança real
		if updated {
			saveLeaderboard()
		}

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

	sort.Slice(scores, func(i, j int) bool { return scores[i].Level > scores[j].Level })

	limit := 10
	if len(scores) < 10 {
		limit = len(scores)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scores[:limit])
}

func handleInventories(w http.ResponseWriter, r *http.Request) {
	if setupCORS(&w, r) {
		return
	}

	id := r.URL.Query().Get("id")
	key := r.URL.Query().Get("key")
	if id == "" || key == "" {
		http.Error(w, "missing id or key", http.StatusBadRequest)
		return
	}

	url := fmt.Sprintf("https://skyapi.onrender.com/skyblock/player/inventories?id=%s&key=%s", id, key)

	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, "failed to call skyapi", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// --- MAIN ---

func main() {
	// 1. Tentar carregar dados salvos antes de iniciar
	loadLeaderboard()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/profile", proxyHandler)
	http.HandleFunc("/update-top", updateTopHandler)
	http.HandleFunc("/top", getTopHandler)
	http.HandleFunc("/inventories", handleInventories)

	fmt.Printf("🚀 Servidor iniciado na porta %s\n", port)
	fmt.Printf("📌 Rotas: /profile, /top, /update-top, /inventories\n")

	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Printf("❌ Erro ao iniciar servidor: %v\n", err)
	}
}
