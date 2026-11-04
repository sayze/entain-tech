# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project context

This is the Entain backend technical test (see `README.md` for the full brief). Two independent Go modules (Go 1.16, module paths `git.neds.sh/matty/entain/{api,racing}`) make up the starting point. The brief asks for five incremental changes, each delivered as its own PR stacked on the previous one:

1. Add a `ListRaces` filter so callers can ask for visible races only, while still being able to fetch all races. Name it following https://cloud.google.com/apis/design/standard_methods#list
2. Order results by `advertised_start_time`. Bonus: let the caller choose the sort field and direction.
3. Add a derived `status` field (`OPEN`/`CLOSED`): races with an `advertised_start_time` in the past are `CLOSED`.
4. Add an RPC that fetches a single race by ID (Google API "Get" standard method).
5. Add a separate `sports` service with a `ListEvents` RPC that works like racing. An event needs at least `id`, `name` and `advertised_start_time`. It must be its own service and module, reached through the `api` gateway (see `example.png`), not added onto `racing`.

The reviewers want the code documented and commented. The README points to the Uber Go style guide.

## Commands

Run each module from its own directory. There is no root module.

```bash
# Run racing gRPC service (listens on :9000)
cd racing && go build && ./racing

# Run REST gateway (listens on localhost:8000, proxies to localhost:9000)
cd api && go build && ./api

# Smoke test
curl -X POST http://localhost:8000/v1/list-races -H 'Content-Type: application/json' -d '{"filter": {}}'

# Regenerate protobuf code after editing a .proto (run in api/ and racing/)
go generate ./...

# One-time install of the protoc plugins needed by go generate (protoc must also be installed)
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 google.golang.org/grpc/cmd/protoc-gen-go-grpc google.golang.org/protobuf/cmd/protoc-gen-go

# Tests (none exist yet), e.g. a single test:
cd racing && go test ./db -run TestName -v
```

CI (`.gitlab-ci.yml`) only runs `go generate ./... && go build` in `racing` and then `api`. It runs no tests and no linter.

## Architecture

The request flow is: HTTP client → `api` (grpc-gateway REST proxy) → `racing` (gRPC server) → SQLite.

- **Proto definitions are duplicated.** `racing/proto/racing/racing.proto` and `api/proto/racing/racing.proto` describe the same service and must be kept in sync by hand. The `api` copy adds `google.api.http` annotations, which define the REST routes (e.g. `POST /v1/list-races`, `body: "*"`), and also generates the `*.pb.gw.go` gateway code. The `go:generate` directives live in `racing/proto/racing.go` and `api/proto/api.go`, and their `protoc` flags differ: racing uses `require_unimplemented_servers=false`, api uses `paths=source_relative` and the gateway plugin. Generated `*.pb.go` files are committed, so regenerate them whenever you change a proto.
- **Gateway wiring**: `api/main.go` registers each backend service with `Register<Service>HandlerFromEndpoint` on one `runtime.ServeMux`. A new service (e.g. sports) needs its own annotated proto copy under `api/proto/`, a `go:generate` entry, another registration call, and its own `--*-endpoint` flag.
- **racing layering**: `main.go` (wiring) → `service/` (gRPC handler implementing the generated `RacingServer` interface; it delegates to the repo) → `db/` (repository).
  - `db/races.go`: `RacesRepo` interface. `List` builds SQL from the base query in `db/queries.go` and adds `WHERE` clauses in `applyFilter` (parameterized `?` placeholders joined with `AND`). `scanRaces` maps rows to proto `Race` and converts times with `ptypes.TimestampProto`.
  - `db/db.go`: `seed()` creates the `races` table and runs `INSERT OR IGNORE` for 100 faker-generated races, with start times between now−1 day and now+2 days. It runs once via `sync.Once` in `Init()`.
  - `db/racing.db` is a committed SQLite file opened through the relative path `./db/racing.db`, so the binary must run from inside `racing/`. Because inserts are `OR IGNORE`, existing rows (and their start times) are never refreshed. If you change the schema, the existing DB file will not pick up the change from `CREATE TABLE IF NOT EXISTS`. Delete or migrate it.
- **Gotcha**: `racing/main.go` hardcodes `net.Listen("tcp", ":9000")`. The `--grpc-endpoint` flag only changes the log message.
- `tools.go` in each module (build tag `tools`) pins the protoc plugin versions in `go.mod`.
