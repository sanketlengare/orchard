# Orchard implementation guide

This describes the code **as it exists now**. For the changing status and future work, see [ROADMAP.md](ROADMAP.md). The finished ambition is multiple Macs acting as workers on one Wi-Fi network; the present implementation runs on localhost and requires manual scheduling and one-shot worker invocation.

## Vocabulary and process boundary

- **Job:** a command plus a resource request and lifecycle status. Example: `echo` with argument `hello orchard`, requesting one CPU core and 512 MiB.
- **Worker record:** an ID and total/available CPU and memory capacity held by the server. Registering the record does not start a process on that worker Mac.
- **Control plane:** the server-side code that stores jobs/workers, schedules jobs, and records completion.
- **Worker runner:** client-side code that checks an assignment, executes one command locally, and reports completion.
- **Assignment:** the current server-side mapping from job ID to worker ID. It exists only while that job is running and is removed on finish.
- **HTTP boundary:** the worker and server communicate through requests, even when both processes run on one Mac. Later this boundary can span Macs once configuration and security are added.

Current runtime shape:

```text
submitter (curl)                         server process: orchard server
       │ HTTP                                   │
       └──────────────────────────────► HTTP handler
                                               │
                                         control-plane service
                                          /       |         \
                                     job store  scheduler  worker store
                                               │
                                          assignment map

worker process: orchard worker-run worker1 job1
       │
       ├── HTTP client ───────────────────────► server
       └── runner ──► executor ──► local OS process
```

The server and worker process both target `127.0.0.1:8080` today, so this is **not** yet cross-Mac execution. `internal` is Go's mechanism for packages used only within this module; `cmd/orchard` is the binary entry point.

## Startup and composition

`cmd/orchard/main.go` reads arguments and calls `runWithServer`. `version` prints `orchard dev`; `server` calls `serveHTTP`; `worker-run` calls `runWorker`. Missing/unknown arguments return usage exit code 2, operational errors return 1, success returns 0. `main` passes `os.Args[1:]`, `os.Stdout`, and `os.Stderr`, then exits with that code. Passing writers and a server-running function as arguments makes CLI behavior testable without relying on a terminal or starting a server.

`cmd/orchard/server.go` is the server composition root: it creates `MemoryJobStore` and `MemoryWorkerStore`, passes them to `controlplane.NewService`, passes the service to `httpapi.NewHandler`, and starts `http.Server` on `127.0.0.1:8080` with a five-second read-header timeout. All state belongs to this server process and is lost when it exits.

`cmd/orchard/worker.go` is the worker composition root: it validates `<worker-id> <job-id>`, constructs a `client.Client` for `http://127.0.0.1:8080`, injects `executor.Run` into `agent.New`, and calls `RunJob` with a context canceled by Ctrl+C. It prints command stdout/stderr, prints an error if any, and returns a CLI exit code. This command runs **one named job**, then stops. It neither registers itself nor searches for work.

## Domain model: `internal/job`

`job.ResourceRequest` holds positive `CPUCores` and `MemoryMiB` integers. `job.New` trims and validates ID, checks the command has a nonblank executable, checks resources are positive, copies the command slice, and initializes status to `pending`. The copy means a caller changing its original slice cannot silently alter the stored command; `Job.Command()` also returns a copy.

`Status` is a named string type with four values: `pending`, `running`, `succeeded`, `failed`. `CanTransitionTo` allows only `pending → running` and `running → succeeded|failed`. `TransitionTo` acquires the job mutex, checks that rule, and updates the status. `Status()` uses a read lock. The job has no separate `assigned` state; assignment is tracked in the service map.

## Worker accounting: `internal/worker`

`worker.Capacity` contains CPU and memory integers. `worker.New` validates ID and positive total capacity, and initializes available capacity equal to total. `CanRun` compares a job's request with current available capacity. `Reserve` checks and subtracts both resources; `Release` adds both back but rejects releasing more than is currently used. Access to mutable available capacity uses a mutex. `canRunLocked` is a private helper called while the lock is already held, avoiding a self-deadlock.

These are **logical reservations** used by the scheduler. They do not restrict the CPU or RAM consumed by a spawned OS process.

## Storage: `internal/store`

`MemoryJobStore` and `MemoryWorkerStore` each use a map keyed by ID plus a read/write mutex. `Create`/`Register` reject duplicates; `Get` returns the stored pointer or a typed not-found error. `List` returns a new, ID-sorted slice of pointers. The sorted list makes today's first-fit worker choice deterministic. A copied list is not a copied job or worker: callers still hold the actual stored objects. Neither store persists anything to disk.

## Scheduling: `internal/scheduler`

`Select(workers, request)` validates the request and returns the first non-nil worker whose available capacity fits. `Assign(workers, job)` requires a pending job, selects a worker, reserves its capacity, and transitions the job to `running`. If the transition fails after reservation, it tries to release that reservation and returns the error(s). `Finish(worker, job, finalStatus)` requires a running job and `succeeded` or `failed`; it releases capacity, transitions the job to the final status, and attempts to re-reserve if the transition fails. `errors.Join` preserves both primary and rollback errors when needed.

The policy is deliberately simple first-fit. Selection strategy can change later without rewriting the job and worker models.

## Orchestration: `internal/controlplane`

`Service` receives a `JobStore` and `WorkerStore` interface. It also owns an in-memory `assignments map[string]string` (`jobID → workerID`) and a lock for multi-step operations.

- `SubmitJob`: call `job.New`, then store `Create`.
- `RegisterWorker`: call `worker.New`, then store `Register`.
- `GetJob`: retrieve from the job store.
- `ScheduleJob`: lock the service, retrieve the job and sorted workers, call `scheduler.Assign`, then record the assignment. A no-fit or non-pending error leaves no new assignment.
- `AssignedWorker`: read-lock the assignment map, resolve the worker ID through the worker store, and return that worker.
- `FinishJob`: lock the service, retrieve job/assignment/worker, call `scheduler.Finish`, and delete the assignment **only after** successful finish.

The service lock protects coordination across the assignment map, job state, and worker capacity. The lower-level job/worker/store locks protect their own state. In-process locks do not give cross-process transactions or persistence; those are future design work.

## HTTP server: `internal/httpapi`

`Handler` implements `http.Handler` through `ServeHTTP`, delegating routing to `http.ServeMux`. Request and response structs carry JSON; they are deliberately separate from domain objects. For example, `submitJobRequest` decodes a JSON resource object, the handler converts it to `job.ResourceRequest`, and `mapJobToResponse` converts the created job back to JSON-friendly fields.

| Method and path | Handler → service call | Success |
| --- | --- | --- |
| `POST /v1/jobs` | `submitJob` → `SubmitJob` | `201` job JSON |
| `GET /v1/jobs/{id}` | `getJob` → `GetJob` | `200` job JSON |
| `POST /v1/workers` | `registerWorker` → `RegisterWorker` | `201` worker JSON |
| `POST /v1/jobs/{id}/schedule` | `scheduleJob` → `ScheduleJob` | `200` assignment JSON |
| `GET /v1/jobs/{id}/assignment` | `getAssignment` → `AssignedWorker` | `200` assignment JSON |
| `POST /v1/jobs/{id}/finish` | `finishJob` → `FinishJob` | `204` no body |

POST handlers decode JSON and reject unknown fields. The handlers map known domain errors to `400` (invalid input), `404` (missing job/assignment), or `409` (duplicate or incompatible lifecycle/capacity), and hide unexpected server errors behind a generic `500` message. There is no authentication or worker authorization. In particular, the finish endpoint does not verify that the caller is the assigned worker. This is safe only as a localhost prototype, not as a LAN command service.

## Worker-side HTTP client: `internal/client`

`Client` stores a base URL and an `http.Client` with a ten-second timeout. `GetAssignment` and `GetJob` create context-aware GET requests, URL-escape the job ID, expect `200`, and decode response JSON into small client-side DTOs (`Assignment` and `Job`). `FinishJob` sends `{"status":"succeeded"}` or `{"status":"failed"}` as JSON and expects `204`. Unexpected statuses become errors. These DTOs represent the wire format; they are not the server's in-memory `job.Job` objects.

## Local execution: `internal/executor`

`Run(ctx, command)` validates the command and calls `exec.CommandContext(ctx, command[0], command[1:]...)`. It captures stdout and stderr separately, waits for completion, and returns `Result{Stdout, Stderr, ExitCode}` plus an error if starting/running failed. It uses exit code `-1` when no process exit status is available. `CommandContext` allows cancellation of the OS process when the context is canceled. There is **no implicit shell**: `['echo', 'hello']` is an executable plus argument, while pipes, redirects, and shell expansion are not automatically interpreted.

The executor runs commands with the operating-system permissions of the worker process. There is no container, sandbox, resource enforcement, output-size bound, or artifact upload yet.

## Assigned-job runner: `internal/agent`

`Runner.RunJob(ctx, jobID)` performs these checks and actions in order:

1. Fetch the assignment. If it is missing, stop before execution.
2. Check both assignment job ID and worker ID against the requested job and this runner's worker ID.
3. Fetch the job. Check its ID and that status is `running`.
4. Call the injected execute function with the job command.
5. Choose `succeeded` only if execution returned no error **and** exit code 0; otherwise choose `failed`.
6. Call `FinishJob` to report that final status. The reporting context has a ten-second timeout and is detached from cancellation of the execution context, giving completion reporting a chance after Ctrl+C.
7. Return command output/result plus any execution and reporting errors, combined with `errors.Join` if both occurred.

Pre-execution failures return an exit code of `-1`. The runner's worker-ID check prevents an accidental wrong-worker execution by this client. It is **not** server-side authorization: a malicious HTTP caller could bypass the runner. There is also no atomic claim today, so launching two worker-run processes for the same assigned job can execute it twice; one of the completion reports may then fail. Exclusive claim/lease is planned in C5.

## Interfaces and dependency injection, concretely

An interface states **which methods a component needs**, not which concrete type it must use. Go checks interface satisfaction implicitly: a type with the required method signatures can be supplied without writing an `implements` declaration. The interface is not a cast, and it does not create an implementation.

For example, `controlplane.JobStore` requires `Create`, `Get`, and `List`. `*store.MemoryJobStore` has those methods, so `NewService(jobStore, workerStore)` can receive it. Later, `s.jobStore.Get(id)` invokes the real `MemoryJobStore.Get` method on the stored object. A database-backed store could be passed in the future only after we implement those methods and wire it at startup.

The same boundary pattern appears three times:

| Consumer | What it requires | Real value today | Why tests can substitute |
| --- | --- | --- | --- |
| `controlplane.Service` | `JobStore`, `WorkerStore` | In-memory stores | Fake stores can return controlled failures. |
| `httpapi.Handler` | `ControlPlaneService` | `*controlplane.Service` | A fake service can isolate JSON/status behavior. |
| `agent.Runner` | `ControlPlaneClient` | `*client.Client` | A fake client can test runner decisions without HTTP. |

`agent.ExecuteFunc` is related but is a **function type**, not an interface. Real code passes `executor.Run` without calling it; the runner stores it and invokes it later. A test can pass a small function returning a chosen `Result`, so no OS process needs to start. `io.Writer` used for CLI output is a standard-library interface: real values are `os.Stdout`/`os.Stderr`, while tests use buffers. Prefer these narrow boundaries over making an interface for every struct.

## Example current manual flow

With `./bin/orchard server` running in another terminal:

```sh
curl -i http://127.0.0.1:8080/v1/workers \
  -H 'Content-Type: application/json' \
  -d '{"id":"worker1","capacity":{"cpu_cores":2,"memory_mib":1024}}'

curl -i http://127.0.0.1:8080/v1/jobs \
  -H 'Content-Type: application/json' \
  -d '{"id":"job1","command":["echo","hello orchard"],"resources":{"cpu_cores":1,"memory_mib":512}}'

curl -i -X POST http://127.0.0.1:8080/v1/jobs/job1/schedule
./bin/orchard worker-run worker1 job1
curl -i http://127.0.0.1:8080/v1/jobs/job1
```

The schedule call is mandatory in this version. If the job is only pending, `GET /assignment` returns `404` and `worker-run` cannot execute it. Job IDs and worker IDs must be fresh if you repeat the flow against the same running server. If the server restarts, its in-memory records disappear, so register and submit again.

## Tests and boundaries of confidence

Each package has focused `_test.go` files. Model tests check validation and transitions; scheduler/service tests check resource and assignment invariants; HTTP tests check JSON and status mappings; client tests use an HTTP test server; executor tests check process results; runner tests inject fake HTTP/executor dependencies; CLI tests inject output writers and a server function. Run `go test ./...` and `go test -race ./...` locally. A sandbox that forbids local socket listeners may prevent `httptest.NewServer` client tests from running even when the code compiles.

The code does **not** yet provide automatic scheduling, worker polling, exclusive claims, retries, worker heartbeats, durable history, server-side stdout/stderr, LAN connectivity, authentication, or true resource isolation. These are explicit roadmap checkpoints, not hidden capabilities.
