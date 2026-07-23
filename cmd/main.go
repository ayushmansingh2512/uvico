package main

import (
	"fmt"
	"net/http"
	"os"

	"universal-copilot/internal/database"
	"universal-copilot/internal/handlers"
)

func main() {
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

	// 4. Register Test Route
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "knowledge/test_portfolio.html")
	})

	// Dynamic Port for Hugging Face Spaces
	port := os.Getenv("PORT")
	if port == "" {
		port = "7860"
	}

	fmt.Printf("Starting Universal Copilot Engine on :%s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
