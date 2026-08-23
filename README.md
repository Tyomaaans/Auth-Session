# Auth & Session Management API

A production-ready REST API for authentication and multi-session management built with Go. Designed for high security and scalability — featuring refresh token rotation, token hashing at rest, multi-device tracking, and instant session revocation via Redis.

> ⚠️ This project is actively under development. Core authentication, token rotation, and multi-session controls are stable.

---

## Architecture Overview

```
Client
  │
  ▼
Gin HTTP Server
  │
  ├── Auth Middleware & Admin Secret Middleware
  │     └── Token validation via Redis (no DB round-trip)
  │
  ├── Handlers → Services → Repositories
  │                             │
  │                         PostgreSQL (User primary datastore)
  │
  └── Redis
        ├── Refresh token storage (hashed SHA-256, with TTL)
        └── Multi-session state store (tracked by session ID / sid)

```

---

## Tech Stack

| Layer | Technology | Reason |
| --- | --- | --- |
| Language | Go | Performance, strong typing, excellent concurrency |
| Framework | Gin | Lightweight, fast HTTP router |
| Database | PostgreSQL | Reliable relational datastore for user profiles |
| Cache & Token Store | Redis | Ultra-low latency storage for session state, hashed tokens, and fast revocation checks |
| Containerization | Docker & Docker Compose | Consistent local and production environments |

---

## Features

### Auth & Token Rotation

* **JWT Authentication:** Short-lived access tokens paired with long-lived refresh tokens.
* **Refresh Token Rotation:** Every refresh request invalidates the old refresh token and issues a new pair, mitigating replay attacks.
* **Hashed at Rest:** Refresh tokens are stored in Redis as SHA-256 hashes — if Redis is compromised, raw tokens are not exposed.

### Multi-Session Control

* **Multi-Device Tracking:** Users can log in across multiple devices, each tracked with a unique Session ID (`sid`).
* **Granular Self-Revocation:** Users can view active logins, revoke specific sessions, wipe all other sessions except the active one, or terminate all sessions at once.
* **Admin Force Revocation:** Administrators can inspect any target user's active sessions and forcibly terminate specific or all active sessions.

---

## API Endpoints

<details>
<summary><strong>Auth</strong> — 4 endpoints</summary>

| Method | Endpoint | Access |
| --- | --- | --- |
| POST | `/api/v1/auth/register` | Public |
| POST | `/api/v1/auth/login` | Public |
| POST | `/api/v1/auth/refresh` | Public |
| POST | `/api/v1/auth/logout` | Authenticated |

</details>

<details>
<summary><strong>Users</strong> — 2 endpoints</summary>

| Method | Endpoint | Access |
| --- | --- | --- |
| GET | `/api/v1/users/me` | Authenticated |
| PATCH | `/api/v1/users/me` | Authenticated |

</details>

<details>
<summary><strong>Sessions</strong> — 4 endpoints</summary>

| Method | Endpoint | Access |
| --- | --- | --- |
| GET | `/api/v1/users/me/sessions` | Authenticated |
| DELETE | `/api/v1/users/me/sessions/:sid` | Authenticated |
| DELETE | `/api/v1/users/me/sessions/others` | Authenticated |
| DELETE | `/api/v1/users/me/sessions` | Authenticated |

</details>

<details>
<summary><strong>Admin</strong> — 6 endpoints</summary>

| Method | Endpoint | Access |
| --- | --- | --- |
| GET | `/api/v1/admin/users` | Authenticated + Admin Secret |
| GET | `/api/v1/admin/users/:sub` | Authenticated + Admin Secret |
| PATCH | `/api/v1/admin/users/:sub` | Authenticated + Admin Secret |
| GET | `/api/v1/admin/users/:sub/sessions` | Authenticated + Admin Secret |
| DELETE | `/api/v1/admin/users/:sub/sessions/:sid` | Authenticated + Admin Secret |
| DELETE | `/api/v1/admin/users/:sub/sessions` | Authenticated + Admin Secret |

</details>

---

## Getting Started

### Prerequisites

* [Docker](https://docs.docker.com/get-docker/) & Docker Compose
* Go 1.22+

### Run Locally

```bash
# Clone the repository
git clone https://github.com/Tyomaaans/Auth-Session.git
cd auth-session-api

# Copy environment variables
cp .env.example .env

# Start PostgreSQL and Redis containers
docker compose up -d

```

### Environment Variables

See `.env.example` for all required variables. Key configs:

```env
# App
APP_ENV=
APP_PORT=
APP_URL=

# Database
POSTGRES_USER=
POSTGRES_PASSWORD=
POSTGRES_DB=
DATABASE_URL=

# Redis
REDIS_ADDR=
REDIS_PASSWORD=

# Token
JWT_SECRET_KEY=
JWT_EXPIRY=
DEFAULT_REFRESH_EXPIRY=
SHORT_REFRESH_EXPIRY=

# Admin
ADMIN_SECRET_KEY=

```

---

## Project Status

| Feature | Status | Notes |
| --- | --- | --- |
| User Registration & Login | ✅ Done |  |
| Short-lived JWT Access Token | ✅ Done |  |
| Refresh Token Rotation (SHA-256 in Redis) | ✅ Done | Old refresh tokens invalidated on reuse |
| Multi-Session Management (User) | ✅ Done | Get active sessions, revoke single, others, or all |
| Admin User & Session Revocation | ✅ Done | Protected by JWT and Admin Secret Key |
| Profile Management (`/users/me`) | ✅ Done | Get and patch current user details |