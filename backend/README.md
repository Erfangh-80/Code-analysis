# Backend

Independent Go module for the local Code Risk Analyzer API.

## Run locally

```bash
cd backend
go run ./cmd/server
```

The server binds to `127.0.0.1:8080` by default. Its only implemented endpoint at this foundation stage is:

```bash
curl http://127.0.0.1:8080/api/v1/health
```

Expected response:

```json
{"status":"ok","version":"0.1.0"}
```

The API exposes no repository-analysis endpoint yet; it does not claim analysis capability until the static-analysis, Git-history, and scoring services are implemented and tested.

## Test

```bash
go test ./...
```

Use `go run ./cmd/server -addr 127.0.0.1:8081` to select another explicit local address.
