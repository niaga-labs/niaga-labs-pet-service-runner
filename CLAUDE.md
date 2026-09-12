# Kilat Pet Delivery - service-runner

Runner profiles, vehicle and crate capabilities, PostGIS proximity search, and the pet-shop directory.
Jira project **KPD** - GitHub `Kilat-Pet-Delivery/service-runner` - stack **Go 1.24 - Gin - GORM - PostgreSQL - Kafka**. Global rules live in `~/.claude/`;
this file only adds what is specific here.

## Orient here first

- `.claude/memory/project_state.md` - **resume here** (`/continue` reads it, `/recap` rewrites it).
- `README.md` - how to run it. `CHANGELOG.md` - what changed.
- The workspace map: `~/Documents/kilat-pet-delivery/CLAUDE.md`.

## Commands

| Task | Command |
|---|---|
| install | `go mod download` |
| run | `go run ./cmd/server` (copy `.env.example` to `.env` first) |
| test | `go test ./...` |
| integration tests | none in this repo |
| lint | `gofmt -l . && go vet ./...` |
| build | `go build ./...` |
| migrate | `go run ./cmd/migrate` - applies `migrations/` and exits |

Needs the dev-infra stack: Postgres database `kilat_runner`, Kafka on `localhost:9092` -> `cd ~/Documents/dev-infra; ./dev.ps1 up kilat`.

## Conventions that differ from the global rules

- **Ticket branches and PRs** - company repo, never commit on `main` (`branch-guard` enforces it).
- **One migration path.** `migrations/` owns the schema in every environment including development, and `cmd/server` applies it at startup. There is deliberately no GORM AutoMigrate branch - that is what let six services drift (KPD-56 through KPD-61).
- Protected paths (never edited in place, see `.claude/protected-paths.txt`): `migrations/*.sql`.

## Testing

No test files in any package. `go test ./...` has nothing to run.

## Where things are

- `cmd/server` - `cmd/migrate` - `internal/domain/{runner,petshop}` - `internal/repository/petshop_seed.go` seeds sample shops in development only

## Worth knowing

- Uses PostGIS for proximity, so the database needs the extension. The dev-infra image has it.
