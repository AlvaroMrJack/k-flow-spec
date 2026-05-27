# k-flow-spec

> Automated QA for WhatsApp workflows on Kapso — Spec-driven, open-source, Go.

[![Go Reference](https://pkg.go.dev/badge/github.com/AlvaroMrJack/k-flow-spec.svg)](https://pkg.go.dev/github.com/AlvaroMrJack/k-flow-spec)
[![CI](https://github.com/AlvaroMrJack/k-flow-spec/actions/workflows/test.yml/badge.svg)](https://github.com/AlvaroMrJack/k-flow-spec/actions/workflows/test.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

```bash
# Option 1 — One-liner (auto-install + configure PATH)
curl -fsSL https://raw.githubusercontent.com/AlvaroMrJack/k-flow-spec/main/install.sh | bash

# Option 2 — Go only
go install github.com/AlvaroMrJack/k-flow-spec/cmd/kfs@latest

# Option 3 — Docker
docker run ghcr.io/AlvaroMrJack/k-flow-spec:latest kfs spec run --mock
```

---

## Why?

Testing a WhatsApp workflow today means grabbing your phone, typing messages to your test number, and hoping things work. No CI/CD, no versionable specs, no programmatic validation.

**Every deploy is a leap of faith.**

`k-flow-spec` fixes this: a CLI that discovers workflows, broadcasts, and flows; auto-generates specs; runs them against the real API or a mock; captures webhooks in real time; and deploys with a single command.

### The stack

```
k-flow-spec  ← Automated QA layer (you are here)
    ↓
Kapso        ← Workflow engine (workflows, AI, broadcasts, flows)
    ↓
Meta API     ← Official WhatsApp Business API
```

`k-flow-spec` is a QA layer on top of [Kapso](https://kapso.ai), which orchestrates WhatsApp conversations using Meta's official Business API. You write specs once and run them against your Kapso workflows — either against the real API or the embedded mock server.

---

## Quick Start

```bash
# 1. Try it instantly (no API key, no connection needed)
cd your-project/
kfs init
kfs spec run --mock

# 2. Connect to the real API
kfs spec generate                # Generate specs from your Kapso workflows
kfs spec learn                   # Or record specs by running the flow live
kfs spec run                     # Run against the real API
```

Output:

```
✓ support-router (3.2s) — path: 6/6, decisions: 2/2, snapshot: ✓
✓ booking-flow (5.1s) — path: 8/8, decisions: 3/3, snapshot: ✓
✗ cancel-flow (12.4s) — path: 5/7 (expected: confirm_cancel, got: timeout)

Results: 2 passed, 1 failed, 0 skipped (20.7s)
```

---

## Installation

```bash
# Go
go install github.com/AlvaroMrJack/k-flow-spec/cmd/kfs@latest

# Docker
docker run ghcr.io/AlvaroMrJack/k-flow-spec:latest kfs run

# One-liner
curl -fsSL https://raw.githubusercontent.com/AlvaroMrJack/k-flow-spec/main/install.sh | bash
```

---

## How It Works

1. **`kfs init`** — Creates `kfs.yaml` with base config and `.env.example`
2. **`kfs spec generate`** — Discovers workflows via API, generates YAML specs. Use `-i` for interactive wizard
3. **`kfs spec learn`** — Records a spec by running the live workflow step by step from the terminal
4. **`kfs spec run`** — Runs specs against the real API. Use `--mock` for offline development
5. **`kfs spec ls`** — Lists all specs with name, messages, path length, and snapshot status
6. **`kfs spec fix`** — Analyzes broken specs and repairs them automatically
7. **`kfs tool ...`** — Advanced tools: `deploy`, `webhook`, `broadcast`, `flow`, `ui`, `mcp`, `mock`

### Testing cycle

```
POST /workflows/{id}/executions → 202 (tracking_id)
  ↓ poll every 500ms
GET /workflow_executions/{id}   → status: "waiting"
  ↓
POST /workflow_executions/{id}/resume {message:{kind:"payload", data:"reply"}}
  ↓ poll
GET /workflow_executions/{id}   → "waiting" | "ended" | "failed"
  ↓ repeat for each spec message
GET /workflow_executions/{id}/events → validate path + snapshot
  ↓
PATCH /workflow_executions/{id} → {workflow_execution:{status:"ended"}}  # cleanup
```

---

## Spec Format

```yaml
# kfs-specs/order-pizza.yml
name: "Order Pizza - Happy Path"
workflow: pizza-bot

given:
  variables:
    city: "Buenos Aires"
  phone_number: "+541100000000"

when:
  messages:
    - user: "I want to order a pizza"
    - user: "Mozzarella"
    - user: "Large"
    - user: "Yes, confirm"

then:
  path: ["start", "menu", "ask_pizza", "ask_size", "confirm", "done"]
  terminal_status: "ended"
  decisions:
    intent: "order"
    confirm: "yes"
  variables_set:
    pizza: "Mozzarella"
    size: "Large"
  snapshot: true
```

`kfs spec generate` creates these stubs automatically — you just edit the messages and expected paths.

### Button messages

Some workflows send interactive buttons. Use the `button` field to reply:

```yaml
messages:
  - button:
      id: "btn_confirm"
      title: "Yes, confirm"
  - user: "Av. Corrientes 1234"
```

---

## Reports

| Format | Use |
|--------|-----|
| Colored CLI | Local development |
| JSON | Integration with other tools |
| JUnit XML | CI/CD (GitHub Actions, GitLab CI) |
| TAP | Test Anything Protocol |

```bash
kfs spec run --ci        # JUnit XML in kfs-reports/
kfs spec run --format json  # JSON output
```

---

## Snapshot Testing

Each spec captures the full `execution_context` as a snapshot. On subsequent runs, it compares against the baseline and reports differences. Ideal for catching invisible regressions (variable changes, AI responses, unexpected paths).

```bash
kfs spec run --update-snapshots  # Update all snapshots
```

---

## CI/CD

```yaml
# .github/workflows/test.yml
name: Test
on:
  push:
    paths: ["workflows/**"]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.26"
      - run: go install github.com/AlvaroMrJack/k-flow-spec/cmd/kfs@latest
      - run: kfs spec run --ci
        env:
          KAPSO_API_KEY: ${{ secrets.KAPSO_API_KEY }}
```

---

## `kfs spec run --mock` — No API key, no connection needed

The embedded mock server simulates the entire WhatsApp API. Perfect for onboarding, development, and CI when you don't want to depend on upstream.

```bash
# Start mock + run specs, all in one
kfs spec run --mock

# Start mock server standalone
kfs tool mock
```

It ships with a pre-loaded example workflow, so `kfs spec run --mock` works from the first `go install`, with zero configuration.

---

## `kfs spec learn` — Interactive spec recording

Runs the real workflow and asks you for each message one by one. When you're done, it generates the spec with the actual path, AI decisions, and your replies.

```bash
# Pick a workflow from the list
kfs spec learn

# Specify a workflow directly
kfs spec learn --workflow <workflow-id>

# Example session:
$ kfs spec learn --workflow pizza-bot-123
  Connecting to Kapso API...
  ✓ Execution started

  ╔══════════════════════════════════════╗
  ║  Recording flow — type 'done'        ║
  ║  to finish, Ctrl+C to exit           ║
  ╚══════════════════════════════════════╝

  ─── Workflow waiting at: ask_pizza ───
  You > Mozzarella
  ─── Workflow waiting at: ask_size ───
  You > Large
  ─── Workflow waiting at: ask_address ───
  You > Av. Corrientes 1234
  ...
  You > done

  ✓ Spec saved: kfs-specs/pizza-bot-learned.yaml
    - 3 messages recorded
    - 8 nodes in path
    - 2 decisions captured
```

Unlike `kfs spec generate` (which creates a static spec with placeholders), `kfs spec learn` runs the real flow and captures exactly what happened: the path the AI took, the decisions at each router, and the messages you typed.

---

## `kfs spec fix` — Auto-repair

Detects and repairs broken specs automatically.

```bash
# Analyze specs
kfs spec fix

# Apply repairs
kfs spec fix --apply

# Interactive mode
kfs spec fix --interactive
```

| Problem | Repair |
|---------|--------|
| Node renamed | Updates path + decisions |
| Branch removed | Removes stale decisions |
| Variable changed | Renames in given.variables |
| Outdated snapshot | kfs spec run --update-snapshots |

---

## `kfs ui` — Local web dashboard

```bash
# Start dashboard
kfs tool ui

# Dashboard + integrated mock
kfs tool ui --mock

# Export static HTML
kfs tool ui --export ./dashboard/
```

Shows execution history, 7-day trends, visual snapshot diffs, and lets you run specs from the browser.

---

## `kfs tool broadcast` — Broadcast testing

Validates Broadcast API campaigns: creation, recipients, sending, and metrics.

```bash
# Launch a test broadcast
kfs tool broadcast --spec kfs-specs/promo-july.yml

# Against mock (no real sends)
kfs tool broadcast --mock

# Validate recipients only, no send
kfs tool broadcast --dry-run
```

---

## `kfs tool flow` — WhatsApp Flow testing

Simulates user navigation inside a Flow (native WhatsApp form).

```bash
# Run a flow spec
kfs tool flow --spec kfs-specs/checkout-flow.yml

# With mock
kfs tool flow --mock

# Open flow in browser for visual debugging
kfs tool flow --open
```

---

## `kfs tool webhook` — Webhook Receiver

Starts a temporary HTTP server, registers it as a webhook, captures events, and validates them against the spec.

```bash
# Webhook receiver + run spec
kfs tool webhook --spec kfs-specs/order-pizza.yml

# Watch events in real time
kfs tool webhook --verbose

# With automatic ngrok tunnel
kfs tool webhook --tunnel
```

---

## `kfs tool deploy` — Deploy + Test Pipeline

Single command: build → push → test.

```bash
# Build + push + test
kfs tool deploy

# Build + test only (no push)
kfs tool deploy --dry-run

# Full deploy + broadcast + webhook
kfs tool deploy --full
```

---

## `kfs tool mcp` — MCP Server

`kfs` works as an MCP server for AI assistants:

```json
{
  "mcpServers": {
    "k-flow-spec": {
      "command": "kfs",
      "args": ["tool", "mcp"],
      "env": { "KAPSO_API_KEY": "..." }
    }
  }
}
```

---

## Commands

### Core

| Command | Description |
|---------|-------------|
| `kfs init` | Creates `kfs.yaml` and `.env.example` |
| `kfs spec generate` | Discovers workflows and generates YAML specs |
| `kfs spec generate -i` | Step-by-step generation wizard |
| `kfs spec generate --workflow <id>` | Generate spec for a specific workflow |
| `kfs spec learn` | Records a spec by running the live workflow from the terminal |
| `kfs spec learn --workflow <id>` | Records a specific workflow directly |
| `kfs spec run` | Runs all specs against the real API |
| `kfs spec run --mock` | Runs against the embedded mock server (no API key needed) |
| `kfs spec run --spec <file>` | Runs a specific spec |
| `kfs spec run --watch` | Re-runs when spec files change |
| `kfs spec run --ci` | JUnit XML output + strict exit codes |
| `kfs spec run --update-snapshots` | Updates existing snapshots |
| `kfs spec run --format <fmt>` | Report format: json, junit, tap, markdown |
| `kfs spec ls` | Lists all specs with workflow name, messages, path, and snapshot status |
| `kfs spec fix` | Analyzes and repairs broken specs |
| `kfs spec fix --apply` | Applies repairs automatically |

### Advanced tools

| Command | Description |
|---------|-------------|
| `kfs tool deploy` | Build + push + test pipeline |
| `kfs tool deploy --dry-run` | Build + test without push |
| `kfs tool webhook` | Webhook receiver + real-time validator |
| `kfs tool webhook --tunnel` | Auto-tunnel with ngrok |
| `kfs tool broadcast` | Tests Broadcast API campaigns |
| `kfs tool broadcast --mock` | Broadcast against mock |
| `kfs tool flow` | Simulates WhatsApp Flow navigation |
| `kfs tool flow --mock` | Flows against mock |
| `kfs tool ui` | Local web dashboard |
| `kfs tool ui --mock` | Dashboard + integrated mock |
| `kfs tool mcp` | MCP server for AI assistants |
| `kfs tool mock` | Standalone mock server |

---

## Configuration

```yaml
# kfs.yaml — Full configuration
project: "my-project"
base_url: "https://api.kapso.ai/platform/v1"
api_key: "${KAPSO_API_KEY}"
phone_number: "+56900000000"

specs_dir: "kfs-specs"
snapshots_dir: "kfs-snapshots"
reports_dir: "kfs-reports"

rate_limit:
  max_burst: 5
  general_rpm: 100

defaults:
  timeout: 60
  snapshot: true
  poll_interval_ms: 500
  poll_max_retries: 60

notifications:
  slack_webhook: "https://hooks.slack.com/..."

deploy:
  environment: "staging"
  auto_generate: true
  auto_run: true
  workflows: []
```

---

## Architecture & Internal Design

`k-flow-spec` is built for robustness and extensibility:

- **Auto-Discovery**: Like `git`, `kfs` walks up the directory tree looking for your `kfs.yaml`. This lets you run tests from any subfolder without relative path issues.
- **Core as a Library (`pkg/`)**: The test engine, spec parser, and HTTP client are publicly exposed. Import `github.com/AlvaroMrJack/k-flow-spec/pkg/...` in your own Go projects.
- **Graceful Shutdown**: Safe signal handling (`Ctrl+C`). If you cancel a running execution, the engine finishes in-flight requests cleanly and generates a partial report before exiting.
- **Structured Logging**: Clean terminal output backed by async debug dumps to file using Go 1.22+'s `slog`.

---

## Stack

| Layer | Technology |
|-------|-----------|
| Language | Go |
| CLI | cobra |
| HTTP | net/http (stdlib) |
| Serialization | encoding/json + gopkg.in/yaml.v3 |
| Async | goroutines + channels |
| Errors | fmt.Errorf + errors.Is |
| Logging | slog (stdlib) |

---

## Roadmap

| Phase | Time | Deliverable |
|-------|------|-------------|
| 1 | Day 1 | `kfs init` + `kfs spec generate` with/without API key |
| 2 | Day 2 | `kfs spec run` (real + --mock) + polling + validation + `kfs spec fix` |
| 3 | Day 3 | CI + snapshot + reports + `kfs tool ui` |
| 4 | Day 4 | MCP server + `kfs tool webhook` + `kfs tool flow` + parallel runner |
| 5 | Day 5 | `kfs tool broadcast` + `kfs tool deploy` + Docker + docs |

---

## Contributing

```bash
git clone https://github.com/AlvaroMrJack/k-flow-spec
cd k-flow-spec
go build ./cmd/kfs
go test ./...
```

PRs welcome. Issues welcome. Keep it simple.

---

## License

MIT — do whatever you want with it.
