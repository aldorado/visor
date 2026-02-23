# Visor — next-gen agent runtime

## Core idea
Rewrite ubik from scratch in a compiled language (Rust or Go) for a faster, leaner agent runtime. New project in its own folder (`visor/`).

## Why
- TypeScript/Node overhead is noticeable — cold starts, memory, pi process spawning
- A compiled binary could serve webhooks and manage the agent process with minimal latency
- Chance to rethink the architecture from scratch with lessons learned from jarvis → ubik

## Language choice: needs thought
- **Go**: faster to write, excellent stdlib for HTTP/JSON/processes, goroutines for concurrency, single binary deployment. pragmatic choice.
- **Rust**: faster runtime, smaller binary, better memory safety. slower to iterate.
- This isn't just a webhook server though — it's a full agent runtime with:
  - persistent memory (embeddings, semantic search, parquet storage)
  - cron/scheduling system
  - voice pipeline (OpenAI transcription, ElevenLabs TTS)
  - session management
  - agent process manager (RPC-based)
  - message dedup + queuing
  - self-edit + restart capability
- For the memory/embeddings piece: Rust has better ML/embedding ecosystem (candle, ort). Go would need CGo bindings or shell out to Python.
- For concurrency (webhooks + cron + agent process + voice): both are strong. Go's goroutines are simpler, Rust's tokio is more performant.
- Leaning Rust if we want native embeddings. Leaning Go if we keep shelling out to external APIs for everything.

## Architecture
- Single binary as core runtime
- Webhook server (Telegram)
- Agent process manager (persistent child process, RPC over stdin/stdout)
- Voice pipeline (HTTP calls to OpenAI + ElevenLabs)
- Memory system (embeddings + vector search, parquet-first)
- Cron/scheduling (in-process scheduler, persistent to disk)
- Session logging (append-only JSONL files)
- Human-readable observability baseline (request lifecycle logs, traceback quality, verbose/normal modes, optional OTEL export to SigNoz)
- Native email in core capabilities (receive + send)
- Level-up infra alongside visor via Docker Compose extensions
- Self-edit: the agent edits source, triggers rebuild + restart


## Storage direction (research update)
- For Go implementation, prefer `parquet-go/parquet-go` for core read/write path.
- Treat parquet as immutable: append via new chunk files/row groups + periodic compaction.
- Keep `apache/arrow-go` optional for interoperability/export, not mandatory runtime core.

## Telegram integration direction (research update)
- v1 default: raw HTTP with Go stdlib (`net/http`) to keep runtime simple and dependency-light.
- keep framework options (`telebot`, `gotgbot`, `go-telegram/bot`) as fallback paths if ergonomics outweigh control later.

## Agent backend: swappable CLIs (free tier exploitation)
Key design decision: visor should be *agent-agnostic*. It manages the runtime (webhooks, memory, voice, cron) and delegates thinking to a CLI agent via a standard process interface.

The big win: AI companies give away free tokens with their CLIs to attract developers. Visor can rotate between backends to maximize free usage:
- **pi CLI** (`pi --mode rpc`) — free tokens from OpenAI/Codex
- **Claude Code** (`claude -p --output-format stream-json --verbose`) — practical CLI integration path (process-per-request)
- **Gemini CLI** (`gemini -p ... --output-format stream-json`) — available via `@google/gemini-cli`, headless JSON event stream support
- **GitHub Copilot CLI** (`@github/copilot`) — supports `-p` prompt mode and `--acp` server mode; useful secondary backend
- potentially others as they launch

Strategy: primary path is pi.dev as auth/runtime hub, with backend/model switching at runtime. visor can auto-switch when one provider/subscription hits limits.

quota strategy: combine static limits where officially documented (eg Gemini tiers) with runtime-discovered rate-limit signals (eg Claude `rate_limit_event`) and plan/subscription checks (eg Copilot premium request budget).

current finding: claude code currently exposes reliable `--print` + `stream-json` output, but no clear `--mode rpc` flag. so adapter should treat claude as process-per-request unless deeper stdin streaming contract is validated.

current finding: pi `--mode rpc` is stable and emits richer event streams than just text (`response`, `thinking_*`, `tool_execution_*`, `agent_end`). adapter should fail fast on `response success:false` and can optionally surface reasoning/tool telemetry.

current finding: gemini cli is available and supports headless `stream-json` output with documented events (`init`, `message`, `tool_use`, `tool_result`, `error`, `result`). best integration is process-per-request; `--experimental-acp` should be treated as unstable for now.

Visor just needs a unified interface per backend: send prompt → get response. Each backend gets an adapter that translates the common protocol to the CLI's specific RPC format.

This means visor is the "body" and the CLI agent is the "brain". Swap brains without changing anything else — and ride the free tier wave.

## Multi-subagent direction
Visor should support coordinated multi-agent thinking with pi subagents.
- v1: on-demand (explicit user call) fan-out to multiple pi subagents
- subagents have fixed domain stations (starship style), each with dedicated task area
- each station has a ranked model ladder (best -> fallback), e.g. engineering: opus high rank, haiku low rank
- station model ladders are JSON-configurable so routing can be tuned without code changes
- coordinator merges sub-results into one final answer
- later: automatic activation via complexity heuristics and latency/budget guardrails

## Skill parity bootstrap
Visor should start with the same skill surface as ubik.
- copied baseline skill pack from `ubik/.pi/skills` into `visor/skills/`
- includes all currently available skills (chat-history, memory-lookup, scheduling, obsidian, forge flows, minecraft, etc.)
- this is a bootstrap snapshot; visor can later evolve manifests/runtime details while preserving behavior parity

## Open questions
- How to handle the skill system in a compiled language? (scripts? WASM plugins? just let the agent write code?)
- Native embeddings vs API calls? (native = faster + offline, API = simpler)
- Unified RPC protocol or adapter per backend?
- SQLite for memories instead of parquet?

#project #visor #rewrite #go #rust

> **promoted to forge** — see [forge/visor](../forge/visor.md) for the execution plan
