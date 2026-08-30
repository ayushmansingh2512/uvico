package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"universal-copilot/internal/calendar"
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
	<script src="https://unpkg.com/htmx.org@1.9.12"></script>
	<style>
		/* ======================================================
		   ICHIBAN LUXURY DARK DESIGN + FIXED UI
		   ====================================================== */

		html, body {
			margin: 0;
			padding: 0;
			width: 100%;
			height: 100%;
			overflow: hidden;
			background: transparent !important;
		}

		.copilot-circle-wrapper {
			position: fixed;
			bottom: 1.5rem;
			right: 1.5rem;
			z-index: 9999;
			width: 3.8rem;
			height: 3.8rem;
			background: radial-gradient(circle at 32% 28%, #29292e 0%, #16161a 75%);
			border: 1px solid rgba(255, 255, 255, 0.10);
			border-radius: 50%;
			padding: 0;
			box-sizing: border-box;
			box-shadow: 0 4px 18px rgba(0, 0, 0, 0.45), 0 0 26px rgba(216, 168, 74, 0.20);
			cursor: pointer;
			display: flex;
			flex-direction: column;
			justify-content: space-between;
			overflow: hidden;
			font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
			color: #ffffff;

			transition: transform 0.3s cubic-bezier(0.16, 1, 0.3, 1),
						box-shadow 0.35s cubic-bezier(0.16, 1, 0.3, 1),
						background 0.35s cubic-bezier(0.16, 1, 0.3, 1),
						border-color 0.3s ease,
						width 0.35s cubic-bezier(0.16, 1, 0.3, 1),
						height 0.35s cubic-bezier(0.16, 1, 0.3, 1),
						border-radius 0.35s cubic-bezier(0.16, 1, 0.3, 1),
						padding 0.35s cubic-bezier(0.16, 1, 0.3, 1);
		}

		.copilot-circle-wrapper:not(.is-open):hover {
			transform: translateY(-2px) scale(1.05);
			background: radial-gradient(circle at 32% 28%, #333339 0%, #1c1c20 75%);
			box-shadow: 0 6px 22px rgba(0, 0, 0, 0.5), 0 0 34px rgba(216, 168, 74, 0.35);
			border-color: rgba(216, 168, 74, 0.4);
		}

		.copilot-circle-wrapper:not(.is-open):active {
			transform: scale(0.96);
		}

		.copilot-circle-wrapper.is-expanded-width {
			width: min(24rem, calc(100vw - 3rem));
			height: 3.8rem;
			background-color: #242424;
			border: 1px solid rgba(255, 255, 255, 0.12);
			border-radius: 1.25rem;
			padding: 1rem;
			box-shadow: 0 10px 28px rgba(0, 0, 0, 0.4);
		}

		.copilot-circle-wrapper.is-open {
			width: min(24rem, calc(100vw - 3rem));
			height: 28rem;
			background-color: #242424;
			border: 1px solid rgba(255, 255, 255, 0.12);
			border-radius: 1.25rem;
			padding: 1.25rem;
			cursor: default;
			transform: translateY(0) scale(1);
			box-shadow: 0 14px 36px rgba(0, 0, 0, 0.45);
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

		.copilot-toggle-icon svg {
			transition: transform 0.35s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.2s ease;
		}

		.copilot-circle-wrapper.is-open .copilot-toggle-icon svg {
			transform: rotate(90deg);
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
			scrollbar-width: none;
			-ms-overflow-style: none;
			background: #181818;
			border: 1px solid rgba(255, 255, 255, 0.08);
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

		.copilot-chat-history::-webkit-scrollbar {
			display: none;
		}

		.copilot-chat-bubble {
			animation: chatBubbleIn 0.32s cubic-bezier(0.16, 1, 0.3, 1) forwards;
		}

		@keyframes chatBubbleIn {
			from {
				opacity: 0;
				transform: translateY(8px) scale(0.97);
			}
			to {
				opacity: 1;
				transform: translateY(0) scale(1);
			}
		}

		.site-pet__svg {
			overflow: visible;
			filter: drop-shadow(0 4px 14px rgba(0, 0, 0, 0.45));
			animation: petIdleBob 3.2s ease-in-out infinite;
			transition: transform 0.3s cubic-bezier(0.16, 1, 0.3, 1);
		}

		@keyframes petIdleBob {
			0%, 100% { transform: translateY(0px); }
			50% { transform: translateY(-3.5px); }
		}

		.site-pet__shadow, .site-pet__shadow-ambient {
			transform-box: fill-box;
			transform-origin: center;
			animation: petShadowPulse 3.2s ease-in-out infinite;
		}

		@keyframes petShadowPulse {
			0%, 100% { transform: scaleX(1); opacity: 0.85; }
			50% { transform: scaleX(0.72); opacity: 0.40; }
		}

		.site-pet__eyes {
			transition: transform 0.15s ease-out;
			transform-box: fill-box;
			transform-origin: center;
			animation: petEyeJoyShake 3.2s cubic-bezier(0.4, 0, 0.2, 1) infinite;
		}

		@keyframes petEyeJoyShake {
			0%, 28%, 75%, 100% {
				transform: translate(0, 0) scaleY(1);
			}
			36% {
				transform: translate(-1.2px, -1.2px) scaleY(0.38);
			}
			44% {
				transform: translate(1.2px, -0.6px) scaleY(0.30);
			}
			52% {
				transform: translate(-1px, -1px) scaleY(0.35);
			}
			60% {
				transform: translate(1px, -0.6px) scaleY(0.30);
			}
			68% {
				transform: translate(0, -0.4px) scaleY(0.65);
			}
		}

		.copilot-circle-wrapper:hover .site-pet__eyes {
			animation: petHoverHappyEyes 0.35s ease-in-out infinite alternate;
		}

		@keyframes petHoverHappyEyes {
			0% { transform: translateY(-1.2px) scaleY(0.30) rotate(-2deg); }
			100% { transform: translateY(-1.5px) scaleY(0.25) rotate(2deg); }
		}

		.site-pet__eye {
			transform-box: fill-box;
			transform-origin: center;
			animation: petBlink 3.8s infinite ease-in-out;
			filter: drop-shadow(0 0 3px rgba(232, 196, 104, 0.85));
		}

		@keyframes petBlink {
			0%, 88%, 94%, 100% { transform: scaleY(1); }
			91% { transform: scaleY(0.1); }
		}

		.pet-hat[data-hat-variant="sprout"] ellipse:first-of-type {
			transform-box: fill-box;
			transform-origin: bottom right;
			animation: leafLeft 2.4s ease-in-out infinite alternate;
		}

		.pet-hat[data-hat-variant="sprout"] ellipse:last-of-type {
			transform-box: fill-box;
			transform-origin: bottom left;
			animation: leafRight 2.4s ease-in-out infinite alternate;
		}

		@keyframes leafLeft {
			0% { transform: rotate(22deg); }
			100% { transform: rotate(34deg); }
		}

		@keyframes leafRight {
			0% { transform: rotate(-22deg); }
			100% { transform: rotate(-34deg); }
		}

		.copilot-circle-wrapper:hover .site-pet__leg--left {
			animation: legTapLeft 0.45s ease-in-out infinite alternate;
		}

		.copilot-circle-wrapper:hover .site-pet__leg--right {
			animation: legTapRight 0.45s ease-in-out 0.22s infinite alternate;
		}

		@keyframes legTapLeft {
			0% { transform: translateY(0); }
			100% { transform: translateY(-2px); }
		}

		@keyframes legTapRight {
			0% { transform: translateY(0); }
			100% { transform: translateY(-2px); }
		}

		.copilot-circle-wrapper:not(.is-open):hover .site-pet__svg {
			animation: petWiggle 0.6s ease-in-out infinite alternate;
		}

		@keyframes petWiggle {
			0% { transform: translateY(-3px) rotate(-3deg); }
			100% { transform: translateY(-3px) rotate(3deg); }
		}

		.site-pet__arm--left {
			transform-box: fill-box;
			transform-origin: 7px 3px;
			animation: leftArmWave 3.2s cubic-bezier(0.4, 0, 0.2, 1) infinite;
		}

		@keyframes leftArmWave {
			0%, 15% {
				transform: rotate(0deg);
			}
			32% {
				transform: rotate(138deg) translateY(-2px);
			}
			42% {
				transform: rotate(165deg) translateY(-3px);
			}
			52% {
				transform: rotate(122deg) translateY(-3px);
			}
			62% {
				transform: rotate(160deg) translateY(-3px);
			}
			75% {
				transform: rotate(45deg);
			}
			88%, 100% {
				transform: rotate(0deg);
			}
		}

		.copilot-circle-wrapper:hover .site-pet__arm--left {
			animation: leftArmExcited 0.35s ease-in-out infinite alternate;
		}

		@keyframes leftArmExcited {
			0% { transform: rotate(130deg) translateY(-2px); }
			100% { transform: rotate(168deg) translateY(-4px); }
		}

		.copilot-input-group {
			display: flex;
			gap: 0.5rem;
			position: relative;
		}

		.copilot-input {
			flex: 1;
			background: #242424;
			border: 1px solid rgba(255, 255, 255, 0.12);
			border-radius: 0.5rem;
			padding: 0.6rem 0.8rem;
			color: #ffffff;
			font-size: 0.8rem;
			outline: none;
			transition: border-color 0.2s ease, box-shadow 0.2s ease;
		}

		.copilot-input:focus {
			border-color: rgba(255, 255, 255, 0.35);
			box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.05);
		}

		.copilot-chips {
			display: flex;
			gap: 6px;
			flex-wrap: wrap;
			margin-bottom: 0.75rem;
		}

		.copilot-chip-btn {
			background: #2e2e2e;
			color: #d1d1d6;
			border: 1px solid rgba(255, 255, 255, 0.1);
			padding: 5px 10px;
			border-radius: 6px;
			font-size: 0.75rem;
			font-weight: 500;
			cursor: pointer;
			transition: all 0.2s ease;
		}

		.copilot-chip-btn:hover {
			background: #383838;
			color: #ffffff;
			border-color: rgba(255, 255, 255, 0.2);
			transform: translateY(-1px);
		}

		.copilot-form {
			display: flex;
			gap: 6px;
		}

		.copilot-send-btn {
			background: #ffffff;
			color: #1a1a1a;
			border: none;
			padding: 0.6rem 1rem;
			border-radius: 0.5rem;
			font-size: 0.8rem;
			font-weight: 600;
			cursor: pointer;
			transition: background-color 0.2s ease, transform 0.15s ease;
		}

		.copilot-send-btn:hover {
			background: #e1e1e6;
			transform: translateY(-1px);
		}

		.copilot-send-btn:active {
			transform: scale(0.97);
		}

		.copilot-send-btn:disabled {
			background: #383838;
			color: #77777c;
			cursor: not-allowed;
			transform: none;
		}

		.copilot-chips {
			display: flex;
			flex-wrap: wrap;
			gap: 6px;
			margin-top: 8px;
		}

		.copilot-chip-btn {
			background: #2a2a2a;
			color: #f3e9d6;
			border: 1px solid rgba(255, 255, 255, 0.15);
			padding: 5px 10px;
			border-radius: 6px;
			font-size: 0.72rem;
			font-weight: 500;
			cursor: pointer;
			transition: all 0.2s ease;
			display: inline-block;
		}

		.copilot-chip-btn:hover {
			background: #383838;
			border-color: rgba(255, 255, 255, 0.35);
			transform: translateY(-1px);
		}

		.copilot-starter-chips {
			display: flex;
			flex-direction: column;
			gap: 6px;
			margin-top: 10px;
		}

		.copilot-starter-chip {
			background: #28282e;
			color: #f3e9d6;
			border: 1px solid rgba(216, 168, 74, 0.3);
			padding: 7px 11px;
			border-radius: 8px;
			font-size: 0.76rem;
			font-weight: 500;
			cursor: pointer;
			transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
			text-align: left;
			display: flex;
			align-items: center;
			gap: 6px;
			font-family: inherit;
		}

		.copilot-starter-chip:hover {
			background: #34343c;
			border-color: rgba(216, 168, 74, 0.7);
			color: #ffffff;
			transform: translateX(2px);
		}

		.copilot-starter-chip:active {
			transform: scale(0.98);
		}
	</style>

	<div id="copilotWrapper" class="copilot-circle-wrapper">
		<div class="copilot-content-wrapper">
			<div class="copilot-header">
				<h3>AI Copilot</h3>
			</div>
			
			<div id="copilot-chat-history" class="copilot-chat-history">
				<p style="margin:0; color:#94a3b8; font-size: 0.82rem; line-height: 1.4;">Hey there! I am an AI Assistant. Ask me anything or select a quick option:</p>
				<div class="copilot-starter-chips">
					<button type="button" class="copilot-starter-chip" onclick="quickSend('Schedule an interview / meeting with {{CLIENT_NAME}}')">📅 Schedule Meeting</button>
					<button type="button" class="copilot-starter-chip" onclick="quickSend('What programming languages and tech stack does {{CLIENT_NAME}} use?')">💻 Tech Stack & Languages</button>
					<button type="button" class="copilot-starter-chip" onclick="quickSend('Tell me about the key projects and engineering work of {{CLIENT_NAME}}')">🚀 Key Projects</button>
					<button type="button" class="copilot-starter-chip" onclick="quickSend('Who is {{CLIENT_NAME}} and what is the background?')">👤 About {{CLIENT_NAME}}</button>
				</div>
			</div>

			<form class="copilot-form"
			      hx-post="/copilot/chat" 
			      hx-target="#copilot-chat-history" 
			      hx-swap="beforeend" 
			      hx-on::before-request="document.getElementById('copilot-send-btn').disabled = true; document.getElementById('copilot-send-btn').innerText = 'Thinking...';"
			      hx-on::after-request="this.reset(); document.getElementById('copilot-send-btn').disabled = false; document.getElementById('copilot-send-btn').innerText = 'Send'; var elem = document.getElementById('copilot-chat-history'); elem.scrollTop = elem.scrollHeight;">
				<input type="hidden" name="app_id" value="{{APP_ID}}" />
				<input type="text" id="copilot-msg-input" name="message" placeholder="Ask anything..." class="copilot-input" />
				<button id="copilot-send-btn" type="submit" class="copilot-send-btn">Send</button>
			</form>
		</div>

		<div id="copilotToggleIcon" class="copilot-toggle-icon">
			<svg id="copilotIconSvg" width="34" height="44" viewBox="-10 -8 58 64" class="site-pet__svg" fill="none">
				<ellipse class="site-pet__shadow-ambient" cx="22" cy="53.5" rx="16" ry="2.6" fill="#000000" opacity="0.65"></ellipse>
				<ellipse class="site-pet__shadow" cx="22" cy="53.5" rx="12" ry="1.8" fill="#000000" opacity="0.9"></ellipse>
				<rect x="4" y="6" width="36" height="38" rx="5" fill="#2b2b30" stroke="rgba(216,168,74,0.3)" stroke-width="1"></rect>
				<rect x="4" y="6" width="36" height="2" rx="1" fill="#e8c468" opacity="0.25"></rect>
				<g class="site-pet__eyes">
					<rect class="site-pet__eye" x="14" y="22" width="4" height="5" rx="1" fill="#e8c468"></rect>
					<rect class="site-pet__eye" x="26" y="22" width="4" height="5" rx="1" fill="#e8c468"></rect>
				</g>
				<rect class="site-pet__leg site-pet__leg--left" x="10" y="42" width="8" height="10" rx="1.5" fill="#2b2b30" stroke="rgba(216,168,74,0.3)" stroke-width="1"></rect>
				<rect class="site-pet__leg site-pet__leg--right" x="26" y="42" width="8" height="10" rx="1.5" fill="#2b2b30" stroke="rgba(216,168,74,0.3)" stroke-width="1"></rect>
				<rect class="site-pet__arm site-pet__arm--left" x="-5" y="17" width="8" height="17" rx="4" fill="#2b2b30" stroke="rgba(216,168,74,0.3)" stroke-width="1"></rect>
				<g class="pet-hat" data-hat-variant="sprout">
					<rect x="21.2" y="-6" width="1.6" height="12" fill="#1f8a4c"></rect>
					<ellipse cx="17" cy="-7" rx="5" ry="2.8" fill="#34d399" transform="rotate(28 17 -7)"></ellipse>
					<ellipse cx="27" cy="-7" rx="5" ry="2.8" fill="#34d399" transform="rotate(-28 27 -7)"></ellipse>
				</g>
			</svg>
		</div>
	</div>

	<script>
		(function() {
			const copilotWrapper = document.getElementById("copilotWrapper");
			const copilotToggleIcon = document.getElementById("copilotToggleIcon");
			const copilotIconSvg = document.getElementById("copilotIconSvg");
			let isAnimating = false;

			window.quickSend = function(text) {
				const input = document.getElementById('copilot-msg-input');
				const form = input.closest('form');
				input.value = text;
				htmx.trigger(form, 'submit');
			};

			const robotSvg = '<ellipse class="site-pet__shadow-ambient" cx="22" cy="53.5" rx="16" ry="2.6" fill="#000000" opacity="0.65"></ellipse><ellipse class="site-pet__shadow" cx="22" cy="53.5" rx="12" ry="1.8" fill="#000000" opacity="0.9"></ellipse><rect x="4" y="6" width="36" height="38" rx="5" fill="#2b2b30" stroke="rgba(216,168,74,0.3)" stroke-width="1"></rect><rect x="4" y="6" width="36" height="2" rx="1" fill="#e8c468" opacity="0.25"></rect><g class="site-pet__eyes"><rect class="site-pet__eye" x="14" y="22" width="4" height="5" rx="1" fill="#e8c468"></rect><rect class="site-pet__eye" x="26" y="22" width="4" height="5" rx="1" fill="#e8c468"></rect></g><rect class="site-pet__leg site-pet__leg--left" x="10" y="42" width="8" height="10" rx="1.5" fill="#2b2b30" stroke="rgba(216,168,74,0.3)" stroke-width="1"></rect><rect class="site-pet__leg site-pet__leg--right" x="26" y="42" width="8" height="10" rx="1.5" fill="#2b2b30" stroke="rgba(216,168,74,0.3)" stroke-width="1"></rect><rect class="site-pet__arm site-pet__arm--left" x="-5" y="17" width="8" height="17" rx="4" fill="#2b2b30" stroke="rgba(216,168,74,0.3)" stroke-width="1"></rect><g class="pet-hat" data-hat-variant="sprout"><rect x="21.2" y="-6" width="1.6" height="12" fill="#1f8a4c"></rect><ellipse cx="17" cy="-7" rx="5" ry="2.8" fill="#34d399" transform="rotate(28 17 -7)"></ellipse><ellipse cx="27" cy="-7" rx="5" ry="2.8" fill="#34d399" transform="rotate(-28 27 -7)"></ellipse></g>';
			const closeSvg = '<line x1="12" y1="18" x2="32" y2="38" stroke="#ffffff" stroke-width="3.5" stroke-linecap="round"></line><line x1="32" y1="18" x2="12" y2="38" stroke="#ffffff" stroke-width="3.5" stroke-linecap="round"></line>';

			// Live cursor glance tracking for pet eyes
			window.addEventListener("mousemove", (e) => {
				const eyesGroup = document.querySelector(".site-pet__eyes");
				if (!eyesGroup) return;
				const rect = eyesGroup.getBoundingClientRect();
				const eyeCenterX = rect.left + rect.width / 2;
				const eyeCenterY = rect.top + rect.height / 2;
				const dx = e.clientX - eyeCenterX;
				const dy = e.clientY - eyeCenterY;
				const maxOffset = 2.2;
				const dist = Math.hypot(dx, dy) || 1;
				const offsetX = Math.max(-maxOffset, Math.min(maxOffset, (dx / dist) * maxOffset));
				const offsetY = Math.max(-maxOffset, Math.min(maxOffset, (dy / dist) * maxOffset));
				eyesGroup.style.transform = "translate(" + offsetX.toFixed(2) + "px, " + offsetY.toFixed(2) + "px)";
			});

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

	clientName := database.GetClientNameForApp(appID)
	if clientName == "" {
		clientName = "Ayushman"
	}

	finalHTML := strings.ReplaceAll(rawTmpl, "{{APP_ID}}", appID)
	finalHTML = strings.ReplaceAll(finalHTML, "{{CLIENT_NAME}}", clientName)
	fmt.Fprint(w, finalHTML)
}

// HandleChat processes chat requests and fetches response via Gemini API
func HandleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	appID := strings.TrimSpace(r.URL.Query().Get("app_id"))
	if appID == "" {
		appID = strings.TrimSpace(r.FormValue("app_id"))
	}

	if appID == "" {
		http.Error(w, "Missing app_id parameter", http.StatusBadRequest)
		return
	}

	userMsg := r.FormValue("message")

	if strings.TrimSpace(userMsg) == "" {
		return
	}

	ownerEmail := database.GetCalendarEmailForApp(appID)
	clientName := database.GetClientNameForApp(appID)
	if clientName == "" {
		clientName = appID
	}

	// 1. Check if user provided an email + time to auto-book a meeting on Google Calendar
	booked, bookingReply, _ := calendar.CheckAndBookMeeting(r.Context(), ownerEmail, clientName, userMsg)
	if booked {
		responseHTML := fmt.Sprintf(`
			<div class="copilot-chat-bubble" style="margin-top:8px; text-align:right;">
				<span style="background:#2e2e2e; border:1px solid rgba(255,255,255,0.1); color:#ffffff; padding:6px 10px; border-radius:6px; font-size:12px; display:inline-block;">%s</span>
			</div>
			<div class="copilot-chat-bubble" style="margin-top:8px; text-align:left;">
				<div style="background:#242424; border:1px solid rgba(255,255,200,0.08); padding:8px 12px; border-radius:6px; font-size:12px; color:#e1e1e6; white-space:pre-wrap; margin-top:4px;">%s</div>
			</div>
			<script>
				var elem = document.getElementById('copilot-chat-history');
				elem.scrollTop = elem.scrollHeight;
			</script>
		`, userMsg, bookingReply)
		fmt.Fprint(w, responseHTML)
		return
	}

	// 2. Check if user is asking about schedule, availability, meetings, free slots, or booking
	calendarInfo := ""
	lowerMsg := strings.ToLower(userMsg)
	if strings.Contains(lowerMsg, "free") || strings.Contains(lowerMsg, "available") || strings.Contains(lowerMsg, "schedule") || strings.Contains(lowerMsg, "book") || strings.Contains(lowerMsg, "meet") || strings.Contains(lowerMsg, "call") || strings.Contains(lowerMsg, "time") || strings.Contains(lowerMsg, "slot") || strings.Contains(lowerMsg, "appointment") {
		calendarInfo = calendar.GetAvailableSlotsSummary(r.Context(), ownerEmail, 7)
	}

	contextData := database.GetContextForApp(appID, userMsg)
	
	prompt := fmt.Sprintf(`System Instruction: You are an AI Copilot assistant for '%s' (App ID: '%s', Calendar: '%s').
Analyze the provided Context Data and Live Calendar Availability below to answer the user's question directly, accurately, politely, and concisely.

CRITICAL CALENDAR & SCHEDULING RULES:
1. When asked about scheduling an interview, meeting, or checking %s's availability:
   - Check the Live Calendar Status below for real-time open slots.
   - Present 3-4 specific recommended free options (e.g. Option A, Option B, Option C) during working hours (10:00 AM – 6:00 PM).
   - ALWAYS append interactive clickable buttons at the very bottom of your response in this exact HTML structure:
   <div class="copilot-chips">
     <button type="button" class="copilot-chip-btn" onclick="document.getElementById('copilot-msg-input').value = this.innerText + ', my email is '; document.getElementById('copilot-msg-input').focus();">Tomorrow at 11:00 AM</button>
     <button type="button" class="copilot-chip-btn" onclick="document.getElementById('copilot-msg-input').value = this.innerText + ', my email is '; document.getElementById('copilot-msg-input').focus();">Tomorrow at 03:00 PM</button>
     <button type="button" class="copilot-chip-btn" onclick="document.getElementById('copilot-msg-input').value = this.innerText + ', my email is '; document.getElementById('copilot-msg-input').focus();">Monday at 04:00 PM</button>
   </div>
2. If the user wants to book or schedule a call, ask for their preferred time slot and email address.

CRITICAL FORMATTING & LANGUAGE RULES:
1. STRICT LANGUAGE MATCHING: Respond strictly in the same language as the User Query. If written in English, reply ONLY in plain, natural English.
2. ABSOLUTELY NO ASTERISKS: Do NOT use any asterisks (*) or double asterisks (**) anywhere in the generated text.
3. CLEAN LISTS: Format bullet points using simple hyphens (-) or numbers.

--- LIVE CALENDAR STATUS ---
%s
--- END CALENDAR STATUS ---

--- CONTEXT DATA START ---
%s
--- CONTEXT DATA END ---

User Query: %s`, clientName, appID, ownerEmail, clientName, calendarInfo, contextData, userMsg)

	log.Printf("[Copilot] Handling chat for App ID '%s' (%s | %s) | Query: %s", appID, clientName, ownerEmail, userMsg)
	apiKey := database.GetAPIKeyForApp(appID)
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
		if apiKey != "" {
			log.Println("[Copilot] Using fallback GEMINI_API_KEY from environment")
		} else {
			log.Printf("[Copilot] ⚠️ Warning: No API key found for '%s' in DB or ENV!", appID)
		}
	} else {
		log.Printf("[Copilot] Retrieved API key from DB for '%s'", appID)
	}

	aiReply := callGeminiAPI(prompt, apiKey)

	// Regex filter to completely strip out any rogue asterisks from the text
	re := regexp.MustCompile(`\*+`)
	aiReplyClean := re.ReplaceAllString(aiReply, "")

	responseHTML := fmt.Sprintf(`
		<div class="copilot-chat-bubble" style="margin-top:8px; text-align:right;">
			<span style="background:#2e2e2e; border:1px solid rgba(255,255,255,0.1); color:#ffffff; padding:6px 10px; border-radius:6px; font-size:12px; display:inline-block;">%s</span>
		</div>
		<div class="copilot-chat-bubble" style="margin-top:8px; text-align:left;">
			<div style="background:#242424; border:1px solid rgba(255,255,200,0.08); padding:8px 12px; border-radius:6px; font-size:12px; color:#e1e1e6; white-space:pre-wrap; margin-top:4px;">%s</div>
		</div>
		<script>
			var elem = document.getElementById('copilot-chat-history');
			elem.scrollTop = elem.scrollHeight;
		</script>
	`, userMsg, aiReplyClean)

	fmt.Fprint(w, responseHTML)
}

// callGeminiAPI sends the prompt directly to Gemini 2.5 Flash model
func callGeminiAPI(promptText, apiKey string) string {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		log.Println("[Gemini] Error: Missing API Key.")
		return "⚠️ No Gemini API Key found in Database or Environment Variables! Please register this App ID in /admin with your Gemini API key."
	}

	maskedKey := "invalid"
	if len(apiKey) > 8 {
		maskedKey = apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
	}
	log.Printf("[Gemini] Calling Gemini 2.5 Flash with API key: %s (length: %d)", maskedKey, len(apiKey))

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

	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		log.Printf("[Gemini] Network error calling Gemini 2.5 Flash: %v", err)
		return fmt.Sprintf("Error calling Gemini API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "Error reading response body."
	}

	log.Printf("[Gemini] Gemini 2.5 Flash responded with HTTP %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		log.Printf("[Gemini] API error response: %s", string(body))
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
