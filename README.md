# Olympsis Server

The core backend service for Olympsis — a REST API written in Go that powers users,
clubs, organizations, events, venues, social posts, and notifications.

## Overview

`olympsis-server` is a monolithic HTTP service organized into self-contained domain
modules (one Go package per domain). Each module exposes an `api.go` that registers its
routes against a shared [`gorilla/mux`](https://github.com/gorilla/mux) router and a
`service/` package that holds the business logic. A single `ServerInterface`
(see `server/models.go`) is constructed in `main.go` and passed into every module,
giving each one access to the shared dependencies (database, cache, auth, storage, etc.).

### Tech stack

| Concern              | Technology                                              |
| -------------------- | ------------------------------------------------------- |
| Language             | Go 1.24                                                 |
| HTTP router          | `gorilla/mux`                                            |
| Primary database     | MongoDB (`mongo-driver/v2`)                              |
| Cache                | Redis (`go-redis/v9`)                                    |
| Authentication       | Firebase Auth                                            |
| Payments             | Stripe (`stripe-go/v82`) — club finance                 |
| Push notifications   | Apple Push Notification service (APNs, `apns2`)          |
| File storage / media | Google Cloud Storage + Vision (image moderation)        |
| Maps                 | Apple MapKit (map snapshots + server tokens)            |
| Models               | Shared `github.com/olympsis/models` package             |

## Architecture

```
main.go                  # Wires up dependencies, builds ServerInterface, registers all APIs, starts HTTP server
server/                  # ServerInterface — the shared dependency container passed to every module
database/                # MongoDB connection + wrapper
redis/                   # Redis client wrapper (cache)
middleware/              # Auth (user/admin/club-admin/super-admin), gzip, logging, JSON, chain
notifications/           # APNs notification service
storage/                 # GCP Storage upload service (other modules depend on it)
aggregations/            # MongoDB aggregation pipelines per domain
utils/                   # Config loading, secrets, MapKit, validators, helpers
types/                   # Shared interfaces (e.g. StorageUploader)
tools/                   # Init scripts, nginx config, prod setup helpers

# Domain modules (each: api.go + service/)
announcement/  auth/  user/  club/  organization/  event/
post/  venue/  report/  locales/  health/  map-snapshots/  system/
```

### Module pattern

Each domain module follows the same shape:

1. `NewXxxAPI(serverInterface)` — constructs the module with shared dependencies.
2. `Ready(...)` — registers the module's routes on the router.

Routes are composed with `middleware.Chain(handler, ...middleware)`, where the
innermost handler is the service method and middleware wraps it (auth, logging, etc.).
The storage module is initialized **first** in `main.go` because other modules depend
on it for media uploads.

## API surface

All routes are versioned under `/v1`. A non-exhaustive map of the domains:

| Domain         | Base path(s)                                            |
| -------------- | ------------------------------------------------------- |
| Auth           | `/v1/auth/{login,register,modify,delete}`               |
| Users          | `/v1/users`, `/v1/users/search/*`, `/v1/users/check-in` |
| Clubs          | `/v1/clubs`, `/v1/clubs/{id}/...` (members, posts, finance) |
| Organizations  | `/v1/organizations`, `/v1/organizations/{id}/...`       |
| Events         | `/v1/events`, `/v1/events/{id}/...`, `/v1/events/location`, `/v1/events/past/...` |
| Posts          | `/v1/posts`, `/v1/posts/{id}/{comments,likes}`          |
| Venues         | `/v1/venues`, `/v1/venues/{id}/units`                   |
| Announcements  | `/v1/announcements`                                     |
| Reports        | `/v1/report/{bugs,events,fields,members,posts}`         |
| Locales        | `/v1/locales/countries`, `/v1/locales/.../administrativeAreas` |
| Notifications  | `/v1/notifications`                                     |
| Storage        | `/v1/storage/{fileBucket}`                              |
| System         | `/v1/system/config`, `/v1/system/mapkit-server-token`   |
| Health         | `/v1/health`, `/v1/health/wsg`                          |
| Map snapshots  | `/v1/map-snapshot`                                      |

Club finance (`/v1/clubs/{id}/finance/...`) is backed by Stripe and covers accounts,
payouts, transactions, and customer sheets.

## Configuration

Configuration is read from environment variables (loaded via the `secrets` manager in
`utils/secrets`). Copy `.env.example` and fill in the values:

```sh
cp .env.example .env.dev
```

Key variables:

| Variable                          | Purpose                                            |
| --------------------------------- | -------------------------------------------------- |
| `PORT`                            | Listen port (defaults to `80`)                     |
| `MODE`                            | `DEVELOPMENT` or `PRODUCTION`                       |
| `HTTP`                            | `SECURE` (TLS) or `UNSECURE` (plain HTTP)          |
| `KEY_FILE_PATH` / `CERT_FILE_PATH`| TLS key/cert paths (required when `HTTP=SECURE`)   |
| `MONGO_ADDRESS` / `_USERNAME` / `_PASSWORD` | MongoDB connection                       |
| `REDIS_ADDRESS`                   | Redis connection                                   |
| `FIREBASE_FILE_PATH`              | Firebase service-account credentials               |
| `APPLE_KEY_ID` / `APPLE_TEAM_ID` / `APNS_FILE_PATH` | APNs auth (`.p8` key)            |
| `STORAGE_FILE_PATH`               | GCP credentials for Storage + Vision               |
| `MAPKIT_TOKEN` / `MAPKIT_FILE_PATH` / `MAPKIT_KEY_ID` | Apple MapKit tokens             |

> Collection names (MongoDB) are also configured via environment variables. The
> production defaults are baked into the [`Dockerfile`](./Dockerfile); see `utils/config.go`
> (`GetCollectionsConfig`) for the full list.

When `MODE` is not `PRODUCTION`, the config falls back to local credential files under
`./files/` (Firebase, APNs key, etc.).

## Development

Requires **Go 1.24+**. The `Makefile` wraps the common workflows.

### Run locally

```sh
# Run directly against your .env (uses go run)
make run

# or build the binary and run it
make build
./olympsis-server
```

### Run the full local stack (Docker Compose)

Brings up the server alongside MongoDB, Redis, and an APISIX gateway using
`compose.dev.yaml`. Expects credential files in `./files/` (see the `secrets` and
`volumes` blocks in the compose file).

```sh
make dev-up     # start the stack (detached)
make dev-down   # tear it down
```

### Tests & quality

```sh
make test   # go test -short ./...
make race   # run with the data-race detector
make lint   # golint
```

## Building & deployment

```sh
make docker-build   # build a local (unsecure) image
make artifact       # build, tag, and push the release image to GCP Artifact Registry
make server         # build a TLS image (local CA certs) and run it on :443
make unsecure-server# build a plain-HTTP image and run it on :80
```

Images are published to GCP Artifact Registry under
`us-central1-docker.pkg.dev/olympsis-485522/server/release:<VERSION>`.

> The compiled `olympsis-server` binary is tracked with **Git LFS** (see
> `.gitattributes`). The `make mac-mini` target builds the binary, ships it to the
> deployment host, and tags the release. Bump `VERSION` at the top of the `Makefile`
> before cutting a release.

## Runtime behavior

- A default `Content-Type: application/json` and gzip compression are applied to all routes globally.
- The server runs over plain HTTP or TLS depending on `HTTP` (`UNSECURE` / `SECURE`).
- Graceful shutdown is wired to `SIGINT` / `SIGTERM` with a 30s drain timeout.
