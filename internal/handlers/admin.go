package handlers

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dslipak/pdf"
	"universal-copilot/internal/database"
	"universal-copilot/internal/docs"
)

// ── shared CSS tokens ──
const antidoteCSS = `
	:root {
		--background: #888eff;
		--antidote-black: #020034;
		--primary: #0000ff;
		--light-grey: #f4f4f4;
		--white: #ffffff;
		--grey-blue: rgba(2, 0, 52, 0.58);
	}
	* {
		box-sizing: border-box;
		margin: 0;
		padding: 0;
		-webkit-font-smoothing: antialiased;
		-moz-osx-font-smoothing: grayscale;
	}
	body {
		font-family: 'Space Grotesk', sans-serif;
		background-color: var(--background);
		color: var(--antidote-black);
		text-transform: lowercase;
		min-height: 100vh;
		font-size: 1rem;
		line-height: 1.5;
	}
	.navbar {
		background-color: var(--background);
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1.5rem 5%;
		position: sticky;
		top: 0;
		z-index: 100;
	}
	.navbar::after {
		content: '';
		position: absolute;
		bottom: 0;
		left: 5%;
		right: 5%;
		height: 1px;
		background: var(--antidote-black);
		opacity: 0.15;
	}
	.navbar-logo {
		display: flex;
		align-items: center;
		gap: 10px;
		text-decoration: none;
		color: var(--antidote-black);
		font-weight: 400;
		font-size: 1.25rem;
		letter-spacing: -0.5px;
	}
	.navbar-right {
		display: flex;
		align-items: center;
		gap: 2rem;
	}
	.nav-link {
		color: var(--antidote-black);
		text-decoration: none;
		font-weight: 400;
		font-size: 1rem;
		padding: 0.5rem 1rem;
		transition: color 0.3s;
		letter-spacing: -0.5px;
	}
	.nav-link:hover { color: var(--primary); }
	.btn-pill {
		border: 2px solid var(--antidote-black);
		background-color: var(--background);
		color: var(--antidote-black);
		text-align: center;
		border-radius: 64px;
		padding: 0.75rem 2.5rem;
		font-family: 'Space Grotesk', sans-serif;
		font-weight: 500;
		font-size: 1rem;
		text-decoration: none;
		text-transform: lowercase;
		cursor: pointer;
		box-shadow: -3px 3px 0 0 var(--antidote-black);
		transition: all 0.3s ease-in-out;
		display: inline-flex;
		justify-content: center;
		align-items: center;
		height: 48px;
	}
	.btn-pill:hover {
		box-shadow: none;
		color: var(--primary);
	}
	.padding-global { padding-left: 5%; padding-right: 5%; }
	.divider { width: 100%; height: 1px; background: var(--antidote-black); opacity: 0.12; margin: 1rem 0; }

	.badge-coming-soon {
		background: #fef08a;
		color: #713f12;
		font-size: 0.68rem;
		font-weight: 700;
		padding: 3px 10px;
		border-radius: 100px;
		border: 1px solid rgba(113, 63, 18, 0.3);
		font-family: 'Space Mono', monospace;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		display: inline-flex;
		align-items: center;
		gap: 4px;
	}
	.skill-card.is-featured .badge-coming-soon {
		background: rgba(254, 240, 138, 0.22);
		color: #fef08a;
		border-color: rgba(254, 240, 138, 0.4);
	}
	.pwd-modal-overlay {
		position: fixed;
		top: 0; left: 0; right: 0; bottom: 0;
		background: rgba(2, 0, 52, 0.68);
		backdrop-filter: blur(8px);
		-webkit-backdrop-filter: blur(8px);
		z-index: 99999;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1.5rem;
	}
	.pwd-modal-box {
		background: var(--white);
		border: 2.5px solid var(--antidote-black);
		border-radius: 24px;
		padding: 2.5rem;
		max-width: 420px;
		width: 100%;
		box-shadow: -8px 8px 0 0 var(--antidote-black);
		position: relative;
		text-align: center;
		animation: modalPop 0.25s cubic-bezier(0.175, 0.885, 0.32, 1.275);
	}
	@keyframes modalPop {
		0% { transform: scale(0.9); opacity: 0; }
		100% { transform: scale(1); opacity: 1; }
	}
	.pwd-modal-close {
		position: absolute;
		top: 1.25rem; right: 1.25rem;
		background: none;
		border: none;
		font-size: 1.25rem;
		cursor: pointer;
		color: var(--antidote-black);
		font-weight: 700;
		line-height: 1;
	}
	.pwd-modal-icon { font-size: 2.5rem; margin-bottom: 0.75rem; }
	.pwd-modal-title {
		font-family: 'Space Grotesk', sans-serif;
		font-size: 1.6rem;
		font-weight: 700;
		letter-spacing: -0.02em;
		color: var(--antidote-black);
		margin-bottom: 0.5rem;
	}
	.pwd-modal-desc {
		font-size: 0.88rem;
		color: var(--grey-blue);
		line-height: 1.4;
		margin-bottom: 1.5rem;
	}
	.pwd-input {
		width: 100%;
		background: rgba(2, 0, 52, 0.04);
		border: 2px solid var(--antidote-black);
		border-radius: 12px;
		padding: 0.85rem 1rem;
		font-family: 'Space Mono', monospace;
		font-size: 0.95rem;
		color: var(--antidote-black);
		outline: none;
		margin-bottom: 0.75rem;
		box-sizing: border-box;
		transition: all 0.2s;
		text-align: center;
	}
	.pwd-input:focus {
		background: #ffffff;
		border-color: var(--primary);
		box-shadow: 0 0 0 3px rgba(0, 0, 255, 0.15);
	}
	.pwd-error {
		font-family: 'Space Mono', monospace;
		font-size: 0.75rem;
		color: #dc2626;
		font-weight: 700;
		margin-bottom: 0.75rem;
	}
	.pwd-submit-btn {
		width: 100%;
		background: var(--antidote-black);
		color: var(--white);
		border: 2px solid var(--antidote-black);
		border-radius: 64px;
		padding: 0.9rem;
		font-family: 'Space Grotesk', sans-serif;
		font-size: 1rem;
		font-weight: 700;
		text-transform: lowercase;
		cursor: pointer;
		box-shadow: -3px 3px 0 0 rgba(2, 0, 52, 0.3);
		transition: all 0.2s;
	}
	.pwd-submit-btn:hover {
		background: var(--primary);
		border-color: var(--primary);
		transform: translateY(-2px);
		box-shadow: -1px 1px 0 0 var(--antidote-black);
	}
	@keyframes shake {
		0%, 100% { transform: translateX(0); }
		20%, 60% { transform: translateX(-6px); }
		40%, 80% { transform: translateX(6px); }
	}
	.shake {
		animation: shake 0.4s ease-in-out;
		border-color: #dc2626 !important;
	}
`

const antidoteRobotCSS = `
	@keyframes robotFloat {
		0%, 100% { transform: translateY(0); }
		50% { transform: translateY(-12px); }
	}
	@keyframes robotEyeBlink {
		0%, 92%, 100% { opacity: 1; }
		95%, 97% { opacity: 0.1; }
	}
	@keyframes antennaPulse {
		0%, 100% { r: 3.5; opacity: 1; }
		50% { r: 5; opacity: 0.6; }
	}
	@keyframes armWave {
		0%, 100% { transform: rotate(0deg); }
		25% { transform: rotate(-8deg); }
		75% { transform: rotate(8deg); }
	}
	.robot-animated {
		animation: robotFloat 3s ease-in-out infinite;
	}
	.robot-eye { animation: robotEyeBlink 4s ease-in-out infinite; }
	.robot-eye-right { animation: robotEyeBlink 4s ease-in-out 0.15s infinite; }
	.robot-antenna { animation: antennaPulse 2s ease-in-out infinite; }
	.robot-arm-left { transform-origin: 3px 25px; animation: armWave 2.5s ease-in-out infinite; }
`

const antidoteNavbar = `
	<div class="navbar">
		<a href="/admin" class="navbar-logo">
			<svg width="30" height="30" viewBox="0 0 30 30" fill="none">
				<circle cx="15" cy="15" r="14" stroke="#020034" stroke-width="1.5" fill="none"/>
				<circle cx="11" cy="13" r="2" fill="#020034"/>
				<circle cx="19" cy="13" r="2" fill="#020034"/>
				<path d="M10 19 Q15 23 20 19" stroke="#020034" stroke-width="1.5" fill="none" stroke-linecap="round"/>
			</svg>
			<span>universal copilot</span>
		</a>
		<div class="navbar-right">
			<a href="/test" class="nav-link">demo</a>
			<a href="/admin/docs" class="nav-link">doc studio</a>
			<a href="/admin/configure" class="btn-pill">create bot</a>
		</div>
	</div>
`

const antidoteRobotSVG = `
	<svg width="90" height="108" viewBox="-12 -12 62 72" fill="none">
		<ellipse cx="22" cy="56" rx="16" ry="2" fill="#020034" opacity="0.1"/>
		<rect x="4" y="6" width="36" height="38" rx="6" fill="#020034"/>
		<rect x="4" y="6" width="36" height="3" rx="1.5" fill="#888eff" opacity="0.3"/>
		<rect class="robot-eye" x="13" y="21" width="5" height="6" rx="1.5" fill="#888eff"/>
		<rect class="robot-eye-right" x="26" y="21" width="5" height="6" rx="1.5" fill="#888eff"/>
		<rect x="17" y="33" width="10" height="2" rx="1" fill="#888eff" opacity="0.4"/>
		<rect x="10" y="43" width="8" height="11" rx="2" fill="#020034"/>
		<rect x="26" y="43" width="8" height="11" rx="2" fill="#020034"/>
		<g class="robot-arm-left"><rect x="-6" y="16" width="9" height="18" rx="4.5" fill="#020034"/></g>
		<rect x="41" y="16" width="9" height="18" rx="4.5" fill="#020034"/>
		<rect x="21" y="-6" width="2" height="13" rx="1" fill="#020034"/>
		<circle class="robot-antenna" cx="22" cy="-8" r="3.5" fill="#020034"/>
	</svg>
`

const antidoteFonts = `
	<link rel="preconnect" href="https://fonts.googleapis.com">
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
	<link href="https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@300;400;500;600;700&family=Space+Mono:wght@400;700&family=Inter:wght@300;400;500;600&display=swap" rel="stylesheet">
`

// ────────────────────────────────────────────────
// PAGE 1: /admin — Skills Catalog (NO form)
// ────────────────────────────────────────────────
func HandleAdminUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>universal copilot — teach your assistant anything</title>
	%s
	<style>
		%s
		%s

		.section-hero {
			text-align: center;
			padding-top: 4rem;
			padding-bottom: 3rem;
		}
		h1.hero-title {
			font-family: 'Space Grotesk', sans-serif;
			font-size: clamp(3rem, 8vw, 7.5rem);
			font-weight: 400;
			line-height: 1.1;
			color: var(--antidote-black);
			letter-spacing: -0.03em;
			margin-bottom: 1.5rem;
		}
		.hero-subtitle {
			font-size: 1.25rem;
			font-weight: 400;
			color: var(--antidote-black);
			font-style: italic;
			max-width: 600px;
			margin: 0 auto 2.5rem;
			line-height: 1.4;
			opacity: 0.85;
		}
		.section-skills { padding-top: 3rem; padding-bottom: 4rem; }
		.section-heading {
			font-family: 'Space Grotesk', sans-serif;
			font-size: clamp(2rem, 5vw, 4.5rem);
			font-weight: 400;
			line-height: 1.15;
			color: var(--antidote-black);
			letter-spacing: -0.02em;
			margin-bottom: 0.75rem;
		}
		.section-subtext {
			font-size: 1.125rem;
			color: var(--grey-blue);
			margin-bottom: 2.5rem;
		}
		.skills-grid {
			display: grid;
			grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
			gap: 1.5rem;
		}
		.skill-card {
			background: var(--white);
			border: 2px solid var(--antidote-black);
			border-radius: 20px;
			padding: 2.25rem 2rem;
			cursor: pointer;
			transition: transform 0.25s ease, box-shadow 0.25s ease, border-color 0.25s ease;
			display: flex;
			flex-direction: column;
			justify-content: space-between;
			min-height: 280px;
			position: relative;
			text-decoration: none;
			color: inherit;
			box-shadow: -4px 4px 0 0 var(--antidote-black);
		}
		.skill-card:hover {
			transform: translateY(-4px);
			box-shadow: -6px 6px 0 0 var(--antidote-black);
		}
		.skill-card:hover .skill-name { color: var(--primary); }
		.skill-card.is-featured {
			background: var(--antidote-black);
			border: 2px solid var(--antidote-black);
			border-radius: 20px;
			box-shadow: -4px 4px 0 0 rgba(2, 0, 52, 0.4);
		}
		.skill-card.is-featured:hover {
			transform: translateY(-4px);
			box-shadow: -6px 6px 0 0 rgba(2, 0, 52, 0.6);
		}
		.skill-card.is-featured .skill-name,
		.skill-card.is-featured .skill-desc { color: var(--white); }
		.skill-card.is-featured .skill-desc { opacity: 0.75; }
		.skill-card.is-featured .skill-badge {
			background: rgba(136, 142, 255, 0.25);
			color: #b4b8ff;
			border-color: rgba(136, 142, 255, 0.3);
		}
		.skill-card.is-featured .skill-cta { color: var(--background); }
		.skill-card.is-featured:hover .skill-name { color: var(--background); }
		.skill-top-row {
			display: flex;
			justify-content: space-between;
			align-items: flex-start;
			margin-bottom: 2rem;
		}
		.skill-icon {
			width: 48px;
			height: 48px;
			border-radius: 50%%;
			display: flex;
			align-items: center;
			justify-content: center;
			font-size: 22px;
			background: rgba(2, 0, 52, 0.06);
		}
		.skill-icon.is-dark {
			background: radial-gradient(circle at 32%% 28%%, #29292e 0%%, #16161a 75%%);
		}
		.skill-badge {
			background: rgba(2, 0, 52, 0.06);
			color: var(--antidote-black);
			font-size: 0.7rem;
			font-weight: 500;
			padding: 4px 12px;
			border-radius: 100px;
			border: 1px solid rgba(2, 0, 52, 0.1);
		}
		.skill-name {
			font-family: 'Space Grotesk', sans-serif;
			font-size: 1.5rem;
			font-weight: 700;
			color: var(--antidote-black);
			letter-spacing: -0.3px;
			margin-bottom: 0.5rem;
			transition: color 0.3s;
		}
		.skill-desc {
			font-size: 0.9rem;
			color: var(--grey-blue);
			line-height: 1.5;
			margin-bottom: 1.5rem;
		}
		.skill-cta {
			font-size: 0.875rem;
			font-weight: 600;
			color: var(--primary);
			display: flex;
			align-items: center;
			gap: 6px;
			transition: gap 0.2s;
		}
		.skill-card:hover .skill-cta { gap: 10px; }
		@media (max-width: 768px) {
			h1.hero-title { font-size: 2.8rem; }
			.skills-grid { grid-template-columns: 1fr; gap: 1.25rem; }
		}
	</style>
</head>
<body>
	%s

	<main>
		<section class="section-hero padding-global">
			<div class="robot-animated" style="margin-bottom: 1.5rem;">
				%s
			</div>
			<h1 class="hero-title">teach your assistant<br>anything</h1>
			<p class="hero-subtitle"><em>skills give your agents new capabilities. browse the catalog, pick what you need, and deploy with a single embed.</em></p>
			<a href="/admin/configure" class="btn-pill">get started →</a>
		</section>

		<div class="padding-global"><div class="divider"></div></div>

		<section class="section-skills padding-global">
			<h2 class="section-heading">brand communication that<br>engages all senses</h2>
			<p class="section-subtext">put simply, when you look good, we look good ✨</p>

			<div class="skills-grid">
				<a href="/admin/configure" class="skill-card is-featured">
					<div>
						<div class="skill-top-row">
							<div class="skill-icon is-dark">
								<svg width="24" height="28" viewBox="-10 -8 58 64" fill="none">
									<rect x="4" y="6" width="36" height="38" rx="5" fill="#2b2b30"/>
									<rect x="14" y="22" width="4" height="5" rx="1" fill="#e8c468"/>
									<rect x="26" y="22" width="4" height="5" rx="1" fill="#e8c468"/>
									<rect x="10" y="42" width="8" height="10" rx="1.5" fill="#2b2b30"/>
									<rect x="26" y="42" width="8" height="10" rx="1.5" fill="#2b2b30"/>
								</svg>
							</div>
							<div style="display: flex; gap: 6px; align-items: center;">
								<span class="skill-badge">★ core</span>
								<span class="badge-live" style="background: rgba(34, 197, 94, 0.15); color: #4ade80; border: 1px solid rgba(34, 197, 94, 0.3); font-size: 0.68rem; padding: 2px 7px; border-radius: 9999px; font-weight: 500;">✓ ready</span>
							</div>
						</div>
						<div class="skill-name">ai copilot</div>
						<div class="skill-desc">interactive floating mascot that greets visitors, answers questions from your docs, and auto-books meetings.</div>
					</div>
					<div class="skill-cta">configure copilot →</div>
				</a>

				<a href="/admin/configure" class="skill-card" onclick="openSkill('/admin/configure'); return false;">
					<div>
						<div class="skill-top-row">
							<div class="skill-icon">✉️</div>
							<div style="display: flex; gap: 6px; align-items: center;">
								<span class="skill-badge">email</span>
								<span class="badge-coming-soon">🔒 coming soon</span>
							</div>
						</div>
						<div class="skill-name">gmail</div>
						<div class="skill-desc">drafting, automated confirmations, meeting dispatches, and attendee invites.</div>
					</div>
					<div class="skill-cta">unlock with password 🔒 →</div>
				</a>

				<a href="/admin/docs" class="skill-card" onclick="openSkill('/admin/docs'); return false;">
					<div>
						<div class="skill-top-row">
							<div class="skill-icon">📄</div>
							<div style="display: flex; gap: 6px; align-items: center;">
								<span class="skill-badge">documents</span>
								<span class="badge-coming-soon">🔒 coming soon</span>
							</div>
						</div>
						<div class="skill-name">google docs</div>
						<div class="skill-desc">generate, format, and export notes, proposals, and briefs directly into your google drive.</div>
					</div>
					<div class="skill-cta">unlock with password 🔒 →</div>
				</a>

				<a href="/admin/configure" class="skill-card" onclick="openSkill('/admin/configure'); return false;">
					<div>
						<div class="skill-top-row">
							<div class="skill-icon">📅</div>
							<div style="display: flex; gap: 6px; align-items: center;">
								<span class="skill-badge">calendar</span>
								<span class="badge-coming-soon">🔒 coming soon</span>
							</div>
						</div>
						<div class="skill-name">google calendar</div>
						<div class="skill-desc">view events, create and manage, check real-time availability for auto-booking.</div>
					</div>
					<div class="skill-cta">unlock with password 🔒 →</div>
				</a>
			</div>
		</section>
	</main>

	<!-- Password Gate Modal -->
	<div id="pwd-modal-overlay" class="pwd-modal-overlay" style="display: none;">
		<div class="pwd-modal-box">
			<button class="pwd-modal-close" onclick="closePwdModal()" type="button">✕</button>
			<div class="pwd-modal-icon">🔐</div>
			<h3 class="pwd-modal-title">enter password</h3>
			<p class="pwd-modal-desc">this skill is coming soon. please enter the access password to open.</p>
			<form id="pwd-modal-form" onsubmit="handlePwdSubmit(event)">
				<input type="password" id="pwd-modal-input" class="pwd-input" placeholder="Enter password..." autocomplete="current-password" required />
				<div id="pwd-error-msg" class="pwd-error" style="display: none;">❌ incorrect password</div>
				<button type="submit" class="pwd-submit-btn">unlock &amp; open →</button>
			</form>
		</div>
	</div>

	<script>
		const ACCESS_PASSWORD = 'Aysuh@90052';
		const AUTH_KEY = 'uvico_auth_unlocked';
		let pendingDestination = '';

		function openSkill(targetUrl) {
			if (sessionStorage.getItem(AUTH_KEY) === 'true') {
				window.location.href = targetUrl;
				return;
			}
			pendingDestination = targetUrl;
			const modal = document.getElementById('pwd-modal-overlay');
			const input = document.getElementById('pwd-modal-input');
			const errorMsg = document.getElementById('pwd-error-msg');
			if (errorMsg) errorMsg.style.display = 'none';
			if (input) input.value = '';
			if (modal) modal.style.display = 'flex';
			setTimeout(function() { if (input) input.focus(); }, 80);
		}

		function closePwdModal() {
			const modal = document.getElementById('pwd-modal-overlay');
			if (modal) modal.style.display = 'none';
		}

		function handlePwdSubmit(e) {
			e.preventDefault();
			const input = document.getElementById('pwd-modal-input');
			const errorMsg = document.getElementById('pwd-error-msg');
			if (input && input.value === ACCESS_PASSWORD) {
				sessionStorage.setItem(AUTH_KEY, 'true');
				closePwdModal();
				if (pendingDestination) {
					window.location.href = pendingDestination;
				}
			} else {
				if (errorMsg) errorMsg.style.display = 'block';
				if (input) {
					input.classList.add('shake');
					setTimeout(function() { input.classList.remove('shake'); }, 500);
					input.select();
				}
			}
		}

		document.addEventListener('keydown', function(e) {
			if (e.key === 'Escape') closePwdModal();
		});
	</script>
</body>
</html>`, antidoteFonts, antidoteCSS, antidoteRobotCSS, antidoteNavbar, antidoteRobotSVG)
}

// ────────────────────────────────────────────────
// PAGE 2: /admin/configure — Create & Configure AI Copilot Form
// ────────────────────────────────────────────────
func HandleConfigureUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	defaultEmail := os.Getenv("CALENDAR_OWNER_EMAIL")
	if defaultEmail == "" {
		defaultEmail = "ayushmansingh2512@gmail.com"
	}
	defaultAPIKey := os.Getenv("GEMINI_API_KEY")

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>universal copilot — create your ai assistant</title>
	%s
	<style>
		%s
		%s

		.section-config {
			padding-top: 2.5rem;
			padding-bottom: 4rem;
			max-width: 840px;
			margin: 0 auto;
		}
		.config-hero {
			text-align: center;
			margin-bottom: 2rem;
		}
		.config-title {
			font-family: 'Space Grotesk', sans-serif;
			font-size: clamp(2rem, 4.5vw, 3.8rem);
			font-weight: 700;
			letter-spacing: -0.02em;
			color: var(--antidote-black);
			margin-bottom: 0.5rem;
			line-height: 1.1;
		}
		.config-subtitle {
			font-size: 1.05rem;
			color: var(--grey-blue);
			max-width: 620px;
			margin: 0 auto;
			line-height: 1.4;
		}

		.tab-nav {
			display: flex;
			gap: 0.75rem;
			justify-content: center;
			margin-bottom: 2rem;
			flex-wrap: wrap;
		}
		.tab-chip {
			display: inline-flex;
			align-items: center;
			gap: 8px;
			padding: 0.6rem 1.4rem;
			border-radius: 999px;
			font-family: 'Space Mono', monospace;
			font-size: 0.8rem;
			font-weight: 700;
			text-decoration: none;
			text-transform: lowercase;
			border: 2px solid var(--antidote-black);
			color: var(--antidote-black);
			background: var(--white);
			box-shadow: -2px 2px 0 0 var(--antidote-black);
			transition: all 0.2s ease;
		}
		.tab-chip.active {
			background: var(--antidote-black);
			color: var(--white);
			box-shadow: none;
		}
		.tab-chip:hover:not(.active) {
			transform: translateY(-2px);
			box-shadow: -4px 4px 0 0 var(--antidote-black);
			color: var(--primary);
		}

		.form-card {
			background: var(--white);
			border: 2.5px solid var(--antidote-black);
			border-radius: 24px;
			padding: 2.5rem;
			box-shadow: -6px 6px 0 0 var(--antidote-black);
		}
		.form-group {
			margin-bottom: 1.6rem;
		}
		.form-label-row {
			display: flex;
			justify-content: space-between;
			align-items: baseline;
			margin-bottom: 0.45rem;
		}
		.form-label {
			font-family: 'Space Mono', monospace;
			font-size: 0.78rem;
			font-weight: 700;
			text-transform: uppercase;
			letter-spacing: 0.05em;
			color: var(--antidote-black);
		}
		.form-hint {
			font-size: 0.75rem;
			color: var(--grey-blue);
			font-family: 'Space Grotesk', sans-serif;
		}
		.form-input, .form-textarea {
			width: 100%%;
			background: rgba(2, 0, 52, 0.03);
			border: 2px solid var(--antidote-black);
			border-radius: 12px;
			padding: 0.85rem 1.1rem;
			font-family: 'Space Mono', monospace;
			font-size: 0.9rem;
			color: var(--antidote-black);
			outline: none;
			transition: all 0.2s;
			box-sizing: border-box;
		}
		.form-input:focus, .form-textarea:focus {
			background: #ffffff;
			border-color: var(--primary);
			box-shadow: 0 0 0 3px rgba(0, 0, 255, 0.12);
		}

		.info-box {
			background: rgba(0, 0, 255, 0.06);
			border: 1.5px solid rgba(0, 0, 255, 0.25);
			border-radius: 10px;
			padding: 0.75rem 1rem;
			margin-top: 0.6rem;
			font-size: 0.75rem;
			color: var(--antidote-black);
			line-height: 1.5;
			font-family: 'Space Mono', monospace;
		}
		.info-box code {
			background: #ffffff;
			padding: 2px 6px;
			border-radius: 4px;
			border: 1px solid rgba(2, 0, 52, 0.15);
			font-size: 0.72rem;
		}

		.file-dropzone {
			border: 2px dashed var(--antidote-black);
			border-radius: 14px;
			padding: 1.75rem 1.5rem;
			text-align: center;
			background: rgba(2, 0, 52, 0.02);
			cursor: pointer;
			transition: all 0.2s;
			position: relative;
		}
		.file-dropzone:hover {
			background: rgba(0, 0, 255, 0.04);
			border-color: var(--primary);
		}
		.file-dropzone input[type="file"] {
			position: absolute;
			top: 0;
			left: 0;
			width: 100%%;
			height: 100%%;
			opacity: 0;
			cursor: pointer;
		}
		.file-drop-icon { font-size: 2rem; margin-bottom: 0.35rem; }
		.file-drop-text {
			font-family: 'Space Mono', monospace;
			font-size: 0.85rem;
			color: var(--antidote-black);
			font-weight: 700;
		}
		.file-drop-subtext {
			font-size: 0.75rem;
			color: var(--grey-blue);
			margin-top: 4px;
		}
		.file-chosen-banner {
			display: none;
			margin-top: 8px;
			font-family: 'Space Mono', monospace;
			font-size: 0.78rem;
			font-weight: 700;
			color: #0d8a4e;
			background: rgba(13, 138, 78, 0.1);
			padding: 4px 10px;
			border-radius: 6px;
		}

		.or-divider {
			display: flex;
			align-items: center;
			text-align: center;
			margin: 1.25rem 0;
			color: var(--grey-blue);
			font-family: 'Space Mono', monospace;
			font-size: 0.72rem;
			font-weight: 700;
			letter-spacing: 0.05em;
		}
		.or-divider::before, .or-divider::after {
			content: '';
			flex: 1;
			border-bottom: 1px solid rgba(2, 0, 52, 0.15);
		}
		.or-divider span { padding: 0 1rem; }

		.btn-submit {
			width: 100%%;
			background: var(--antidote-black);
			color: var(--white);
			border: 2px solid var(--antidote-black);
			border-radius: 64px;
			padding: 1.1rem;
			font-family: 'Space Grotesk', sans-serif;
			font-size: 1.15rem;
			font-weight: 700;
			text-transform: lowercase;
			cursor: pointer;
			box-shadow: -4px 4px 0 0 rgba(2, 0, 52, 0.3);
			transition: all 0.25s ease;
			display: flex;
			align-items: center;
			justify-content: center;
			gap: 10px;
			margin-top: 2rem;
		}
		.btn-submit:hover {
			background: var(--primary);
			border-color: var(--primary);
			box-shadow: -2px 2px 0 0 var(--antidote-black);
			transform: translateY(-2px);
		}
		.btn-submit:active {
			transform: translateY(0);
			box-shadow: none;
		}
	</style>
</head>
<body>
	%s

	<main>
		<section class="section-config padding-global">
			<!-- Mode switcher tabs -->
			<div class="tab-nav">
				<a href="/admin/configure" class="tab-chip active">🤖 configure ai copilot</a>
				<a href="/admin/docs" class="tab-chip">📄 google docs studio</a>
				<a href="/admin" class="tab-chip">🏛️ all skills</a>
			</div>

			<div class="config-hero">
				<div class="robot-animated" style="margin-bottom: 1rem; display: inline-block;">
					%s
				</div>
				<h1 class="config-title">create your ai copilot</h1>
				<p class="config-subtitle">train your personal assistant on your resume, pdf, or portfolio data. get an embed code to drop onto any website in under a minute.</p>
			</div>

			<div class="form-card">
				<form action="/admin/ingest" method="POST" enctype="multipart/form-data">
					<!-- Field 1: App ID -->
					<div class="form-group">
						<div class="form-label-row">
							<label class="form-label" for="app_id">01 — Unique App ID *</label>
							<span class="form-hint">identifier used in your embed snippet</span>
						</div>
						<input type="text" id="app_id" name="app_id" class="form-input" placeholder="e.g. ayushman_portfolio, rahul_dev, clinic_assistant" required />
					</div>

					<!-- Field 2: Client Name -->
					<div class="form-group">
						<div class="form-label-row">
							<label class="form-label" for="client_name">02 — Your Name / Brand Name</label>
							<span class="form-hint">displayed to website visitors</span>
						</div>
						<input type="text" id="client_name" name="client_name" class="form-input" placeholder="e.g. Ayushman Singh" />
					</div>

					<!-- Field 3: Calendar & Notification Email -->
					<div class="form-group">
						<div class="form-label-row">
							<label class="form-label" for="calendar_email">03 — Calendar & Notification Email</label>
							<span class="form-hint">for auto-booking & meeting alerts</span>
						</div>
						<input type="email" id="calendar_email" name="calendar_email" class="form-input" placeholder="e.g. yourname@gmail.com" value="%s" />
						<div class="info-box">
							📅 <strong>To enable live Google Calendar booking:</strong> Share your Google Calendar with <code>calendar-copilot@ai-interviewer-475814.iam.gserviceaccount.com</code> with permission <em>Make changes to events</em>.
						</div>
					</div>

					<!-- Field 4: Security PIN -->
					<div class="form-group">
						<div class="form-label-row">
							<label class="form-label" for="app_passcode">04 — Account Security PIN *</label>
							<span class="form-hint">needed to update or re-index your bot later</span>
						</div>
						<input type="password" id="app_passcode" name="app_passcode" class="form-input" placeholder="Create a secret PIN..." required />
					</div>

					<!-- Field 5: Gemini API Key -->
					<div class="form-group">
						<div class="form-label-row">
							<label class="form-label" for="gemini_api_key">05 — Google Gemini API Key *</label>
							<span class="form-hint">AES-256 encrypted at rest</span>
						</div>
						<input type="password" id="gemini_api_key" name="gemini_api_key" class="form-input" placeholder="AIzaSy..." value="%s" required />
						<div style="margin-top: 0.4rem; font-size: 0.75rem;">
							<a href="https://aistudio.google.com/app/apikey" target="_blank" style="color: var(--primary); text-decoration: none; font-weight: 600;">Get a free Gemini API key from Google AI Studio →</a>
						</div>
					</div>

					<!-- Field 6 & 7: Knowledge Doc Upload or Text -->
					<div class="form-group">
						<div class="form-label-row">
							<label class="form-label">06 — Upload Knowledge Doc / PDF / Resume</label>
							<span class="form-hint">accepts .pdf, .txt, .md</span>
						</div>
						<div class="file-dropzone">
							<input type="file" id="doc_file" name="doc_file" accept=".pdf,.txt,.md" onchange="handleFileSelect(this)" />
							<div class="file-drop-icon">📄</div>
							<div class="file-drop-text" id="file-drop-text">click to browse or drag &amp; drop document</div>
							<div class="file-drop-subtext">pdf, markdown, or plain text documents</div>
							<div class="file-chosen-banner" id="file-chosen-banner"></div>
						</div>

						<div class="or-divider">
							<span>OR PASTE PLAIN TEXT INFORMATION</span>
						</div>

						<textarea name="raw_text" rows="5" class="form-textarea" placeholder="Paste website details, FAQs, skills, bio, project history, or contact information..."></textarea>
					</div>

					<button type="submit" class="btn-submit">
						🚀 Ingest Data &amp; Generate Embed Code
					</button>
				</form>
			</div>
		</section>
	</main>

	<script>

		function handleFileSelect(input) {
			const banner = document.getElementById('file-chosen-banner');
			const label = document.getElementById('file-drop-text');
			if (input.files && input.files[0]) {
				const f = input.files[0];
				banner.style.display = 'inline-block';
				banner.textContent = '✅ Selected: ' + f.name + ' (' + (f.size / 1024).toFixed(1) + ' KB)';
				label.textContent = f.name;
			} else {
				banner.style.display = 'none';
				label.textContent = 'click to browse or drag & drop document';
			}
		}
	</script>
</body>
</html>`, antidoteFonts, antidoteCSS, antidoteRobotCSS, antidoteNavbar, antidoteRobotSVG, defaultEmail, defaultAPIKey)
}

// ────────────────────────────────────────────────
// PAGE 3: /admin/docs — Google Docs Creator Studio
// ────────────────────────────────────────────────
func HandleDocsStudioUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	defaultEmail := os.Getenv("CALENDAR_OWNER_EMAIL")
	if defaultEmail == "" {
		defaultEmail = "ayushmansingh2512@gmail.com"
	}
	defaultAPIKey := os.Getenv("GEMINI_API_KEY")

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>universal copilot — google docs creator</title>
	%s
	<style>
		%s
		%s

		.section-studio {
			padding-top: 2.5rem;
			padding-bottom: 4rem;
			max-width: 960px;
			margin: 0 auto;
		}
		.studio-hero {
			text-align: center;
			margin-bottom: 2rem;
		}
		.studio-title {
			font-family: 'Space Grotesk', sans-serif;
			font-size: clamp(2rem, 4.5vw, 3.8rem);
			font-weight: 700;
			letter-spacing: -0.02em;
			color: var(--antidote-black);
			margin-bottom: 0.5rem;
			line-height: 1.1;
		}
		.studio-subtitle {
			font-size: 1.05rem;
			color: var(--grey-blue);
			max-width: 620px;
			margin: 0 auto;
			line-height: 1.4;
		}

		/* Config Bar */
		.config-strip {
			background: var(--white);
			border: 2px solid var(--antidote-black);
			border-radius: 18px;
			padding: 1.25rem 1.5rem;
			margin-bottom: 1.5rem;
			box-shadow: -4px 4px 0 0 var(--antidote-black);
			display: grid;
			grid-template-columns: 1.2fr 1fr 1fr auto;
			gap: 0.85rem;
			align-items: center;
		}
		.config-field {
			display: flex;
			flex-direction: column;
			gap: 4px;
		}
		.config-field label {
			font-family: 'Space Mono', monospace;
			font-size: 0.65rem;
			font-weight: 700;
			text-transform: uppercase;
			color: var(--antidote-black);
			opacity: 0.7;
			letter-spacing: 0.05em;
		}
		.config-field input {
			background: rgba(2, 0, 52, 0.04);
			border: 1.5px solid var(--antidote-black);
			border-radius: 8px;
			padding: 0.6rem 0.85rem;
			font-family: 'Space Mono', monospace;
			font-size: 0.85rem;
			color: var(--antidote-black);
			outline: none;
			transition: all 0.2s ease;
		}
		.config-field input:focus {
			background: #ffffff;
			border-color: var(--primary);
			box-shadow: 0 0 0 2px rgba(0, 0, 255, 0.15);
		}
		.config-status {
			display: flex;
			align-items: center;
			gap: 6px;
			font-family: 'Space Mono', monospace;
			font-size: 0.75rem;
			font-weight: 700;
			color: #0d8a4e;
			background: rgba(13, 138, 78, 0.1);
			border: 1px solid rgba(13, 138, 78, 0.2);
			padding: 0.5rem 1rem;
			border-radius: 100px;
			align-self: flex-end;
			height: 40px;
			box-sizing: border-box;
		}

		/* Chat Card */
		.chat-card {
			background: #18181c;
			border: 2px solid var(--antidote-black);
			border-radius: 24px;
			box-shadow: -6px 6px 0 0 var(--antidote-black);
			display: flex;
			flex-direction: column;
			height: 600px;
			overflow: hidden;
		}
		.chat-header {
			background: #111114;
			border-bottom: 1px solid rgba(255, 255, 255, 0.08);
			padding: 1rem 1.5rem;
			display: flex;
			justify-content: space-between;
			align-items: center;
		}
		.chat-header-left {
			display: flex;
			align-items: center;
			gap: 12px;
		}
		.chat-header-title {
			font-family: 'Space Grotesk', sans-serif;
			font-size: 1.15rem;
			font-weight: 700;
			color: #ffffff;
			letter-spacing: -0.3px;
		}
		.chat-header-badge {
			font-family: 'Space Mono', monospace;
			font-size: 0.65rem;
			background: rgba(136, 142, 255, 0.2);
			color: #b4b8ff;
			padding: 3px 10px;
			border-radius: 100px;
			border: 1px solid rgba(136, 142, 255, 0.3);
		}

		/* Messages container */
		.chat-messages {
			flex: 1;
			overflow-y: auto;
			padding: 1.5rem;
			display: flex;
			flex-direction: column;
			gap: 1rem;
			scrollbar-width: thin;
			scrollbar-color: rgba(255, 255, 255, 0.2) transparent;
		}
		.chat-bubble {
			max-width: 85%%;
			padding: 1rem 1.25rem;
			border-radius: 16px;
			font-size: 0.92rem;
			line-height: 1.6;
			text-transform: none;
			word-break: break-word;
		}
		.user-bubble {
			align-self: flex-end;
			background: var(--primary);
			color: #ffffff;
			border-bottom-right-radius: 4px;
			box-shadow: 0 4px 14px rgba(0, 0, 255, 0.3);
		}
		.bot-bubble {
			align-self: flex-start;
			background: #232328;
			border: 1px solid rgba(255, 255, 255, 0.08);
			color: #e2e2e8;
			border-bottom-left-radius: 4px;
			box-shadow: 0 4px 16px rgba(0, 0, 0, 0.35);
		}
		.error-bubble {
			background: rgba(235, 87, 87, 0.15);
			border-color: rgba(235, 87, 87, 0.4);
			color: #ff9b9b;
		}
		.bot-msg-title {
			font-family: 'Space Grotesk', sans-serif;
			font-size: 1.1rem;
			font-weight: 700;
			color: #ffffff;
			margin-bottom: 0.5rem;
			display: flex;
			align-items: center;
			gap: 8px;
		}
		.bot-msg-preview {
			font-size: 0.85rem;
			color: #bbbbc4;
			max-height: 220px;
			overflow-y: auto;
			padding: 0.85rem;
			background: rgba(0, 0, 0, 0.35);
			border-radius: 10px;
			border: 1px solid rgba(255, 255, 255, 0.06);
			margin: 0.75rem 0;
			white-space: pre-wrap;
			line-height: 1.55;
			font-family: 'Space Grotesk', sans-serif;
		}
		.bot-notice {
			font-size: 0.75rem;
			color: #f59e0b;
			margin-top: 4px;
		}
		.doc-actions {
			margin-top: 0.85rem;
		}
		.doc-btn-primary {
			display: inline-flex;
			align-items: center;
			gap: 8px;
			background: var(--background);
			color: var(--antidote-black);
			border: 2px solid var(--antidote-black);
			padding: 0.7rem 1.4rem;
			border-radius: 10px;
			font-family: 'Space Grotesk', sans-serif;
			font-weight: 700;
			font-size: 0.9rem;
			text-decoration: none;
			box-shadow: -3px 3px 0 0 var(--antidote-black);
			transition: all 0.2s ease;
		}
		.doc-btn-primary:hover {
			transform: translateY(-2px);
			box-shadow: -5px 5px 0 0 var(--antidote-black);
			background: #ffffff;
		}

		/* Prompt Chips */
		.prompt-chips {
			display: flex;
			gap: 8px;
			flex-wrap: wrap;
			margin-top: 0.85rem;
		}
		.prompt-chip {
			background: rgba(255, 255, 255, 0.06);
			border: 1px solid rgba(255, 255, 255, 0.12);
			color: #d1d1d8;
			padding: 6px 14px;
			border-radius: 100px;
			font-size: 0.78rem;
			cursor: pointer;
			transition: all 0.2s;
		}
		.prompt-chip:hover {
			background: rgba(136, 142, 255, 0.25);
			border-color: #888eff;
			color: #ffffff;
			transform: translateY(-1px);
		}

		/* Input Section */
		.chat-input-container {
			background: #111114;
			border-top: 1px solid rgba(255, 255, 255, 0.08);
			padding: 1rem 1.25rem;
		}
		.chat-input-form {
			display: flex;
			gap: 10px;
		}
		.chat-input-form input {
			flex: 1;
			background: #202026;
			border: 1.5px solid rgba(255, 255, 255, 0.12);
			border-radius: 14px;
			padding: 0.9rem 1.25rem;
			font-family: 'Space Grotesk', sans-serif;
			font-size: 0.95rem;
			color: #ffffff;
			outline: none;
			transition: all 0.2s;
			text-transform: none;
		}
		.chat-input-form input:focus {
			border-color: var(--background);
			box-shadow: 0 0 0 2px rgba(136, 142, 255, 0.2);
		}
		.chat-send-btn {
			background: var(--background);
			color: var(--antidote-black);
			border: 2px solid var(--antidote-black);
			border-radius: 14px;
			padding: 0 1.6rem;
			font-family: 'Space Grotesk', sans-serif;
			font-size: 0.95rem;
			font-weight: 700;
			cursor: pointer;
			box-shadow: -3px 3px 0 0 var(--antidote-black);
			transition: all 0.2s;
			white-space: nowrap;
			display: flex;
			align-items: center;
			gap: 6px;
		}
		.chat-send-btn:hover {
			transform: translateY(-2px);
			box-shadow: -5px 5px 0 0 var(--antidote-black);
			background: #ffffff;
		}
		.chat-send-btn:disabled {
			opacity: 0.5;
			cursor: not-allowed;
			transform: none;
			box-shadow: none;
		}

		/* Loading Dots */
		.typing-indicator {
			display: inline-flex;
			gap: 4px;
			align-items: center;
			padding: 0.4rem 0.6rem;
		}
		.typing-dot {
			width: 6px;
			height: 6px;
			background: #888eff;
			border-radius: 50%%;
			animation: dotPulse 1.4s infinite ease-in-out both;
		}
		.typing-dot:nth-child(1) { animation-delay: -0.32s; }
		.typing-dot:nth-child(2) { animation-delay: -0.16s; }
		@keyframes dotPulse {
			0%%, 80%%, 100%% { transform: scale(0); }
			40%% { transform: scale(1); }
		}

		.tab-nav {
			display: flex;
			gap: 0.75rem;
			justify-content: center;
			margin-bottom: 2rem;
			flex-wrap: wrap;
		}
		.tab-chip {
			display: inline-flex;
			align-items: center;
			gap: 8px;
			padding: 0.6rem 1.4rem;
			border-radius: 999px;
			font-family: 'Space Mono', monospace;
			font-size: 0.8rem;
			font-weight: 700;
			text-decoration: none;
			text-transform: lowercase;
			border: 2px solid var(--antidote-black);
			color: var(--antidote-black);
			background: var(--white);
			box-shadow: -2px 2px 0 0 var(--antidote-black);
			transition: all 0.2s ease;
		}
		.tab-chip.active {
			background: var(--antidote-black);
			color: var(--white);
			box-shadow: none;
		}
		.tab-chip:hover:not(.active) {
			transform: translateY(-2px);
			box-shadow: -4px 4px 0 0 var(--antidote-black);
			color: var(--primary);
		}

		@media (max-width: 768px) {
			.config-strip { grid-template-columns: 1fr; }
			.chat-card { height: 500px; }
			.chat-bubble { max-width: 95%%; }
		}
	</style>
</head>
<body>
	%s

	<main>
		<section class="section-studio padding-global">
			<!-- Mode switcher tabs -->
			<div class="tab-nav">
				<a href="/admin/configure" class="tab-chip">🤖 configure ai copilot</a>
				<a href="/admin/docs" class="tab-chip active">📄 google docs studio</a>
				<a href="/admin" class="tab-chip">🏛️ all skills</a>
			</div>

			<div class="studio-hero">
				<div class="robot-animated" style="margin-bottom: 1rem; display: inline-block;">
					%s
				</div>
				<h1 class="studio-title">universal docs copilot</h1>
				<p class="studio-subtitle">tell the bot what document to write. it will generate the complete content and create an editable google doc in your drive instantly.</p>
			</div>

			<!-- Control Bar: API Key, Email, and Folder -->
			<div class="config-strip">
				<div class="config-field">
					<label>🔑 GEMINI API KEY</label>
					<input type="password" id="gemini_api_key_input" placeholder="AIzaSy..." value="%s" />
				</div>
				<div class="config-field">
					<label>📧 GOOGLE EMAIL</label>
					<input type="email" id="user_email_input" placeholder="yourname@gmail.com" value="%s" />
				</div>
				<div class="config-field">
					<label>📁 FOLDER LINK / ID (COPILOT)</label>
					<input type="text" id="user_folder_input" placeholder="Paste Drive link or ID" />
				</div>
				<div class="config-status">
					<span>●</span> Connected
				</div>
			</div>

			<div style="background: rgba(2, 0, 52, 0.04); border: 1.5px dashed rgba(2, 0, 52, 0.25); border-radius: 12px; padding: 0.75rem 1.25rem; margin-bottom: 1.5rem; font-family: 'Space Mono', monospace; font-size: 0.75rem; color: var(--antidote-black); line-height: 1.5;">
				💡 <strong>Google Drive Sync:</strong> Ensure your folder (e.g. <code>copilot</code>) is shared with <code style="background: #ffffff; padding: 2px 6px; border-radius: 4px; border: 1px solid rgba(2,0,52,0.15);">calendar-copilot@ai-interviewer-475814.iam.gserviceaccount.com</code> (Editor). You can also click <strong>Copy link</strong> on your folder in Drive and paste it in the box above!
			</div>

			<!-- Chat Studio Card -->
			<div class="chat-card">
				<div class="chat-header">
					<div class="chat-header-left">
						<span style="font-size: 1.25rem;">📄</span>
						<span class="chat-header-title">google docs assistant</span>
						<span class="chat-header-badge">gemini 2.5 flash + docs api</span>
					</div>
					<a href="/admin" style="color: #a0a0aa; font-size: 0.8rem; text-decoration: none; font-family: 'Space Mono', monospace;">← Hub</a>
				</div>

				<div class="chat-messages" id="chat-messages-container">
					<div class="chat-bubble bot-bubble">
						<div class="bot-msg-title">✨ What would you like me to write?</div>
						<div>Ask me to write any document (e.g. <em>"Write a project proposal for a full-stack AI web app"</em> or <em>"Draft an executive briefing for Q4"</em>). I'll generate the full content and create a Google Doc directly in your Google Drive.</div>
						<div class="prompt-chips">
							<button type="button" class="prompt-chip" onclick="applyPrompt(this.innerText)">📄 Write a Project Proposal for AI Copilot</button>
							<button type="button" class="prompt-chip" onclick="applyPrompt(this.innerText)">📝 Draft a Technical Architecture Document</button>
							<button type="button" class="prompt-chip" onclick="applyPrompt(this.innerText)">📊 Create an Executive Summary for Q4</button>
							<button type="button" class="prompt-chip" onclick="applyPrompt(this.innerText)">✉️ Write a Client Onboarding & Welcome Guide</button>
						</div>
					</div>
				</div>

				<div class="chat-input-container">
					<form id="chat-form" class="chat-input-form" onsubmit="handleChatSubmit(event)">
						<input type="text" id="chat-user-input" placeholder="Ask me to write any document (e.g. 'Write a complete PRD for...')" autocomplete="off" required />
						<button type="submit" id="chat-submit-btn" class="chat-send-btn">
							<span>Generate Doc</span> <span>→</span>
						</button>
					</form>
				</div>
			</div>
		</section>
	</main>

	<script>
		const apiKeyInput = document.getElementById('gemini_api_key_input');
		const emailInput = document.getElementById('user_email_input');
		const folderInput = document.getElementById('user_folder_input');
		const chatMessages = document.getElementById('chat-messages-container');
		const chatInput = document.getElementById('chat-user-input');
		const submitBtn = document.getElementById('chat-submit-btn');

		if (localStorage.getItem('gemini_api_key') && !apiKeyInput.value) {
			apiKeyInput.value = localStorage.getItem('gemini_api_key');
		}
		if (localStorage.getItem('user_email') && !emailInput.value) {
			emailInput.value = localStorage.getItem('user_email');
		}
		if (localStorage.getItem('user_folder_id') && !folderInput.value) {
			folderInput.value = localStorage.getItem('user_folder_id');
		}

		apiKeyInput.addEventListener('change', () => {
			localStorage.setItem('gemini_api_key', apiKeyInput.value.trim());
		});
		emailInput.addEventListener('change', () => {
			localStorage.setItem('user_email', emailInput.value.trim());
		});
		folderInput.addEventListener('change', () => {
			localStorage.setItem('user_folder_id', folderInput.value.trim());
		});

		function applyPrompt(text) {
			chatInput.value = text.replace(/^[^\s]+\s+/, '');
			chatInput.focus();
		}

		function scrollToBottom() {
			chatMessages.scrollTop = chatMessages.scrollHeight;
		}

		async function handleChatSubmit(e) {
			e.preventDefault();
			const msg = chatInput.value.trim();
			if (!msg) return;

			const apiKey = apiKeyInput.value.trim();
			const email = emailInput.value.trim();
			const folder = folderInput.value.trim();

			if (apiKey) localStorage.setItem('gemini_api_key', apiKey);
			if (email) localStorage.setItem('user_email', email);
			if (folder) localStorage.setItem('user_folder_id', folder);

			// Append user bubble
			const userBubble = document.createElement('div');
			userBubble.className = 'chat-bubble user-bubble';
			userBubble.textContent = msg;
			chatMessages.appendChild(userBubble);
			chatInput.value = '';
			scrollToBottom();

			// Append loading indicator bubble
			const loadingBubble = document.createElement('div');
			loadingBubble.className = 'chat-bubble bot-bubble';
			loadingBubble.id = 'active-loading-bubble';
			loadingBubble.innerHTML = '<div style="display:flex; align-items:center; gap:8px; font-size:0.85rem; color:#888eff;"><span>⚡ Generating document with Gemini & syncing to Google Docs</span><div class="typing-indicator"><div class="typing-dot"></div><div class="typing-dot"></div><div class="typing-dot"></div></div></div>';
			chatMessages.appendChild(loadingBubble);
			scrollToBottom();

			submitBtn.disabled = true;

			try {
				const formData = new FormData();
				formData.append('message', msg);
				formData.append('api_key', apiKey);
				formData.append('email', email);
				formData.append('folder_id', folder);
				formData.append('name', 'Ayushman');

				const resp = await fetch('/admin/chat-docs', {
					method: 'POST',
					body: formData
				});

				const html = await resp.text();
				loadingBubble.remove();

				const tempDiv = document.createElement('div');
				tempDiv.innerHTML = html;
				while (tempDiv.firstChild) {
					if (tempDiv.firstChild.classList && tempDiv.firstChild.classList.contains('user-bubble')) {
						tempDiv.removeChild(tempDiv.firstChild);
					} else {
						chatMessages.appendChild(tempDiv.firstChild);
					}
				}
				scrollToBottom();
			} catch (err) {
				loadingBubble.remove();
				const errBubble = document.createElement('div');
				errBubble.className = 'chat-bubble bot-bubble error-bubble';
				errBubble.textContent = '❌ Network error: ' + err.message;
				chatMessages.appendChild(errBubble);
				scrollToBottom();
			} finally {
				submitBtn.disabled = false;
				chatInput.focus();
			}
		}
	</script>

	<!-- Password Gate Modal -->
	<div id="pwd-modal-overlay" class="pwd-modal-overlay" style="display: none;">
		<div class="pwd-modal-box">
			<a href="/admin" class="pwd-modal-close" style="text-decoration: none;">✕</a>
			<div class="pwd-modal-icon">🔐</div>
			<h3 class="pwd-modal-title">access required</h3>
			<p class="pwd-modal-desc">this page is password-protected. please enter the password to open.</p>
			<form onsubmit="handleDirectGate(event)">
				<input type="password" id="pwd-modal-input" class="pwd-input" placeholder="Enter password..." autocomplete="current-password" required />
				<div id="pwd-error-msg" class="pwd-error" style="display: none;">❌ incorrect password</div>
				<button type="submit" class="pwd-submit-btn">unlock &amp; continue →</button>
			</form>
		</div>
	</div>

	<script>
		const ACCESS_PASSWORD = 'Aysuh@90052';
		const AUTH_KEY = 'uvico_auth_unlocked';

		function checkAccess() {
			if (sessionStorage.getItem(AUTH_KEY) !== 'true') {
				const modal = document.getElementById('pwd-modal-overlay');
				if (modal) modal.style.display = 'flex';
				const input = document.getElementById('pwd-modal-input');
				if (input) setTimeout(function() { input.focus(); }, 80);
			}
		}

		function handleDirectGate(e) {
			e.preventDefault();
			const input = document.getElementById('pwd-modal-input');
			const errorMsg = document.getElementById('pwd-error-msg');
			if (input && input.value === ACCESS_PASSWORD) {
				sessionStorage.setItem(AUTH_KEY, 'true');
				document.getElementById('pwd-modal-overlay').style.display = 'none';
			} else {
				if (errorMsg) errorMsg.style.display = 'block';
				if (input) {
					input.classList.add('shake');
					setTimeout(function() { input.classList.remove('shake'); }, 500);
					input.select();
				}
			}
		}

		checkAccess();
	</script>
</body>
</html>`, antidoteFonts, antidoteCSS, antidoteRobotCSS, antidoteNavbar, antidoteRobotSVG, defaultAPIKey, defaultEmail)
}

// ────────────────────────────────────────────────
// POST /admin/chat-docs — Direct Docs Bot handler
// ────────────────────────────────────────────────
func HandleDirectDocChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userMsg := strings.TrimSpace(r.FormValue("message"))
	apiKey := strings.TrimSpace(r.FormValue("api_key"))
	email := strings.TrimSpace(r.FormValue("email"))
	folderID := strings.TrimSpace(r.FormValue("folder_id"))
	clientName := strings.TrimSpace(r.FormValue("name"))

	if userMsg == "" {
		return
	}

	if clientName == "" {
		clientName = "Ayushman"
	}
	if email == "" {
		email = os.Getenv("CALENDAR_OWNER_EMAIL")
		if email == "" {
			email = "ayushmansingh2512@gmail.com"
		}
	}
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if apiKey == "" {
		fmt.Fprintf(w, `
			<div class="chat-bubble user-bubble">%s</div>
			<div class="chat-bubble bot-bubble error-bubble">
				⚠️ Please enter your <strong>Gemini API Key</strong> in the field above to generate and export Google Docs!
			</div>
		`, userMsg)
		return
	}

	docPrompt := fmt.Sprintf(`System Instruction: You are an expert document author and AI Copilot for '%s'.
The user asked: "%s".

Write a comprehensive, professional, well-structured document based on their request.
Format strictly as follows:
Line 1: TITLE: <Clear concise document title>
Line 2: (empty line)
Line 3+: The full body content with clean sections, bullet points with hyphens (-), and clear paragraphs. Do NOT use any asterisks (*) anywhere in the text.`, clientName, userMsg)

	log.Printf("[DirectDocsBot] Calling Gemini for query: %s", userMsg)
	generatedDoc := callGeminiAPI(docPrompt, apiKey)

	title := "Generated Document"
	content := generatedDoc

	lines := strings.Split(generatedDoc, "\n")
	if len(lines) > 0 && strings.HasPrefix(strings.ToUpper(lines[0]), "TITLE:") {
		title = strings.TrimSpace(strings.TrimPrefix(lines[0], "TITLE:"))
		title = strings.TrimSpace(strings.TrimPrefix(title, "Title:"))
		if len(lines) > 1 {
			content = strings.TrimSpace(strings.Join(lines[1:], "\n"))
		}
	}

	re := regexp.MustCompile(`\*+`)
	content = re.ReplaceAllString(content, "")

	docURL, err := docs.CreateAndShareDoc(r.Context(), email, folderID, title, content)
	if err == nil && docURL != "" {
		fmt.Fprintf(w, `
			<div class="chat-bubble user-bubble">%s</div>
			<div class="chat-bubble bot-bubble">
				<div class="bot-msg-title">📄 Google Doc Created: "%s"</div>
				<div class="bot-msg-preview">%s</div>
				<div class="doc-actions">
					<a href="%s" target="_blank" class="doc-btn-primary">🚀 Open in Google Docs →</a>
				</div>
			</div>
		`, userMsg, title, strings.ReplaceAll(content, "\n", "<br>"), docURL)
	} else {
		log.Printf("[DirectDocsBot] Notice on Google Doc creation: %v", err)
		fmt.Fprintf(w, `
			<div class="chat-bubble user-bubble">%s</div>
			<div class="chat-bubble bot-bubble">
				<div class="bot-msg-title">📄 Drafted Document: "%s"</div>
				<div class="bot-msg-preview">%s</div>
				<div class="bot-notice">(Notice: Google Drive sync error: %v)</div>
			</div>
		`, userMsg, title, strings.ReplaceAll(content, "\n", "<br>"), err)
	}
}

// ────────────────────────────────────────────────
// POST /admin/ingest — Process form submission
// ────────────────────────────────────────────────
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

	var existingPasscode string
	checkErr := database.DB.QueryRow("SELECT app_passcode FROM applications WHERE id = ?", appID).Scan(&existingPasscode)
	if checkErr == nil && existingPasscode != passcode {
		http.Error(w, "⛔ Unauthorized: This App ID already exists and the Security PIN is incorrect!", http.StatusForbidden)
		return
	}

	encryptedKey, err := database.Encrypt(apiKey)
	if err != nil {
		http.Error(w, "Error encrypting key", http.StatusInternalServerError)
		return
	}

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

	_, _ = database.DB.Exec("DELETE FROM knowledge_assets WHERE app_id = ?;", appID)

	assetQuery := `
		INSERT INTO knowledge_assets (app_id, category, title, content_chunk, keywords)
		VALUES (?, ?, ?, ?, ?);
	`
	_, _ = database.DB.Exec(assetQuery, appID, "Full_Document", "Portfolio Knowledge Base", rawText, "about, skills, projects, contacts")

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	hostURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>copilot ready — %s</title>
	%s
	<style>
		%s
		%s

		.section-success { padding: 3rem 5%%; display: flex; justify-content: center; align-items: center; min-height: 80vh; }
		.success-card {
			width: 100%%;
			max-width: 700px;
			background: var(--white);
			border: 2px solid var(--antidote-black);
			overflow: hidden;
		}
		.success-header {
			background: var(--antidote-black);
			color: var(--white);
			padding: 1.5rem 2rem;
		}
		.success-header .badge-label {
			font-family: 'Space Mono', monospace;
			font-size: 0.7rem;
			text-transform: uppercase;
			letter-spacing: 0.1em;
			opacity: 0.6;
			margin-bottom: 0.5rem;
		}
		.success-header h2 {
			font-family: 'Space Mono', monospace;
			font-size: 1.5rem;
			font-weight: 700;
			text-transform: uppercase;
		}
		.success-body { padding: 2rem; }
		p.desc {
			color: rgba(2, 0, 52, 0.6);
			font-size: 0.9rem;
			line-height: 1.5;
			margin-bottom: 1.5rem;
			text-transform: none;
		}
		.snippet-header {
			display: flex;
			justify-content: space-between;
			align-items: center;
			margin-top: 1.25rem;
			margin-bottom: 0.5rem;
			font-family: 'Space Mono', monospace;
			font-size: 0.7rem;
			font-weight: 700;
			text-transform: uppercase;
			letter-spacing: 0.05em;
		}
		.code-box {
			background: rgba(2, 0, 52, 0.04);
			border: 2px solid var(--antidote-black);
			padding: 1rem;
			font-family: 'Space Mono', monospace;
			font-size: 0.7rem;
			color: var(--antidote-black);
			overflow-x: auto;
			line-height: 1.6;
			text-transform: none;
		}
		.copy-btn {
			background: transparent;
			border: 1.5px solid var(--antidote-black);
			color: var(--antidote-black);
			padding: 3px 10px;
			font-family: 'Space Mono', monospace;
			font-size: 0.65rem;
			font-weight: 700;
			cursor: pointer;
			transition: all 0.2s;
			text-transform: uppercase;
		}
		.copy-btn:hover { background: var(--antidote-black); color: var(--white); }
		.actions-row {
			display: flex;
			gap: 0;
			margin-top: 2rem;
			border: 2px solid var(--antidote-black);
		}
		.action-btn {
			flex: 1;
			text-align: center;
			text-decoration: none;
			padding: 1rem;
			font-family: 'Space Mono', monospace;
			font-size: 0.85rem;
			font-weight: 700;
			transition: all 0.3s;
			text-transform: uppercase;
		}
		.btn-dark { background: var(--antidote-black); color: var(--white); }
		.btn-dark:hover { background: var(--primary); }
		.btn-light { background: var(--white); color: var(--antidote-black); border-left: 2px solid var(--antidote-black); }
		.btn-light:hover { background: var(--background); }
	</style>
</head>
<body>
	%s
	<section class="section-success">
		<div class="success-card">
			<div class="success-header">
				<div class="badge-label">✅ INGESTION COMPLETE</div>
				<h2>COPILOT:READY — "%s"</h2>
			</div>
			<div class="success-body">
				<p class="desc">Your knowledge documents are processed. Gemini API key encrypted with AES-256. Copy the embed code below and paste it into your website.</p>

				<div class="snippet-header">
					<span>01 — IFRAME EMBED (RECOMMENDED)</span>
					<button class="copy-btn" onclick="navigator.clipboard.writeText(document.getElementById('code-iframe').innerText); this.innerText='COPIED';">COPY</button>
				</div>
				<div id="code-iframe" class="code-box">&lt;iframe
  src="%s/copilot/embed?app_id=%s"
  style="position: fixed; bottom: 20px; right: 20px; z-index: 99999; border: none; width: 380px; height: 520px; background: transparent;"
  allowtransparency="true"&gt;
&lt;/iframe&gt;</div>

				<div class="snippet-header">
					<span>02 — HTMX EMBED</span>
					<button class="copy-btn" onclick="navigator.clipboard.writeText(document.getElementById('code-htmx').innerText); this.innerText='COPIED';">COPY</button>
				</div>
				<div id="code-htmx" class="code-box">&lt;script src="https://unpkg.com/htmx.org@1.9.12"&gt;&lt;/script&gt;
&lt;aside id="ai-copilot-root"
       hx-get="%s/copilot/embed?app_id=%s"
       hx-trigger="load"
       hx-swap="innerHTML"&gt;
&lt;/aside&gt;</div>

				<div class="actions-row">
					<a href="/copilot/embed?app_id=%s" target="_blank" class="action-btn btn-dark">🚀 LIVE PREVIEW</a>
					<a href="/admin" class="action-btn btn-light">← BACK TO HUB</a>
				</div>
			</div>
		</div>
	</section>
</body>
</html>`, appID, antidoteFonts, antidoteCSS, antidoteRobotCSS, antidoteNavbar, appID, hostURL, appID, hostURL, appID, appID)
}
