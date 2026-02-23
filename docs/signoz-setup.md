# signoz setup (otel)

visor can export traces/log-context to signoz via otlp http.

## 1) set env

add to your `.env` (or service env):

```env
OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
OTEL_SERVICE_NAME=visor
OTEL_ENVIRONMENT=dev
OTEL_INSECURE=true
```

notes:
- for local signoz, `http://localhost:4318` is the usual otlp http endpoint.
- if signoz runs elsewhere, set endpoint accordingly.
- keep `OTEL_ENABLED=false` for local-only mode without export.

## 2) start visor and generate traffic

send a telegram message to visor so webhook + agent spans are created.

## 3) verify in signoz

in signoz, filter by:
- service: `visor`
- environment: `dev` (or your configured value)

expected span names:
- `webhook.handle`
- `agent.process`

## 4) quick sanity checks

- if no traces appear, check visor startup logs for `otel init failed`.
- verify endpoint/protocol is OTLP HTTP, not gRPC.
- confirm firewall/network allows access to the configured endpoint.
