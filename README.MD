# Job Cruiser

Job Cruiser is an automated job search, intelligent matching, and application tracking platform. It ingests job listings across direct ATS APIs (Greenhouse, Lever, Ashby, SmartRecruiters, Workday) and job boards, evaluates opportunities against user profiles using a hybrid AI pipeline, generates tailored LaTeX resumes and cover letters via an Open-Overleaf MCP service, and provides a centralized CRM application tracker.

---

## Architecture Overview

```
Flutter Client (Web & Android)
       |
       v REST (JSON)
Go Backend (Gin)
   ├── Database: PostgreSQL 17 (Single writer)
   ├── AI Matcher: NVIDIA NIM (Llama-3.1-70B) + Gemini Fallback
   └── AI Tailor: Gemini + Open-Overleaf MCP LaTeX Engine
       ^
       | POST /api/scraper/ingest-raw
Python Scraper (Cloud Run Job)
   ├── ATS Connectors (Greenhouse, Lever, Ashby, ...)
   └── JobSpy Board Aggregators
```

---

## Key Modules

### 1. Ingestion Pipeline (`scrapper/`)
- Direct REST clients for Greenhouse, Lever, Ashby, SmartRecruiters, and Workday.
- Board scrapers for LinkedIn, Indeed, Glassdoor, ZipRecruiter, Google Jobs, and RemoteOK.
- Normalization, deduplication on company/title/location hash, and chunked ingestion transmission to the Go backend.

### 2. Matching Engine (`backend/services/`)
- Hybrid two-tier matcher: Primary evaluation with NVIDIA NIM (`meta/llama-3.1-70b-instruct`) and automatic fallback to Google Gemini (`gemini-2.5-flash`).
- Circuit breaker mechanism protecting against API rate limits and sequential failures.
- Strict evaluation heuristics enforcing location compatibility, experience requirements, and structured multi-line match rationale.

### 3. Application Client (`client/`)
- **Web & Desktop Experience**: Responsive split master-detail layout on displays 960px and wider, providing simultaneous feed exploration and deep-dive analysis with inline document tailoring.
- **Mobile Experience**: Single-column feed with bottom navigation and dedicated details views.
- **Dynamic Filtering Engine**: Multi-criteria filters covering match scope (all, matched, unmatched), date recency presets and custom day inputs, match score range sliders, work model arrangements, and view statuses.
- **Persistent Storage**: All filter and sort configurations are persisted to `SharedPreferences` and restored on application startup.
- **Application Tracker**: CRM pipeline managing application statuses from initial bookmark to offer or rejection.

### 4. Document Tailoring Engine (`backend/services/resume_tailor_service.go`)
- Reads baseline LaTeX templates (`templates/resume.tex` and `templates/cover_letter.tex`) hosted in Open-Overleaf.
- Generates job-specific LaTeX source code preserving custom macros, typography, and visual structure.
- Multi-pass tightening loop enforcing single-page constraints.
- Programmatic compilation and PDF synchronization through the Open-Overleaf MCP interface.

---

## Repository Structure

```
Job-cruiser/
├── backend/                  # Go (Gin) REST backend service
│   ├── handlers/             # HTTP endpoint handlers
│   ├── middleware/           # Auth, CORS, and Ingest middleware
│   ├── models/               # Data structures and domain models
│   ├── services/             # AI matchers, tailoring service, MCP client
│   ├── tests/                # Unit and handler test suites
│   └── utils/                # AES-256 encryption, logger, utilities
├── client/                   # Flutter application (Android and Web)
│   ├── lib/
│   │   ├── models/           # Dart data models and filter state
│   │   ├── screens/          # Document version management screens
│   │   ├── services/         # API client, notifications, local storage
│   │   ├── widgets/          # UI components, filter bar, detail panel
│   │   ├── main.dart         # Main feed and master-detail layout
│   │   ├── tracker.dart      # Application CRM tracker
│   │   ├── profile.dart      # User profile screen
│   │   └── preferences.dart  # Target preferences and Overleaf setup
│   └── test/                 # Flutter unit test suites
└── scrapper/                 # Python ingestion scripts and ATS connectors
    ├── job_sources/          # ATS API clients
    └── scrape_all.py         # Scraper orchestrator
```

---

## Local Development & Setup

### Prerequisites
- Go 1.22+
- Flutter 3.24+
- Python 3.11+
- PostgreSQL 17

### 1. Backend Service

```bash
cd backend

# Create .env configuration
cat <<EOF > .env
PORT=8080
DATABASE_URL=postgres://jobcruiser:PASSWORD@localhost:6432/jobcruiser?sslmode=disable
JWT_SECRET=your-jwt-secret
ENCRYPTION_KEY=32-character-encryption-key-here
NVIDIA_NIM_API_KEY=your-nim-api-key
GEMINI_API_KEY=your-gemini-api-key
EOF

# Run test suite
go test ./...

# Start backend service
go run main.go
```

### 2. Flutter Client

```bash
cd client

# Create .env configuration
cat <<EOF > .env
API_BASE_URL=http://localhost:8080/api
GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
EOF

# Install dependencies
flutter pub get

# Run tests and analysis
flutter test
flutter analyze

# Run on Web (Chrome)
flutter run -d chrome

# Run on Android
flutter run -d android
```

### 3. Ingestion Scraper

```bash
cd scrapper

# Set up virtual environment
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt

# Execute ingestion run
python scrape_all.py
```

---

## Security Specifications

- **Database Isolation**: Direct database access is restricted exclusively to the Go backend service. All client and scraper interactions are mediated via authenticated REST endpoints.
- **Credential Encryption**: GitHub tokens and Open-Overleaf credentials are encrypted with AES-256-GCM before persistent database storage.
- **Access Control**: Account registration is protected by an administrative email whitelist mechanism.
- **Data Integrity**: Listings dismissed from the user feed remain stored in the database for tracking and telemetry analysis.
