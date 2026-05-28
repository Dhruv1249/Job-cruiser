# Job Cruiser API Documentation

**Base URL:** `http://localhost:8080/api`

---

# Authentication

## Signup

Creates a new user and returns a JWT token.

* **URL:** `/signup`
* **Method:** `POST`
* **Authentication Required:** No

### Request Body

```json id="d1a9kq"
{
  "full_name": "John Doe",
  "primary_email": "john@example.com",
  "password": "securepassword123"
}
```

### Success Response

**Status:** `201 Created`

```json id="5j8cvm"
{
  "message": "User created successfully",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user_id": "uuid-string"
}
```

### Error Response

**Status:** `400 Bad Request`

```json id="n4r2xy"
{
  "error": "Email already exists"
}
```

---

## Login

Authenticates a user and returns a JWT token.

* **URL:** `/login`
* **Method:** `POST`
* **Authentication Required:** No

### Request Body

```json id="8f3vla"
{
  "primary_email": "test@example.com",
  "password": "password123"
}
```

### Success Response

**Status:** `200 OK`

```json id="k9z1tr"
{
  "message": "Login successful",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user_id": "15141970-3050-4598-b233-995296c6af72"
}
```

### Error Response

**Status:** `401 Unauthorized`

```json id="0x7mep"
{
  "error": "Invalid email or password"
}
```
---

## Google SSO Login

Authenticates a user using Google Single Sign-On and returns a custom JWT token.

* **URL:** `/auth/google`
* **Method:** `POST`
* **Authentication Required:** No

### Description

The backend verifies the provided Google `id_token` against the configured Google Client ID.
After successful verification, the server:

1. Extracts the user's Google profile information.
2. Creates or updates the user in the database.
3. Generates and returns a custom JWT token for authenticated requests.

---

### Request Headers

```http
Content-Type: application/json
```

---

### Request Body

```json
{
  "id_token": "eyJhbGciOiJSUzI1NiIs..."
}
```

> **Note:**
> The `id_token` is the raw JWT received from the Flutter `google_sign_in` package.

---

### Success Response

**Status:** `200 OK` (Existing User)

or

**Status:** `201 Created` (New User)

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "full_name": "Jane Doe",
    "primary_email": "jane.doe@gmail.com",
    "avatar_url": "https://lh3.googleusercontent.com/a/..."
  }
}
```

> **Note:**
> The returned `token` is your application's custom JWT signed using `JWT_SECRET`.
> Use this token as a Bearer token for protected endpoints.

Example:

```http
Authorization: Bearer <your_jwt_token>
```

---

### Error Responses

#### Missing Token

**Status:** `400 Bad Request`

```json
{
  "error": "id_token is required"
}
```

---

#### Invalid Google Token

**Status:** `401 Unauthorized`

```json
{
  "error": "invalid google token"
}
```

---

#### Internal Server Error

**Status:** `500 Internal Server Error`

```json
{
  "error": "internal server error"
}
```


---

# Jobs

## Get Jobs (Paginated)

Fetches the latest scraped jobs.

* **URL:** `/jobs?page=1&limit=20`
* **Method:** `GET`
* **Authentication Required:** Yes (Bearer Token)

### Headers

```http id="3qu9wn"
Authorization: Bearer <your_jwt_token>
```

### Query Parameters

| Parameter | Type    | Description             | Default |
| --------- | ------- | ----------------------- | ------- |
| `page`    | integer | Page number             | `1`     |
| `limit`   | integer | Number of jobs per page | `20`    |

### Success Response

**Status:** `200 OK`

```json id="y7m2du"
{
  "data": [
    {
      "id": "job-uuid",
      "title": "Go Developer",
      "company": "Tech Corp",
      "location": "Remote",
      "job_type": "Full Time",
      "posted_at": "2025-05-20T10:30:00Z"
    }
  ],
  "page": 1,
  "limit": 20
}
```

### Error Response

**Status:** `401 Unauthorized`

```json id="m5a1gh"
{
  "error": "Unauthorized"
}
```

---

# Authentication Flow

1. Create an account using the `/signup` endpoint.
2. Login using the `/login` endpoint.
3. Copy the returned JWT token.
4. Pass the token in the `Authorization` header for protected routes.

Example:

```http id="9r4vbc"
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

---

# Response Format

All API responses are returned in JSON format.

### Success Example

```json id="e2q7ks"
{
  "message": "Success"
}
```

### Error Example

```json id="f8n6wp"
{
  "error": "Something went wrong"
}
```

