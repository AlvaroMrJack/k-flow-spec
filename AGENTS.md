# k-flow-spec

QA automatizado para flujos WhatsApp sobre Kapso. Spec-driven, open-source, Go.

## Setup

```bash
make build        # compilar → bin/kfs
make install      # go install → $GOPATH/bin/kfs
make test         # go test ./...
make lint         # go vet ./...
```

## Code style

- Go 1.26+, sin generics innecesarios
- `fmt.Errorf` + `errors.Is` para errores, nada de libraries externas
- `slog` para logging (stdlib)
- Nombres en inglés aunque los mensajes al usuario puedan ir en español
- Tests en `_test.go` al lado del código que prueban
- imports ordenados: stdlib → terceros → internos → pkg

## CLI structure

```
kfs
├── init               # crear proyecto (único setup)
├── spec               # ciclo de vida del spec (core)
│   ├── generate       # crear stubs desde API
│   ├── learn          # grabar spec ejecutando el flujo real
│   ├── run            # ejecutar specs (--mock para offline)
│   ├── ls             # listar specs con estado
│   └── fix            # reparar specs rotos
├── tool               # herramientas avanzadas
│   ├── deploy, webhook, broadcast, flow, ui, mcp, mock
└── completion          # autocompletado (cobra built-in)
```

## Key design decisions

- **Polling con step-change detection**: `PollUntil` recibe `prevStep` y no retorna hasta que el `CurrentStep` realmente cambie. Sin esto, el learn loop cicla entre dos nodos.
- **Mensajes estructurados**: `MessagePayload{Kind, Data}` soporta texto (`payload`) y botones (`button_reply`).
- **Terminal status**: el validador soporta `"ended"`, `"failed"` y `"waiting"` como estados terminales.
- **AI routing no determinista**: el mismo mensaje puede rutear distinto cada vez. Los specs deben usar mensajes específicos o tolerar múltiples paths.
- **Mock simplificado**: termina tras 1 mensaje, solo para dev offline.

## Commit messages

Usar [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(cli): add kfs spec ls command
fix(poller): wait for CurrentStep change before returning
refactor: group commands under kfs spec and kfs tool
chore: remove corteya-specific references
docs: update README with pizza ordering example
```

## Testing

- `make test` para toda la suite
- `make lint` antes de commitear
- Para probar contra proyecto real: `kfs spec run`
- Para probar offline: `kfs spec run --mock`
- Nuevo comando → crear archivo en `internal/cli/`, registrar en `specCmd`, `toolCmd` o `RootCmd`
- Nueva lógica de polling → `pkg/runner/poller.go`
- Nuevo método de API → `pkg/kapso/client.go` + `pkg/kapso/types.go`

## Adding a new command

1. Crear `internal/cli/<name>.go`
2. Registrar en `specCmd.AddCommand(...)`, `toolCmd.AddCommand(...)` o `RootCmd.AddCommand(...)`
3. Si necesita flags, declararlas como vars de paquete en el archivo
4. Seguir el patrón de los comandos existentes (root discovery → config → exec)
