# Orchard roadmap

## Goal and definition of done

Build a small, understandable job-execution system: one Mac hosts a control plane; worker processes on that Mac or other Macs on the same trusted Wi-Fi network register, receive work, run commands, and report results. A user can submit a job and inspect its status and output without manually choosing or launching the worker for each job. Jobs and assignments survive a server restart, and a lost worker does not leave a job permanently stuck. Network execution must be authenticated before the server is exposed beyond localhost.

This is a learning-scale system, **not** a full AWS or Kubernetes clone. Containers, multi-tenant isolation, autoscaling, billing, and production-grade arbitrary-code sandboxing are out of scope unless explicitly added later. Requested CPU and memory are scheduling quantities until real enforcement is deliberately implemented.

## How to use this roadmap

- Checkboxes are evidence-based: `[x]` means implemented **and verified**; `[ ]` means not yet verified or not built. Do not infer completion from a file's presence.
- Every checkpoint has a definition of done (DoD). Complete a checkpoint only when all its DoD items pass. An unfinished checkpoint may still have completed subparts.
- After each subpart or checkpoint, update both its checkbox(es) and **Current progress** below with the result, evidence (test/manual command), and next action. Record blockers and deferred verification honestly.
- Keep the **Next action** specific enough that a new developer or agent can pick up the work immediately. Preserve history in the progress log; do not silently rewrite past results.
- Tests should be focused on new behavior and boundaries; avoid duplicating exhaustive tests already covered by lower layers.

## Checkpoints

### C0 — Bootstrap the Go CLI

User story: as a developer, I can build and invoke Orchard from the command line.

DoD:

- [x] Initialize the Go module, Git repository, and `cmd/orchard` entry point.
- [x] `version`, missing-command, and unknown-command behavior have useful output and exit codes.
- [x] Build and focused CLI tests pass.

### C1 — Define jobs, workers, and capacity

User story: as the scheduler, I have validated models and legal state changes.

DoD:

- [x] Jobs validate ID, command, and positive CPU/memory request.
- [x] Jobs move only `pending → running → succeeded|failed`.
- [x] Workers validate capacity and safely check, reserve, and release it.
- [x] Focused model tests and race-relevant tests pass.

### C2 — In-memory scheduling and control plane

User story: as a submitter, I can create a job and have it assigned to an available worker without losing capacity accounting.

DoD:

- [x] Thread-safe in-memory job and worker stores reject duplicate IDs and support lookup/listing.
- [x] First-fit selection, assignment, completion, rollback on failed transitions, and capacity release are implemented.
- [x] The control-plane service tracks active `job ID → worker ID` assignments and serializes scheduling/completion.
- [x] Focused store, scheduler, and service tests pass.

### C3 — Local HTTP control plane

User story: as a local client, I can drive the job lifecycle through an HTTP API.

DoD:

- [x] `POST /v1/jobs`, `GET /v1/jobs/{id}`, and `POST /v1/workers` work.
- [x] `POST /v1/jobs/{id}/schedule`, `GET /v1/jobs/{id}/assignment`, and `POST /v1/jobs/{id}/finish` work.
- [x] Requests are validated, domain errors map to appropriate HTTP status codes, and handler tests pass.
- [x] A manual curl lifecycle demonstrated capacity exhaustion and reuse after finish.

### C4 — Run one assigned job on one Mac (active)

User story: as a local worker operator, I can run a scheduled job and see its terminal status and command output.

DoD:

- [x] Local executor runs a command with context and captures stdout, stderr, and exit code.
- [x] Worker HTTP client can fetch job/assignment and report completion; runner checks identity/status and reports success/failure.
- [x] `worker-run <worker-id> <job-id>` is wired into the CLI, with argument tests.
- [ ] On the developer Mac, run the full suite (`go test ./...` and `go test -race ./...`) and build the binary.
- [ ] Manually register → submit → **schedule** → `worker-run` → inspect a succeeded job and released capacity.
- [ ] Manually run a failing command and verify `failed`, nonzero CLI exit, and released capacity.
- [ ] Review and commit the CLI changes, then mark C4 complete.

### C5 — Autonomous worker and safe dispatch

User story: as an operator, I start a worker once; it finds and executes eligible jobs without a job ID or manual schedule call.

DoD:

- [ ] Define the dispatch protocol and ownership rules: who schedules, how a worker claims work, and how duplicate execution is prevented.
- [ ] Implement worker registration/startup, configurable polling or long-polling, and graceful shutdown on one Mac.
- [ ] Make claim/lease operations atomic; define lease expiry/reclaim and behavior after worker loss so jobs do not remain `running` forever.
- [ ] Show with two local worker processes that one job executes at most once and another worker can make progress.
- [ ] Add focused concurrency and failure tests for claim, lease, and recovery behavior.

### C6 — Durable state and observable results

User story: as a user, I do not lose job history on restart and can inspect what a command produced.

DoD:

- [ ] Choose and document a persistence format/database and migration approach; implement it behind storage boundaries.
- [ ] Persist jobs, workers/leases as appropriate, assignments, terminal status, exit code, and bounded stdout/stderr or artifact references.
- [ ] Define restart reconciliation for in-flight jobs and verify no permanent capacity leak or silently lost assignment.
- [ ] Expose result/status retrieval through API and CLI; verify succeeded and failed jobs before and after restart.
- [ ] Document output-size, timeout, and retention limits; test failure paths.

### C7 — Secure LAN configuration

User story: as a Mac owner, I can deliberately connect trusted Macs without leaving an open remote command runner.

DoD:

- [ ] Make the server listen address and worker server URL configurable; localhost remains the safe default.
- [ ] Authenticate workers and submitters and authorize assignment/finish operations; do not trust a caller-supplied worker ID alone.
- [ ] Protect traffic and credentials appropriately for the chosen LAN threat model (for example TLS or a documented secure tunnel); reject unauthenticated requests.
- [ ] Document setup, secret handling, firewall/network requirements, and a safe opt-in LAN mode.
- [ ] Verify authenticated traffic works and unauthorized or wrong-worker requests fail.

### C8 — End-to-end multi-Mac release

User story: as a user, I submit on one Mac and see a job run on another Mac on the same Wi-Fi.

DoD:

- [ ] Run the server on Mac A and a worker process on Mac B; verify registration, dispatch, execution, logs/results, and terminal status.
- [ ] Verify capacity behavior with concurrent jobs, worker disconnect/reconnect, server restart, and a failed command.
- [ ] Provide an operator guide for setup, running, troubleshooting, and clean shutdown; keep README accurate.
- [ ] Run the relevant automated suite and record a reproducible two-Mac smoke test. If a second Mac is unavailable, leave this checkpoint open and use two local processes as an interim test.

## Current progress

**Status:** C0–C3 complete. C4 is partially implemented; C5–C8 have not started. The most recent functional code commit in this snapshot is `9249c5c` (`feat: add assigned job runner`). `cmd/orchard/main.go` is modified and `cmd/orchard/worker.go` plus `cmd/orchard/worker_test.go` are untracked; these C4 CLI changes are not committed. Re-check `git status` before relying on this snapshot.

**Verified so far:** domain, store, scheduler, service, HTTP handler, executor, runner, and CLI-focused tests passed during development. The manual HTTP lifecycle of register/submit/schedule/finish and capacity reuse was demonstrated. In the most recent manual CLI attempt, `worker-run worker1 job1` returned `404 Not Found` because `job1` had been submitted but **not scheduled**; that is expected, not a successful execution. The full suite must still be run on the developer Mac for C4; sandboxed test environments may be unable to open `httptest` sockets.

**Next action:** with the server running, call `POST /v1/jobs/job1/schedule` (or create fresh IDs if the server restarted), then run `./bin/orchard worker-run worker1 job1` and inspect `GET /v1/jobs/job1`. Repeat with a failing command; run the full tests, review the current CLI diff, and commit if correct. Do not begin C5 or claim cross-Mac operation before C4 is closed.

**Known limitations:** hard-coded localhost address; manual scheduling and one-job worker invocation; in-memory state; no job output stored by the server; no worker authentication/authorization; no exclusive claim/lease; no actual CPU/memory limits; no network deployment. These are future checkpoints, not completed features.

### Progress log

| Date | Change / evidence | Result / next action |
| --- | --- | --- |
| 2026-11-21 | Reconciled roadmap against repository history and working tree. Existing focused tests and prior manual curl lifecycle are the evidence for C0–C3. | C4 remains open pending a scheduled-job CLI smoke test, a failure-path smoke test, local full-suite verification, and commit. |
