package models

import "time"

// Application represents a client using the Copilot Engine
type Application struct {
	ID                string    `json:"id"`
	ClientName        string    `json:"client_name"`
	GeminiAPIKey      string    `json:"gemini_api_key"`
	AppPasscode       string    `json:"app_passcode"`
	SystemInstruction string    `json:"system_instruction"`
	CreatedAt         time.Time `json:"created_at"`
}

// KnowledgeAsset represents a chunked document or detail stored in SQLite
type KnowledgeAsset struct {
	ID           int64  `json:"id"`
	AppID        string `json:"app_id"`
	ContentChunk string `json:"content_chunk"`
	MetaTag      string `json:"meta_tag"` // e.g., 'DBMS', 'DP_Pattern', 'About_Me'
}

// AgentSession tracks chat state and current page context
type AgentSession struct {
	SessionID               string    `json:"session_id"`
	AppID                   string    `json:"app_id"`
	CurrentPageContext      string    `json:"current_page_context"`
	ConversationHistoryJSON string    `json:"conversation_history_json"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// PromptRequest represents incoming message from client UI
type PromptRequest struct {
	AppID       string `json:"app_id"`
	UserMessage string `json:"user_message"`
	PageContext string `json:"page_context"`
}
