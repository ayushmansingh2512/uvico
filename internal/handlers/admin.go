package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/dslipak/pdf"
	"universal-copilot/internal/database"
)

func HandleAdminUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Universal AI Copilot - Self-Serve Onboarding</title>
		<style>
			body { font-family: 'Inter', system-ui, sans-serif; background: #0f172a; color: white; padding: 40px 20px; margin: 0; }
			.form-box { max-width: 580px; margin: 0 auto; background: #1e293b; padding: 30px; border-radius: 16px; border: 1px solid #334155; box-shadow: 0 10px 25px -5px rgba(0,0,0,0.5); }
			h2 { color: #60a5fa; margin-top: 0; font-size: 24px; }
			p { font-size: 14px; color: #94a3b8; line-height: 1.5; }
			label { font-size: 13px; color: #cbd5e1; display: block; margin-top: 18px; font-weight: 600; }
			input, textarea { width: 100%; padding: 12px; margin-top: 6px; border-radius: 8px; border: 1px solid #334155; background: #0f172a; color: white; box-sizing: border-box; font-size: 14px; }
			.file-input-wrapper { margin-top: 8px; border: 2px dashed #334155; padding: 20px; border-radius: 8px; text-align: center; background: #111827; }
			button { background: #2563eb; color: white; font-weight: 600; border: none; padding: 14px; border-radius: 8px; cursor: pointer; width: 100%; margin-top: 24px; font-size: 15px; transition: 0.2s; }
			button:hover { background: #1d4ed8; }
			.code-box { background: #090d16; padding: 12px; border-radius: 6px; font-family: monospace; font-size: 12px; color: #38bdf8; overflow-x: auto; margin-top: 6px; }
		</style>
	</head>
	<body>
		<div class="form-box">
			<h2>⚡ Universal AI Copilot</h2>
			<p>Get a custom AI Chatbot for your website, portfolio, or organization in 1 minute. Upload your knowledge doc, set your Gemini key, and embed the 2-line script!</p>

			<form action="/admin/ingest" method="POST" enctype="multipart/form-data">
				<label>1. Unique App ID (e.g. ayushman_dev, city_clinic, saas_copilot):</label>
				<input type="text" name="app_id" placeholder="e.g. ayushman_dev" required />

				<label>2. Your Name / Brand Name (Displayed to visitors in chat):</label>
				<input type="text" name="client_name" placeholder="e.g. Ayushman Singh" />

				<label>3. Google Calendar & Notification Email (For auto-scheduling & confirmations):</label>
				<input type="email" name="calendar_email" placeholder="e.g. yourname@gmail.com" />
				<div style="background: rgba(37, 99, 235, 0.1); border: 1px solid rgba(59, 130, 246, 0.3); border-radius: 8px; padding: 10px; margin-top: 6px; font-size: 12px; color: #93c5fd;">
					📅 <strong>To enable live Google Calendar booking:</strong> Share your Google Calendar with <code>calendar-copilot@ai-interviewer-475814.iam.gserviceaccount.com</code> (Permission: <em>Make changes to events</em>).
				</div>

				<label>4. Set Account Security PIN (To update your data later):</label>
				<input type="password" name="app_passcode" placeholder="Create a secret PIN..." required />

				<label>5. Your Google Gemini API Key (AES-256 Encrypted & Saved):</label>
				<input type="password" name="gemini_api_key" placeholder="AIzaSy..." required />

				<label>6. Upload Knowledge Doc / PDF / Resume:</label>
				<div class="file-input-wrapper">
					<input type="file" name="doc_file" accept=".pdf,.txt,.md" />
				</div>

				<label>Or Paste Plain Text Information:</label>
				<textarea name="raw_text" rows="5" placeholder="Paste website details, FAQs, services, bio, or contact info..."></textarea>

				<button type="submit">🚀 Ingest Data & Generate Embed Code</button>
			</form>
		</div>
	</body>
	</html>
	`
	fmt.Fprint(w, html)
}

func HandleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseMultipartForm(10 << 20)

	appID := strings.TrimSpace(r.FormValue("app_id"))
	clientName := strings.TrimSpace(r.FormValue("client_name"))
	calendarEmail := strings.TrimSpace(r.FormValue("calendar_email"))
	passcode := strings.TrimSpace(r.FormValue("app_passcode"))
	apiKey := strings.TrimSpace(r.FormValue("gemini_api_key"))
	rawText := strings.TrimSpace(r.FormValue("raw_text"))

	if clientName == "" {
		clientName = appID
	}

	if appID == "" || passcode == "" || apiKey == "" {
		http.Error(w, "❌ Please fill all required fields (App ID, Security PIN, and Gemini API Key)!", http.StatusBadRequest)
		return
	}

	// Read File if uploaded
	file, header, err := r.FormFile("doc_file")
	if err == nil {
		defer file.Close()
		fileBytes, readErr := io.ReadAll(file)
		if readErr == nil && len(fileBytes) > 0 {
			ext := strings.ToLower(filepath.Ext(header.Filename))
			if ext == ".pdf" {
				reader, pdfErr := pdf.NewReader(bytes.NewReader(fileBytes), int64(len(fileBytes)))
				if pdfErr == nil {
					var pdfTextBuilder strings.Builder
					for pageNum := 1; pageNum <= reader.NumPage(); pageNum++ {
						page := reader.Page(pageNum)
						if !page.V.IsNull() {
							content, _ := page.GetPlainText(nil)
							pdfTextBuilder.WriteString(content)
						}
					}
					rawText = pdfTextBuilder.String()
				}
			} else {
				rawText = string(fileBytes)
			}
		}
	}

	if strings.TrimSpace(rawText) == "" {
		http.Error(w, "❌ Content is empty! Upload a PDF/TXT or paste text.", http.StatusBadRequest)
		return
	}

	// Verify if App ID exists and check passcode
	var existingPasscode string
	checkErr := database.DB.QueryRow("SELECT app_passcode FROM applications WHERE id = ?", appID).Scan(&existingPasscode)
	if checkErr == nil && existingPasscode != passcode {
		http.Error(w, "⛔ Unauthorized: This App ID already exists and the Security PIN is incorrect!", http.StatusForbidden)
		return
	}

	// Encrypt Gemini Key before saving to SQLite
	encryptedKey, err := database.Encrypt(apiKey)
	if err != nil {
		http.Error(w, "Error encrypting key", http.StatusInternalServerError)
		return
	}

	// Save or Update App with clientName and calendarEmail
	appQuery := `
		INSERT INTO applications (id, client_name, calendar_email, gemini_api_key, app_passcode) 
		VALUES (?, ?, ?, ?, ?) 
		ON CONFLICT(id) DO UPDATE SET 
			client_name = excluded.client_name,
			calendar_email = excluded.calendar_email,
			gemini_api_key = excluded.gemini_api_key,
			app_passcode = excluded.app_passcode;
	`
	_, err = database.DB.Exec(appQuery, appID, clientName, calendarEmail, encryptedKey, passcode)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error saving app: %v", err), http.StatusInternalServerError)
		return
	}

	// Clear previous assets and save fresh content
	_, _ = database.DB.Exec("DELETE FROM knowledge_assets WHERE app_id = ?;", appID)

	assetQuery := `
		INSERT INTO knowledge_assets (app_id, category, title, content_chunk, keywords)
		VALUES (?, ?, ?, ?, ?);
	`
	_, _ = database.DB.Exec(assetQuery, appID, "Full_Document", "Portfolio Knowledge Base", rawText, "about, skills, projects, contacts")

	// Host Origin URL
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	hostURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Chatbot Ready!</title>
		<style>
			body { font-family: sans-serif; background: #0f172a; color: white; padding: 40px; text-align: center; }
			.card { max-width: 600px; margin: 0 auto; background: #1e293b; padding: 30px; border-radius: 12px; border: 1px solid #334155; text-align: left; }
			.code-box { background: #090d16; padding: 15px; border-radius: 8px; font-family: monospace; font-size: 13px; color: #38bdf8; word-break: break-all; margin-top: 10px; }
			h2 { color: #4ade80; }
		</style>
	</head>
	<body>
		<div class="card">
			<h2>🎉 Chatbot Configured for "%s"!</h2>
			<p>Your PDF data was parsed and your Gemini API Key is safely encrypted using AES-256.</p>
			
			<p><strong>Option 1: iFrame Method (Easiest - Works everywhere without HTMX):</strong></p>
			<div class="code-box">
&lt;iframe<br>
&nbsp;&nbsp;src="%s/copilot/embed?app_id=%s"<br>
&nbsp;&nbsp;style="position: fixed; bottom: 20px; right: 20px; z-index: 99999; border: none; width: 380px; height: 520px; background: transparent;"<br>
&nbsp;&nbsp;allowtransparency="true"&gt;<br>
&lt;/iframe&gt;
			</div>

			<br>
			<p><strong>Option 2: HTMX Method:</strong></p>
			<div class="code-box">
&lt;script src="https://unpkg.com/htmx.org@1.9.12"&gt;&lt;/script&gt;<br>
&lt;aside id="ai-copilot-root"<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;hx-get="%s/copilot/embed?app_id=%s"<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;hx-trigger="load"<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;hx-swap="innerHTML"&gt;<br>
&lt;/aside&gt;
			</div>

			<br>
			<a href="/admin" style="color:#60a5fa;">← Configure Another App</a>
		</div>
	</body>
	</html>
	`, appID, hostURL, appID, hostURL, appID)
}
