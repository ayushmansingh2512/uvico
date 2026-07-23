package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"universal-copilot/internal/database"
)

// Gemini API Request/Response Structures
type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// HandleEmbed returns the Ichiban-styled AI Copilot
func HandleEmbed(w http.ResponseWriter, r *http.Request) {
	appID := r.URL.Query().Get("app_id")
	if appID == "" {
		http.Error(w, "Missing app_id parameter", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	rawTmpl := `
	<style>
		/* ======================================================
		   ICHIBAN LUXURY DARK DESIGN + FIXED UI
		   ====================================================== */

		.copilot-circle-wrapper {
			position: fixed;
			bottom: 1.5rem;
			right: 1.5rem;
			z-index: 9999;
			width: 3.8rem;
			height: 3.8rem;
			background-color: #19191a;
			border: 1px solid rgba(255, 255, 255, 0.12);
			border-radius: 2rem;
			padding: 0;
			box-sizing: border-box;
			box-shadow: inset 0 8px 32px rgba(255, 255, 255, 0.03),
			            0 12px 32px rgba(0, 0, 0, 0.75),
			            inset 0 2px 2px rgba(255, 255, 255, 0.15);
			cursor: pointer;
			display: flex;
			flex-direction: column;
			justify-content: space-between;
			overflow: hidden;
			font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
			color: #ffffff;

			transition: transform 0.6s cubic-bezier(0.16, 1, 0.3, 1),
						opacity 0.5s cubic-bezier(0.16, 1, 0.3, 1),
						height 0.35s cubic-bezier(0.16, 1, 0.3, 1) 0s,
						width 0.35s cubic-bezier(0.16, 1, 0.3, 1) 0.35s,
						border-radius 0.35s cubic-bezier(0.16, 1, 0.3, 1) 0.35s,
						padding 0.35s cubic-bezier(0.16, 1, 0.3, 1) 0.35s;
		}

		.copilot-circle-wrapper.is-expanded-width {
			width: min(24rem, calc(100vw - 3rem));
			height: 3.8rem;
			border-radius: 2rem;
			padding: 1rem;

			transition: transform 0.6s cubic-bezier(0.16, 1, 0.3, 1),
						opacity 0.5s cubic-bezier(0.16, 1, 0.3, 1),
						width 0.4s cubic-bezier(0.16, 1, 0.3, 1) 0s,
						height 0.4s cubic-bezier(0.16, 1, 0.3, 1) 0.38s,
						border-radius 0.4s cubic-bezier(0.16, 1, 0.3, 1) 0.38s,
						padding 0.4s cubic-bezier(0.16, 1, 0.3, 1) 0.38s;
		}

		.copilot-circle-wrapper.is-open {
			width: min(24rem, calc(100vw - 3rem));
			height: 28rem;
			border-radius: 1.25rem;
			padding: 1.25rem;
			cursor: default;
		}

		.copilot-toggle-icon {
			position: absolute;
			top: 0;
			right: 0;
			z-index: 10;
			display: flex;
			justify-content: center;
			align-items: center;
			width: 3.8rem;
			height: 3.8rem;
			cursor: pointer;
			color: #ffffff;
			transition: transform 0.3s cubic-bezier(0.16, 1, 0.3, 1);
		}

		.copilot-circle-wrapper.is-expanded-width .copilot-toggle-icon,
		.copilot-circle-wrapper.is-open .copilot-toggle-icon {
			top: 0.9rem;
			right: 1.1rem;
			width: 2rem;
			height: 2rem;
		}

		.copilot-content-wrapper {
			width: 100%;
			height: 100%;
			display: flex;
			flex-direction: column;
			justify-content: space-between;
			opacity: 0;
			visibility: hidden;
			transition: opacity 0.3s cubic-bezier(0.16, 1, 0.3, 1), visibility 0.3s;
		}

		.copilot-circle-wrapper.is-open .copilot-content-wrapper {
			opacity: 1;
			visibility: visible;
			transition-delay: 0.4s;
		}

		.copilot-header {
			border-bottom: 1px solid rgba(255, 255, 255, 0.08);
			padding-bottom: 0.75rem;
			margin-bottom: 0.75rem;
		}

		.copilot-header h3 {
			margin: 0;
			font-size: 0.9rem;
			font-weight: 500;
			color: #e1e1e6;
			letter-spacing: 0.04em;
			text-transform: uppercase;
		}

		.copilot-chat-history {
			flex: 1;
			overflow-y: auto;
			background: #111112;
			border: 1px solid rgba(255, 255, 255, 0.05);
			padding: 0.75rem;
			border-radius: 0.75rem;
			font-size: 0.82rem;
			color: #c5c5c9;
			line-height: 1.5;
			margin-bottom: 0.75rem;
			display: flex;
			flex-direction: column;
			gap: 8px;
		}

		.copilot-chips {
			display: flex;
			gap: 6px;
			flex-wrap: wrap;
			margin-bottom: 0.75rem;
		}

		.copilot-chip-btn {
			background: #222226;
			color: #d1d1d6;
			border: 1px solid rgba(255, 255, 255, 0.08);
			padding: 5px 10px;
			border-radius: 6px;
			font-size: 0.75rem;
			font-weight: 500;
			cursor: pointer;
			transition: all 0.2s ease;
		}

		.copilot-chip-btn:hover {
			background: #2f2f35;
			color: #ffffff;
			border-color: rgba(255, 255, 255, 0.2);
		}

		.copilot-form {
			display: flex;
			gap: 6px;
		}

		.copilot-input {
			flex: 1;
			padding: 0.6rem 0.75rem;
			border-radius: 0.5rem;
			border: 1px solid rgba(255, 255, 255, 0.1);
			background: #111112;
			color: #ffffff;
			font-size: 0.8rem;
			outline: none;
			transition: border-color 0.2s ease;
		}

		.copilot-input:focus {
			border-color: rgba(255, 255, 255, 0.3);
		}

		.copilot-send-btn {
			background: #ffffff;
			color: #0d0d0e;
			border: none;
			padding: 0.6rem 1rem;
			border-radius: 0.5rem;
			font-size: 0.8rem;
			font-weight: 600;
			cursor: pointer;
			transition: background-color 0.2s ease;
		}

		.copilot-send-btn:hover {
			background: #e1e1e6;
		}

		.copilot-send-btn:disabled {
			background: #333336;
			color: #77777c;
			cursor: not-allowed;
		}
	</style>

	<div id="copilotWrapper" class="copilot-circle-wrapper">
		<div class="copilot-content-wrapper">
			<div class="copilot-header">
				<h3>AI Copilot ({{APP_ID}})</h3>
			</div>
			
			<div id="copilot-chat-history" class="copilot-chat-history">
				<p style="margin:0; color:#8a8a8e;">Hey there! I am Ayushman's AI Assistant. Ask me anything about his skills or projects!</p>
			</div>

			<div class="copilot-chips">
				<button class="copilot-chip-btn"
				        hx-post="/copilot/chat?app_id={{APP_ID}}" 
				        hx-vals='{"message": "What is your background and key technical skills?"}' 
				        hx-target="#copilot-chat-history" 
				        hx-swap="beforeend">
					👤 About & Skills
				</button>

				<button class="copilot-chip-btn"
				        hx-post="/copilot/chat?app_id={{APP_ID}}" 
				        hx-vals='{"message": "What are your top projects and tech stacks?"}' 
				        hx-target="#copilot-chat-history" 
				        hx-swap="beforeend">
					🛠️ Top Projects
				</button>
			</div>

			<form class="copilot-form"
			      hx-post="/copilot/chat?app_id={{APP_ID}}" 
			      hx-target="#copilot-chat-history" 
			      hx-swap="beforeend" 
			      hx-on::before-request="document.getElementById('copilot-send-btn').disabled = true; document.getElementById('copilot-send-btn').innerText = 'Thinking...';"
			      hx-on::after-request="this.reset(); document.getElementById('copilot-send-btn').disabled = false; document.getElementById('copilot-send-btn').innerText = 'Send'; var elem = document.getElementById('copilot-chat-history'); elem.scrollTop = elem.scrollHeight;">
				<input type="text" id="copilot-msg-input" name="message" placeholder="Ask anything..." class="copilot-input" />
				<button id="copilot-send-btn" type="submit" class="copilot-send-btn">Send</button>
			</form>
		</div>

		<div id="copilotToggleIcon" class="copilot-toggle-icon">
			<svg id="copilotIconSvg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<rect x="3" y="11" width="18" height="10" rx="2"></rect>
				<circle cx="12" cy="5" r="2"></circle>
				<path d="M12 7v4"></path>
				<line x1="8" y1="16" x2="8" y2="16.01"></line>
				<line x1="16" y1="16" x2="16" y2="16.01"></line>
			</svg>
		</div>
	</div>

	<script>
		(function() {
			const copilotWrapper = document.getElementById("copilotWrapper");
			const copilotToggleIcon = document.getElementById("copilotToggleIcon");
			const copilotIconSvg = document.getElementById("copilotIconSvg");
			let isAnimating = false;

			const robotSvg = '<rect x="3" y="11" width="18" height="10" rx="2"></rect><circle cx="12" cy="5" r="2"></circle><path d="M12 7v4"></path><line x1="8" y1="16" x2="8" y2="16.01"></line><line x1="16" y1="16" x2="16" y2="16.01"></line>';
			const closeSvg = '<line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line>';

			function openCopilot() {
				if (isAnimating) return;
				isAnimating = true;

				copilotWrapper.classList.add("is-expanded-width");
				copilotIconSvg.innerHTML = closeSvg;

				setTimeout(() => {
					copilotWrapper.classList.add("is-open");
					setTimeout(() => { isAnimating = false; }, 400);
				}, 380);
			}

			function closeCopilot() {
				if (isAnimating) return;
				isAnimating = true;

				copilotWrapper.classList.remove("is-open");
				copilotIconSvg.innerHTML = robotSvg;

				setTimeout(() => {
					copilotWrapper.classList.remove("is-expanded-width");
					setTimeout(() => { isAnimating = false; }, 350);
				}, 350);
			}

			function toggleCopilot(e) {
				if (e) e.stopPropagation();
				const isOpen = copilotWrapper.classList.contains("is-open") || copilotWrapper.classList.contains("is-expanded-width");
				if (!isOpen) {
					openCopilot();
				} else {
					closeCopilot();
				}
			}

			copilotWrapper.addEventListener("click", (e) => {
				if (!copilotWrapper.classList.contains("is-open")) {
					toggleCopilot(e);
				}
			});

			copilotToggleIcon.addEventListener("click", (e) => {
				e.stopPropagation();
				toggleCopilot(e);
			});
		})();
	</script>
	`

	finalHTML := strings.ReplaceAll(rawTmpl, "{{APP_ID}}", appID)
	fmt.Fprint(w, finalHTML)
}

// HandleChat processes chat requests and fetches response via Gemini API
func HandleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	appID := r.URL.Query().Get("app_id")
	userMsg := r.FormValue("message")

	if strings.TrimSpace(userMsg) == "" {
		return
	}

	// ✅ Updated Code:
	contextData := database.GetContextForApp(appID, userMsg)
	
	prompt := fmt.Sprintf(`System Instruction: You are an AI Copilot assistant for Ayushman's portfolio ('%s').
Analyze the provided Context Data below (retrieved from SQLite) and answer the user's question directly and concisely.

CRITICAL FORMATTING & LANGUAGE RULES:
1. STRICT LANGUAGE MATCHING: Respond strictly in the same language as the User Query. If the user query is written in English (e.g. "What is your background?", "Aapka intro..."), reply ONLY in plain, natural English. Never reply in Hindi script or Hinglish unless the user explicitly speaks in Hindi.
2. ABSOLUTELY NO ASTERISKS: Do NOT use any asterisks (*) or double asterisks (**) anywhere in the generated text.
3. CLEAN LISTS: Format bullet points using simple hyphens (-) or numbers.

--- CONTEXT DATA START ---
%s
--- CONTEXT DATA END ---

User Query: %s`, appID, contextData, userMsg)

	apiKey := database.GetAPIKeyForApp(appID)
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	aiReply := callGeminiAPI(prompt, apiKey)

	// Regex filter to completely strip out any rogue asterisks from the text
	re := regexp.MustCompile(`\*+`)
	aiReplyClean := re.ReplaceAllString(aiReply, "")

	responseHTML := fmt.Sprintf(`
		<div style="margin-top:8px; text-align:right;">
			<span style="background:#222226; border:1px solid rgba(255,255,255,0.08); color:#ffffff; padding:4px 8px; border-radius:6px; font-size:12px; display:inline-block;">%s</span>
		</div>
		<div style="margin-top:8px; text-align:left;">
			<div style="background:#19191a; border:1px solid rgba(255,255,255,0.08); padding:8px 10px; border-radius:6px; font-size:12px; color:#e1e1e6; white-space:pre-wrap; margin-top:4px;">%s</div>
		</div>
		<script>
			var elem = document.getElementById('copilot-chat-history');
			elem.scrollTop = elem.scrollHeight;
		</script>
	`, userMsg, aiReplyClean)

	fmt.Fprint(w, responseHTML)
}

// callGeminiAPI sends the prompt to Gemini 2.5 Flash model
func callGeminiAPI(promptText, apiKey string) string {
	if apiKey == "" {
		return "⚠️ No Gemini API Key found in Database or Environment Variables!"
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=%s", apiKey)

	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{Parts: []GeminiPart{{Text: promptText}}},
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "Error encoding request JSON."
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Sprintf("Error calling Gemini API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "Error reading response body."
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("⚠️ Gemini API Error (Status %d): %s", resp.StatusCode, string(body))
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "Error parsing Gemini response JSON."
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text
	}

	return "No response content in Gemini candidate."
}
