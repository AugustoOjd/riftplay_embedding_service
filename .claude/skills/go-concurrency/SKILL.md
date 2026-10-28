---
name: go-concurrency
description: >-
  Use when writing, reviewing, or debugging concurrent Go code. Covers goroutine
  lifecycle management, channels, errgroup, mutexes, atomics, sync.Map, and
  synchronous-first design. Based on Google and Uber style guides.
version: "2.0.0"
license: MIT
metadata:
  author: saisudhir14
  tags: golang go concurrency goroutines channels mutex atomic errgroup
  category: languages
---

# Go Concurrency

Patterns for safe, efficient concurrent Go code.

## Channel Size

Channels should have size zero (unbuffered) or one. Any other size requires justification about what prevents filling under load.

```go
// Wrong: arbitrary buffer
c := make(chan int, 64)

// Correct
c := make(chan int)    // unbuffered: synchronous handoff
c := make(chan int, 1) // buffered: allows one pending send
```

## Goroutine Lifetimes

Document when and how goroutines exit. Goroutines blocked on channels will not be garbage collected even if the channel is unreachable.

```go
// Run exits when ctx is canceled, when w.jobs is closed, or on the first
// processing error.
func (w *Worker) Run(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case job, ok := <-w.jobs:
            if !ok {
                return nil // channel closed: clean exit
            }
            if err := w.process(ctx, job); err != nil {
                return fmt.Errorf("process job: %w", err)
            }
        }
    }
}
```

## Use errgroup for Concurrent Operations

Prefer `errgroup.Group` over manual `sync.WaitGroup` for error-returning goroutines.

```go
import "golang.org/x/sync/errgroup"

func processItems(ctx context.Context, items []Item) error {
    g, ctx := errgroup.WithContext(ctx)

    for _, item := range items {
        g.Go(func() error {
            return process(ctx, item)
        })
    }

    return g.Wait() // returns first error, cancels others via ctx
}

// With concurrency limit
func processItemsLimited(ctx context.Context, items []Item) error {
    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(10) // max 10 concurrent goroutines

    for _, item := range items {
        g.Go(func() error {
            return process(ctx, item)
        })
    }

    return g.Wait()
}
```

## Prefer Synchronous Functions

Synchronous functions are easier to reason about and test. Let callers add concurrency when needed.

```go
// Wrong: forces concurrency on caller
func Fetch(url string) <-chan Result

// Correct: caller can wrap in goroutine if needed
func Fetch(url string) (Result, error)
```

## Zero Value Mutexes

The zero value of `sync.Mutex` is valid. Do not use pointers to mutexes or embed them in exported structs.

```go
// Wrong
mu := new(sync.Mutex)

// Wrong: exposes Lock/Unlock in API
type SMap struct {
    sync.Mutex
    data map[string]string
}

// Correct
type SMap struct {
    mu   sync.Mutex
    data map[string]string
}
```

## Atomic Operations (Go 1.19+)

Use the standard library's typed atomics.

```go
import "sync/atomic"

type Counter struct {
    value atomic.Int64
}

func (c *Counter) Inc() { c.value.Add(1) }
func (c *Counter) Value() int64 { return c.value.Load() }

// Also available: atomic.Bool, atomic.Pointer[T], atomic.Uint32, etc.
```

## sync.Map Performance (Go 1.24+)

The `sync.Map` implementation was significantly improved in Go 1.24. Modifications of disjoint sets of keys are much less likely to contend on larger maps, and there is no longer any ramp-up time required to achieve low-contention loads.

## Context Cancellation

Always select on `ctx.Done()` in long-running goroutines to allow clean cancellation.

```go
func longOperation(ctx context.Context) error {
    resultCh := make(chan result, 1)

    go func() {
        // Note: if ctx is canceled, this goroutine still runs to completion
        // and sends to the buffered channel (then gets GC'd).
        // Pass ctx into the work function if it supports cancellation.
        resultCh <- expensiveWork()
    }()

    select {
    case <-ctx.Done():
        return ctx.Err()
    case r := <-resultCh:
        return r.err
    }
}
```

## Graceful Shutdown

Tie the process lifetime to OS signals with `signal.NotifyContext`, then give
in-flight work a bounded window to finish:

```go
func run() error {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    // ... start workers with ctx ...

    <-ctx.Done() // signal received

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    return client.Stop(shutdownCtx) // e.g. River: waits for running jobs, then returns
}
```

Use a fresh context for shutdown: the signal context is already canceled.