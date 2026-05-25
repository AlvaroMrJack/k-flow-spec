# k-flow-spec

> QA automatizado para flujos WhatsApp — Spec-driven, open-source, Go.
> Un solo comando y ya estás testeando.

```bash
# Instalación (elegí una)
brew install k-flow-spec
go install github.com/AlvaroMrJack/k-flow-spec/cmd/kfs@latest
npx k-flow-spec

# Usarlo
kfs init                    # Crea config en 1 segundo
kfs generate                # Descubre + genera specs automáticamente
kfs run                     # Ejecuta todos los specs
kfs run --mock              # Corre specs contra mock server (sin API key)
kfs run --watch             # Modo watch: corre al cambiar specs
kfs test                    # generate + run en un comando
kfs mock                    # Inicia mock server de la API Kapso
kfs fix                     # Auto-repara specs rotos
kfs ui                      # Dashboard web local
kfs run-broadcast            # Testea broadcast campaigns
kfs flow                     # Testea WhatsApp Flows
kfs webhook                  # Webhook receiver + validator
kfs deploy                   # Build + push + test pipeline
kfs completions              # Genera autocompletado para tu shell
```

## Problema

Hoy no existe forma de hacer QA automatizado de workflows dentro de WhatsApp. Todo se prueba manualmente: escribiendo mensajes desde el celular al número de pruebas. No hay CI/CD, no hay specs versionables, no hay validación programática de rutas y decisiones.

Cada deploy de un workflow es un salto de fe.

## Solución

Un CLI en Go que:

1. **Escanea** proyectos Kapso y descubre todos los workflows, broadcasts y flows via API
2. **Genera automáticamente** spec stubs a partir de la definición del grafo (nodos, aristas, configs)
3. **Mockea** la API de Kapso embebida para desarrollo sin conexión
4. **Ejecuta** los specs contra la Platform API o contra el mock y valida resultados paso a paso
5. **Reporta** en JSON, Markdown, JUnit XML (para CI), o TAP (para integración con test runners)
6. **Repara** specs rotos automáticamente al detectar cambios en los workflows
7. **Visualiza** historial, tendencias y snapshots en un dashboard web local
8. **Testea broadcasts** campañas masivas con validación de métricas
9. **Simula WhatsApp Flows** navegación de formularios nativos
10. **Captura webhooks** en tiempo real para validación integral
11. **Despliega** build + push + test en un solo comando

## API de Kapso (verificada contra docs)

Base URL: `https://api.kapso.ai/platform/v1`
Auth: `X-API-Key: <tu_api_key>`

| Operación | Endpoint | Propósito |
|-----------|----------|-----------|
| Listar workflows | `GET /workflows` | `kfs generate` descubre qué workflows existen |
| Leer definición | `GET /workflows/{workflow_id}/definition` | Analiza nodos, aristas, configs, variables |
| Leer variables | `GET /workflows/{workflow_id}/variables` | Descubre variables con sample values |
| Iniciar ejecución | `POST /workflows/{workflow_id}/executions` | Lanza test con `phone_number` + `variables` |
| Ver estado + context | `GET /workflow_executions/{execution_id}` | Polling: status, current_step, execution_context completo |
| Simular input | `POST /workflow_executions/{execution_id}/resume` | **Clave** — envía `{message:{kind:"payload", data:"text"}}` al `wait_for_response` |
| Forzar fin | `PATCH /workflow_executions/{execution_id}` → `{workflow_execution:{status:"ended"}}` | Cleanup de ejecuciones colgadas + timeout safety |
| Ver eventos | `GET /workflow_executions/{execution_id}/events` | Traza paso a paso para validar path y decisiones |
| Ver triggers | `GET /workflows/{workflow_id}/triggers` | Sabe si el workflow es inbound vs api_call |

### Rate Limiting (CRÍTICO para el runner)

| Límite | Legacy/Free | Pro | Enterprise/Platform |
|--------|-------------|-----|---------------------|
| General (por minuto) | 100 | 500 | 1000 |
| Burst (por segundo, por workflow) | 5 | 15 | 30 |

El runner debe:
- Respetar burst limiter: max N req/s por workflow (configurable en `kfs.yaml`)
- Manejar `429` con `Retry-After` y backoff exponencial
- Usar `X-Burst-RateLimit-Remaining` headers para adaptive pacing

### Ciclo de testing

```
POST /workflows/{workflow_id}/executions
  → 202 { data: { message, id, workflow_id, tracking_id } }
  ↓ polling c/500ms con backoff (max 30s)
GET /workflow_executions/{execution_id}
  → { data: { id, status, current_step, execution_context, events } }
  ↓ status: "waiting" (llegó a wait_for_response)
  ↓
POST /workflow_executions/{execution_id}/resume
  { message: { kind: "payload", data: "respuesta del spec" } }
  ↓ polling
GET /workflow_executions/{execution_id}
  → status: "waiting" | "ended" | "failed"
  ↓ repetir por cada mensaje del spec
  ↓ timeout global por spec (default: 60s, configurable)
GET /workflow_executions/{execution_id}/events
  → { data: [...] } validar path vs expected + snapshot execution_context
  ↓
PATCH /workflow_executions/{execution_id}
  → { data: { status: "ended" } }  # cleanup
```

## Formato Spec (YAML)

```yaml
# kfs-specs/support-router.yml
name: "Support Router - Happy Path"
workflow: support-router

# Timeout global para este spec (opcional, default 60s)
timeout: 120

given:
  variables:
    customer_name: "Juan"
    service: "corte clásico"
  # Teléfono de prueba (opcional, fallback a kfs.yaml)
  phone_number: "+56912345678"

when:
  messages:
    - user: "Tengo un problema con mi factura"
    - user: "Sí, necesito ayuda"
    - user: "Gracias"

then:
  # Path esperado: nodos por los que debe pasar
  path: ["start", "intro", "wait_reply", "classify", "handoff"]

  # Status terminal esperado
  terminal_status: "ended"

  # Decisiones esperadas en nodos decide
  decisions:
    classify: "billing"  # label esperado

  # Variables que deben estar seteadas al final
  variables_set:
    customer_name: "Juan"

  # Snapshot: captura execution_context completo y compara
  # Si no existe baseline, se crea en primera ejecución
  snapshot: true

  # Eventos clave que deben ocurrir (affirmative matching)
  events_include:
    - eventType: "decision_evaluated"
      edgeLabel: "billing"
```

### Auto-generación

`kfs generate` parsea `GET /workflows/{id}/definition` y crea stubs:

- `{{vars.*}}` en mensajes `send_text`/`send_interactive` → `given.variables` con placeholders
- Cada nodo `wait_for_response` → un `when.messages[{user: "__EDIT_ME__"}]` + hint del `save_response_to`
- Cada nodo `decide` con `conditions` → `then.decisions` con los labels
- Path lineal inferido del grafo (branches como placeholders con `__CHOOSE__`)
- Si el workflow nunca se ejecutó, marca variables sin sample values como `__FILL_ME__`

## Stack

| Capa | Tecnología | Por qué |
|------|-----------|---------|
| Lenguaje | **Go** | Binario único, cross-compile nativo, goroutines para polling |
| CLI | `cobra` | Subcomandos, autocompletado, help generado |
| HTTP | `net/http` (stdlib) | Sin dependencias externas, pool de conexiones built-in |
| Serialización | `encoding/json` + `gopkg.in/yaml.v3` | API Kapso JSON ↔ Specs YAML |
| Async | goroutines + channels | Polling loops, paralelismo entre specs, sin runtime externo |
| Errors | `fmt.Errorf` + `errors.Is` | Errores idiomáticos de Go, wrapping con %w |
| Logging | `slog` (stdlib, Go 1.21+) | Logs estructurados sin dependencias |
| Testing | `testing` (stdlib) + `testify` | Snapshot testing, asserts |
| Instalación | Homebrew + npm + go install + Docker + curl | "par de clicks y avanzar" |

## Estructura del proyecto

```
k-flow-spec/
├── go.mod
├── go.sum
├── README.md                    # Open-source intro, cómo contribuir
├── install.sh                   # Script one-liner: curl | bash
├── Dockerfile                   # Para CI sin Go toolchain
├── Makefile                     # build, test, lint, cross-compile
├── .github/
│   └── workflows/
│       ├── test.yml             # CI: go test + go vet en cada push
│       └── release.yml          # CD: cross-compile + publish a todos los canales
├── action/                      # GitHub Action (reusable)
│   └── action.yml
├── examples/
│   ├── corte-ya/
│   │   ├── booking.yml
│   │   └── support.yml
│   └── kfs.yml                  # Config de ejemplo
├── tests/
│   ├── fixtures/                # Workflow definitions de prueba
│   └── integration/             # Tests de integración contra mock server
├── cmd/
│   └── kfs/
│       └── main.go              # cobra: kfs {init,generate,run,mock,fix,ui,completions}
├── pkg/                         # Core expuesto como librería open-source
│   ├── kapso/
│   │   ├── client.go            # KapsoApiClient (net/http con rate limiting)
│   │   ├── types.go             # WorkflowDefinition, Node, Edge, etc.
│   │   └── error.go             # Errores tipados del API (429, 422, etc.)
│   ├── spec/
│   │   ├── types.go             # Spec, Given, When, Then, Snapshot
│   │   ├── parser.go            # Leer y validar YAML
│   │   └── generator.go         # WorkflowDefinition → Spec stub
│   ├── runner/
│   │   ├── engine.go            # Ciclo: start → poll → resume → validate
│   │   ├── scheduler.go         # Coordina specs paralelos respetando burst
│   │   ├── poller.go            # Polling con backoff y timeout
│   │   └── validator.go         # Path, decisions, variables, snapshot, events
├── internal/
│   ├── config/
│   │   └── config.go            # KfsConfig (API key, project, phone, etc.)
│   ├── discovery/
│   │   └── finder.go            # Escala directorios hacia arriba buscando kfs.yaml (como git)
│   ├── logger/
│   │   └── slog.go              # Wrapper de slog para CLI (colores, spinners, JSON)
│   ├── signal/
│   │   └── context.go           # Manejo de SIGINT/SIGTERM para graceful shutdown
│   ├── report/
│   │   ├── report.go            # Report interface
│   │   ├── json.go              # Reporte JSON
│   │   ├── markdown.go          # Reporte Markdown
│   │   ├── junit.go             # JUnit XML para CI
│   │   └── tap.go               # TAP format
│   ├── mock/
│   │   ├── server.go            # Servidor HTTP (net/http embebido)
│   │   ├── handlers.go          # Handlers que simulan cada endpoint Kapso
│   │   └── fixtures.go          # Carga de fixtures YAML → estados internos
│   ├── fix/
│   │   ├── analyzer.go          # Compara spec vs definition actual
│   │   ├── repair.go            # Apply repairs automáticos o interactivos
│   │   └── issues.go            # Tipos de issues detectables
│   └── ui/
│       ├── server.go            # Sirve HTML embebido + API REST local
│       ├── embed.go             # go:embed para index.html, app.js, style.css
│       ├── index.html           # Dashboard HTML
│       ├── app.js               # Lógica frontend (sin framework)
│       └── style.css            # Tema oscuro
```

## `kfs.yaml` — Configuración

```yaml
# kfs.yaml
project: corte-ya
api_key: ${KAPSO_API_KEY}         # O literal, pero env var es mejor
base_url: https://api.kapso.ai/platform/v1

# Teléfono dummy para testing
phone_number: "+56900000000"

# Rate limiting
rate_limit:
  max_burst: 5                    # Requests por segundo por workflow
  general_rpm: 100                # Requests por minuto totales

# Defaults para todos los specs
defaults:
  timeout: 60                     # Timeout global por spec (segundos)
  snapshot: true                  # Snapshot execution_context por defecto
  poll_interval_ms: 500           # Intervalo de polling (ms)
  poll_max_retries: 60            # Máximo de polls antes de timeout

# Directorios
specs_dir: kfs-specs
snapshots_dir: kfs-snapshots
reports_dir: kfs-reports

# Notificaciones (opcional)
notifications:
  slack_webhook: ${SLACK_WEBHOOK}
```

## Instalación

```bash
# Opción 1: Homebrew (recomendado)
# Opción 2: Go
go install github.com/AlvaroMrJack/k-flow-spec/cmd/kfs@latest

# Opción 3: npm (wrapper)
npx k-flow-spec init

# Opción 4: Docker (para CI)
docker run ghcr.io/AlvaroMrJack/k-flow-spec:latest kfs run

# Opción 5: One-liner
curl -fsSL https://raw.githubusercontent.com/AlvaroMrJack/k-flow-spec/main/install.sh | bash
```

## MVP — Listo para usar

### Fase 1: `kfs init` + `kfs mock` + `kfs generate` (Día 1)

```bash
kfs init
# → Crea kfs.yaml + kfs-specs/ + kfs-snapshots/ + kfs-mock-fixtures/
# → Pregunta interactiva por API key (opcional)
# → Si no hay API key, configura modo mock por defecto

kfs mock
# → Inicia mock server embebido (localhost:4172)
# → Viene con un workflow de ejemplo para probar
# → Sin API key, sin conexión a Kapso

kfs generate
# → Descubre todos los workflows via API (o desde fixtures en modo mock)
# → Genera .yml stubs en kfs-specs/
# → Markea placeholders con __EDIT_ME__
# → Detecta node types y crea hints precisos

kfs generate --workflow support-router
# → Solo un workflow específico

kfs generate --save-fixtures
# → Además de specs, guarda los fixtures para modo mock
```

### Fase 2: `kfs run` + `kfs fix` (Día 2)

```bash
kfs run --mock
# → Corre todos los specs contra mock server embebido
# → Sin API key, ideal para onboarding y development
# → Viene con workflow de ejemplo pre-cargado

kfs run
# → Ejecuta todos los specs contra Kapso real en paralelo
# → Reporta pass/fail por spec
# → Crea snapshots en primera ejecución
# → Exit code: 0 si todos pasan, 1 si algún fail

kfs run --spec kfs-specs/booking.yml
# → Solo un spec

kfs run --watch
# → File watcher: re-ejecuta specs al cambiar archivos

kfs run --ci
# → JUnit XML output + exit codes estrictos

kfs fix
# → Analiza specs rotos contra definition actual
# → Detecta nodos renombrados, ramas eliminadas, variables cambiadas
# → kfs fix --apply aplica reparaciones automáticamente
```

### Fase 3: `kfs test` + `kfs ui` + CI (Día 3)

```bash
kfs test
# = kfs generate + kfs run en un solo comando

kfs ui
# → Dashboard local en localhost:4173
# → Historial, tendencias, snapshot diff, run manual

kfs ui --mock
# → Dashboard + mock server integrado, todo local

# En GitHub Actions:
# .github/workflows/test.yml
# - name: Test Kapso Workflows
#   run: kfs test
#   env:
#     KAPSO_API_KEY: ${{ secrets.KAPSO_API_KEY }}
```

## Formato de reporte

### CLI output (coloreado)

```
✓ support-router (3.2s) — path: 6/6, decisions: 2/2, snapshot: ✓
✓ booking-flow (5.1s) — path: 8/8, decisions: 3/3, snapshot: ✓
✗ cancel-flow (12.4s) — path: 5/7 ✗ (esperado: "confirm_cancel", recibió: "timeout")
  ↓ diff:
    - "confirm_cancel"
    + "timeout"
  ⚠ snapshot mismatch: kfs-snapshots/cancel-flow.yml

Results: 2 passed, 1 failed, 0 skipped (20.7s)
```

### JSON

```json
{
  "summary": { "passed": 2, "failed": 1, "skipped": 0, "duration_ms": 20700 },
  "specs": [
    {
      "name": "cancel-flow",
      "passed": false,
      "errors": [
        {
          "type": "path_mismatch",
          "expected": ["start", "menu", "cancel", "confirm_cancel"],
          "actual": ["start", "menu", "cancel", "timeout"],
          "snapshot_diff": true
        }
      ]
    }
  ]
}
```

### JUnit XML

```xml
<testsuite name="k-flow-spec" tests="3" failures="1">
  <testcase name="support-router" classname="support-router" time="3.2"/>
  <testcase name="booking-flow" classname="booking-flow" time="5.1"/>
  <testcase name="cancel-flow" classname="cancel-flow" time="12.4">
    <failure message="path mismatch">
      Expected confirm_cancel, got timeout
    </failure>
  </testcase>
</testsuite>
```

## Snapshot Testing

El killer feature. Cada spec genera un snapshot del `execution_context` completo en `kfs-snapshots/{workflow}.snap.yml`:

```yaml
# kfs-snapshots/support-router.snap.yml
workflow: support-router
run_at: "2026-05-25T12:00:00Z"

execution_context:
  vars:
    customer_name: "Juan"
    service: "corte clásico"
    classify_result: "billing"
  system:
    flow_id: "flow_abc123"
    started_at: "2026-05-25T12:00:00Z"
  context:
    phone_number: "+56912345678"

events:
  - eventType: "execution_started"
    timestamp: "2026-05-25T12:00:00Z"
  - eventType: "step_entered"
    step: { identifier: "intro" }
    timestamp: "2026-05-25T12:00:01Z"
  - eventType: "step_entered"
    step: { identifier: "wait_reply" }
    timestamp: "2026-05-25T12:00:02Z"
  - eventType: "user_input_received"
    step: { identifier: "wait_reply" }
    payload: { content: "Tengo un problema..." }
    timestamp: "2026-05-25T12:00:03Z"
  - eventType: "decision_evaluated"
    step: { identifier: "classify" }
    edgeLabel: "billing"
    payload: { model: "gpt-4o-mini", reasoning: "..." }
    timestamp: "2026-05-25T12:00:04Z"
  - eventType: "execution_ended"
    timestamp: "2026-05-25T12:00:05Z"
```

En ejecuciones posteriores, se compara contra el snapshot. Las diferencias se reportan como fallo o warning (configurable).

## CI/CD Integrado

### GitHub Action

```yaml
# .github/workflows/test.yml
name: Test Kapso Workflows
on:
  push:
    paths: ["workflows/**"]
  pull_request:
    paths: ["workflows/**"]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: k-flow-spec/action@v1
        with:
          api_key: ${{ secrets.KAPSO_API_KEY }}
          phone_number: ${{ secrets.TEST_PHONE }}
          specs_dir: kfs-specs
      - uses: dorny/test-results@v1
        if: always()
        with:
          path: kfs-reports/*.xml
```

### Docker

```dockerfile
FROM ghcr.io/AlvaroMrJack/k-flow-spec:latest AS tester
COPY kfs-specs/ /specs/
RUN kfs run --ci
```

## `kfs mock` — Mock Server embebido

Un servidor HTTP que simula la Platform API de Kapso. Permite desarrollar y testear specs **sin conexión a Kapso**, sin API key, y con respuestas predecibles.

```bash
# Iniciar mock server
kfs mock

# En otra terminal, correr specs contra el mock
kfs run --mock

# Mock en puerto custom
kfs mock --port 9999
kfs run --mock --mock-port 9999

# Cargar fixtures custom
kfs mock --fixtures ./test-fixtures/
```

### Qué simula

- `GET /workflows` → lista de workflows desde fixtures
- `GET /workflows/{id}/definition` → grafo completo con nodos y aristas
- `GET /workflows/{id}/variables` → fixed + discovered variables
- `POST /workflows/{id}/executions` → 202 con tracking_id
- `GET .../executions/{id}` → status machine que avanza con cada resume
- `POST .../executions/{id}/resume` → simula recepción de mensaje
- `PATCH .../executions/{id}/status` → permite forzar fin
- `GET .../executions/{id}/events` → eventos según el grafo y los resumes

### Fixtures

Las definiciones de workflow se cargan desde `kfs-mock-fixtures/` por defecto:

```yaml
# kfs-mock-fixtures/support-router.yml
name: support-router
status: active
definition:
  nodes:
    - id: start
      data: { node_type: "start" }
    - id: intro
      data: { node_type: "send_text", config: { message: "Hola {{customer_name}}" } }
    - id: wait_reply
      data: { node_type: "wait_for_response", config: { save_response_to: "user_input" } }
    - id: classify
      data:
        node_type: "decide"
        config:
          decision_type: "ai"
          conditions:
            - { label: "billing", description: "Facturación" }
            - { label: "technical", description: "Soporte técnico" }
    - id: handoff
      data: { node_type: "handoff" }
  edges:
    - { source: "start", target: "intro", label: "next" }
    - { source: "intro", target: "wait_reply", label: "next" }
    - { source: "wait_reply", target: "classify", label: "next" }
    - { source: "classify", target: "handoff", label: "billing" }
```

### Automático con `kfs generate`

Si ya tienes acceso a Kapso, `kfs generate --save-fixtures` genera los fixtures automáticamente:

```bash
kfs generate --save-fixtures
# → Crea specs en kfs-specs/ + fixtures en kfs-mock-fixtures/
# → Después puedes correr kfs run --mock sin conexión
```

Después del init, el mock server viene pre-cargado con un workflow de ejemplo para que el usuario pueda probar `kfs run --mock` inmediatamente sin configuración.

---

## `kfs fix` — Auto-reparación de specs

Detecta specs rotos y sugiere o aplica reparaciones automáticas.

```bash
# Analizar todos los specs en busca de problemas
kfs fix

# Aplicar reparaciones automáticamente
kfs fix --apply

# Modo interactivo: pregunta antes de cada cambio
kfs fix --interactive

# Analizar un spec específico
kfs fix --spec kfs-specs/booking.yml
```

### Qué detecta y repara

| Problema | Detección | Reparación |
|----------|-----------|------------|
| Nodo renombrado | Path del spec referencia nodo que ya no existe en definition | Actualiza path + decisions al nuevo nombre |
| Rama eliminada | `then.decisions` referencia un label de condition que ya no existe | Elimina el decision obsoleto |
| Variable renombrada | `given.variables` usa key que cambió en el grafo | Renombra la variable |
| Mensaje `__EDIT_ME__` | Spec tiene placeholders sin editar | Warning (no auto-repara, necesita input humano) |
| Snapshot desactualizado | Snapshot baseline difiere de la ejecución real | `kfs fix --update-snapshots` |
| Path incompleto | Faltan nodos intermedios en el path esperado | Completa el path según el grafo actual |
| Spec YAML inválido | Error de parseo | Sugiere la línea exacta y el fix |

### Ejemplo

```bash
$ kfs fix
🔍 Analizando 3 specs...

✗ kfs-specs/support-router.yml
  → Nodo "classify_intent" renombrado a "classify" (línea 42)
  → Variable "user_name" ya no existe en el grafo, se usaba en línea 17
  ⚠ Mensaje #2 es __EDIT_ME__ (sin reparar)

✓ kfs-specs/booking.yml — sin problemas
✓ kfs-specs/cancel-flow.yml — snapshot desactualizado

Reparaciones disponibles: 2
  kfs fix --apply          # Aplica todo
  kfs fix --interactive    # Revisa uno por uno
```

---

## `kfs ui` — Dashboard Web Local

Interfaz web HTML embebida en el binario (sin dependencias externas). Se sirve en `localhost:4173`.

```bash
# Iniciar dashboard
kfs ui

# Puerto custom
kfs ui --port 8080

# Con mock server integrado (no necesita Kapso)
kfs ui --mock

# Solo generar HTML estático (para deploy)
kfs ui --export ./kfs-dashboard/
```

### Qué muestra

**Historial de ejecuciones** — todas las runs anteriores con fecha, duración, pass/fail:

```
┌──────────────────────────────────────────────────────┐
│  k-flow-spec Dashboard                        v0.1  │
│──────────────────────────────────────────────────────│
│  Últimas ejecuciones                                 │
│  ┌──────────────────────────────────────────────────┐│
│  │ ✓ support-router  3.2s  path 6/6  dec 2/2  snap ││
│  │ ✓ booking-flow     5.1s  path 8/8  dec 3/3  snap ││
│  │ ✗ cancel-flow     12.4s  path 5/7  snap diff    ││
│  │ ✓ support-router  2.8s  path 6/6  dec 2/2       ││
│  └──────────────────────────────────────────────────┘│
│                                                       │
│  Tendencias (últimos 7 días)                          │
│  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐    │
│  │100% │ │100% │ │ 66% │ │100% │ │100% │ │100% │    │
│  │  ✓  │ │  ✓  │ │  ✗  │ │  ✓  │ │  ✓  │ │  ✓  │    │
│  │ lun │ │ mar │ │ mie │ │ jue │ │ vie │ │ sab │    │
│  └─────┘ └─────┘ └─────┘ └─────┘ └─────┘ └─────┘    │
└──────────────────────────────────────────────────────┘
```

**Snapshot diff visual** — comparación lado a lado de execution_context, con colores en líneas agregadas/eliminadas.

**Specs explorer** — lista de todos los specs, su estado, último resultado.

**Run manual** — botón para ejecutar un spec desde el browser (via MCP server interno).

### Stack del UI

- HTML estático + CSS + JS vanilla (embebido en el binario con `//go:embed`)
- Sin frameworks, sin node_modules, sin build step
- Chart.js inline para tendencias
- Fetch a `http://localhost:4173/api/*` servido por el mismo `kfs ui`
- Tema oscuro por defecto (coherente con CLI)

---

## MCP Server incluido

El binario `kfs` también sirve como MCP server:

```bash
# En tu opencode.json o claude_desktop_config.json:
{
  "mcpServers": {
    "k-flow-spec": {
      "command": "kfs",
      "args": ["mcp"],
      "env": {
        "KAPSO_API_KEY": "..."
      }
    }
  }
}
```

Esto expone tools MCP que el asistente IA puede llamar:
- `generate_spec` — Genera specs desde descripción o desde la API
- `run_specs` — Ejecuta tests y devuelve resultados
- `get_status` — Consulta resultados de ejecuciones anteriores
- `update_snapshots` — Actualiza snapshots existentes
- `fix_specs` — Analiza y repara specs rotos
- `run_broadcast` — Lanza broadcast de prueba y valida resultados
- `run_flow_test` — Simula navegación de WhatsApp Flow
- `deploy_and_test` — Build + push + test en un solo paso

---

## `kfs run-broadcast` — Testing de Broadcasts

Valida campañas de Broadcast API completas: creación, destinatarios, envío, y métricas.

```bash
# Lanzar broadcast de prueba
kfs run-broadcast --spec kfs-specs/promo-july.yml

# Contra mock (sin enviar a reales)
kfs run-broadcast --mock

# Solo validar recipients sin enviar
kfs run-broadcast --dry-run
```

### Formato Spec para Broadcasts

```yaml
# kfs-specs/promo-july.yml
name: "Promo July - Broadcast Test"
kind: broadcast

given:
  template_id: "promo_july_2026"
  phone_number_id: "1134735729717664"
  recipients:
    - phone: "+56900000001"
      variables:
        first_name: "Juan"
        discount: "50%"
    - phone: "+56900000002"
      variables:
        first_name: "María"
        discount: "30%"

when:
  send: {}  # Solo dispara el envío

then:
  status: "completed"
  metrics:
    sent_count: 2
    failed_count: 0
    delivery_rate: 1.0  # 100%
  # Por cada recipient validar entrega
  recipients:
    - phone: "+56900000001"
      status: "sent"
    - phone: "+56900000002"
      status: "sent"
```

### Ciclo de testing

```
POST /whatsapp/broadcasts → 201 (broadcast_id)
  ↓ agregar recipients
POST /whatsapp/broadcasts/{id}/recipients
  ↓ enviar
POST /whatsapp/broadcasts/{id}/send → status: "sending"
  ↓ polling c/5s (max 5 min)
GET /whatsapp/broadcasts/{id}
  → status: "completed" | "failed"
  → validar sent_count, failed_count, delivery_rate
  ↓
GET /whatsapp/broadcasts/{id}/recipients
  → validar status individual por recipient
```

### Mock support

`kfs mock` también simula Broadcasts API:
- Crea broadcast en estado `draft`
- Acepta recipients (validando formato)
- Envío simulado: avanza a `completed` después de 3s
- Métricas simuladas: 100% delivery rate

---

## `kfs flow` — Testing de WhatsApp Flows

Simula la navegación de un usuario dentro de un WhatsApp Flow (formulario nativo).

```bash
# Ejecutar spec de flow
kfs flow --spec kfs-specs/checkout-flow.yml

# Con mock server
kfs flow --mock

# Abrir flow en browser para debug visual
kfs flow --open
```

### Formato Spec para Flows

```yaml
# kfs-specs/checkout-flow.yml
name: "Checkout - Happy Path"
kind: flow

given:
  flow_id: "123456789012345"
  screen: "WELCOME"
  initial_data:
    phone_number: "+56900000000"
    customer_name: "Juan"

when:
  # Navegación: pantalla → acción → campos
  screens:
    - screen: "WELCOME"
      action: "next"
    - screen: "SERVICES"
      action: "select"
      fields:
        service: "corte clásico"
        professional: "any"
    - screen: "SCHEDULE"
      action: "submit"
      fields:
        date: "2026-06-01"
        time: "15:00"
    - screen: "CONFIRM"
      action: "confirm"

then:
  terminal_screen: "THANK_YOU"
  # Datos enviados por el flow al webhook
  submitted_data:
    service: "corte clásico"
    date: "2026-06-01"
    time: "15:00"
  # Webhook recibido correctamente
  webhook_received: true
  webhook_payload:
    status: "completed"
```

### Qué valida

- Navegación entre pantallas (`action: "next"`, `"select"`, `"submit"`, `"confirm"`)
- Renderizado de campos en cada screen
- Datos enviados vs esperados
- Webhook de salida recibido con payload correcto
- Errores de validación en campos

### Mock support

`kfs mock --flows` simula el servidor de WhatsApp Flows:
- Responde con screens según el Flow JSON cargado
- Valida campos requeridos
- Simula el webhook de salida
- Permite testear errores: campo inválido, timeout, etc.

---

## `kfs webhook` — Webhook Receiver Embebido

Levanta un servidor HTTP temporal, lo registra como webhook en Kapso, captura eventos en tiempo real, y los valida contra el spec.

```bash
# Iniciar webhook receiver + correr spec
kfs webhook --spec kfs-specs/support-router.yml

# Puerto custom
kfs webhook --port 9000

# Webhook + workflow execution en un solo comando
kfs webhook --run

# Ver eventos en tiempo real por consola
kfs webhook --verbose
```

### Cómo funciona

```
kfs webhook --spec booking.yml
  ↓
1. Inicia servidor HTTP en localhost:9000
2. Registra webhook en Kapso vía API:
   POST /whatsapp/phone_numbers/{id}/webhooks
   { url: "https://{tunnel}.ngrok.io/webhook",
     events: ["workflow.execution.completed", "message.sent"] }
3. Ejecuta el workflow spec
4. Captura eventos en tiempo real
5. Valida eventos contra `then.events_include` del spec
6. Limpia: elimina webhook al terminar
```

### Formato Spec extendido

```yaml
# kfs-specs/booking-with-webhook.yml
then:
  path: ["start", "menu", "booking", "confirm"]
  terminal_status: "ended"

  # Webhook validation
  webhook:
    events_expected:
      - eventType: "workflow.execution.started"
      - eventType: "workflow.execution.completed"
    # El webhook debe recibirse antes de N segundos
    timeout_seconds: 10
    # Payload debe contener estos campos
    payload_contains:
      execution_id: string
      workflow_id: string
      status: "ended"
```

### Tunnel automático (ngrok)

Si `ngrok` está instalado, `kfs webhook` crea un tunnel automáticamente:

```bash
# Sin ngrok: solo recibe si Kapso reachable (local network)
kfs webhook

# Con ngrok: auto-tunnel
kfs webhook --tunnel

# Forzar tunnel en puerto específico
kfs webhook --tunnel --port 9000
```

---

## `kfs deploy` — Pipeline Deploy + Test

Un solo comando para compilar, desplegar y testear un workflow Kapso.

```bash
# Build + push + test
kfs deploy

# Solo build + test (sin push)
kfs deploy --dry-run

# Deploy a entorno específico
kfs deploy --env staging

# Deploy + broadcast test + webhook validation
kfs deploy --full
```

### Pipeline

```
kfs deploy
  ↓
1. kfs build                    # Compila workflow.js → definition.json
2. kfs push workflow <name>     # Despliega a Kapso prod
3. kfs generate                   # Regenera specs frescos
4. kfs run --ci                    # Ejecuta tests contra workflow deployeado
5. Report: pass/fail + duración   # Resultado final
```

### Configuración en kfs.yaml

```yaml
# kfs.yaml
deploy:
  auto_generate: true        # Regenerar specs después de deploy
  auto_run: true             # Correr tests después de deploy
  environment: production    # production | staging
  # Workflows a deployar (vacío = todos)
  workflows:
    - support-router
    - booking-flow
```

### En CI/CD

```yaml
# .github/workflows/deploy.yml
name: Deploy & Test
on:
  push:
    paths: ["workflows/**"]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: k-flow-spec/action@v1
        with:
          api_key: ${{ secrets.KAPSO_API_KEY }}
          command: deploy
```

| Caso | Qué pasa |
|------|----------|
| Workflow no existe | Error claro: `Workflow "foo" not found in project` |
| API key inválida | Error 401 con diagnóstico |
| Rate limited (429) | Backoff automático + log warning |
| Timeout en wait_for_response | Se detecta via `system.last_resume.reason === "timeout"` |
| Workflow sin ejecuciones previas | `GET /variables` devuelve solo fixed → mark como `__FILL_ME__` |
| Múltiples triggers | Se listan todos en el spec generado |
| Ejecución colgada | Timeout global + PATCH cleanup forzado |
| Decide node con AI vs function | El spec detecta el modo y ajusta validación |
| Snapshot inicial vs diff | Primera run = baseline, segunda run = diff |
| Specs rotos (YAML inválido) | Error de parseo con línea exacta + sugerencia |
| Broadcast sin recipients | Error: `broadcast debe tener al menos 1 recipient` |
| Flow screen no existe | Error: `Screen "CHECKOUT" no encontrada en el flow` |
| Webhook no llega | Timeout: `webhook no recibido en 10s` + log de eventos disponibles |
| ngrok no instalado | Warning + fallback a localhost sin tunnel |
| Deploy sin cambios | Skip: `No hay cambios en workflows/` |

## Roadmap resumido

| Fase | Tiempo | Entregable |
|------|--------|------------|
| Fase 1 | Día 1 | `kfs init` + `kfs mock` + `kfs generate` con/sin API key |
| Fase 2 | Día 2 | `kfs run` (real + --mock) + polling + validación + `kfs fix` |
| Fase 3 | Día 3 | `kfs test` + CI + snapshot + reportes + `kfs ui` |
| Fase 4 | Día 4 | MCP server + `kfs webhook` + `kfs flow` + parallel runner |
| Fase 5 | Día 5 | `kfs run-broadcast` + `kfs deploy` + Homebrew + Docker + docs |

## Conclusión

De 0 a QA automatizado de workflows WhatsApp en **5 días**, con:

- **Un binario** — `kfs` lo hace todo
- **Un comando** — `kfs test` descubre, genera y ejecuta
- **Zero config** — `kfs init` + pegar API key
- **CI ready** — GitHub Action, Docker, JUnit XML
- **AI ready** — MCP server para asistentes
- **Broadcast ready** — Testea campañas masivas
- **Flow ready** — Simula formularios nativos de WhatsApp
- **Pipeline ready** — `kfs deploy` buildea, pushea y testea

```bash
# Día 1: Instalar + configurar
go install github.com/AlvaroMrJack/k-flow-spec/cmd/kfs@latest && kfs init

# Día 1: Generar specs automáticamente
kfs generate

# Día 1: Ejecutar tests
kfs run

# Día 2: Integrar en CI
# Agregar el GitHub Action al repo
# kfs test corre en cada push
```
