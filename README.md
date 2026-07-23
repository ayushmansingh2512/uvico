# ⚡ Universal AI Copilot (`uvico`)

> **Multi-Tenant Self-Serve AI Chatbot Platform for Portfolios & Websites**  
> Easily embed a custom, resume-aware AI Copilot into any website with just a 2-line HTML snippet.

---

## 🚀 Overview

**Universal AI Copilot** is a high-performance, multi-tenant micro-SaaS backend built with **Go** and **SQLite**. It enables creators, developers, and organizations to onboard their own portfolio AI chatbot in under 1 minute.

### Key Architecture Highlights
- 🔒 **100% Multi-Tenant Isolation**: Each user gets a unique `App ID` secured by an account **Security PIN**.
- 🔐 **AES-256 Encryption**: User-provided Google Gemini API keys are encrypted at rest using AES-256 GCM.
- 📄 **Automatic PDF/Document Parsing**: Directly uploads resumes, PDFs, or Markdown docs and indexes knowledge chunks into SQLite.
- ⚡ **Lightweight HTMX Embed**: Add a dynamic, responsive AI assistant to any portfolio with zero complex frontend build steps.

---

## 🎨 Self-Serve Onboarding Flow

```
                                 ┌───────────────────────────┐
                                 │   Universal Copilot UI    │
                                 │   http://localhost:8080   │
                                 └─────────────┬─────────────┘
                                               │
                         ┌─────────────────────┴─────────────────────┐
                         │                                           │
             App ID: "ayushman_portfolio"                App ID: "rahul_portfolio"
             Passcode: "ayushman@123"                    Passcode: "rahul@456"
                         │                                           │
                         ▼                                           ▼
          ┌─────────────────────────────┐             ┌─────────────────────────────┐
          │ SQLite: Knowledge Asset 1   │             │ SQLite: Knowledge Asset 2   │
          │ Encrypted Gemini Key 1      │             │ Encrypted Gemini Key 2      │
          └─────────────────────────────┘             └─────────────────────────────┘
```

1. **Visit `/admin`**: Enter your unique `App ID`, set your **Security PIN**, and provide your **Gemini API Key**.
2. **Upload Resume/Doc**: Upload a `.pdf`, `.txt`, or `.md` file or paste plain text information about your skills, projects, and contact info.
3. **Get 2-Line Embed Code**: Copy the generated HTMX script into your website:

```html
<script src="https://unpkg.com/htmx.org@1.9.12"></script>
<aside id="ai-copilot-root"
       hx-get="https://your-domain.com/copilot/embed?app_id=your_app_id"
       hx-trigger="load"
       hx-swap="innerHTML">
</aside>
```

---

## 🛠️ Tech Stack

- **Backend**: Go (1.22+)
- **Database**: SQLite with modern `WAL` mode (`modernc.org/sqlite`)
- **AI Model**: Google Gemini API (`gemini-2.5-flash`)
- **PDF Extraction**: `github.com/dslipak/pdf`
- **Encryption**: AES-256 GCM
- **Frontend Widget**: HTML5, Vanilla CSS (Ichiban Dark Aesthetic), HTMX

---

## 💻 Local Setup & Development

### 1. Prerequisites
- [Go 1.22+](https://go.dev/dl/) installed.
- A free [Google Gemini API Key](https://aistudio.google.com/).

### 2. Run Locally

```bash
# Clone the repository
git clone https://github.com/ayushmansingh2512/uvico.git
cd uvico

# Install dependencies
go mod download

# Set encryption secret (optional)
export ENCRYPTION_SECRET="32-byte-long-secret-key-for-aes"

# Start server
go run cmd/main.go
```

The server will start on `http://localhost:8080`.

- **Self-Serve Dashboard**: `http://localhost:8080/admin`
- **Embed API**: `http://localhost:8080/copilot/embed?app_id=your_app_id`

---

## 🐳 Docker Deployment (Render / HuggingFace Spaces)

You can containerize and deploy this application effortlessly using Docker.

```bash
docker build -t universal-copilot .
docker run -p 8080:8080 universal-copilot
```

---

## 📜 License

Distributed under the MIT License. Built with ❤️ by [Ayushman Singh](https://github.com/ayushmansingh2512).
