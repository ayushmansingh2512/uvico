package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"universal-copilot/internal/database"
)

type ParserRequest struct {
	AppID   string `json:"app_id"`
	RawText string `json:"raw_text"`
}

type ParsedChunkItem struct {
	ContentChunk string `json:"content_chunk"`
	MetaTag      string `json:"meta_tag"`
}

type ParserResponse struct {
	AppID  string            `json:"app_id"`
	Chunks []ParsedChunkItem `json:"chunks"`
}

// ParseAndStoreKnowledge sends text to Python Local LLM for parsing & saves to SQLite
func ParseAndStoreKnowledge(appID, rawText string) int {
	parserURL := "http://localhost:8000/parse-and-tag"

	reqPayload := ParserRequest{
		AppID:   appID,
		RawText: rawText,
	}

	jsonBytes, err := json.Marshal(reqPayload)
	if err != nil {
		fmt.Println("Error encoding parser request:", err)
		return 0
	}

	resp, err := http.Post(parserURL, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		fmt.Println("Local LLM Parser unreachable! Saving raw chunk as fallback...")
		database.InsertParsedChunk(appID, rawText, "General")
		return 1
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0
	}

	var parserResp ParserResponse
	if err := json.Unmarshal(body, &parserResp); err != nil {
		fmt.Println("Error parsing LLM response JSON:", err)
		return 0
	}

	savedCount := 0
	for _, chunk := range parserResp.Chunks {
		err := database.InsertParsedChunk(appID, chunk.ContentChunk, chunk.MetaTag)
		if err == nil {
			savedCount++
		}
	}

	return savedCount
}
