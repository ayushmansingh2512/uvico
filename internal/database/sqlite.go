package database

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

// Secret Key for AES Encryption (Must be 32 bytes for AES-256)
func getEncryptionKey() []byte {
	secret := os.Getenv("ENCRYPTION_SECRET")
	if len(secret) < 32 {
		// Fallback 32-byte default key if env is not set
		return []byte("32-byte-long-secret-key-for-aes")
	}
	return []byte(secret[:32])
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
	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Failed to open SQLite database: %v", err)
	}

	_, err = DB.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		log.Fatalf("Failed to set WAL mode: %v", err)
	}

	fmt.Println("SQLite Database connected successfully with WAL mode!")

	createTables()
	return DB
}

func createTables() {
	appTable := `
	CREATE TABLE IF NOT EXISTS applications (
		id TEXT PRIMARY KEY NOT NULL,
		client_name TEXT NOT NULL,
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

	fmt.Println("Universal Database tables initialized successfully!")
}

// GetContextForApp searches targeted contacts & info based on user query
func GetContextForApp(appID string, userQuery string) string {
	searchTerm := "%" + strings.TrimSpace(userQuery) + "%"

	// Targeted Search for specific Department/Contact/Query
	query := `
		SELECT category, title, content_chunk, COALESCE(contact_number, ''), COALESCE(email_address, '')
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
			var category, title, content, phone, email string
			if err := rows.Scan(&category, &title, &content, &phone, &email); err == nil {
				hasResults = true
				contextBuilder.WriteString(fmt.Sprintf("- Category: %s | Title: %s\n  Details: %s\n", category, title, content))
				if phone != "" {
					contextBuilder.WriteString(fmt.Sprintf("  Phone/Ext: %s\n", phone))
				}
				if email != "" {
					contextBuilder.WriteString(fmt.Sprintf("  Email: %s\n", email))
				}
				contextBuilder.WriteString("\n")
			}
		}
	}

	// Fallback: If no direct keyword match, fetch default assets for this app
	if !hasResults {
		fallbackQuery := `
			SELECT category, title, content_chunk, COALESCE(contact_number, ''), COALESCE(email_address, '')
			FROM knowledge_assets WHERE app_id = ? LIMIT 10;
		`
		fallbackRows, err := DB.Query(fallbackQuery, appID)
		if err == nil {
			defer fallbackRows.Close()
			for fallbackRows.Next() {
				var category, title, content, phone, email string
				if err := fallbackRows.Scan(&category, &title, &content, &phone, &email); err == nil {
					contextBuilder.WriteString(fmt.Sprintf("- Title: %s (%s)\n  Details: %s\n  Phone: %s | Email: %s\n\n", title, category, content, phone, email))
				}
			}
		}
	}

	return contextBuilder.String()
}

// GetAPIKeyForApp fetches the configured Gemini API Key from DB
func GetAPIKeyForApp(appID string) string {
	var key string
	err := DB.QueryRow("SELECT gemini_api_key FROM applications WHERE id = ?", appID).Scan(&key)
	if err != nil {
		return ""
	}
	decryptedKey, err := Decrypt(key)
	if err == nil && decryptedKey != "" {
		return decryptedKey
	}
	return key
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
