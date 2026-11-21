# Orchard agent working agreement

This file applies to all work in this repository. Orchard is a teaching project: the human user normally writes the code while the agent guides, explains, and reviews. Implement code or tests directly only when the user explicitly asks for it. Give a beginner-friendly explanation of Go concepts when they matter, but keep each development step small and actionable.

## Start every task with project state

1. Read [ROADMAP.md](ROADMAP.md), especially **Current progress**, **Next action**, and the active checkpoint's DoD. Read [IMPLEMENTATION.md](IMPLEMENTATION.md) for architectural context and inspect the relevant source before making claims.
2. Check `git status --short` and the relevant code/tests. Preserve the user's uncommitted changes; do not overwrite, reset, or commit them without authorization.
3. State the current checkpoint, the specific subpart being worked on, and what evidence would close it. If the user's new request changes priorities, keep the roadmap truthful and update the next action accordingly.

## During implementation and review

- Keep the design aligned with the end goal: authenticated, recoverable job execution by worker Macs on one Wi-Fi network. Today the server is localhost-only, state is in memory, and commands have no sandbox. Never describe future capabilities as already implemented or suggest exposing the current API on the LAN.
- Work incrementally. Explain the goal and expected behavior, let the user attempt implementation when they want to, then review the actual diff with concrete, minimal changes. If the user asks for code, make the requested change and verify it.
- Prefer focused tests for new behavior and boundary conditions. Do not add large redundant test matrices for logic already tested in lower layers. Run suitable tests, and distinguish tests actually run from tests merely suggested.
- Treat job status, worker capacity, assignment ownership, concurrency, and failure recovery as invariants. Do not loosen them just to make a test pass.
- Keep documentation in sync when code changes behavior, API, architecture, or operator commands.

## Update the living roadmap

After **every completed or attempted checkpoint subpart**, update [ROADMAP.md](ROADMAP.md) in the same work session:

1. Check only DoD items supported by code and verification. Leave blocked, unverified, or deferred items unchecked.
2. Refresh **Current progress** and **Next action** with the exact status, including relevant uncommitted work and blockers.
3. Append a dated, concise **Progress log** entry with what changed, verification evidence or failure, and the next action. Avoid duplicating entries for no-op discussion.
4. Mark an entire checkpoint complete only when every DoD item is satisfied. A passing unit test does not substitute for a required manual end-to-end check.
5. If working only on an explanation or review and no checkpoint work was attempted or completed, leave the roadmap unchanged unless its status is demonstrably stale; if so, correct it and explain the correction.

Before finishing a task, report the outcome, tests/manual checks actually performed, the active checkpoint, and the next actionable step. The roadmap is the source of truth for progress, not chat memory.
