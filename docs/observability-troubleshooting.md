# observability troubleshooting

## no logs visible

- check `LOG_LEVEL` (set `info` or `debug`)
- ensure service stdout/stderr is collected by your runner/systemd/docker
- set `LOG_VERBOSE=true` for source locations

## missing request_id

- request ids are injected by `RequestIDMiddleware`
- ensure requests pass through http server middleware stack
- verify `X-Request-ID` is present in response headers

## no trace_id/span_id in logs

- trace fields appear when a span is active in context
- webhook and agent paths should show them when otel is enabled
- set `OTEL_ENABLED=true` and restart visor

## otel init failed on startup

common causes:
- missing `OTEL_EXPORTER_OTLP_ENDPOINT`
- invalid endpoint format
- collector not reachable

fix:
- set endpoint correctly (eg `http://localhost:4318`)
- ensure collector/signoz is running

## traces not arriving in signoz

- verify signoz endpoint and protocol (otlp http)
- test network reachability from visor host
- check env:
  - `OTEL_ENABLED=true`
  - `OTEL_EXPORTER_OTLP_ENDPOINT=...`
  - `OTEL_SERVICE_NAME=visor`
- generate fresh traffic after startup (send a message to webhook)

## too much log noise

- set `LOG_LEVEL=info` for normal mode
- keep `LOG_VERBOSE=false` unless debugging
- only enable `debug` temporarily
