package main

import (
	"fmt"
	"io"
	"net/http"
	"os" // Para ler variáveis de ambiente, se necessário
)

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	// Pega a chave da variável de ambiente que configuraremos no Render
	apiKey := os.Getenv("SKYBLOCK_API_KEY")

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Monta a URL usando a chave protegida
	targetURL := fmt.Sprintf("https://skyapi.onrender.com/skyblock/player/profile?%s&key=%s", r.URL.RawQuery, apiKey)

	resp, err := http.Get(targetURL)
	if err != nil {
		http.Error(w, "Erro na API externa", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func main() {
	// O Render injeta automaticamente a variável PORT
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Fallback para rodar localmente
	}

	http.HandleFunc("/profile", proxyHandler)

	fmt.Printf("🚀 Proxy Online na porta %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}
