# User Management API

Go backend challenge built with Gin, MongoDB, and JWT.

## Features

- User registration and login
- JWT authentication with HS256
- List, get, update, and delete users
- Update/delete restricted to the authenticated user's own resource
- Bcrypt password hashing
- MongoDB unique email index
- Request logging middleware
- Background worker logging user count every 10 seconds
- Graceful shutdown
- Docker and Docker Compose support

## Tech Stack

- Go 1.27
- Gin
- MongoDB official driver
- JWT v5
- Bcrypt
- `log/slog`
- Docker with a Distroless runtime image

## Project Structure

```text
cmd/api                    Application entrypoint
internal/app               Application wiring and lifecycle
internal/application       Application services
internal/user              User domain and repository port
internal/adapters/http     HTTP handlers, router, and middleware
internal/adapters/mongodb  MongoDB adapter
internal/adapters/security JWT and password adapters
internal/adapters/worker   Background workers
```

## Environment Setup

Copy `.env.example` to `.env`:

macOS / Linux:

```bash
cp .env.example .env
```

Windows PowerShell:

```powershell
Copy-Item .env.example .env
```

Windows Command Prompt:

```cmd
copy .env.example .env
```

Configure these variables in `.env`:

```env
SERVICE_NAME=user-management
PORT=8080
MONGO_URI=mongodb://localhost:27017
MONGO_DATABASE=user_management
JWT_SECRET=replace-with-a-secure-secret
```

## Run Locally

```bash
docker compose up -d mongodb
go run ./cmd/api
```

The API is available at `http://localhost:8080`.

## Run with Docker Compose

```bash
docker compose up --build
```

Compose connects the API to MongoDB using `mongodb://mongodb:27017`.

## API Endpoints

```text
API
├── Health
├── Authentication
│   ├── Register
│   └── Login
└── Users
    ├── List
    ├── Get by ID
    ├── Update by ID
    └── Delete by ID
```

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/health` | No | Health check |
| POST | `/auth/register` | No | Register user |
| POST | `/auth/login` | No | Login and receive JWT |
| GET | `/users` | Bearer | List users |
| GET | `/users/:id` | Bearer | Get user by ID |
| PATCH | `/users/:id` | Bearer + owner | Update own user |
| DELETE | `/users/:id` | Bearer + owner | Delete own user |

## Request / Response Examples

### Health

GET `/health`

Response:

```http
200 OK
```

```json
{
  "status": "ok"
}
```

### Register

POST `/auth/register`

```json
{
  "name": "Thitiphum",
  "email": "thiti@example.com",
  "password": "secret"
}
```

Response:

```http
201 Created
```

### Login

POST `/auth/login`

```json
{
  "email": "thiti@example.com",
  "password": "secret"
}
```

Response:

```http
200 OK
```

```json
{
  "token": "<jwt>",
  "user": {
    "id": "68ca...",
    "name": "Thitiphum",
    "email": "thiti@example.com"
  }
}
```

### List Users

GET `/users`

Response:

```json
[
  {
    "id": "68ca...",
    "name": "Thitiphum",
    "email": "thiti@example.com",
    "created_at": "2026-09-18T10:00:00Z"
  }
]
```

### Get User by ID

GET `/users/:id`

Response:

```json
{
  "id": "68ca...",
  "name": "Thitiphum",
  "email": "thiti@example.com",
  "created_at": "2026-09-18T10:00:00Z"
}
```

### Update User

PATCH `/users/:id`

```json
{
  "name": "Thitiphum Updated"
}
```

Response:

```http
204 No Content
```

### Delete User

DELETE `/users/:id`

Response:

```http
204 No Content
```

## Authentication

Login returns a JWT. Send it with protected requests:

```yaml
Authorization: Bearer <token>
```

JWT claims: `sub`, `iat`, and `exp`.

`PATCH` and `DELETE` require the token subject to match `:id`.

Example:

```bash
curl http://localhost:8080/users \
  -H "Authorization: Bearer <token>"
```

## Testing

```bash
go test ./...
```

Tests cover application services and HTTP handlers using Go's standard `testing` package.

## Design Notes

- Hexagonal-lite architecture with consumer-owned interfaces.
- MongoDB adapter owns BSON and ObjectID mapping.
- Domain/application layers do not depend on Gin or MongoDB.
- Email is normalized before persistence.
- MongoDB's unique index enforces email uniqueness safely under concurrency.
- JWT stores the authenticated user ID in the `sub` claim; mutable user data is not embedded in the token.

## Assumptions

- No RBAC or admin role.
- Any authenticated user may list and read user records.
- Users may update or delete only their own record.
- The optional gRPC bonus is not implemented.

## Graceful Shutdown

On `SIGINT` or `SIGTERM`, the application:

1. Shuts down the HTTP server.
2. Stops and waits for the background worker.
3. Disconnects from MongoDB last.
