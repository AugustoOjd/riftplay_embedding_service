# riftplay embedding service

Module: `github.com/AugustoOjd/riftplay_embedding_service`

Go service that will own the Riftbound rules corpus: fragments, their
embeddings, the daily Rules Hub sync and `POST /search`. The Python `api`
calls it over HTTP. Design and open decisions:
[RAG_DESIGN.md](RAG_DESIGN.md). Conventions mirror `riftplay_ingest_service`.

The domain is being built in pair programming: don't add packages, tables or
endpoints ahead of what's been agreed.

## Stack

| Area | Choice |
|---|---|
| Language | Go 1.27 (pinned in `go.mod`), standard toolchain |
| HTTP | stdlib `net/http` |
| Logging | `log/slog` (JSON handler) |
| Config | env vars, parsed and validated in `internal/config` |

Do not add dependencies outside this table without asking first.

## Layout

```
cmd/embedding/      entry point: config, HTTP server, graceful shutdown
internal/config/    env → validated Config (every error reported at once)
```

## Commands

`make help` lists them. `make check` runs what `/push_github` runs
(lint, tests with `-race`, govulncheck, `go mod tidy -diff`).
Local env goes in `.env` (copy `.env.example`); the Makefile exports it.

## Conventions

- `main.go` stays thin; logic lives under `internal/`.
- Config: add each new variable to `Config`, `Load`, `config_test.go` and
  `.env.example` together. Sensitive values never get logged.
- Errors wrapped with `%w` and context (`"http server: %w"`).
- Table-driven tests, `t.Parallel()`.
