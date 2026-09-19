# Jev CI/CD

A CI/CD pipeline system with an **AI decision layer**: every failure classification, log triage, issue routing, and notification filtering is a small, typed judgment made by [Jev](https://typesafe.ai) (TypeSafe AI's System One model, served via the Vercel AI Gateway) — composed into ordinary pipeline logic, never generating prose.

Forked from [`custom-cicd`](https://github.com/jkalend/custom-cicd) and re-architected: the Python/Flask backend is now **Go**, the existing Go CLI is unchanged in contract, and the Next.js/TypeScript frontend gained an AI dashboard. The original frontend also shipped without two files every page imported (`src/lib/api.ts`, `src/lib/utils.ts`) — a fresh clone couldn't build; they're restored here.

## Architecture

```
              nginx (:80)
             /          \
       frontend:3000   /api/* → backend:8000 (Go)
        Next.js/TS             engine ── executes shell steps
                              /    \
                          jev client  persistence (JSON)
                              |
                    Vercel AI Gateway ── typesafe-ai/jev
```

- **backend/** — Go. Pipeline engine (runs, steps, retries, timeouts, cancellation), REST API (wire-compatible with the original Flask contract: wrapped `{data, success, error}`), Jev client with offline fallback, and the four decision modules.
- **frontend/** — Next.js/TS. Original dashboard plus the AI Decision Layer page: live decision log, notification feed, and playgrounds for log triage and issue routing.
- **cli/** — Go (cobra). Unchanged contract: `cicd --api-url http://localhost:8000 pipeline list` etc.
- **nginx.conf** — reverse proxy; `/api/*` → backend, everything else → frontend.

## The Jev decision layer

Four modules, each a small set of typed questions (`boolean` / `choice` / `score`) asked in one batched request per event, with plain Go applying thresholds:

| Module | Fires when | Questions |
| --- | --- | --- |
| `classify_failure` | a step fails | failure kind (flaky/build/config/infra/code), worth retrying?, severity 0–3 |
| `triage_logs` | step output captured | severity 0–3, action (ignore/investigate/restart/page), page engineer? |
| `route_issue` | user routes an issue | is bug?, component, severity, regression? → labels |
| `filter_notification` | a run finishes | level (ignore/defer/notify/urgent), interrupt now? |

Principles: **Jev advises, Go decides** — e.g. a "page" only fires when both the action choice and the page-engineer boolean agree; urgency likewise needs two signals. Every decision is appended to `data/jev_decisions.jsonl` (provider, latency, state, questions, answers) and surfaced in the frontend. With no API key — or if the gateway is unreachable — a deterministic heuristic provider keeps everything working; the system never fails because the AI layer did.

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

Enable live Jev: copy `backend/.env.example` to `backend/.env`, set `AI_GATEWAY_API_KEY` (Vercel AI Gateway key; the account needs a credit card on file to serve requests), restart the backend. Without it you run in heuristic mode — everything still works, decisions are just deterministic placeholders.

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
