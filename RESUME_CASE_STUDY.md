# 🚀 Project Case Study: Universal AI Copilot (`uvico`)
**Candidate:** Ayushman Singh | Full-Stack Systems & AI Engineer  
**Project:** Universal AI Copilot (`uvico`) — Multi-Tenant Self-Serve AI Chatbot Engine  
**Repository:** [github.com/ayushmansingh2512/uvico](https://github.com/ayushmansingh2512/uvico)  
**Primary Tech Stack:** Go (1.22+), SQLite (WAL Mode), Google Gemini 2.5 Flash, AES-256 GCM Encryption, HTMX, Docker, Render  

---

## 📌 Executive Summary & Resume Overview

**Universal AI Copilot (`uvico`)** is a lightweight, high-performance Micro-SaaS backend designed to allow developers, creators, and organizations to deploy context-aware, document-backed AI chatbots on any website with a single 2-line HTMX or iFrame snippet.

I designed and engineered the platform from scratch to overcome traditional RAG complexity, high vector-database hosting costs, security risks (unencrypted API keys), and heavy frontend build requirements.

---

## 🎯 Resume Bullet Points (STAR / XYZ Method)

*Copy & paste directly into your Software Engineering / AI Engineer resume:*

* **Architected a multi-tenant micro-SaaS backend** in Go and SQLite enabling instant AI chatbot deployment to any website via a 2-line HTMX embed code, reducing onboarding time to **<60 seconds**.
* **Engineered a zero-trust security model** with **AES-256 GCM encryption at rest** for user-provided Gemini API keys and isolated tenant security PINs to prevent data tampering.
* **Designed an efficient SQL-driven RAG pipeline** featuring native PDF/Markdown extraction and targeted keyword searching, delivering sub-second response times (**~500ms latency**).
* **Optimized backend system performance** utilizing SQLite Write-Ahead Logging (WAL) mode and Go concurrency, achieving an ultralight idle memory footprint of **<30 MB RAM**.
* **Built a responsive, zero-framework luxury UI ("Ichiban Aesthetic")** with smooth CSS cubic-bezier expanding animations and HTMX dynamic DOM swapping without React/Next.js overhead.

---

## 🛠️ The Technical Deep-Dive

### 1. Key Engineering Challenges & Solutions

| Engineering Challenge | How I Solved It |
| :--- | :--- |
| **Cross-Tenant API Key Security** | Implemented Go `crypto/aes` GCM 256-bit encryption with dynamic nonces. Keys are encrypted at rest before storing in SQLite and decrypted in-memory only during API requests. |
| **Fast & Low-Cost Context Retrieval (RAG)** | Replaced expensive external vector databases with an in-database text-chunking and keyword-indexed SQLite retrieval strategy, paired with Gemini 2.5 Flash. |
| **Zero Frontend Build Dependencies** | Served self-contained, HTMX-driven widget HTML directly from the Go backend. Site owners paste 2 lines of code without needing React, Webpack, or npm installations. |
| **High Concurrent Read/Write Throughput** | Enabled SQLite `PRAGMA journal_mode=WAL` (Write-Ahead Logging), allowing concurrent read queries during background document parsing. |

---

## 🏗️ System Architecture & Data Flow

```
                                  ┌───────────────────────────┐
                                  │   Universal Copilot UI    │
                                  │   http://localhost:8080   │
                                  └─────────────┬─────────────┘
                                                │
                         ┌─────────────────────┴─────────────────────┐
                         │                                           │
             App ID: "ayushman_portfolio"                App ID: "city_hospital"
             Passcode: "ayushman@123"                    Passcode: "hospital@456"
                         │                                           │
                         ▼                                           ▼
          ┌─────────────────────────────┐             ┌─────────────────────────────┐
          │ SQLite: Knowledge Asset 1   │             │ SQLite: Knowledge Asset 2   │
          │ Encrypted Gemini Key 1      │             │ Encrypted Gemini Key 2      │
          └─────────────────────────────┘             └─────────────────────────────┘
```

1. **Admin Onboarding Layer**: Admin uploads a `.pdf` resume or pastes Markdown details.
2. **Document Ingestion Engine**: Native PDF text extraction occurs on upload, splitting content into tagged chunks stored in `knowledge_assets`.
3. **Embed Serving Engine**: Serving `/copilot/embed?app_id=XXX` loads a floating action widget with a 2-stage animated state machine.
4. **Chat Execution Layer**: Incoming user messages query SQLite context, assemble the context-augmented prompt, and call Gemini 2.5 Flash REST API securely.

---

## 💡 System Design Interview Talking Points

*When discussing this project in technical interviews:*

### Q: Why Go instead of Python or Node.js for an AI backend?
> **Answer:** "Python is great for AI model training, but for serving high-concurrency micro-SaaS backends, Go provides vastly superior execution speed, native concurrency with goroutines, a single compiled binary, and an extremely low memory footprint (<30 MB RAM vs 200MB+ in Node/Python)."

### Q: How did you handle API key security and multi-tenancy?
> **Answer:** "I implemented strict multi-tenant isolation in SQLite using `app_id` foreign keys and PIN protection (`app_passcode`). To protect user Gemini keys, I used AES-256 GCM authenticated encryption with crypto-random nonces. Keys are never stored in plain text and are decrypted in memory strictly for the duration of the API call."

### Q: Why choose SQLite over PostgreSQL + pgvector?
> **Answer:** "For single-binary deployment and micro-SaaS efficiency, SQLite in WAL mode handles thousands of read requests per second with microsecond local access times. It eliminates database server networking overhead, simplifies Docker containerization, and costs $0 to host."

---

## 📊 Quantifiable Resume Summary

- **Primary Languages**: Go (1.22+), HTML5/CSS3, SQL, Python
- **Frameworks & Libraries**: HTMX, `modernc.org/sqlite`, `github.com/dslipak/pdf`
- **Security & Cloud**: AES-256 GCM, Docker, Render, REST APIs, Google Gemini API
- **Key Metrics**: < 60s Onboarding, ~500ms AI Latency, <30 MB Memory Footprint
