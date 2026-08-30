package database

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

// Secret Key for AES Encryption (Always guaranteed 32 bytes for AES-256)
func getEncryptionKey() []byte {
	secret := os.Getenv("ENCRYPTION_SECRET")
	if secret == "" {
		secret = "universal-copilot-default-aes-secret-key-2026"
	}
	hash := sha256.Sum256([]byte(secret))
	return hash[:]
}

// Encrypt encrypts plain text string into Base64 encoded encrypted string
func Encrypt(plainText string) (string, error) {
	block, err := aes.NewCipher(getEncryptionKey())
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// Decrypt decrypts Base64 encoded string back to plain text
func Decrypt(cipherText string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(getEncryptionKey())
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, cipherTextBytes := data[:nonceSize], data[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, cipherTextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plainText), nil
}

func InitDB(dbPath string) *sql.DB {
	tursoURL := os.Getenv("TURSO_DATABASE_URL")
	tursoToken := os.Getenv("TURSO_AUTH_TOKEN")

	var err error
	if tursoURL != "" {
		fullURL := tursoURL
		if tursoToken != "" && !strings.Contains(fullURL, "authToken=") {
			if strings.Contains(fullURL, "?") {
				fullURL = fmt.Sprintf("%s&authToken=%s", fullURL, tursoToken)
			} else {
				fullURL = fmt.Sprintf("%s?authToken=%s", fullURL, tursoToken)
			}
		}
		fmt.Printf("Connecting to Turso Cloud Database: %s...\n", tursoURL)
		DB, err = sql.Open("libsql", fullURL)
		if err != nil {
			log.Fatalf("Failed to open Turso database: %v", err)
		}
		if err = DB.Ping(); err != nil {
			log.Fatalf("Failed to ping Turso database: %v", err)
		}
		fmt.Println("⚡ Turso Cloud Database connected successfully!")
	} else {
		dir := filepath.Dir(dbPath)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				log.Printf("⚠️ Directory creation warning: %v\n", err)
			}
		}

		DB, err = sql.Open("sqlite", dbPath)
		if err != nil {
			log.Fatalf("Failed to open SQLite database: %v", err)
		}

		_, err = DB.Exec("PRAGMA journal_mode=WAL;")
		if err != nil {
			fmt.Printf("⚠️ WAL mode notice: %v. Switching to DELETE mode...\n", err)
			_, err = DB.Exec("PRAGMA journal_mode=DELETE;")
			if err != nil {
				log.Fatalf("Failed to set journal mode: %v", err)
			}
		}

		fmt.Println("SQLite Local Database connected successfully!")
	}

	createTables()
	return DB
}


func createTables() {
	appTable := `
	CREATE TABLE IF NOT EXISTS applications (
		id TEXT PRIMARY KEY NOT NULL,
		client_name TEXT NOT NULL,
		calendar_email TEXT NOT NULL DEFAULT '',
		gemini_api_key TEXT NOT NULL,
		app_passcode TEXT NOT NULL,
		system_instruction TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	knowledgeTable := `
	CREATE TABLE IF NOT EXISTS knowledge_assets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		app_id TEXT NOT NULL,
		category TEXT NOT NULL,
		title TEXT NOT NULL,
		content_chunk TEXT NOT NULL,
		keywords TEXT,
		FOREIGN KEY(app_id) REFERENCES applications(id) ON DELETE CASCADE
	);`

	if _, err := DB.Exec(appTable); err != nil {
		log.Fatalf("Failed to create applications table: %v", err)
	}

	if _, err := DB.Exec(knowledgeTable); err != nil {
		log.Fatalf("Failed to create knowledge_assets table: %v", err)
	}

	// Migration for existing database files
	_, _ = DB.Exec("ALTER TABLE applications ADD COLUMN app_passcode TEXT NOT NULL DEFAULT '';")
	_, _ = DB.Exec("ALTER TABLE applications ADD COLUMN calendar_email TEXT NOT NULL DEFAULT '';")
	_, _ = DB.Exec("ALTER TABLE applications ADD COLUMN client_name TEXT NOT NULL DEFAULT '';")

	fmt.Println("Universal Database tables initialized successfully!")
}

// GetContextForApp searches targeted contacts & info based on user query
func GetContextForApp(appID string, userQuery string) string {
	searchTerm := "%" + strings.TrimSpace(userQuery) + "%"

	// Targeted Search for specific Department/Contact/Query
	query := `
		SELECT category, title, content_chunk
		FROM knowledge_assets
		WHERE app_id = ? AND (
			title LIKE ? OR 
			category LIKE ? OR 
			keywords LIKE ? OR 
			content_chunk LIKE ?
		)
		LIMIT 5;
	`

	rows, err := DB.Query(query, appID, searchTerm, searchTerm, searchTerm, searchTerm)
	var contextBuilder strings.Builder
	hasResults := false

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var category, title, content string
			if err := rows.Scan(&category, &title, &content); err == nil {
				hasResults = true
				contextBuilder.WriteString(fmt.Sprintf("- Category: %s | Title: %s\n  Details: %s\n\n", category, title, content))
			}
		}
	}

	// Fallback: If no direct keyword match, fetch default assets for this app
	if !hasResults {
		fallbackQuery := `
			SELECT category, title, content_chunk
			FROM knowledge_assets WHERE app_id = ? LIMIT 10;
		`
		fallbackRows, err := DB.Query(fallbackQuery, appID)
		if err == nil {
			defer fallbackRows.Close()
			for fallbackRows.Next() {
				var category, title, content string
				if err := fallbackRows.Scan(&category, &title, &content); err == nil {
					contextBuilder.WriteString(fmt.Sprintf("- Title: %s (%s)\n  Details: %s\n\n", title, category, content))
				}
			}
		}
	}

	return contextBuilder.String()
}

// GetAPIKeyForApp fetches the configured Gemini API Key from DB
func GetAPIKeyForApp(appID string) string {
	var key string
	err := DB.QueryRow("SELECT gemini_api_key FROM applications WHERE id = ?", strings.TrimSpace(appID)).Scan(&key)
	if err != nil {
		log.Printf("[Database] No app record found for ID '%s'", appID)
		return ""
	}
	decryptedKey, err := Decrypt(key)
	if err == nil && decryptedKey != "" {
		return strings.TrimSpace(decryptedKey)
	}
	if strings.HasPrefix(key, "AIzaSy") {
		return strings.TrimSpace(key)
	}
	log.Printf("[Database] Warning: Stored API key for '%s' was encrypted with an older key. Re-save it in /admin.", appID)
	return ""
}

// GetCalendarEmailForApp returns the registered Google Calendar email for this app or fallback
func GetCalendarEmailForApp(appID string) string {
	var email string
	err := DB.QueryRow("SELECT calendar_email FROM applications WHERE id = ?", strings.TrimSpace(appID)).Scan(&email)
	if err == nil && strings.TrimSpace(email) != "" {
		return strings.TrimSpace(email)
	}
	return os.Getenv("CALENDAR_OWNER_EMAIL")
}

// GetClientNameForApp returns the registered client/owner name for this app or defaults to App ID
func GetClientNameForApp(appID string) string {
	var name string
	err := DB.QueryRow("SELECT client_name FROM applications WHERE id = ?", strings.TrimSpace(appID)).Scan(&name)
	if err == nil && strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	return appID
}

// InsertParsedChunk inserts chunked text into SQLite
func InsertParsedChunk(appID, contentChunk, metaTag string) error {
	query := `
		INSERT INTO knowledge_assets (app_id, category, title, content_chunk, keywords)
		VALUES (?, ?, ?, ?, ?);
	`
	_, err := DB.Exec(query, appID, metaTag, metaTag, contentChunk, metaTag)
	return err
}
