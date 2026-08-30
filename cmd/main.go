package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"

	"universal-copilot/internal/database"
	"universal-copilot/internal/handlers"
	"universal-copilot/internal/pinger"
)

// loadEnv parses .env file into environment variables
func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			os.Setenv(k, v)
		}
	}
}

func main() {
	// 0. Load .env variables
	loadEnv()

	// Read database directory path
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "."
	}
	dbPath := fmt.Sprintf("%s/copilot.db", dataDir)

	// 1. Initialize SQLite Database
	db := database.InitDB(dbPath)
	defer db.Close()

	// 2. Register Copilot Engine Routes
	http.HandleFunc("/copilot/embed", handlers.HandleEmbed)
	http.HandleFunc("/copilot/chat", handlers.HandleChat)

	// 3. Register Admin Ingestion Routes
	http.HandleFunc("/admin", handlers.HandleAdminUI)
	http.HandleFunc("/admin/ingest", handlers.HandleIngest)

	// 4. Register Health / Ping Route
	http.HandleFunc("/ping", pinger.HandlePing)

	// 5. Register Test Route
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "knowledge/test_portfolio.html")
	})

	// Dynamic Port for Render / Cloud Deployment
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	// 6. Start Self-Pinging Background Service (default 30 min interval)
	pinger.Start(port)

	fmt.Printf("Starting Universal Copilot Engine on :%s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
