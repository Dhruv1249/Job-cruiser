# Job Cruiser API Documentation

**Base URL:** `http://localhost:8080/api`

---

## Authentication

### 1. Signup

Creates a new user with an email and password, returning a JWT token.

- **URL:** `/signup`
- **Method:** `POST`
- **Authentication Required:** No

#### Request Body

```json
{
  "primary_email": "john@example.com",
  "password": "securepassword123"
}
```

#### Success Response

**Status:** `201 Created`

```json
{
  "message": "User created successfully",
  "token": "eyJhbGciOiJIUzI1NiIsIn...",
  "user_id": "uuid-string",
  "is_new_user": true
}
```

#### Error Response

**Status:** `409 Conflict`

```json
{
  "error": "Email already exists or database error"
}
```

---

### 2. Login

Authenticates an existing user and returns a JWT token.

- **URL:** `/login`
- **Method:** `POST`
- **Authentication Required:** No

#### Request Body

```json
{
  "primary_email": "john@example.com",
  "password": "securepassword123"
}
```

#### Success Response

**Status:** `200 OK`

```json
{
  "message": "Login successful",
  "token": "eyJhbGciOiJIUzI1NiIsIn...",
  "user_id": "uuid-string"
}
```

#### Error Response

**Status:** `401 Unauthorized`

```json
{
  "error": "Invalid credentials"
}
```

---

### 3. Google SSO Login

Authenticates a user via Google. If the user doesn't exist, it creates a new record.

- **URL:** `/auth/google`
- **Method:** `POST`
- **Authentication Required:** No

#### Request Body

```json
{
  "id_token": "eyJhbGciOiJSUzI1NiIs..."
}
```

> The `id_token` is the raw JWT received from the Flutter `google_sign_in` package.

#### Success Response

**Status:** `200 OK`

```json
{
  "message": "Google Login successful",
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user_id": "uuid-string",
  "is_new_user": true,
  "suggested_name": "John Doe"
}
```

**Notes:**

- Use `is_new_user` on the frontend to redirect users to the Preferences onboarding screen.
- Use `suggested_name` to pre-fill the user's name during onboarding.

#### Error Response

**Status:** `401 Unauthorized`

```json
{
  "error": "Invalid Google token"
}
```

---

## User Settings

### 4. Update Preferences

Creates or updates a user's job search preferences. This is the primary onboarding endpoint for new users.

- **URL:** `/preferences`
- **Method:** `POST`
- **Authentication Required:** Yes (Bearer Token)

#### Headers

```http
Authorization: Bearer <your_jwt_token>
```

#### Request Body

```json
{
  "full_name": "John Doe",
  "target_roles": [
    "Backend Engineer",
    "Go Developer"
  ],
  "work_models": [
    "remote",
    "hybrid"
  ],
  "min_salary": 120000,
  "currency": "USD"
}
```

#### Success Response

**Status:** `200 OK`

```json
{
  "message": "Preferences saved successfully"
}
```

---

## Jobs Feed

### 5. Get Jobs (Paginated)

Fetches the latest scraped jobs from the database.

- **URL:** `/jobs?page=1&limit=20`
- **Method:** `GET`
- **Authentication Required:** Yes (Bearer Token)

#### Headers

```http
Authorization: Bearer <your_jwt_token>
```

#### Query Parameters

| Parameter | Type | Description | Default |
|------------|---------|-------------|---------|
| page | integer | Page number | 1 |
| limit | integer | Number of jobs per page | 20 |

#### Success Response

**Status:** `200 OK`

```json
{
  "data": [
    {
      "id": "job-uuid",
      "company_id": "company-uuid",
      "title": "Senior Go Developer",
      "location": "New York, NY",
      "salary_min": 130000,
      "salary_max": 160000,
      "currency": "USD",
      "experience_required": "5+ years",
      "job_type": "Full-time",
      "is_easy_apply": true,
      "is_remote": false,
      "source": "LinkedIn",
      "url": "https://linkedin.com/jobs/...",
      "posted_date": "2 hours ago",
      "tags": [
        "golang",
        "postgres",
        "docker"
      ],
      "scraped_at": "2026-05-29T10:30:00Z"
    }
  ],
  "page": 1,
  "limit": 20
}
```

> Nullable fields such as `location`, `salary_min`, and `posted_date` may return `null` if the scraper could not find them.

---

## General Authentication Flow

1. Authenticate via one of:
   - `/signup`
   - `/login`
   - `/auth/google`

2. Extract the `token` from the JSON response.

3. Pass the token in the `Authorization` header for all protected routes:

```http
Authorization: Bearer <your_jwt_token>
```

---

### 6. Get Preferences

Fetches the user's currently saved job search preferences.

- **URL:** `/preferences`
- **Method:** `GET`
- **Authentication Required:** Yes (Bearer Token)

#### Headers

```http
Authorization: Bearer <your_jwt_token>
```

#### Success Response

**Status:** `200 OK`

```json
{
  "data": {
    "full_name": "John Doe",
    "target_roles": [
      "Backend Engineer",
      "Go Developer"
    ],
    "work_models": [
      "remote",
      "hybrid"
    ],
    "min_salary": 120000,
    "currency": "USD"
  }
}
```

> If the user has never saved preferences, `data` will return `null`.

---

## Applications (Kanban Pipeline)

### 7. Save / Apply to Job

Adds a job to the user's application pipeline.

- **URL:** `/applications`
- **Method:** `POST`
- **Authentication Required:** Yes (Bearer Token)

#### Headers

```http
Authorization: Bearer <your_jwt_token>
```

#### Request Body

```json
{
  "job_id": "uuid-string-of-the-job",
  "status": "bookmarked"
}
```

**Valid statuses:**

- `bookmarked`
- `applied`
- `interviewing`
- `rejected`

> Defaults to `bookmarked` if omitted.

#### Success Response

**Status:** `201 Created`

```json
{
  "message": "Job saved successfully",
  "application_id": "uuid-string"
}
```

#### Error Response

**Status:** `409 Conflict`

```json
{
  "error": "Job already saved"
}
```

> Returned when the user has already saved the specified job.

---

### 8. Get Pipeline

Fetches all jobs the user has saved, applied to, or interviewed for.

- **URL:** `/applications`
- **Method:** `GET`
- **Authentication Required:** Yes (Bearer Token)

#### Headers

```http
Authorization: Bearer <your_jwt_token>
```

#### Success Response

**Status:** `200 OK`

```json
{
  "data": [
    {
      "application_id": "app-uuid",
      "job_id": "job-uuid",
      "company_id": "company-uuid",
      "title": "Senior Go Developer",
      "location": "New York, NY",
      "status": "interviewing",
      "applied_at": "2026-05-29T10:30:00Z"
    }
  ]
}
```

---

### 9. Update Pipeline Status

Moves a job across the user's Kanban board (for example, from `applied` to `interviewing`).

- **URL:** `/applications/:id/status`
- **Method:** `PUT`
- **Authentication Required:** Yes (Bearer Token)

> Replace `:id` with the `application_id`.

#### Headers

```http
Authorization: Bearer <your_jwt_token>
```

#### Request Body

```json
{
  "status": "interviewing"
}
```

#### Success Response

**Status:** `200 OK`

```json
{
  "message": "Status updated successfully"
}
```
