# Orchard

Orchard is a Go-based, local-first job-execution platform. A central server accepts command jobs, tracks worker capacity, assigns jobs, and records completion. A worker process fetches an assigned job, runs its command, and reports the result. The intended end state is multiple Macs acting as workers on the same trusted Wi-Fi network—a small learning-scale counterpart to cloud job schedulers, not an AWS or Kubernetes replacement.

**Current state:** the server, HTTP API, scheduler, in-memory stores, executor, and assigned-job runner exist. The one-shot `worker-run` CLI is in progress. The server and worker client currently use `127.0.0.1:8080`; scheduling and worker invocation are manual. Data disappears when the server stops. Do not expose this unauthenticated command-execution API to a network.

- [ROADMAP.md](ROADMAP.md): checkpoints, definitions of done, current progress, and next actions.
- [IMPLEMENTATION.md](IMPLEMENTATION.md): architecture, code-level tour, request flows, interfaces, and limitations.
- [AGENTS.md](AGENTS.md): working agreement for AI coding guides and progress updates.

## Build and test

From the repository root:

```sh
go test ./...
go test -race ./...
go build -o bin/orchard ./cmd/orchard
```

Run `./bin/orchard server` in one terminal. The current manual flow is: register a worker, submit a job, schedule it, then run `./bin/orchard worker-run <worker-id> <job-id>` in another terminal. See [ROADMAP.md](ROADMAP.md#current-progress) for what has and has not been verified.
