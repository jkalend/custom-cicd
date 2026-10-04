# Laya CI/CD

A CI/CD pipeline system with an **AI decision layer**: every failure classification, log triage, issue routing, and notification filtering is a small, typed judgment made by [Laya](https://huggingface.co/convaiinnovations/laya) (Convai Innovation's open-weights System One decision model, Apache 2.0, served locally via the Laya gateway) — composed into ordinary pipeline logic, never generating prose.

A ground-up re-architecture of the original Python/Flask pipeline manager — that first version is preserved on the [`archive/pre-jev-cicd`](https://github.com/jkalend/custom-cicd/tree/archive/pre-jev-cicd) branch. The backend is now **Go** (pipeline engine plus a REST API wire-compatible with the original Flask contract), the Go CLI is unchanged in contract, and the Next.js/TypeScript frontend gained an AI dashboard. The original frontend also shipped without two files every page imported (`src/lib/api.ts`, `src/lib/utils.ts`) — a fresh clone couldn't build; they're restored here.

## Architecture

```
              nginx (:80)
             /          \
       frontend:3000   /api/* → backend:8000 (Go)
        Next.js/TS             engine ── executes shell steps
                              /    \
                          laya client  persistence (JSON)
                              |
                    Laya gateway (:8128) ── convaiinnovations/laya
                    (local, self-hosted, keyless)
```

- **backend/** — Go. Pipeline engine (runs, steps, retries, timeouts, cancellation), REST API (wire-compatible with the original Flask contract: wrapped `{data, success, error}`), Laya client with offline fallback, and the four decision modules.
- **frontend/** — Next.js/TS. Original dashboard plus the AI Decision Layer page: live decision log, notification feed, and playgrounds for log triage and issue routing.
- **cli/** — Go (cobra). Unchanged contract: `cicd --api-url http://localhost:8000 pipeline list` etc.
- **nginx.conf** — reverse proxy; `/api/*` → backend, everything else → frontend.

## The Laya decision layer

Four modules, each a small set of typed questions (`boolean` / `choice` / `score`) asked in one batched request per event, with plain Go applying thresholds:

| Module | Fires when | Questions |
| --- | --- | --- |
| `classify_failure` | a step fails | failure kind (flaky/build/config/infra/code), worth retrying?, severity 0–3 |
| `triage_logs` | step output captured | severity 0–3, action (ignore/investigate/restart/page), page engineer? |
| `route_issue` | user routes an issue | is bug?, component, severity, regression? → labels |
| `filter_notification` | a run finishes | level (ignore/defer/notify/urgent), interrupt now? |

Principles: **Laya advises, Go decides** — e.g. a "page" only fires when both the action choice and the page-engineer boolean agree; urgency likewise needs two signals. Every decision is appended to `data/laya_decisions.jsonl` (provider, latency, state, questions, answers) and surfaced in the frontend. With the gateway unreachable — or during its checkpoint load — a deterministic heuristic provider keeps everything working; the system never fails because the AI layer did.

## Run it

Docker (nginx + backend + frontend):

```bash
docker-compose up -d          # http://localhost
```

Local development:

```bash
# backend (Go 1.22+)
cd backend && go run ./cmd/server          # :8000

# frontend (Node 18+)
cd frontend && npm install
BACKEND_URL=http://localhost:8000 npm run dev   # :3000

# CLI (optional)
cd cli && go build -o cicd .
./cicd --api-url http://localhost:8000 health
```

Enable live Laya: start the local Laya gateway (see below). The backend talks to `http://127.0.0.1:8128/v1/evaluate` by default; override with `LAYA_GATEWAY_URL`. No API key is needed. Without the gateway running you stay in heuristic mode — everything still works, decisions are just deterministic placeholders.

### The Laya gateway (local sidecar)

The decision models are PyTorch checkpoints, so they run in a small Python sidecar that speaks the same `/v1/evaluate` dialect the backend already uses. One-time setup:

```bash
python -m venv laya-venv
laya-venv/Scripts/pip install laya        # or bin/pip on Unix
laya-venv/Scripts/python laya_gateway.py  # :8128, checkpoints load in background
```

`laya_gateway.py` lives at the repo root. It downloads the English and multilingual checkpoints on first start (~1.4 GB), serves `/v1/evaluate` and `/health`, auto-routes per request, and pins ~1.5 GB of VRAM (or runs on CPU at ~200–500 ms per decision). Docker deployments reach it through `host.docker.internal:8128`.

## Using it

1. Open the dashboard, paste a pipeline (or use the template) and **Create & Run**.
2. When a step fails, expand it: the **AI Analysis** panel shows the failure kind, retry recommendation, and log action.
3. Open **🧠 AI Decision Layer** (top-right): the live decision log, notification levels for every run completion, and the two playgrounds (log triage, issue router).

Example pipeline:

```json
{
  "name": "Demo",
  "version": "1.0.0",
  "steps": [
    { "name": "build", "command": "echo building", "timeout": 30 },
    { "name": "tests", "command": "npm test", "timeout": 120, "retry_count": 1 }
  ]
}
```

## License

MIT
