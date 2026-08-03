package pinger

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// HandlePing responds with 200 OK for health/ping checks
func HandlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"alive","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
}

// Start launches a background goroutine to ping the service periodically (default 30 min)
func Start(port string) {
	intervalMinutes := 30
	if envVal := os.Getenv("PING_INTERVAL_MINUTES"); envVal != "" {
		if parsed, err := strconv.Atoi(envVal); err == nil && parsed > 0 {
			intervalMinutes = parsed
		}
	}

	targetURL := getTargetURL(port)
	log.Printf("[Pinger] Initialized self-ping service. Target: %s | Interval: %d minutes\n", targetURL, intervalMinutes)

	go func() {
		// Initial delay to give the server time to start up completely
		time.Sleep(10 * time.Second)

		// Perform an initial ping after startup delay
		pingServer(targetURL)

		ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			pingServer(targetURL)
		}
	}()
}

func getTargetURL(port string) string {
	// 1. Check explicit custom APP_URL env
	if appURL := strings.TrimSpace(os.Getenv("APP_URL")); appURL != "" {
		return cleanURL(appURL) + "/ping"
	}

	// 2. Check Render external URL env (automatically supplied on Render platform)
	if renderURL := strings.TrimSpace(os.Getenv("RENDER_EXTERNAL_URL")); renderURL != "" {
		return cleanURL(renderURL) + "/ping"
	}

	// 3. Fallback to local address
	return fmt.Sprintf("http://127.0.0.1:%s/ping", port)
}

func cleanURL(u string) string {
	u = strings.TrimSpace(u)
	return strings.TrimSuffix(u, "/")
}

func pingServer(u string) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		log.Printf("[Pinger] Self-ping failed for %s: %v\n", u, err)
		return
	}
	defer resp.Body.Close()

	log.Printf("[Pinger] Self-ping succeeded for %s: %s\n", u, resp.Status)
}
