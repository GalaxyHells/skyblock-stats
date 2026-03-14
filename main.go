package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	_ "github.com/lib/pq"
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

// --- PERSISTÊNCIA ---

var (
	// Caches separados para Perfis e Inventários
	profileCache   = make(map[string]CachedData)
	inventoryCache = make(map[string]CachedData)
	cacheMutex     sync.RWMutex

	// Conexão com o banco
	db *sql.DB
)

func initDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL não configurada")
	}

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Erro ao abrir conexão com o banco: %v", err)
	}

	// Opcional: testar conexão
	if err := db.Ping(); err != nil {
		log.Fatalf("Erro ao conectar no banco: %v", err)
	}

	// Criar tabela se não existir
	createTable := `
        CREATE TABLE IF NOT EXISTS leaderboard (
            username TEXT PRIMARY KEY,
            level    INT NOT NULL
        );
    `
	if _, err := db.Exec(createTable); err != nil {
		log.Fatalf("Erro ao criar tabela leaderboard: %v", err)
	}

	log.Println("✅ Banco de dados inicializado e tabela leaderboard pronta")
}

func updateScoreDB(username string, level int) error {
	// Usa UPSERT: se já existir, atualiza somente se o novo level for maior
	query := `
        INSERT INTO leaderboard (username, level)
        VALUES ($1, $2)
        ON CONFLICT (username)
        DO UPDATE
        SET level = GREATEST(leaderboard.level, EXCLUDED.level);
    `
	_, err := db.Exec(query, username, level)
	return err
}

func getTopScoresDB(limit int) ([]ScoreRequest, error) {
	query := `
        SELECT username, level
        FROM leaderboard
        ORDER BY level DESC
        LIMIT $1;
    `
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scores []ScoreRequest
	for rows.Next() {
		var s ScoreRequest
		if err := rows.Scan(&s.Username, &s.Level); err != nil {
			return nil, err
		}
		scores = append(scores, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return scores, nil
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

	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var req ScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" {
		http.Error(w, "JSON inválido ou username ausente", http.StatusBadRequest)
		return
	}

	if err := updateScoreDB(req.Username, req.Level); err != nil {
		log.Printf("Erro ao atualizar score no banco: %v", err)
		http.Error(w, "Erro ao salvar no banco", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func getTopHandler(w http.ResponseWriter, r *http.Request) {
	if setupCORS(&w, r) {
		return
	}

	// Limite opcional via query param, padrão 10
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	scores, err := getTopScoresDB(limit)
	if err != nil {
		log.Printf("Erro ao obter top scores: %v", err)
		http.Error(w, "Erro ao consultar o banco", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scores)
}

// --- MAIN ---

func main() {
	initDB()
	defer db.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/profile", proxyHandler)
	http.HandleFunc("/inventories", handleInventories)
	http.HandleFunc("/update-top", updateTopHandler)
	http.HandleFunc("/top", getTopHandler)

	fmt.Printf("🚀 API Rodando na porta %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}
