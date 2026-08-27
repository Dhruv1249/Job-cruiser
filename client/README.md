# Job Cruiser Client

Flutter client application for Job Cruiser supporting Android and Web deployments.

---

## Overview

The client provides an interface for exploring AI-matched job opportunities, customizing dynamic search criteria, monitoring background application tailoring, and managing the application pipeline CRM.

---

## Key Features

- **Responsive Master-Detail Layout**:
  - Displays with a width of 960px or greater render a two-pane layout: a scrolling left feed with real-time search and filter controls, and a right inspection panel displaying the job description, match analysis, and document tailoring triggers.
  - Displays under 960px render a single-column feed with bottom navigation and separate detail screens.
- **Dynamic Multi-Criteria Filtering**:
  - Match Scope: All jobs, matched only, or unmatched only.
  - Date Recency: Presets (Today, 2 days, 3 days, 7 days, 14 days) and custom day inputs.
  - Match Score: Continuous range slider (0% to 100%) and preset thresholds (60%+, 80%+, 90%+).
  - Sorting: Match score, date scraped (descending or ascending), and salary.
  - Work Arrangement: Remote only or on-site/hybrid.
  - View Status: Unviewed only or viewed only.
- **Persistent Local Settings**:
  - Filter and sort states are serialized to `SharedPreferences` and automatically restored on startup.
- **Document Management**:
  - Inline PDF preview and management for AI-tailored resumes and cover letters compiled via Open-Overleaf.
- **CRM Application Tracker**:
  - Pipeline management across all application phases (Bookmarked, Applied, Outreach, Interview, Offer, Rejected).

---

## Setup & Running

### Environment Configuration

Create a `.env` file in the `client` directory:

```env
API_BASE_URL=http://localhost:8080/api
GOOGLE_CLIENT_ID=your-google-oauth-web-client-id.apps.googleusercontent.com
```

### Installation & Execution

```bash
# Fetch package dependencies
flutter pub get

# Execute test suite
flutter test

# Run static analysis
flutter analyze

# Launch on Chrome (Web)
flutter run -d chrome

# Launch on Android device or emulator
flutter run -d android
```

---

## Architecture & State Structure

```
lib/
├── models/
│   ├── application.dart       # CRM application data model
│   ├── job.dart               # Matched job data model
│   └── job_filter_state.dart  # Filter state model & local persistence
├── screens/
│   ├── cover_letters_screen.dart
│   └── resume_versions_screen.dart
├── services/
│   ├── api_service.dart       # Dio HTTP client and REST endpoints
│   └── notification_service.dart
├── widgets/
│   ├── company_logo_avatar.dart
│   ├── job_description_renderer.dart
│   ├── job_detail_panel.dart  # Modular job inspection and actions panel
│   ├── job_filter_bar.dart    # Horizontal quick chip bar
│   ├── job_filter_dialog.dart # Multi-criteria filter modal sheet
│   ├── notifications_sheet.dart
│   └── tailoring_result_sheet.dart
├── auth.dart                  # Authentication and Google SSO screen
├── details.dart               # Standalone job details route for mobile
├── main.dart                  # Application shell and responsive master-detail feed
├── onboarding.dart            # Multi-step profile and preference setup
├── preferences.dart           # Match preferences and Open-Overleaf config
├── profile.dart               # User profile and documents screen
└── tracker.dart               # Application pipeline CRM tracker
```
