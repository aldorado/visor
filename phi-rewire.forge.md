# phi rewire plan

## milestone
integrate `phi` as primary runtime backend in visor with safe staged rollout.

## iteration 1 — sandbox adapter
- add `phi` backend to visor backend registry
- implement minimal prompt -> response path (text only)
- keep existing `pi` + `echo` behavior untouched

## iteration 2 — stream and event mapping
- map phi stream events to visor internal event model
- support text/progress/error passthrough
- add structured logs for stream lifecycle

## iteration 3 — tool bridge
- connect visor tools (`read`, `write`, `edit`, `bash`) to phi tool calls
- enforce tool timeout + fail-fast errors
- add tool call/result tracing in logs

## iteration 4 — session and memory compatibility
- align session ids and transcript persistence
- keep memory save/lookup behavior stable
- verify no regression on existing session format

## iteration 5 — reliability and fallback
- healthcheck for phi backend
- retry + failover behavior for transient failures
- add timeout/rate-limit tests

## iteration 6 — staged rollout
- gate via `AGENT_BACKEND=phi`
- rollout order: local -> staging -> canary -> broader usage
- define rollback switch back to `pi`

## iteration 7 — cleanup decision
- if stable: remove obsolete adapter paths
- if not stable: keep phi optional and document limits
- finalize docs + operational runbook

## done criteria
- all iterations completed with green `go test ./...`
- no regression in webhook flow, session logging, tool execution
- clear rollback path documented
