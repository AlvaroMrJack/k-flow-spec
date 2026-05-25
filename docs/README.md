# k-flow-spec

> QA automatizado para flujos WhatsApp — Spec-driven, open-source, Go.

[![Go Reference](https://pkg.go.dev/badge/github.com/AlvaroMrJack/k-flow-spec.svg)](https://pkg.go.dev/github.com/AlvaroMrJack/k-flow-spec)
[![CI](https://github.com/AlvaroMrJack/k-flow-spec/actions/workflows/test.yml/badge.svg)](https://github.com/AlvaroMrJack/k-flow-spec/actions/workflows/test.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

```bash
# Opción 1 — Un solo comando (auto-instala + configura PATH)
curl -fsSL https://raw.githubusercontent.com/AlvaroMrJack/k-flow-spec/main/install.sh | bash

# Opción 2 — Solo Go (si ya lo tienes)
go install github.com/AlvaroMrJack/k-flow-spec/cmd/kfs@latest

# Opción 3 — Docker
docker run ghcr.io/AlvaroMrJack/k-flow-spec:latest kfs run --mock
```

---

## ¿Por qué?

Hoy, testear un workflow de WhatsApp es agarrar el celular, escribir mensajes a tu número de pruebas, y esperar que pase lo correcto. No hay CI/CD, no hay specs versionables, no hay validación programática.

**Cada deploy es un salto de fe.**

`k-flow-spec` resuelve esto: un CLI que escanea workflows, broadcasts y flows; genera specs automáticamente; los ejecuta contra la API real o mock; captura webhooks en tiempo real; y despliega con un solo comando.

---

## Quick Start

```bash
# 1. Probar al toque (sin API key, sin conexión)
cd tu-proyecto/
kfs init
kfs run --mock

# 2. Conectar a API real (interactivo)
kfs init --configure             # Crea proyecto + asistente paso a paso
kfs generate -i                  # Genera specs interactivamente
kfs run                          # Ejecuta contra API real
```

Salida:

```
✓ support-router (3.2s) — path: 6/6, decisions: 2/2, snapshot: ✓
✓ booking-flow (5.1s) — path: 8/8, decisions: 3/3, snapshot: ✓
✗ cancel-flow (12.4s) — path: 5/7 (esperado: confirm_cancel, recibió: timeout)

Results: 2 passed, 1 failed, 0 skipped (20.7s)
```

---

## Instalación

```bash
# Go
go install github.com/AlvaroMrJack/k-flow-spec/cmd/kfs@latest

# Docker
docker run ghcr.io/AlvaroMrJack/k-flow-spec:latest kfs run

# One-liner
curl -fsSL https://raw.githubusercontent.com/AlvaroMrJack/k-flow-spec/main/install.sh | bash
```

---

## Cómo funciona

1. **`kfs init`** — Crea `kfs.yaml` con configuración base
2. **`kfs configure`** — Asistente interactivo: API key, teléfono, modo (mock/real), timeouts, rate limits, notificaciones, deploy
3. **`kfs mock`** — Inicia servidor HTTP que simula la API de WhatsApp (para dev sin conexión)
4. **`kfs generate`** o **`kfs generate -i`** — Descubre workflows via API (o fixtures), genera specs YAML. Modo interactivo pregunta mock/real, Flows, selección de workflows
5. **`kfs run`** — Ejecuta specs contra API real (`kfs run`) o contra mock (`kfs run --mock`)
6. **`kfs fix`** — Analiza specs rotos y los repara automáticamente
7. **`kfs ui`** — Dashboard web local con historial, tendencias y snapshot diff
8. **`kfs test`** — `generate` + `run` en un solo comando
9. **`kfs run-broadcast`** — Testea campañas Broadcast API
10. **`kfs flow`** — Simula navegación de WhatsApp Flows
11. **`kfs webhook`** — Webhook receiver + validador en tiempo real
12. **`kfs deploy`** — Build + push + test pipeline

### Ciclo de testing

```
POST /workflows/{id}/executions → 202 (tracking_id)
  ↓ polling c/500ms
GET /workflow_executions/{id}   → status: "waiting"
  ↓
POST /workflow_executions/{id}/resume {message:{kind:"payload", data:"respuesta"}}
  ↓ polling
GET /workflow_executions/{id}   → "waiting" | "ended" | "failed"
  ↓ repetir por cada mensaje del spec
GET /workflow_executions/{id}/events → validar path + snapshot
  ↓
PATCH /workflow_executions/{id} → {workflow_execution:{status:"ended"}}  # cleanup
```

---

## Formato Spec

```yaml
# kfs-specs/booking.yml
name: "Booking - Happy Path"
workflow: booking-flow

given:
  variables:
    customer_name: "Juan"
    service: "corte clásico"
  phone_number: "+56900000000"

when:
  messages:
    - user: "Quiero agendar un corte"
    - user: "Mañana a las 15:00"
    - user: "Sí, confirmar"

then:
  path: ["start", "menu", "booking", "confirm"]
  terminal_status: "ended"
  decisions:
    classify: "booking"
  variables_set:
    customer_name: "Juan"
  snapshot: true
```

`kfs generate` crea estos stubs automáticamente — solo editas los mensajes y paths esperados.

---

## Reportes

| Formato | Uso |
|---------|-----|
| CLI coloreado | Desarrollo local |
| JSON | Integración con otros tools |
| JUnit XML | CI/CD (GitHub Actions, GitLab CI) |
| TAP | Test Anything Protocol |

```bash
kfs run --ci        # JUnit XML en kfs-reports/
kfs run --format json  # JSON output
```

---

## Snapshot Testing

Cada spec captura el `execution_context` completo como snapshot. En la segunda ejecución, compara contra el baseline y reporta diferencias. Ideal para detectar regresiones invisibles (cambios en variables, respuestas de AI, paths inesperados).

```bash
kfs run --update-snapshots  # Actualizar todos los snapshots
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
      - run: kfs test --ci
        env:
          KAPSO_API_KEY: ${{ secrets.KAPSO_API_KEY }}
```

---

## `kfs run --mock` — Sin API key, sin conexión

El mock server embebido simula toda la API de WhatsApp. Ideal para onboarding, desarrollo, y CI cuando no quieres depender de upstream.

```bash
# Iniciar mock + correr specs, todo en uno
kfs run --mock

# Iniciar mock server standalone
kfs mock
```

Viene con un workflow de ejemplo pre-cargado, así que `kfs run --mock` funciona desde el primer `go install`, sin configuración.

---

## `kfs fix` — Auto-reparación

Detecta y repara specs rotos automáticamente.

```bash
# Analizar specs
kfs fix

# Aplicar reparaciones
kfs fix --apply

# Modo interactivo
kfs fix --interactive
```

| Problema | Reparación |
|----------|-----------|
| Nodo renombrado | Actualiza path + decisions |
| Rama eliminada | Elimina decisiones obsoletas |
| Variable cambiada | Renombra en given.variables |
| Snapshot desactualizado | kfs run --update-snapshots |

---

## `kfs ui` — Dashboard web local

```bash
# Iniciar dashboard
kfs ui

# Dashboard + mock integrado
kfs ui --mock

# Exportar HTML estático
kfs ui --export ./dashboard/
```

Muestra historial de ejecuciones, tendencias (7 días), diff visual de snapshots, y permite ejecutar specs desde el browser.

---

## `kfs run-broadcast` — Testing de Broadcasts

Valida campañas de Broadcast API: creación, recipients, envío y métricas.

```bash
# Lanzar broadcast de prueba
kfs run-broadcast --spec kfs-specs/promo-july.yml

# Contra mock (sin enviar a reales)
kfs run-broadcast --mock

# Solo validar recipients sin enviar
kfs run-broadcast --dry-run
```

---

## `kfs flow` — Testing de WhatsApp Flows

Simula navegación de un usuario dentro de un Flow (formulario nativo de WhatsApp).

```bash
# Ejecutar spec de flow
kfs flow --spec kfs-specs/checkout-flow.yml

# Con mock
kfs flow --mock

# Abrir flow en browser para debug visual
kfs flow --open
```

---

## `kfs webhook` — Webhook Receiver

Levanta un servidor HTTP temporal, lo registra como webhook, captura eventos y los valida contra el spec.

```bash
# Webhook receiver + correr spec
kfs webhook --spec kfs-specs/booking.yml

# Ver eventos en tiempo real
kfs webhook --verbose

# Con tunnel ngrok automático
kfs webhook --tunnel
```

---

## `kfs deploy` — Pipeline Deploy + Test

Un solo comando: build → push → test.

```bash
# Build + push + test
kfs deploy

# Solo build + test (sin push)
kfs deploy --dry-run

# Deploy completo + broadcast + webhook
kfs deploy --full
```

---

## MCP Server

`kfs` también funciona como MCP server para asistentes IA:

```json
{
  "mcpServers": {
    "k-flow-spec": {
      "command": "kfs",
      "args": ["mcp"],
      "env": { "KAPSO_API_KEY": "..." }
    }
  }
}
```

---

## Comandos

| Comando | Descripción |
|---------|-------------|
| **Onboarding** | |
| `kfs init` | Crea `kfs.yaml` base |
| `kfs init --configure` | Crea `kfs.yaml` + lanza asistente de configuración |
| `kfs configure` | Asistente interactivo: API key, teléfono, modo, timeouts, rate limits, notificaciones, deploy |
| **Generación de specs** | |
| `kfs generate` | Descubre workflows y genera specs (no interactivo) |
| `kfs generate -i` | Generación paso a paso: mock/real, Flows, selección de workflows |
| `kfs generate --save-fixtures` | Genera specs + fixtures para modo mock |
| `kfs generate --workflow <id>` | Genera spec para un workflow específico |
| **Ejecución** | |
| `kfs run` | Ejecuta todos los specs contra API real |
| `kfs run --mock` | Ejecuta contra mock server embebido (sin API key) |
| `kfs run --spec <file>` | Ejecuta un spec específico |
| `kfs run --watch` | Modo watch: re-ejecuta al cambiar archivos |
| `kfs run --ci` | Modo CI: JUnit XML + exit codes estrictos |
| `kfs run --update-snapshots` | Actualiza snapshots existentes |
| `kfs run --interactive` | Modo debug paso a paso |
| `kfs run --format <fmt>` | Formato de reporte: json, junit, tap, markdown |
| `kfs test` | generate + run en un comando |
| **Mock server** | |
| `kfs mock` | Inicia mock server standalone |
| **Mantenimiento** | |
| `kfs fix` | Analiza y repara specs rotos |
| `kfs fix --apply` | Aplica reparaciones automáticamente |
| **Testing avanzado** | |
| `kfs run-broadcast` | Testea campañas Broadcast API |
| `kfs run-broadcast --mock` | Broadcast contra mock |
| `kfs flow` | Simula navegación de WhatsApp Flows |
| `kfs flow --mock` | Flows contra mock |
| `kfs flow --open` | Abre flow en browser |
| `kfs webhook` | Webhook receiver + validador en tiempo real |
| `kfs webhook --tunnel` | Webhook con ngrok automático |
| **Deploy** | |
| `kfs deploy` | Build + push + test pipeline |
| `kfs deploy --dry-run` | Build + test sin push |
| `kfs deploy --full` | Deploy + broadcast + webhook |
| **Dashboard** | |
| `kfs ui` | Dashboard web local |
| `kfs ui --mock` | Dashboard + mock integrado |
| **Integración** | |
| `kfs mcp` | Inicia MCP server para asistentes IA |
| `kfs completions` | Genera autocompletado para tu shell |

---

## Configuración

```yaml
# kfs.yaml — Configuración completa
project: "mi-proyecto"
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

## Arquitectura y Diseño Interno

`k-flow-spec` está construido para ser robusto y extensible:

- **Auto-Discovery**: Al igual que `git`, `kfs` escala hacia arriba en el árbol de directorios buscando tu `kfs.yaml`. Esto te permite ejecutar tests desde cualquier subcarpeta de tu proyecto sin problemas de paths relativos.
- **Core como Librería (`pkg/`)**: El motor de testing, el parser de specs y el cliente HTTP están expuestos públicamente. Puedes importar `github.com/AlvaroMrJack/k-flow-spec/pkg/...` en tus propios proyectos Go.
- **Graceful Shutdown**: Manejo seguro de señales (`Ctrl+C`). Si cancelas una ejecución en progreso, el motor termina de procesar los requests en vuelo limpiamente y genera un reporte parcial antes de salir.
- **Logging Especializado**: Salida limpia en terminal, respaldado por volcados de debug asíncronos a archivo usando el `slog` de Go 1.22+.

---

## Stack

| Capa | Tecnología |
|------|-----------|
| Lenguaje | Go |
| CLI | cobra |
| HTTP | net/http (stdlib) |
| Serialización | encoding/json + gopkg.in/yaml.v3 |
| Async | goroutines + channels |
| Errors | fmt.Errorf + errors.Is |
| Logging | slog (stdlib) |

---

## Roadmap

| Fase | Tiempo | Entregable |
|------|--------|------------|
| 1 | Día 1 | `kfs init` + `kfs mock` + `kfs generate` con/sin API key |
| 2 | Día 2 | `kfs run` (real + --mock) + polling + validación + `kfs fix` |
| 3 | Día 3 | `kfs test` + CI + snapshot + reportes + `kfs ui` |
| 4 | Día 4 | MCP server + `kfs webhook` + `kfs flow` + parallel runner |
| 5 | Día 5 | `kfs run-broadcast` + `kfs deploy` + Docker + docs |

---

## Contribuir

```bash
git clone https://github.com/AlvaroMrJack/k-flow-spec
cd k-flow-spec
go build ./cmd/kfs
go test ./...
```

PRs bienvenidas. Issues también.

---

## Licencia

MIT — haz lo que quieras con esto.
