---
name: golang-pro
description: Implements concurrent Go patterns using goroutines and channels, designs and builds microservices with gRPC or REST, optimizes Go application performance with pprof, and enforces idiomatic Go with generics, interfaces, and robust error handling. Use when building Go applications requiring concurrent programming, microservices architecture, or high-performance systems. Invoke for goroutines, channels, Go generics, gRPC integration, CLI tools, benchmarks, or table-driven testing.
license: MIT
metadata:
  author: https://github.com/Jeffallan
  version: "1.1.0"
  domain: language
  triggers: Go, Golang, goroutines, channels, gRPC, microservices Go, Go generics, concurrent programming, Go interfaces
  role: specialist
  scope: implementation
  output-format: code
  related-skills: devops-engineer, microservices-architect, test-master
---

# Golang Pro

Senior Go developer with deep expertise in Go 1.21+, concurrent programming, and cloud-native microservices. Specializes in idiomatic patterns, performance optimization, and production-grade systems.

## Core Workflow

1. **Analyze architecture** — Review module structure, interfaces, and concurrency patterns
2. **Design interfaces** — Create small, focused interfaces with composition
3. **Implement** — Write idiomatic Go with proper error handling and context propagation; run `go vet ./...` before proceeding
4. **Lint & validate** — Run `golangci-lint run` and fix all reported issues before proceeding
5. **Optimize** — Profile with pprof, write benchmarks, eliminate allocations
6. **Test** — Table-driven tests with `-race` flag, fuzzing where input is untrusted (parsers, extractors); confirm race detector passes before committing

## Reference Guide

Load detailed guidance based on context:

| Topic | Reference | Load When |
|-------|-----------|-----------|
| Concurrency | `references/concurrency.md` | Goroutines, channels, select, sync primitives |
| Interfaces | `references/interfaces.md` | Interface design, io.Reader/Writer, composition |
| Generics | `references/generics.md` | Type parameters, constraints, generic patterns |
| Testing | `references/testing.md` | Table-driven tests, benchmarks, fuzzing |
| Project Structure | `references/project-structure.md` | Module layout, internal packages, go.mod |

## Core Pattern Example

Bounded concurrent processing with context cancellation and error propagation:

```go
import "golang.org/x/sync/errgroup"

// runPipeline processes jobs with at most 4 in flight. It returns the first
// error; that error cancels ctx, so the remaining jobs stop early.
func runPipeline(ctx context.Context, jobs []Job) error {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(4)

    for _, job := range jobs {
        g.Go(func() error {
            if err := process(ctx, job); err != nil {
                return fmt.Errorf("process job %v: %w", job.ID, err)
            }
            return nil
        })
    }
    return g.Wait()
}
```

Key properties demonstrated: bounded goroutine lifetime via `ctx`, bounded
parallelism via `SetLimit`, error propagation with `%w`, no hand-rolled
channels to leak or deadlock, and success returns immediately (no waiting for
the timeout).

## Project Overrides (riftplay ingest)

This skill is generic. In this repository, when it disagrees with any of the
following, they win:

- **CLAUDE.md** — stack and rules. Postgres via `pgx/v5` + `sqlc` (not
  `database/sql`/`lib/pq`/GORM), logging via `log/slog` (not `zap`/`logrus`), no
  HTTP framework (no `gin`/`echo`), no gRPC.
- **`go-project-layout`** — everything under `internal/`, **no `pkg/`**, Go version
  from `go.mod` (not the `go 1.21` in `references/project-structure.md`).
- **`go-concurrency`** — channels are unbuffered or size 1 unless justified;
  prefer `errgroup` over `sync.WaitGroup` for goroutines that return errors.
- **Mocks** — hand-written fakes of small interfaces (e.g. the `LLM` interface).
  Do not use `github.com/golang/mock` (archived); if a generator is ever needed,
  `go.uber.org/mock`, and only after asking.
- **`any`** instead of `interface{}` in new code.
- **Coverage** — no fixed 80% target; test the behavior that matters (sync
  idempotency, extractors against `testdata/` fixtures, agent loop with a fake LLM).

Treat `references/project-structure.md` as background only; its `go.mod`, `pkg/`
tree and mockgen setup do not apply here.

## Constraints

### MUST DO
- Use gofmt and golangci-lint on all code
- Add context.Context to all blocking operations
- Handle all errors explicitly (no naked returns)
- Write table-driven tests with subtests
- Document all exported functions, types, and packages
- Use `X | Y` union constraints for generics (Go 1.18+)
- Propagate errors with fmt.Errorf("%w", err)
- Run race detector on tests (-race flag)

### MUST NOT DO
- Ignore errors (avoid _ assignment without justification)
- Use panic for normal error handling
- Create goroutines without clear lifecycle management
- Skip context cancellation handling
- Use reflection without performance justification
- Mix sync and async patterns carelessly
- Hardcode configuration (use functional options or env vars)

## Output Templates

When implementing Go features, provide:
1. Interface definitions (contracts first)
2. Implementation files with proper package structure
3. Test file with table-driven tests
4. Brief explanation of concurrency patterns used

## Knowledge Reference

Go 1.21+, goroutines, channels, select, sync package, generics, type parameters, constraints, io.Reader/Writer, gRPC, context, error wrapping, pprof profiling, benchmarks, table-driven tests, fuzzing, go.mod, internal packages, functional options

[Documentation](https://jeffallan.github.io/claude-skills/skills/language/golang-pro/)