# go-service-template

Baseline for Go services, frontend (templ + htmx) and backend (JSON API)
alike. It ships one small example feature, `notes`, that exercises every
layer: a JSON API, an HTML page, a Postgres store with migrations, and tests.

## Start a new service

```bash
go run golang.org/x/tools/cmd/gonew@latest github.com/soockee/go-service-template github.com/soockee/<name>
```

Then replace `internal/notes` with your own feature, and its routes,
handlers, migration and store.

**API-only service:** delete `internal/views`, `assets/`, the page routes in
`internal/server/routes.go`, the page handlers, the `page` adapter and
`render.go`, and the Tailwind lines in the Makefile and Dockerfile.

## Layout

```
cmd/server/main.go        wiring only: config → logger → store → server
internal/config/          every setting, read from env, validated at startup
internal/server/          http.Server, routes.go, middleware, handlers, error mapping
internal/<feature>/       domain types, rules, and the Store interface it needs
internal/store/           Postgres: pool, embedded goose migrations, Store implementations
internal/views/           templ components (frontend)
assets/                   static files, embedded into the binary (frontend)
```

## Conventions

- **Handlers** are `func(w, r) error` methods on `Server`. Return an error
  before writing anything. `toHTTPError` in `respond.go` maps domain errors
  to status codes in one place; unknown errors become a 500 with details only
  in the log.
- **Routes** all live in `routes.go`, using stdlib patterns (`"GET /api/notes/{id}"`).
- **Config** comes from env vars only, read in `internal/config`. See `.env.example`.
- **Logging** uses `log/slog`: text in dev, JSON in prod, and every request
  log line carries `request_id`.
- **Persistence** uses pgx with SQL in the store package. Migrations live in
  `internal/store/migrations` (goose) and are applied at startup. Without
  `DATABASE_URL` the service uses an in-memory store, which only `APP_ENV=dev`
  allows.
- **Generated templ code is committed**, so `go build` works without
  tools. CI fails if it is stale.
- **Tests** go through the real router with `httptest`. Store tests hit a
  real Postgres and are skipped unless `TEST_DATABASE_URL` is set.
- **Commits** follow Conventional Commits. release-please opens release
  PRs, and a `v*` tag publishes a multi-arch image to GHCR.

## Develop

| Task | Command (make) | Without make (e.g. Windows) |
|---|---|---|
| Regenerate templ | `make generate` | `go tool templ generate` |
| Build CSS | `make css` | Tailwind standalone CLI, see Makefile |
| Run (in-memory) | `make run` | `go run ./cmd/server` |
| Run with Postgres | `make db-up`, set `DATABASE_URL` | `docker compose up -d db` |
| Test | `make test` | `go test ./...` |
| Test incl. store | `make test-db` | set `TEST_DATABASE_URL`, `go test ./...` |
| Lint | `make lint` | `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 run` |
| Image | `make docker` | `docker build .` |

Endpoints: `GET /` (page), `GET|POST /api/notes`, `GET /api/notes/{id}`,
`GET /healthz` (liveness), `GET /readyz` (database ping).

## Deploy

The image is `distroless/static:nonroot`, listens on `:8080` and runs with
`APP_ENV=prod`, which requires `DATABASE_URL`. Terminate TLS at a reverse
proxy (Caddy or Traefik) in front of it.

CI secrets: `RELEASE_PLEASE_TOKEN`, a PAT with contents and pull-request
write access, so release tags trigger the publish workflow.
