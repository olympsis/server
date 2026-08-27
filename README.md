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

### Cutting a release

The **git tag is the single source of truth** for a version. Pushing a `v*` tag
triggers `.github/workflows/release.yml`, which builds and publishes everything:

```sh
make release V=v0.9.5      # tags, pushes, and lets CD take over
```

That one tag flows into three places that therefore cannot drift apart:

| Artifact | Destination |
| --- | --- |
| Docker image | `northamerica-northeast1-docker.pkg.dev/olympsis-485522/docker-images/server:<VERSION>` (and `:latest`) |
| darwin/arm64 binary | AR generic repo `go-binaries`, package `olympsis-server`, version `<VERSION>` (no `latest` — generic versions are immutable and `latest` is not a valid version id, so pull by explicit version) |
| Binary build stamp | linked in via `-ldflags`, served at `GET /v1/health` |

Pre-release tags (`v1.0.0-rc1`) publish under their own version but deliberately
do **not** move `latest`, so a `pull_policy: always` host never rolls onto an
unfinished build.

### What the running server reports

```console
$ curl -s https://<host>/v1/health
{"msg":"OK","build":{"version":"v0.9.5","commit":"6351f14","buildTime":"2026-08-27T02:45:00Z"}}
```

Those values come from `package version`, set at **link time** — not from the
environment. An env var can be changed without a rebuild, which would let the
server report a version it is not actually running; a linker-set value is welded
to the binary. A local `go build` reports `dev`/`none`/`unknown`, which is the
correct answer for a non-release build.

### Deploying

GitHub-hosted runners cannot reach the Mac mini, so CD publishes to a registry
the mini **pulls from** rather than pushing to it:

```sh
make deploy-mac-mini V=v0.8.0   # gcloud download -> chmod +x -> pm2 restart
```

For the container host, bump the tag in `compose.yaml` and:

```sh
docker compose pull server && docker compose up -d server
```

### invite-service / notif-service in the stack

Both now run as containers in `compose.yaml` rather than as pm2 processes.
The stack reaches them by container name, so the gateway's four invite and two
notif routes moved from `host.docker.internal:{8082,8083}` to
`invite-service:8082` / `notif-service:8083`.

Three things must be true before `docker compose up`, or the cutover misbehaves
in ways that do not announce themselves:

1. **Stop the pm2 instances first.**
   ```sh
   sudo env PATH=/opt/homebrew/bin:$PATH HOME=/Users/joel pm2 stop invite-service notif-service
   ```
   Both consume from the same RabbitMQ queues. Leaving pm2 running alongside
   the containers means two consumers competing for each message, so roughly
   half of all invites and notifications get handled by the old binary. Nothing
   errors — the work just lands in the wrong process.

2. **Ship the updated `krakend.prod.json`.** It is gitignored (`/files/*`), so
   CD never touches it; copy it to the deploy directory by hand. Until then the
   gateway still resolves `host.docker.internal`, reaching the stopped pm2
   processes and returning 502 on those six routes.

3. **Set `RABBITMQ_URL` in the `.env` beside `compose.yaml`.** It has no
   default on purpose: RabbitMQ still runs on the host under pm2, not in this
   stack, and a wrong broker URL fails open — the services start cleanly and
   silently consume nothing. Compose refuses to start without it.

> The services publish a binary as well as an image, so the pm2 path still
> works as a fallback — `make deploy-mac-mini V=...` in either repo.

### Manual / local builds

```sh
make build          # stamped binary via git describe
make docker-build   # build a local (unsecure) image
make artifact       # manual image push (prefer `make release`)
make server         # build a TLS image (local CA certs) and run it on :443
make unsecure-server# build a plain-HTTP image and run it on :80
```

### The private `models` dependency

`github.com/olympsis/models` lives in its own repo. `go.mod` carries a dev-only
`replace ... => ../models` so local and `compose.dev.yaml` builds resolve it from
the sibling checkout — which also means unpushed edits are picked up without
cutting a tag.

> The repo is currently **public**, so the public proxy can serve it. Publishing
> to Artifact Registry is what makes the setup work if it ever goes private —
> see "Going private" below.

CI cannot use that path, so both release jobs run
`go mod edit -dropreplace=github.com/olympsis/models` on the runner and resolve
the module from Artifact Registry instead:

```sh
GOPROXY=https://northamerica-northeast1-go.pkg.dev/olympsis-485522/go-modules,https://proxy.golang.org,direct
GONOSUMDB=github.com/olympsis/*
```

> Use `GONOSUMDB`, **not** `GOPRIVATE`. `GOPRIVATE` implies `GONOPROXY`, which
> would bypass the very proxy we need.

Publishing a new models version is `olympsis/models`' own
`.github/workflows/publish.yml`, triggered the same way — by pushing a `v*` tag.

#### Going private

Making `olympsis/models` private does **not** break anything currently pinned:
`proxy.golang.org` caches module versions permanently and immutably, so every
version the six services pin today stays resolvable even after the repo is
locked down. It does change three things:

1. **New versions stop resolving publicly.** The five services that still pin
   pseudo-versions (`invite-service`, `notif-service`, `chat`, `notif`,
   `search`) resolve models through the public proxy and carry no `replace`.
   They keep building, but any future bump needs the same treatment `server`
   got: pin a published tag and point `GOPROXY` at the AR repo, or set
   `GOPRIVATE` and give the builder git credentials.
2. **`server` is already immune** — its release workflow resolves from Artifact
   Registry with its own credentials.
3. **It does not retract what is already public.** Everything cached in
   `proxy.golang.org` and notarized in `sum.golang.org` stays world-readable
   forever, source zips included. Going private protects future commits, not
   past ones.

> **Retired:** the old flow committed the 42 MB `olympsis-server` binary to Git
> LFS on every release and scp'd it to the mini (`make mac-mini`). That target
> now errors with a pointer to the commands above.

## Runtime behavior

- A default `Content-Type: application/json` and gzip compression are applied to all routes globally.
- The server runs over plain HTTP or TLS depending on `HTTP` (`UNSECURE` / `SECURE`).
- Graceful shutdown is wired to `SIGINT` / `SIGTERM` with a 30s drain timeout.
