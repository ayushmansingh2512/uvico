package pinger

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHandlePing(t *testing.T) {
	req, err := http.NewRequest("GET", "/ping", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandlePing)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expectedHeader := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
		t.Errorf("Handler returned wrong content-type: got %v want %v", contentType, expectedHeader)
	}
}

func TestGetTargetURL(t *testing.T) {
	// Test default fallback
	os.Unsetenv("APP_URL")
	os.Unsetenv("RENDER_EXTERNAL_URL")

	url := getTargetURL("10000")
	expected := "http://127.0.0.1:10000/ping"
	if url != expected {
		t.Errorf("getTargetURL fallback failed: got %v want %v", url, expected)
	}

	// Test RENDER_EXTERNAL_URL
	os.Setenv("RENDER_EXTERNAL_URL", "https://universal-copilot.onrender.com/")
	url = getTargetURL("10000")
	expected = "https://universal-copilot.onrender.com/ping"
	if url != expected {
		t.Errorf("getTargetURL RENDER_EXTERNAL_URL failed: got %v want %v", url, expected)
	}
	os.Unsetenv("RENDER_EXTERNAL_URL")

	// Test APP_URL override
	os.Setenv("APP_URL", "https://custom-domain.com")
	url = getTargetURL("10000")
	expected = "https://custom-domain.com/ping"
	if url != expected {
		t.Errorf("getTargetURL APP_URL failed: got %v want %v", url, expected)
	}
	os.Unsetenv("APP_URL")
}
