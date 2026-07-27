# URL Shortener

[![CI](https://github.com/ErenKarakus1/URL-Shortener/actions/workflows/ci.yml/badge.svg)](https://github.com/ErenKarakus1/URL-Shortener/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.26.5-00ADD8?logo=go&logoColor=white)
![React](https://img.shields.io/badge/React-19.2.8-20232A?logo=react&logoColor=white)
![MongoDB](https://img.shields.io/badge/MongoDB-8.0-47A248?logo=mongodb&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-8-FF4438?logo=redis&logoColor=white)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A minimal full-stack URL shortener built with React, Go, Gin, MongoDB, and
Redis.

It provides two operations:

- Submit a long URL and receive a deterministic six-character short code.
- Visit a short code in React, review the destination, and continue automatically or manually.

## Demo

[![URL Shortener demo](https://img.youtube.com/vi/mDih5SVSsW8/maxresdefault.jpg)](https://youtu.be/mDih5SVSsW8)

## Architecture

```mermaid
flowchart LR
    User[User] --> Browser[Browser running React]

    Browser <-->|App files and API traffic| Nginx[Nginx]
    Nginx <-->|Proxied /api traffic| Backend[Go + Gin API]

    Backend <-->|URL mappings| MongoDB[(MongoDB)]
    Backend <-->|Rate-limit counters| Redis[(Redis)]
```

Nginx serves the React application and proxies `/api` requests to the Go
backend. MongoDB stores URL mappings, while Redis keeps temporary rate-limit
counters.

## Run with Docker

Docker Compose runs the React frontend, Nginx API proxy, Go backend, MongoDB,
and Redis together:

```powershell
docker compose up --build
```

Open `http://localhost:3000`. Stop the stack with:

```powershell
docker compose down
```

MongoDB data is kept in the `mongodb_data` Docker volume. Copy `.env.example`
to `.env` if you want to change the frontend port or use a different MongoDB
connection.

## Run locally

### Requirements

- Go 1.26.5 or newer
- Node.js 24
- MongoDB running locally or a MongoDB Atlas connection string
- Redis running locally

## Configuration

The backend reads these optional environment variables:

| Variable | Default |
| --- | --- |
| `MONGODB_URI` | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | `url_shortener` |
| `REDIS_ADDRESS` | `localhost:6379` |
| `RATE_LIMIT_REQUESTS` | `30` |
| `RATE_LIMIT_WINDOW_SECONDS` | `60` |
| `SERVER_ADDRESS` | `localhost:8080` |
| `CORS_ALLOWED_ORIGIN` | Empty |

For MongoDB Atlas, set the connection string in PowerShell before starting the backend:

```powershell
$env:MONGODB_URI = "mongodb+srv://USERNAME:PASSWORD@CLUSTER/"
```

Do not commit connection strings containing credentials.

The local Vite proxy and Docker stack use the same origin and do not need CORS.
If the frontend is deployed separately, set `CORS_ALLOWED_ORIGIN` to its exact
origin, such as `https://short.example.com`.

Start the backend:

```powershell
cd backend
go run .
```

In another terminal, install and start the React frontend:

```powershell
cd frontend
npm install
npm run dev
```

Open `http://localhost:5173`. During local development, Vite proxies `/api`
requests to the backend at `http://localhost:8080`.

For a deployed frontend, set `VITE_API_URL` to the public backend API URL and
`VITE_SHORT_URL_BASE` to the public base used for generated links. By default,
generated links use the same origin as the React application.

## Project structure

```text
backend/
├── main.go          # Configuration and application startup
├── server.go        # Router and HTTP handlers
├── url.go           # URL models, validation, and short-code generation
├── mongo_store.go   # MongoDB persistence
├── rate_limiter.go  # Redis rate limiting
└── cors.go          # CORS middleware

frontend/
├── src/             # React application and tests
├── nginx.conf       # SPA routing and /api proxy
└── Dockerfile       # Production frontend image
```

## API

The Go backend exposes the routes below directly. Through the Dockerized Nginx
frontend, prefix them with `/api`, such as `POST /api/shorten`.

### Shorten a URL

```http
POST /shorten
Content-Type: application/json

{
  "longurl": "https://example.com/path"
}
```

A new mapping returns `201 Created`:

```json
{
  "shorturl": "m8ApTP",
  "longurl": "https://example.com/path"
}
```

Submitting the same normalized URL returns the same mapping with `200 OK`.

PowerShell example:

```powershell
Invoke-RestMethod `
  -Method Post `
  -Uri "http://localhost:8080/shorten" `
  -ContentType "application/json" `
  -Body '{"longurl":"https://example.com/path"}'
```

### Resolve a short URL

```http
GET /resolve/m8ApTP
```

Known codes return the destination as JSON with `200 OK`. Unknown codes return
`404 Not Found`. React uses this endpoint to show a five-second countdown page.
**Redirect now** continues immediately. **Cancel** shows a three-second
countdown before returning to the main page.

### Health

```http
GET /health
```

Returns `200 OK` when the Go server is running.

## Behavior

- Only HTTP and HTTPS URLs with a dotted hostname are accepted.
- URLs without a scheme default to HTTPS.
- Surrounding spaces are removed.
- The scheme and hostname are normalized to lowercase.
- Path, query, and fragment casing is preserved.
- The same normalized URL produces the same short code.
- Hash collisions are resolved deterministically by retrying with an incrementing salt.
- Short codes are protected by a unique MongoDB index.
- MongoDB operations time out after three seconds.
- Request bodies are limited to 4 KiB and URLs to 2,048 characters.
- Redis limits each client to 30 shortening requests per 60-second window by default.
- Container health checks cover MongoDB, Redis, the Go API, and Nginx.

## Tests

```powershell
cd backend
go test ./...
```

Frontend tests and production build:

```powershell
cd frontend
npm test
npm run build
```

GitHub Actions runs the backend tests, frontend tests, production build, and
Docker image builds on every push and pull request.

## Known limitations

- URL validation checks structure only; it does not confirm that a domain exists, is reachable, or is safe.
- Dotted hostnames are required, so single-label development hosts such as `localhost` are rejected.
- Links do not expire and cannot be edited or deleted through the application.
- The service has no accounts, custom short codes, analytics, or administrative interface.
- Rate limiting uses a fixed window per client IP, and its Redis counters reset when Redis restarts.
- The health endpoint confirms that the API is running but does not check MongoDB or Redis readiness.
- Unknown short-link pages are rendered by the React SPA with HTTP `200`; the resolve API still returns the correct `404`.
- Short-code generation retries at most 100 deterministic hashes if collisions occur.

## Future improvements

The current version intentionally focuses on the core shortening and redirect
flow. Possible next improvements include:

- Add integration and end-to-end tests for MongoDB, Redis, and browser flows.
- Improve health checks so they report dependency readiness, not only API availability.
- Add structured server logs and request IDs for production troubleshooting.
- Add graceful shutdown so in-flight requests can finish during deployments.
- Replace the fixed-window rate limiter with a smoother sliding-window or token-bucket strategy.
- Add configurable link expiration and automatic cleanup of expired mappings.
- Add optional custom short codes while preserving uniqueness checks.
- Improve accessibility with automated checks and broader keyboard and screen-reader testing.
- Add production deployment documentation covering HTTPS, secrets, backups, and monitoring.

## License

This project is licensed under the [MIT License](LICENSE).
