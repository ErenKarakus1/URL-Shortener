# URL Shortener

A minimal URL shortener backend built with Go, Gin, and MongoDB.

It provides two operations:

- Submit a long URL and receive a deterministic six-character short code.
- Visit a short code and receive a permanent redirect to the long URL.

## Requirements

- Go 1.26 or newer
- MongoDB running locally or a MongoDB Atlas connection string

## Configuration

The backend reads these optional environment variables:

| Variable | Default |
| --- | --- |
| `MONGODB_URI` | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | `url_shortener` |
| `SERVER_ADDRESS` | `localhost:8080` |

For MongoDB Atlas, set the connection string in PowerShell before starting the backend:

```powershell
$env:MONGODB_URI = "mongodb+srv://USERNAME:PASSWORD@CLUSTER/"
```

Do not commit connection strings containing credentials.

## Run

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

For a deployed frontend, set `VITE_API_URL` to the public backend URL when
building the app.

## API

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

### Follow a short URL

```http
GET /m8ApTP
```

Known codes return `301 Moved Permanently`. Unknown codes return `404 Not Found`.

## Behavior

- Only HTTP and HTTPS URLs with a hostname are accepted.
- Surrounding spaces are removed.
- The scheme and hostname are normalized to lowercase.
- Path, query, and fragment casing is preserved.
- The same normalized URL produces the same short code.
- Hash collisions are resolved deterministically by retrying with an incrementing salt.
- Short codes are protected by a unique MongoDB index.
- MongoDB operations time out after three seconds.

## Tests

```powershell
cd backend
go test ./...
```
