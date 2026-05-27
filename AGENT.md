# k-flow-spec — Agent Guide

> Automated QA for WhatsApp workflows on Kapso. Spec-driven, open-source, Go.

## Stack

- **Language**: Go 1.26+
- **CLI**: cobra
- **HTTP**: net/http (stdlib)
- **Serialization**: `encoding/json` + `gopkg.in/yaml.v3`
- **Async**: goroutines + channels
- **Logging**: `slog` (stdlib)
- **Module**: `github.com/AlvaroMrJack/k-flow-spec`

## Project Map

```
k-flow-spec/
├── AGENT.md              ← This file
├── Makefile              # build, install, test, lint, clean
├── go.mod / go.sum
├── cmd/kfs/              # Main binary entrypoint
├── internal/
│   ├── cli/              # cobra commands (learn, run, generate, etc.)
│   ├── config/           # kfs.yaml loading + .env support
│   ├── discovery/        # Workspace root discovery
│   ├── fix/              # Auto-repair broken specs
│   ├── mock/             # Embedded mock WhatsApp API server
│   ├── broadcast/        # Broadcast API testing
│   ├── deploy/           # Build → push → test pipeline
│   ├── flow/             # WhatsApp Flow testing
│   ├── logger/           # Structured logging
│   ├── mcp/              # MCP server for AI assistants
│   ├── report/           # JUnit XML, JSON, TAP reports
│   ├── signal/           # Graceful shutdown
│   ├── ui/               # Web dashboard
│   └── webhook/          # Webhook receiver + validator
├── pkg/
│   ├── kapso/            # Kapso API client + types
│   │   ├── client.go     # HTTP client, do(), all API methods
│   │   ├── types.go      # Workflow, ExecutionStatus, Event, MessagePayload
│   │   └── error.go      # APIError, RateLimitError, NotFoundError, ValidationError
│   ├── runner/           # Test engine + poller + validator
│   │   ├── engine.go     # Engine.Run() — main test loop
│   │   ├── poller.go     # PollUntil() — polls for status + step change
│   │   └── validator.go  # Validate() — path, decisions, terminal_status, variables, events
│   └── spec/             # Spec types + parser + generator
│       ├── types.go      # Spec, Message (text + button), Given, When, Then
│       ├── parser.go     # Load(), Save()
│       └── generator.go  # Generate() — creates stub specs from workflow definition
├── docs/README.md        # Full user-facing docs
├── Dockerfile
└── install.sh
```

## Key Design Decisions

### Polling with step-change detection (`poller.go`)
`PollUntil` accepts `prevStep interface{}`. When the target status is `"waiting"` and `prevStep` is non-nil, it keeps polling until `CurrentStep` actually differs from `prevStep`. This prevents the race where the API briefly returns the old step after `ResumeExecution`.

### Structured messages (`kapso.MessagePayload`)
Messages sent to the Kapso API use `{kind, data}`:
- **Text**: `{kind: "payload", data: "hello"}`
- **Button**: `{kind: "button_reply", data: {id: "btn_1", title: "Option 1"}}`

The spec `Message` type has an optional `Button` field; `ToPayload()` converts it automatically.

### Terminal status validation
`terminal_status` supports three values:
- `"ended"` — expects an `execution_ended` event
- `"failed"` — expects an `execution_failed` event
- `"waiting"` — expects **no** ended/failed event (flow returned to waiting state)

## How to Build & Install

```bash
make build        # ./bin/kfs
make install      # go install → $GOPATH/bin/kfs
make test         # go test ./...
make lint         # go vet ./...
```

## Common Workflows

### Create a spec interactively
```bash
kfs learn --workflow <workflow-id>
```
Type user messages one by one. The tool records the path, decisions, and messages. Ends with `done` or `Ctrl+C`.

### Run a spec
```bash
kfs run                          # all specs in kfs-specs/
kfs run --spec kfs-specs/foo.yaml
kfs run --mock                   # against embedded mock (no API key)
kfs run --ci                     # JUnit XML output
```

### Generate stubs
```bash
kfs generate                     # non-interactive, from API
kfs generate -i                  # step-by-step wizard
```

### Fix broken specs
```bash
kfs fix                          # analyze
kfs fix --apply                  # auto-repair
```

## Important Gotchas

1. **AI routing is non-deterministic.** The same message can route differently on different runs (e.g., `router: reservar` vs `router: unirme`). Specs should use specific messages that the AI classifies consistently, or expect multiple possible paths.
2. **Learn vs Run divergence.** The `kfs learn` command shows only "waiting" steps to the user, but the generated spec path includes all intermediate nodes (processing nodes between user inputs). This is correct behavior.
3. **Mock is simplified.** The mock server always ends after one message. It's for dev/testing without real API access. For real workflow testing, use the real API.
4. **Environment variables.** API keys go in `kfs.yaml` as `${KAPSO_API_KEY}` or in a `.env` file next to `kfs.yaml`.

## Testing a Project (e.g., corteya)

```bash
cd /path/to/project

# 1. Install latest kfs
cd /path/to/k-flow-spec && go install ./cmd/kfs && cd -

# 2. Run all specs
kfs run

# 3. Record a new spec
kfs learn

# 4. Run a specific spec
kfs run --spec kfs-specs/onboarding-profesional.yaml
```

## Modifying kfs

1. **New API method** → `pkg/kapso/client.go` + `pkg/kapso/types.go`
2. **New CLI command** → `internal/cli/<name>.go` + register in `RootCmd`
3. **Polling logic** → `pkg/runner/poller.go`
4. **Validation rules** → `pkg/runner/validator.go`
5. **Spec format** → `pkg/spec/types.go`

Always run `go vet ./...` and `go test ./...` before committing.
