# Visor

## Vision
A fast, compiled agent runtime in Go that serves as the "body" for swappable AI CLI backends. Visor handles everything except thinking: webhooks, memory, voice, scheduling, sessions. The AI brain (pi, claude code, gemini, etc.) plugs in via stdin/stdout RPC. Main motivation: exploit free tokens from every AI company's CLI offering.

## Key decisions
- Language: Go (bottleneck is model latency, not server perf. Go's stdlib covers HTTP/JSON/exec/crypto natively. single binary, fast iteration)
- Storage: Parquet files for memories + embeddings (portable, remote-ready — can sync to cloud/S3 later)
- Voice: API-based (OpenAI Whisper for STT, ElevenLabs for TTS — no need for local models)
- Agent interface: stdin/stdout JSON-lines RPC (each backend gets a thin adapter)
- Primary backend strategy: use pi.dev as the main runtime/auth hub; other providers are accessed through pi.dev where possible.
- Embeddings: OpenAI API (proven pattern from jarvis + ubik, keeps Go binary lean)
- Config: single TOML or env file, no YAML
- Self-evolution: visor can edit its own Go source, git commit + push, rebuild itself, and hot-restart. this is a core feature, not an afterthought.
- Email: first-class in visor (receive + send), not an afterthought skill.
- Level-ups: optional infra sidecars via Docker Compose overlays + `.levelup.env` (visor itself stays host-native, not containerized).
- Himalaya: official reference implementation of the level-up pattern (email as exemplar) and the *first* implemented level-up in the project.
- Obsidian: additional standard level-up shipped by default (knowledge workspace sidecar) via LinuxServer container.
- Skill parity bootstrap: ship visor with the same baseline skills as current ubik by copying them into `visor/skills/` as the initial pack.
- Reverse proxy: level-ups get exposed via subdomain under wildcard DNS (`*.visor.<domain>`), auto SSL via Let's Encrypt, no direct host port exposure. Proxy runs as a base level-up.
- Forgejo: self-hosted git (Forgejo over Gitea, for ethical/community reasons) as a standard level-up. Visor pushes all self-authored code (self-evolve, forge-execution, skills) to its own Forgejo instance automatically.
- Zero-to-running onboarding: new user clones repo, starts pi/claude, gets guided through setup conversationally. No manual config editing. Optional level-ups (Forgejo, Himalaya, Obsidian) presented during setup.

## Research tasks
- [x] Investigate Claude Code RPC mode — does `claude --mode rpc` exist? what's the protocol? document stdin/stdout message format
- [x] Investigate pi CLI RPC protocol differences from what we learned in ubik — any undocumented events?
- [x] Investigate Gemini CLI — does Google offer a coding agent CLI? what RPC options exist? (lower priority — pi + claude first)
- [x] Investigate GitHub Copilot CLI — `gh copilot` capabilities, can it be used as a persistent agent? (lower priority)
- [x] Research Go libraries for Parquet read/write (parquet-go, apache arrow-go) — performance for append + scan
- [x] Research Go libraries for Telegram Bot API (telebot, gotgbot, or raw HTTP)
- [x] Check free tier limits for each CLI: tokens/day, rate limits, model access
- [x] Research Himalaya integration modes: CLI wrapping vs long-running service, IMAP IDLE support, SMTP send flow
- [x] Design compose overlay merge strategy (`docker-compose.yml` + `docker-compose.levelup.<name>.yml`) and env layering with `.levelup.env`

## Research findings log

### 2026-02-19 — Claude Code interface check (step 1)
- `claude --mode rpc` is *not* available in Claude Code `2.1.47` (checked via `claude --help`).
- Viable non-interactive integration path exists via:
  - `claude -p --output-format stream-json --verbose "<prompt>"`
- Observed stream-json output events include:
  - `system` (session/bootstrap metadata)
  - `rate_limit_event`
  - `assistant` (message payload)
  - `result` (final status/usage/cost)
- Input streaming mode exists (`--input-format stream-json`), but exact stdin event contract still needs deeper probing/docs before relying on bidirectional long-lived RPC behavior.
- Practical adapter decision for now: implement Claude adapter as *process-per-request* stream-json mode, not persistent `--mode rpc`.

### 2026-02-19 — pi RPC protocol check (step 2)
- `pi --mode rpc` is available in pi `0.53.0`.
- Prompt command contract confirmed:
  - stdin command: `{"type":"prompt","message":"..."}`
  - immediate ack event: `{"type":"response","command":"prompt","success":true|false,...}`
- Observed top-level events:
  - `agent_start`, `turn_start`, `message_start`, `message_update`, `message_end`, `turn_end`, `agent_end`
  - plus tool lifecycle events when tools are enabled: `tool_execution_start`, `tool_execution_update`, `tool_execution_end`
- Observed `assistantMessageEvent.type` values inside `message_update`:
  - text stream: `text_start`, `text_delta`, `text_end`
  - reasoning stream: `thinking_start`, `thinking_delta`, `thinking_end`
  - tool-call stream: `toolcall_start`, `toolcall_delta`, `toolcall_end`
- Error behavior confirmed:
  - invalid json / unknown command / malformed prompt emit `type:"response", success:false, error:"..."`
- Adapter implication:
  - current ubik extraction via `text_delta` + `agent_end` is still valid
  - but visor adapter should also track `response success:false` as hard failure and optionally expose tool/thinking streams for richer telemetry.

### 2026-02-19 — Gemini CLI capability check (step 3)
- Google offers a coding agent CLI via npm package `@google/gemini-cli` (checked latest `0.29.2`, preview/nightly also published).
- Local binary was not preinstalled, but `npx @google/gemini-cli --help` works and confirms headless mode support.
- Supported non-interactive output formats:
  - `text`
  - `json`
  - `stream-json`
- Official docs confirm stream-json event model for headless mode with event types:
  - `init`, `message`, `tool_use`, `tool_result`, `error`, `result`
- Auth requirement is explicit (without auth it exits with config/auth error), expecting one of:
  - `GEMINI_API_KEY`
  - `GOOGLE_GENAI_USE_VERTEXAI`
  - `GOOGLE_GENAI_USE_GCA`
- There is an `--experimental-acp` flag in CLI reference, but no stable public protocol contract found yet.
- Adapter implication:
  - Gemini is viable today as process-per-request headless adapter via `-p` + `--output-format stream-json`
  - treat ACP as experimental-only until protocol/docs stabilize.

### 2026-02-19 — architecture decision update
- product decision: prefer pi.dev as primary runtime/auth entrypoint.
- keep runtime-level switching between provider/subscription/model as a first-class feature.
- recommended policy:
  - default provider/model per task class (coding/chat/research)
  - automatic fallback on quota/rate/auth failures
  - manual override command for forced provider/model pinning per run

### 2026-02-19 — GitHub Copilot CLI check (step 4)
- legacy `github/gh-copilot` extension is deprecated (README points to new `github/copilot-cli`).
- current Copilot CLI is available as `@github/copilot` (public preview) and supports:
  - interactive mode
  - non-interactive prompt mode: `-p`
  - model selection via `--model`
  - an ACP mode via `--acp` (Agent Client Protocol server mode)
- local run via `npx @github/copilot --help` confirmed flags, including `--acp`, `-p`, `--stream`, `--allow-all-tools`.
- auth is mandatory and can come from:
  - Copilot login flow (`/login`)
  - env token (`COPILOT_GITHUB_TOKEN`, `GH_TOKEN`, `GITHUB_TOKEN`)
  - `gh auth login`
- practical adapter stance right now:
  - process-per-request integration via `-p` is viable once authenticated
  - `--acp` exists and is promising for persistence, but treat as secondary/optional until protocol/docs are validated against visor needs
  - pi.dev remains primary hub per architecture decision.

### 2026-02-19 — Go Parquet libs check (step 5)
- `parquet-go/parquet-go`:
  - explicitly positioned as high-performance Go Parquet library.
  - modern typed APIs (`GenericWriter[T]`, `GenericReader[T]`) fit visor's Go-first codebase well.
  - important note from docs: parquet files are immutable after write; true in-place update/append is not the natural pattern.
  - supports low-level row group/page access and schema conversion utilities.
  - currently pre-v1 (possible occasional breaking API changes).
- `apache/arrow-go` parquet stack (`parquet`, `file`, `pqarrow`):
  - broader ecosystem coverage (Arrow + Parquet + conversion bridge).
  - robust for interoperability and analytics pipelines.
  - heavier mental model (Arrow memory + retain/release lifecycle + extra layers) for simple persistence use cases.
- append + scan implications for visor memory:
  - safest append strategy is *chunked immutable files* (new parquet chunks/row-groups) plus periodic compaction/merge, not in-place mutation.
  - scan path should leverage row-group metadata and chunk-level filtering before full row decode.
- recommendation for visor v1:
  - primary writer/reader: `parquet-go/parquet-go`
  - architecture pattern: append-by-new-file (or append-by-new-row-group in new output artifact), then background compaction
  - keep Arrow integration optional for export/interchange, not core runtime dependency.

### 2026-02-19 — Telegram Go libs check (step 6)
- `telebot` (`tucnak/telebot`):
  - mature and widely used framework with concise routing/middleware API.
  - high ergonomics for quick bot feature delivery.
- `gotgbot` (`PaulSonOfLars/gotgbot/v2`):
  - code-generated API wrapper with strong parity to Telegram Bot API docs.
  - zero third-party dependency style and explicit updater/dispatcher model.
- `go-telegram/bot`:
  - zero-dependency framework, explicitly tracks latest Bot API versions (README currently references 9.4).
  - clean webhook + manual `ProcessUpdate` path.
- Raw HTTP (stdlib only):
  - max control, minimal dependency surface, easiest long-term runtime stability for a self-evolving host process.
  - but requires more internal boilerplate (routing/update parsing/retries/types).
- recommendation for visor v1:
  - keep *raw HTTP as primary* (as already intended in architecture) for minimal moving parts.
  - optionally add thin compatibility adapter layer later if framework migration becomes attractive.

### 2026-02-19 — free-tier/rate-limit check across CLIs (step 7)
- Decision context: pi.dev remains the primary runtime/auth hub; other CLIs are secondary adapters.
- Gemini CLI (well-documented quotas):
  - Google login (Code Assist Individuals): 1000 requests/day, 60 requests/min, Gemini family routing.
  - Unpaid Gemini API key: 250 requests/day, 10 requests/min, Flash-only.
  - Paid tiers increase per-user daily/minute limits; pay-as-you-go available via API key/Vertex.
- Claude Code:
  - explicit static free-tier numbers are not exposed directly in CLI help output.
  - runtime emits `rate_limit_event` with machine-readable fields (`rateLimitType`, `resetsAt`, overage flags), which is enough for dynamic routing decisions.
  - practical stance: treat Claude limits as runtime-discovered rather than hardcoded constants.
- GitHub Copilot CLI:
  - requires active Copilot subscription/auth.
  - docs/README indicate monthly premium-request accounting, but precise allowance depends on plan/subscription context.
- pi.dev hub strategy implications:
  - do not hardcode provider limits except where official and stable (Gemini values above).
  - keep a runtime quota registry fed by real events/errors (rate-limit, quota-exceeded, auth-failed).
  - route selection policy should prioritize: health -> quota headroom -> cost profile -> requested model.

### 2026-02-19 — Himalaya integration modes check (step 8)
- Himalaya is a robust email *CLI* (IMAP + SMTP + multiple backends) with JSON output support (`--output json`), which is suitable for wrapper integration.
- CLI command surface (from source) is command-oriented (`account`, `folder`, `envelope`, `message`, etc.) and does not expose a dedicated long-running daemon/service mode in current core commands.
- No explicit `idle`/`watch`/`notify` command was found in current CLI command tree, so push-style IMAP IDLE is not exposed as a first-class CLI workflow.
- SMTP send flow is cleanly represented in command layer (`message send`, `template send`) and internally calls `send_message_then_save_copy`, matching expected "send + sent copy" behavior.
- Integration recommendation for visor level-up:
  - mode A (v1): *CLI wrapping* as default (spawn short-lived himalaya commands for list/read/send)
  - mode B (later): optional custom sidecar bridge service built by us if we need push/stream semantics
  - inbound strategy in v1: polling envelopes/messages with state checkpoint, not IMAP IDLE.

### 2026-02-19 — compose overlay + env layering design (step 9)
- Compose merge order should be deterministic: base first, then level-up overlays in declared order (`-f base -f overlay1 -f overlay2 ...`).
- Merge semantics to rely on:
  - single-value fields replace previous values
  - selected list fields concatenate
  - map-like fields (eg `environment`, `labels`) merge with later file taking precedence on key conflicts
- Path safety rule: all relative paths must be resolved against the *base* compose file directory.
- Validation rule: always render merged model via `docker compose config` before up/down operations.
- Env layering design for visor level-ups:
  - source layers: `.env` (base) + `.levelup.env` (level-up specific) + process env overrides
  - required level-up keys declared in `levelup.toml`; fail fast if missing
  - secrets stay out of compose YAML; referenced through env and/or compose secrets.
- Scope boundary stays strict:
  - compose manages sidecars only
  - visor host process remains outside compose
  - himalaya email level-up is the first and canonical example integration.

### 2026-02-20 — Git remote strategy + SSH vs HTTP (M11 research #3 + #4)
- **No git hooks needed.** Visor controls the commit process itself — it just runs `git push forgejo main` as part of its commit flow. No post-commit hooks, no complexity.
- **When visor pushes:**
  - Self-evolve commit → push visor's own repo
  - Forge-execution commit → push the project repo
  - Skill creation → push visor's own repo
  - Push runs in background, non-blocking. If forgejo is unreachable → log warning, continue.
- **Remote setup on level-up enable:** visor runs `git remote add forgejo <url>` on its own repo (and any forge-execution project repos). On disable → `git remote remove forgejo`.
- **Decision: HTTP with token auth (not SSH).** Reasons:
  - No SSH key generation/registration needed
  - Visor owns the forgejo instance, no 2FA → no auth issues
  - Local docker network only (forgejo container reachable as `forgejo:3000`)
  - URL format: `http://visor:<token>@forgejo:3000/visor/<repo>.git`
  - Token generated at bootstrap, stored in `.levelup.env`
  - Push-to-create handles repo creation automatically on first push
- **SSH still available** if user wants to expose forgejo's SSH externally (via caddy-l4 TCP forwarding). Not needed for visor's internal push workflow.

### 2026-02-20 — Forgejo API for repo management (M11 research #2)
- **Core need is minimal**: push-to-create + `git push` handles 90% of visor's use case. No API calls needed for basic operation.
- **Push-to-create**: built-in feature, disabled by default. Set `ENABLE_PUSH_CREATE_USER=true` in `app.ini`. Then just `git push` to a non-existent repo URL → repo created automatically. No API call needed. This is the primary integration point.
- **First-run bootstrap**: `FORGEJO__security__INSTALL_LOCK=true` env var skips web installer. `forgejo admin user create --admin` via CLI creates admin user. Fully automatable in docker compose entrypoint.
- **REST API available but optional**: full CRUD for repos, orgs, webhooks at `/api/v1/`. Auth via `Authorization: token <TOKEN>`. Swagger docs at `/api/swagger`. Only needed if visor wants to do more than push (e.g., create webhooks for notification on external pushes). Can be added incrementally later.

### 2026-02-20 — Forgejo vs Gitea (M11 research #1)
- **Decision: Forgejo** (ethical/community reasons + technical merits). Non-profit governance (Codeberg e.V.), pure Free Software, more active development (217 contributors / 3039 commits since July 2024 vs Gitea's 136 / 1228).
- **Docker image**: `codeberg.org/forgejo/forgejo:14.0.2` (latest stable). Same lightweight Go binary as Gitea, comparable resource footprint (~256MB RAM minimal, similar to Gitea).
- **API compatibility**: Forgejo maintains Gitea API compatibility. Version scheme `X.Y.Z+gitea-A.B.C` indicates which Gitea API version is compatible. Existing Gitea tooling (Jenkins plugin, API clients) works with Forgejo.
- **Breaking point**: Forgejo v10+ no longer supports migration from Gitea >= v1.23. The codebases are now a hard fork. For visor this doesn't matter — fresh install, no migration needed.
- **Forgejo Actions (built-in CI)**: GitHub Actions compatible workflows. Requires a separate runner container (`data.forgejo.org/forgejo/runner`). Two modes: Docker socket mount (simple, less secure) or Docker-in-Docker (recommended). Runner registration via shared secret (40-char hex). Could be useful for visor's `scripts/check.sh` as a post-push CI step later, but not required for M11 scope.
- **Compose recipe**: Forgejo + runner can run as a single docker-compose stack. Runner needs Docker access (socket or DinD) to execute workflow containers.

### 2026-02-20 — Reverse proxy comparison: Caddy vs Traefik vs nginx (M10 research #1)
- **nginx**: eliminated. No auto HTTPS, requires certbot sidecar, fully manual static config. Overkill for this use case.
- **Traefik**: native Docker label-based service discovery (no plugin), wildcard certs via DNS challenge (`certificatesResolvers`), 50–200MB RAM. Config is complex (YAML/TOML static config + Docker labels + routers/services/middlewares concept). Best for large-scale Docker/K8s orchestration.
- **Caddy**: simplest config (Caddyfile, ~3 lines per route), built-in auto HTTPS zero-config, 30–100MB RAM. Wildcard certs require custom Docker image with DNS provider plugin (`caddy-dns/cloudflare` etc.) + `tls { dns cloudflare {env.CLOUDFLARE_API_TOKEN} }` in Caddyfile. Docker service discovery via `caddy-docker-proxy` plugin (label-based, auto-generates Caddyfile, zero-downtime reload). Admin API for runtime config updates without restart.
- **Layer 4 (TCP/UDP)**: Caddy supports non-HTTP protocol forwarding via `caddy-l4` plugin (by mholt, Caddy's author). Enables raw TCP proxy for PostgreSQL, SSH, custom protocols. Supports protocol multiplexing on same port (SSH + HTTPS on 443, like GitHub). Plugin is experimental but actively maintained. Must be included in custom Docker image build.
- **Decision**: Caddy. Caddyfile is trivial to template/generate from Go code (visor needs to write routes programmatically). Lightest footprint. caddy-docker-proxy gives Traefik-style label discovery. caddy-l4 covers TCP/UDP forwarding for Gitea SSH, PostgreSQL, etc.
- Custom Docker image recipe (minimal): `xcaddy build --with github.com/mholt/caddy-l4` (DNS plugin only needed if choosing wildcard cert approach over on-demand TLS)

### 2026-02-20 — Cloudflared tunnel coexistence with Caddy (M10 research #4)
- **Two deployment modes for visor, Caddy is useful in both:**
  - *Direct mode* (ports 80/443 open): `client → Caddy (HTTPS, on-demand TLS) → service`. Caddy handles SSL via LE, full control, no external dependency.
  - *Tunnel mode* (no ports open): `client → Cloudflare edge (SSL) → cloudflared → Caddy (HTTP) → service`. Cloudflare terminates TLS at the edge. Caddy runs as internal HTTP-only reverse proxy, routing by Host header. No LE certs needed.
- **Caddy config difference between modes**: same routing logic (host-based reverse proxy), but in tunnel mode Caddy listens on `:80` (HTTP), in direct mode on `:443` with `tls { on_demand }`. Visor generates the Caddyfile accordingly based on `PROXY_MODE=direct|tunnel`.
- **On-demand TLS does NOT work behind cloudflared**: HTTP-01 challenge requires LE to reach port 80 directly — the tunnel intercepts this. If behind tunnel, either skip LE entirely (CF handles SSL at edge) or fall back to DNS-01 for internal HTTPS.
- **Sub-subdomain limitation on free Cloudflare plan**: `*.visor.example.com` is a sub-subdomain wildcard — free CF Universal SSL only covers `*.example.com`, not `*.sub.example.com`. Options:
  - (a) Purchase Advanced Certificate Manager (~$10/month) for multi-level wildcard SSL
  - (b) Use flat subdomains: `visor-obsidian.example.com` instead of `obsidian.visor.example.com`
  - (c) Dedicate a whole domain: `*.visor.dev` (first-level wildcard, works on free plan)
  - Direct mode has no such limitation — Caddy on-demand TLS works with any subdomain depth.
- **Cloudflared tunnel routing**: set wildcard CNAME (`*` or `*.visor`) → `TUNNEL_ID.cfargotunnel.com`. Add matching public hostname in CF dashboard pointing to `caddy:80`. Host header is preserved through tunnel, so Caddy routes correctly.
- **Is cloudflared still needed?** Depends on user's setup:
  - User has ports open + static IP/domain → direct mode, no cloudflared needed for level-ups
  - User is behind NAT / no open ports → tunnel mode, cloudflared required
  - Hybrid: cloudflared for main webhook (Telegram), direct for level-ups → possible but awkward
- **Decision**: visor supports both modes via `PROXY_MODE` env var. Default to `direct` (simpler, no CF dependency). Cloudflared level-up from M0-I5 remains available as opt-in for tunnel mode.

### 2026-02-20 — On-demand TLS vs wildcard cert (M10 research #5)
- **Decision: on-demand TLS**. Eliminates DNS provider API dependency entirely. User only needs one wildcard DNS A record (`*.visor.example.com → server IP`), set once manually. No further DNS interaction needed.
- **How on-demand TLS works**: when a new subdomain is first requested (e.g. `obsidian.visor.example.com`), Caddy issues an individual cert via HTTP-01 challenge on the spot. First request has ~2-3s delay, all subsequent requests are instant. Renewal is automatic in the background.
- **Security: `ask` endpoint required**. Caddy sends `GET http://localhost:<port>/check?domain=obsidian.visor.example.com` before issuing any cert. Visor provides this endpoint — trivial: check if the requested subdomain matches an enabled level-up. Returns 200 = issue cert, anything else = reject. Prevents abuse (random domains pointing to server won't get certs).
- **Caddyfile config** (minimal):
  ```
  {
      on_demand_tls {
          ask http://localhost:9123/check
      }
  }
  https:// {
      tls {
          on_demand
      }
      reverse_proxy {upstream}
  }
  ```
- **Rate limit**: LE allows 50 certs/domain/week. Visor won't have 50 level-ups, so not a concern.
- **No custom Docker image needed for TLS** — standard `caddy:2-alpine` works. Custom image only needed if using `caddy-l4` for TCP/UDP forwarding (SSH, PostgreSQL).
- **Visor env contract simplified**: only `PROXY_DOMAIN` (e.g. `visor.example.com`) needed. No `DNS_PROVIDER`, no `DNS_API_TOKEN`.
- **Wildcard cert still available as fallback**: if user has many subdomains or wants to avoid per-subdomain issuance, they can opt into DNS-01 by providing DNS provider credentials. But on-demand TLS is the default.

### 2026-02-20 — Docker network isolation patterns (M10 research #3)
- **Network topology for visor**: one shared `proxy` network + per-level-up isolated networks. Caddy container joins `proxy` + all level-up networks it needs to reach. Level-up containers join their own internal network + `proxy` network (only for the service that caddy needs to reach).
- **No host port exposure**: level-up compose files must NOT have `ports:` mappings. Services are only reachable through Caddy via the shared docker network. Container-to-container communication uses internal docker DNS (service name resolution).
- **`internal: true` option**: marking a level-up network as `internal: true` prevents outbound internet access from those containers. Useful for databases, but NOT for services that need internet (e.g., himalaya for SMTP/IMAP, gitea for git clone). Decision: default to normal (not internal), let level-up manifest opt-in to `internal: true` if desired.
- **Caddy discovery — two approaches**:
  - (a) `caddy-docker-proxy` with labels: automatic discovery via `CADDY_INGRESS_NETWORKS` env var. Caddy watches docker events, auto-generates Caddyfile from container labels. Zero-config per service.
  - (b) visor generates Caddyfile directly: visor writes/updates Caddyfile when level-ups are enabled/disabled, then triggers `caddy reload`. More explicit control, no label dependency.
  - **Decision: option (b) — visor generates Caddyfile**. Visor already manages the level-up lifecycle and knows which services are enabled/disabled. Generating the Caddyfile directly is simpler than maintaining docker labels across compose overlays. Also works cleanly with on-demand TLS (visor provides the `ask` endpoint AND the routing config).
- **External network pattern**: the `proxy` network is created as an `external: true` network in each level-up compose file. Caddy's compose creates it, level-ups reference it. This allows cross-compose-file communication without being in the same compose project.
- **Practical compose structure per level-up**:
  ```yaml
  services:
    obsidian:
      image: lscr.io/linuxserver/obsidian
      # NO ports: section
      networks:
        - proxy
        - obsidian_internal
  networks:
    proxy:
      external: true
    obsidian_internal:
      driver: bridge
  ```

### 2026-02-20 — Wildcard DNS + cert flow (M10 research #2)
- **Two approaches available**: (a) wildcard cert via DNS-01 challenge (requires DNS provider API), or (b) on-demand TLS via HTTP-01 (no DNS API needed). See research #5 for the decision.
- **Wildcard DNS setup**: user sets one `*.visor.example.com` A record pointing to the server. This is a one-time manual DNS change. After that, ALL subdomains (`obsidian.visor.example.com`, `git.visor.example.com`, etc.) automatically resolve to the server. No further DNS entries needed per level-up.
- **Wildcard cert (DNS-01)**: requires DNS provider API access because LE creates a TXT record at `_acme-challenge.<domain>` on every renewal. 30+ providers supported via `caddy-dns` GitHub org (Cloudflare, Route53, Hetzner, IONOS, etc.). Requires custom Caddy Docker image with DNS plugin.
- **On-demand TLS (HTTP-01)**: no DNS provider API needed at all. Caddy issues individual certs per subdomain when first requested. Requires only that the wildcard DNS points to the server (which it does). See research #5.
- **Cert storage in Docker**: Caddy stores certs in `/data`, config in `/config`. Both MUST be mounted as persistent volumes — losing them means re-issuance, and LE has rate limits (50 certs/domain/week). Renewal is fully automatic (~30 days before expiry), zero-downtime.

### 2026-02-19 — Obsidian standard level-up added
- Added an additional default level-up overlay for Obsidian based on LinuxServer `docker-obsidian`.
- Compose file target: `docker-compose.levelup.obsidian.yml`.
- Env contract added for `.levelup.env`: PUID/PGID/TZ, optional basic auth (CUSTOM_USER/PASSWORD), title/subfolder, host paths, and host ports.
- Positioning: Obsidian is a standard level-up shipped alongside Himalaya (not replacing Himalaya as first canonical example).

### 2026-02-19 — Skill parity + filesystem mount update
- Copied the full current ubik skill pack from `/root/code/ubik/.pi/skills/` to `/root/code/visor/skills/` as visor bootstrap parity.
- Obsidian level-up explicitly uses host bind mounts for `/config` and `/vault` so host-native visor can directly read/write vault files.

## Milestones

### M0: host-native runtime boundary + level-up foundation + native email baseline
Lock the boundary first: visor is host-native, compose is sidecars-only. Then use Himalaya email as canonical first level-up.

#### Iteration 1: level-up framework
- [x] Define level-up manifest format (`levelup.toml`) with name, compose overlay file, required env keys, healthcheck
- [x] Add loader for `.levelup.env` with strict validation (fail fast if required vars missing)
- [x] Implement compose assembly strategy (base + selected overlays)
- [x] Add CLI/admin command to list/enable/disable level-ups
- [x] Compose merge rules in runtime builder:
  - base file first, then overlays in declared order
  - never include visor service in compose (sidecars only)
  - enforce all relative paths resolved from base compose file directory
  - verify final model with `docker compose config` before apply

#### Iteration 2: Himalaya email level-up (reference)
- [x] Add `docker-compose.levelup.email-himalaya.yml` (*first concrete level-up shipped*)
- [x] Add `docker-compose.levelup.obsidian.yml` (standard knowledge workspace level-up)
- [x] Define required secrets in `.levelup.env.example` (IMAP/SMTP host, user, app-password, tls flags + Obsidian auth/paths/ports)
- [x] Implement inbound mail polling/IDLE bridge into visor events
- [x] Implement outbound mail send action from agent structured output
- [x] Add roundtrip tests: receive email → agent sees it → send reply
- [x] Add smoke test: Obsidian sidecar is reachable and persists vault/config mounts
- [x] Ensure Obsidian bind mounts resolve to host filesystem paths accessible by visor runtime
- [x] Add `EMAIL_ALLOWED_SENDERS` env var (comma-separated email addresses). Poller filters inbound mail before it reaches the agent: only messages from listed addresses pass through, all others are silently dropped (logged only). Empty/unset = no filter (all mail passes through).

#### Iteration 3: generalization docs
- [x] Write "how to build a level-up" guide using Himalaya as template
- [x] Add second toy level-up (minimal stub) to prove pattern is generic
- [x] Add failure-mode docs (missing env, container down, auth error)

#### Iteration 4: self-authored level-up skill
- [x] Add visor skill `levelup-creator` that lets visor create its own level-ups end-to-end
- [x] Require this skill to use M0 docs (`levelup-authoring`, `levelup-failure-modes`, `levelup-manifest`)
- [x] Enforce validation + docs sync + iteration-scoped commit in the skill workflow
- [x] Extend skill for operations: update `.levelup.env`, enable/disable level-ups, trigger validation via structured actions

#### Iteration 5: base cloudflared level-up
- [x] Add cloudflared base level-up manifest for first-start connectivity setup
- [x] Add `docker-compose.levelup.cloudflared.yml` with token/env-based tunnel startup
- [x] Add cloudflared env keys to `.levelup.env.example`

### M0b: observability + human-readable logging baseline
Guarantee full processing visibility for humans: readable request lifecycle logs, meaningful call-site context, clear tracebacks, and optional OTEL export to SigNoz.

#### Research notes (2026-02-19)
- Go `log/slog` is the best default baseline for visor (structured logs, stable stdlib, easy level control).
- `slog` can run with `AddSource: true` for file:line. function names should be added via a thin helper using `runtime.CallersFrames`.
- Verbose/normal mode should be pure config: `LOG_LEVEL=debug|info` and `LOG_VERBOSE=true|false`.
- Correlated request visibility needs a request-id in every log line plus span id/trace id fields where available.
- OTEL Go stack to target:
  - `go.opentelemetry.io/otel`
  - `go.opentelemetry.io/otel/sdk`
  - `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp`
  - `go.opentelemetry.io/contrib/bridges/otelslog`
- SigNoz integration path: OTLP exporter endpoint configurable by env (`OTEL_EXPORTER_OTLP_ENDPOINT`, service name/env attrs), with OTEL disable switch for local-only runs.

#### Iteration 1: logging contract + modes
- [x] Define global logging schema (timestamp, level, component, function, request_id, trace_id, message, attrs)
- [x] Implement logger package with normal mode and verbose mode switch
- [x] Add helper wrappers for consistent function/class/component naming
- [x] Add panic/recover middleware that logs compact traceback with request context

#### Iteration 2: request lifecycle visibility
- [x] Add request-id middleware for all webhook/admin paths
- [x] Log request lifecycle events: received, parsed, deduped, authorized, queued, processed, replied
- [x] Add agent lifecycle logs: queue length, start/end, backend used, duration, errors
- [x] Add level-up lifecycle logs: validate, enable/disable, compose config check, apply outcome

#### Iteration 3: OTEL + SigNoz export
- [x] Initialize OTEL provider with env config (`OTEL_EXPORTER_OTLP_ENDPOINT`, service name, environment)
- [x] Add spans around webhook handling, agent processing, and level-up operations
- [x] Bridge slog -> OTEL events/attributes for key log lines
- [x] Add config toggle to disable OTEL cleanly without changing code paths

#### Iteration 4: docs + operability
- [x] Add README section: where to read logs, verbose mode usage, sample output
- [x] Add SigNoz setup doc with minimal env example and verification steps
- [x] Add troubleshooting checklist for missing logs/traces/export failures

#### Iteration 5: full codebase logging sweep
- [x] Replace remaining `log.Printf` hotspots in runtime packages with structured observability logger
- [x] Add component/function aware logging in voice, memory manager, pi/claude adapters, process manager, and email poller
- [x] Ensure all core runtime paths emit consistent human-readable structured logs

### M1: skeleton — webhook + echo
Get a Go binary that receives Telegram webhooks and echoes messages back.

#### Iteration 1: project setup
- [x] Init Go module in `/root/code/visor`
- [x] Basic project structure: `main.go`, `internal/platform/`, `internal/agent/`
- [x] Load config from env vars (TELEGRAM_BOT_TOKEN, USER_PHONE_NUMBER, PORT)
- [x] HTTP server with /webhook and /health routes

#### Iteration 2: telegram integration
- [x] Parse Telegram webhook payloads (text, voice, image, reactions)
- [x] Send text responses via Telegram Bot API
- [x] Webhook signature verification
- [x] Auth check: drop messages not from USER_PHONE_NUMBER
- [x] Message dedup (in-memory set with TTL)

#### Iteration 3: echo bot
- [x] Wire webhook → parse → echo response → send
- [x] Deploy and test end-to-end on telegram (validated with webhook→agent→sendMessage e2e test harness)

### M2: agent process manager
Persistent CLI agent process with RPC over stdin/stdout.

#### Iteration 1: agent interface
- [x] Define Go interface: `Agent { SendPrompt(ctx, prompt) -> (response, error) }`
- [x] Implement process manager: spawn, restart on crash, periodic restart (configurable)
- [x] Message queue: if agent is busy, queue incoming messages

#### Iteration 2: pi adapter
- [x] Implement pi CLI adapter (`pi --mode rpc`)
- [x] Handle JSON-lines protocol: send `{ type: "prompt", message: "..." }`
- [x] Collect `text_delta` events, resolve on `agent_end`
- [x] Handle errors, timeouts (configurable per-prompt timeout)

#### Iteration 3: claude code adapter
- [x] Research Claude Code RPC protocol (depends on research task)
- [x] Implement adapter following same interface
- [x] Test switching between pi and claude via config

### M3: memory system
Persistent memory with semantic search. All data in Parquet files — portable, remote-ready, can sync to S3/cloud later.

#### Iteration 1: parquet storage
- [x] Parquet read/write in Go (parquet-go library or apache arrow)
- [x] Memories schema: id, text, embedding (float32 array), created_at
- [x] Sessions schema: id, user_id, role, content, timestamp
- [x] Store files in `data/memories.parquet` and `data/sessions/` (JSONL per session)
- [x] Basic CRUD: append to parquet, read all, filter by date

#### Iteration 2: embeddings + search
- [x] Call OpenAI embeddings API to generate vectors
- [x] Store embeddings as float32 arrays in parquet columns
- [x] Implement cosine similarity search in Go (load parquet → scan embeddings → rank)
- [x] Memory save: agent response includes memories_to_save → auto-embed and store
- [x] Memory lookup: before each prompt, search relevant memories and inject as context

#### Iteration 3: remote sync (future)
- [ ] Design sync protocol: local parquet ↔ remote storage (S3, R2, or custom)
- [ ] Incremental sync: only push new rows, not full file rewrites

### M4: voice pipeline
Transcription + text-to-speech.

#### Iteration 1: speech-to-text
- [x] Download voice message from Telegram
- [x] Send to OpenAI Whisper API for transcription
- [x] Feed transcribed text to agent as normal message with [Voice message] tag

#### Iteration 2: text-to-speech
- [x] ElevenLabs TTS API integration
- [x] Agent can request voice response (send_voice flag)
- [x] Send audio file back via Telegram

### M5: scheduling + cron
In-process scheduler for reminders and recurring tasks.

#### Iteration 1: scheduler
- [x] In-process cron scheduler (no system crontab dependency)
- [x] Persist scheduled tasks to disk as JSON (survives restarts)
- [x] Support one-shot and recurring schedules
- [x] On trigger: send prompt to agent with context about what was scheduled

#### Iteration 2: agent integration
- [x] Agent can create/modify/delete scheduled tasks via structured output
- [x] List upcoming tasks on request

#### Iteration 2.5: reminder ergonomics (quick actions)
- [x] Add quick actions for triggered reminders: `done`, `snooze <duration>`, `reschedule <time>`
- [x] Add parser for natural short forms (`in 10m`, `tomorrow 09:00`) in reminder follow-ups
- [x] Keep recurrence semantics intact when snoozing recurring tasks (no accidental series drift)
- [x] Add idempotency guard for duplicate quick-action replies
- [x] Add tests for snooze/done/reschedule flows and edge cases (past time, invalid duration, timezone boundaries)

### M6: skills system
Agent can create, edit, import, and execute skills autonomously, and request level-up enablement when infra dependencies are needed.

#### Iteration 1: skill runtime
- [x] Bootstrap parity pack copied from ubik into `visor/skills/`
- [x] Define skill format: executable scripts in `skills/` directory (shell, python, etc.)
- [x] Skill manifest: each skill has a `skill.toml` with name, description, trigger patterns, dependencies
- [x] Skill executor: visor runs skills in a sandboxed subprocess, captures stdout/stderr
- [x] Pass context to skills via env vars or stdin (user message, chat history summary, etc.)

#### Iteration 2: agent-authored skills
- [x] Agent can create new skills via structured output (writes script + manifest)
- [x] Agent can edit existing skills (modify script or manifest)
- [x] Agent can delete skills
- [x] Skill discovery: agent gets list of available skills in its system prompt context
- [x] Auto-trigger: visor matches incoming messages against skill trigger patterns, suggests or auto-runs
- [x] Skill dependency handshake: skill can declare required level-up(s); visor prompts for enablement if missing

#### Iteration 3: skill import
- [x] Import skills from git repos or URLs
- [x] Dependency resolution: skills can declare required tools/packages
- [x] Version tracking: git hash or semver per skill

### M7: multi-backend + auto-switch
Rotate between AI backends based on availability.

#### Iteration 1: backend registry
- [x] Config: list of backends with priority order
- [x] Health check per backend (is the CLI installed? does auth work?)
- [x] Select highest-priority healthy backend

#### Iteration 2: auto-failover
- [x] Detect rate limit / quota errors from backend
- [x] Auto-switch to next available backend
- [x] Log which backend is active
- [x] Notify user on backend switch (optional)

### M8: self-evolution
Visor's agent can modify visor's own source code, commit, push, rebuild, and restart — fully autonomous.

#### Iteration 1: self-edit pipeline
- [x] Agent has read/write access to visor's source directory (its own codebase)
- [x] Structured output includes `code_changes: true` flag when source was modified
- [x] On `code_changes: true`: visor sends the response to the user FIRST, then triggers the build/restart pipeline
- [x] Pipeline: `git add -A && git commit -m "..." && git push` (agent provides commit message)

#### Iteration 2: self-rebuild + restart
- [x] After commit: run `go build -o visor-new .` to compile the new binary
- [x] If build fails: notify user with error, rollback the commit (`git reset HEAD~1`), keep running old binary
- [x] If build succeeds: replace the running binary with the new one
- [x] Graceful restart: finish in-flight requests, shut down agent process, exec() into new binary (or use a supervisor)
- [x] Supervisor approach: visor spawns itself as a child, parent watches and restarts on exit. On self-update, child exits with special code, parent replaces binary and respawns.

#### Iteration 3: safety rails
- [x] Pre-build validation: run `go vet` and basic checks before committing
- [x] Keep last N working binaries as rollback (e.g. `visor.bak.1`, `visor.bak.2`)
- [x] If new binary crashes within 30s of startup: auto-rollback to previous binary
- [x] Log all self-modifications to a dedicated changelog (who changed what, when, which agent backend)
- [x] User can disable self-evolution via config flag

### M8a: release hardening + repo polish
Make the repository publication-ready: clean structure, clear docs, no accidental secrets, and a professional first impression.

#### Iteration 1: repo hygiene baseline
- [x] Add and validate `.gitignore` for Go + env + build/runtime artifacts
- [x] Remove accidental generated files and dead artifacts from tracked repo contents (none found; audit documented)
- [x] Normalize root layout (`cmd/`, `internal/`, `docs/`, `skills/`, `levelups/`) and remove clutter (validated, non-breaking)
- [x] Ensure naming consistency across files/docs (`visor`, `levelup`, command names)
- [x] Add/update `LICENSE` baseline
- [x] Add/update `CONTRIBUTING.md` baseline

#### Iteration 2: documentation pass
- [x] Tighten `README.md` for external users: what it is, quickstart, architecture, status
- [x] Add "first 10 minutes" setup section with copy-paste commands
- [x] Add config reference table (required env vars vs optional)
- [x] Add operational docs for local run, logs, troubleshooting, and updates
- [x] Add release notes/changelog policy (`CHANGELOG.md`)

#### Iteration 3: quality + release gate
- [x] Add/verify formatting and lint checks (`gofmt`, `go vet`) as local pre-push gate
- [x] Add/verify test command (`go test -race ./...`) as local pre-push gate
- [x] Add a pre-release checklist (security scan, secret check, docs check, smoke test)
- [x] Define semantic versioning + tagging flow (`v0.x`, `v1.0.0` criteria)
- [x] Create `M8a release candidate` milestone: all checks green + clean tree + tagged release

## Stack / tech
- Language: Go 1.22+
- HTTP: net/http (stdlib) or chi router
- Memory storage: Parquet files via parquet-go (remote-ready, portable)
- Telegram: raw HTTP calls (keep it simple, no heavy SDK)
- Voice: OpenAI Whisper API + ElevenLabs API (HTTP calls)
- Embeddings: OpenAI API
- Config: env vars (godotenv for .env loading)
- Skills: executable scripts in `skills/` dir, managed by agent

### M10: reverse proxy level-up — automatic service exposure
Visor can expose its docker-compose level-ups to the internet automatically. Each level-up gets its own subdomain under a wildcard DNS entry, with auto-provisioned SSL via Let's Encrypt. Level-ups run in isolated docker networks and never expose ports to the host directly.

#### Research tasks
- [x] Compare reverse proxy options for this use case: Caddy vs Traefik vs nginx. Key criteria: auto Let's Encrypt with wildcard certs, docker-aware service discovery, config-as-code simplicity, resource footprint
- [x] Investigate wildcard DNS + wildcard cert flow: DNS-01 challenge requirements per provider (Cloudflare API, Route53, etc.), cert storage, renewal automation
- [x] Investigate docker network isolation patterns: per-level-up networks, proxy-only shared network, no host port exposure. How does the proxy discover backend containers?
- [x] Check if cloudflared tunnel can coexist with or replace the reverse proxy approach (tunnel per service vs. single ingress point)
- [x] Evaluate Caddy's on-demand TLS vs. wildcard cert via DNS-01 for subdomain auto-provisioning

#### Iteration 1: proxy level-up + network isolation
- [x] Choose proxy (based on research) and add as base level-up (`docker-compose.levelup.proxy.yml`)
- [x] Define network topology: one shared `proxy` network + per-level-up isolated networks. Proxy container joins both.
- [x] Remove direct port mappings from existing level-up compose files (obsidian, himalaya, etc.)
- [x] Add proxy config generation: visor writes proxy routes when level-ups are enabled/disabled
- [x] Add proxy config to `.levelup.env` (`PROXY_DOMAIN`; optional `DNS_PROVIDER` + `DNS_API_TOKEN` for wildcard cert fallback)

#### Iteration 2: dynamic subdomain routing
- [ ] Visor auto-registers `<levelup-name>.visor.<domain>` → `<levelup-container>:<port>` on level-up enable
- [ ] Visor auto-deregisters route on level-up disable
- [ ] Add subdomain field to `levelup.toml` manifest (default: level-up name, overridable)
- [ ] Health endpoint per subdomain (proxy returns 502/503 if backend is down)
- [ ] Add tests for enable/disable/re-enable routing lifecycle

#### Iteration 3: auth + access control (optional)
- [ ] Add optional basic auth or SSO gate per level-up subdomain
- [ ] Add allowlist/denylist per subdomain (IP or user-based)
- [ ] Add admin dashboard subdomain for proxy status/metrics

### M11: Forgejo level-up — self-hosted git for visor-authored code
Visor pushes code it writes (via forge-execution, self-evolution, skill creation) to its own Forgejo instance. Forgejo runs as a standard level-up, optionally enabled at first setup.

#### Research tasks
- [x] Investigate Forgejo vs Gitea: docker image, resource footprint, API compatibility, built-in CI (Forgejo Actions / Runner)
- [x] Investigate Forgejo API for repo management: create repo, push, webhooks, org/user setup — all automatable via REST?
- [x] Design git remote strategy: visor's local repo gets a `forgejo` remote added automatically. Push after self-evolve, forge-execution, and skill creation commits.
- [x] Investigate SSH vs HTTP push: which is simpler for a local-network setup? Token-based HTTP push vs SSH key management.

#### Iteration 1: Forgejo level-up
- [ ] Add `docker-compose.levelup.forgejo.yml` with persistent storage (repos + SQLite)
- [ ] Add Forgejo env keys to `.levelup.env.example` (admin user, token, domain, HTTP port)
- [ ] Add first-run bootstrap: `INSTALL_LOCK=true` + `forgejo admin user create` + push-to-create enabled
- [ ] Integrate with M10 proxy: `git.visor.<domain>` subdomain auto-routed to Forgejo

#### Iteration 2: auto-push integration
- [ ] On level-up enable: visor adds `forgejo` remote to its own repo (and any forge-execution project repos)
- [ ] On self-evolve commit: auto-push to Forgejo remote after local commit
- [ ] On forge-execution commit: auto-push project repo to Forgejo (push-to-create handles repo creation)
- [ ] Add structured output field `git_push: true/false` so agent can control push behavior
- [ ] Add fallback: if Forgejo is unreachable, log warning and continue (don't block commits)

#### Iteration 3: visibility + collaboration
- [ ] Forgejo webhook → visor notification on external push/PR (if someone else pushes)
- [ ] Add repo listing skill: agent can list repos, recent commits, open issues on its Forgejo
- [ ] Add README auto-generation on repo creation (project name, forge link, status)

### M12: interactive setup — zero-to-running guided onboarding
New user clones visor, starts `pi` or `claude` in the repo folder, and gets guided through the entire setup process conversationally. No manual config editing, no reading docs. The agent walks them through everything and visor is running at the end.

#### Research tasks
- [ ] Investigate how CLAUDE.md / .pi/instructions can detect first-run state (no `.env`, no `data/` dir, no running process)
- [ ] Investigate platform-specific setup flows: what needs to happen for Telegram (bot token, webhook URL, chat ID) and potentially Signal
- [ ] Investigate how to validate env vars interactively (test Telegram token, test OpenAI key, etc.)
- [ ] Investigate optional level-up selection UX: how to present Forgejo, Himalaya, Obsidian as opt-in during setup

#### Iteration 1: first-run detection + core setup
- [ ] Add first-run detection in CLAUDE.md / agent instructions: check for `.env`, running process, `data/` dir
- [ ] Agent walks user through platform selection (Telegram, potentially Signal)
- [ ] Agent collects required env vars conversationally, writes `.env`
- [ ] Agent validates credentials (ping Telegram API, test OpenAI key, etc.)
- [ ] Agent sets up webhook (Telegram: run setup script)
- [ ] Agent runs `go build .` and starts visor, confirms it's responding on `/health`

#### Iteration 2: optional level-ups
- [ ] Agent presents available level-ups (Forgejo, Himalaya, Obsidian, Cloudflared) and lets user pick
- [ ] For each selected level-up: collect required env vars, write `.levelup.env`, start compose
- [ ] Forgejo: run bootstrap (admin user, push-to-create config), add git remote
- [ ] Verify each level-up is healthy before moving on

#### Iteration 3: personality + finish
- [ ] Agent asks if user wants to customize personality (edit CLAUDE.md) or keep defaults
- [ ] Agent sends a test message to the user's platform (Telegram) to confirm end-to-end flow
- [ ] Agent writes a summary of what was set up and how to start/stop visor
- [ ] Clean up setup instructions from CLAUDE.md (they're only needed once)

### M9: multi-pi-subagent orchestration (optional)
Visor can spawn multiple pi subagents in parallel, coordinate them, and return one merged final answer. Not needed for core functionality — nice-to-have for complex tasks.

#### Iteration 1: manual orchestration (on-demand)
- [ ] Add explicit trigger path (user command) to start multi-subagent execution
- [ ] Add orchestrator that spawns N pi subagent runs concurrently with bounded worker limit
- [ ] Add role templates per subagent (e.g. planner/researcher/critic/synthesizer)
- [ ] Add domain-specialized subagents ("starship stations") with fixed task areas
- [ ] Add per-domain model rank ladders (e.g. engineering: opus high rank, haiku fallback)
- [ ] Add JSON-configurable station registry (`config/subagent-stations.json`) with model/provider ranks per station
- [ ] Collect sub-results and produce one merged final response via coordinator step
- [ ] Add timeout/cancel handling so one stuck subagent does not block finalization

#### Iteration 2: reliability + observability
- [ ] Add per-subagent run IDs and structured logs (start/end/duration/error)
- [ ] Add partial-failure strategy (continue with surviving agents, mark degraded mode)
- [ ] Add execution report block in final response (which subagents ran, station/domain, model rank used, success/fail, latency)
- [ ] Add tests for fan-out/fan-in correctness and timeout behavior
- [ ] Add tests for rank-based fallback behavior inside one station

#### Iteration 3: automatic orchestration
- [ ] Add policy layer to auto-enable multi-subagent mode for complex tasks
- [ ] Add complexity heuristics (task size, ambiguity, required breadth) for auto-trigger
- [ ] Add station/domain auto-selection + model rank routing based on task classification
- [ ] Add budget/latency guardrails to avoid over-spawning
- [ ] Add fallback to single-agent mode when orchestration is unnecessary

## Open questions
- Skill sandboxing: how strict? Docker/nsjail or just subprocess with timeout?
- Should skills be able to call other skills (skill chaining)?

#forge #visor #go #project

> promoted from [ideas/visor-rewrite](../ideas/visor-rewrite.md)
