# kyzmet — Go guest SDK

Go module targeting TinyGo's WASI Preview 2 component target. Provides these
layers:

- **World bindings** (`world/app/`, `world/auth-type/`,
  `world/local-mcp-service/`, `world/inference-router/`): public, TinyGo-native imports and exports
  generated from the exact Protocol world. Each world has its own import root,
  so one world's packages cannot stand in for another's.
- **Layer 1** (`raw.go`, `raw_wasip2.go`, `raw_stub.go`): the vault and graph
  host calls and the bridge-fetch export, called directly through the generated
  component bindings. Host builds get stubs that return `ErrNotImplemented`, and
  a wasip1 build fails to compile.
- **Layer 2** (`a2ui.go`, `manifest.go`): ergonomic builders with construction-time validation.
- **Connector adapters** (`connectoradapter/`): local MCP services on the
  `local-mcp-service` world, with typed provider HTTP, delegated tools and the
  stable tool-error vocabulary.

## Installing

```sh
go get kyzmet.ai/go@latest
```

The SDK follows Semantic Versioning, with its own version line, independent of
the KDK's. Before 1.0, a breaking change is a minor release, so pin the minor
version you build against.

## Mod types

Each executable mod type is one world. A mod imports that world's packages and
registers its exports by assigning the generated `Exports` functions in `init`:

| Mod type | World packages (`kyzmet.ai/go/...`) | Exports to assign |
|---|---|---|
| app | `world/app/kyzmet/host/<interface>` | `apprender.Exports` (`Render`, `HandleEvent`, `HandleTrigger`) and `bridgefetch.Exports.Handle` |
| auth-type | `world/auth-type/kyzmet/host/<interface>` | `authtypehandler.Exports` (`ValidateConfig`, `BeginRegistration`, `CompleteRegistration`, `BeginVerification`, `CompleteVerification`) |
| local-mcp-service | `world/local-mcp-service/kyzmet/host/<interface>` | `localmcpservicehandler.Exports.Invoke`, or `connectoradapter.Export(handler)` |
| inference-router | `world/inference-router/kyzmet/router/<interface>` | `inferencerouterhandler.Exports` (`RecommendModels`, `Run`) |

```go
package main

import (
	authtypehandler "kyzmet.ai/go/world/auth-type/kyzmet/host/auth-type-handler"
)

func init() {
	authtypehandler.Exports.ValidateConfig = validateConfig
	// ...one assignment per exported function.
}

func main() {}
```

Use only the packages of the world the mod type builds against: KDK selects the
world from the build unit's output role, and a component that imports another
world's interfaces fails to build.

`vendor-journey/surface/go/check` in the KDK proves that each world's every
Kyzmet import is reachable through public packages from a module outside the
SDK's path, and that a stub registering every export builds into a component.
`just go-surface-check` runs it against the checkout's SDK, with a regression
that breaks the export wiring and the `Exports` fields. The vendor journey gate
runs it against the published SDK once there is one
(kyzmet-ai/kyzmet-workspace#2796).

## Connector adapter starter

A connector adapter is a local MCP service: a `Handler` from a `Request` to a
`Response`, registered with `Export`. Provider HTTP goes through
`HTTPRequestDo` (text) and `HTTPRequestBytes` (bytes), delegated tools through
`InvokeTool`. Tool arguments arrive as JSON within `MaxInvocationBytes`;
anything else fails as `invalid_arguments` before the handler runs.

```go
package main

import "kyzmet.ai/go/connectoradapter"

func handle(request connectoradapter.Request) (connectoradapter.Response, error) {
	switch request.Tool {
	case "list_data":
		upstream, err := connectoradapter.HTTPRequestDo(
			connectoradapter.NewHTTPRequest("GET", "https://example.com/api/data").
				Header("Accept", "application/json"),
		)
		if err != nil {
			return connectoradapter.Response{}, err
		}
		return connectoradapter.Success(map[string]any{
			"status": upstream.Status,
			"body":   upstream.Body,
		}), nil
	default:
		return connectoradapter.Response{}, connectoradapter.ErrUnknownTool
	}
}

func init() { connectoradapter.Export(handle) }

func main() {}
```

A handler's error crosses the host boundary as a tool-error code: `ErrUnknownTool`
as `not_found`, a `*ToolError` (from `Error(code, message)`) as its code, a
refused host call (`*HostCallError`) as its failure's code, and any other error
as `provider_unavailable`. Messages stay in the guest.

## Regenerating Preview 2 bindings

The canonical generated bindings and restricted WASI world live in
`kyzmet-protocol`. Regenerate them there, then copy the exact outputs into this
module:

```sh
# From kyzmet-protocol:
# From kyzmet-kdk:
scripts/sync-go-preview2-bindings.sh
```

The bindings land in `world/<world>/`. The `app` bindings are the superset of
the `skill` world, so one SDK distribution builds both: a behavior component
selects the `skill` world and exports bridge fetch; an app selecting the `app`
world must additionally implement UI render.
Run `scripts/check-go-preview2-bindings.sh` and `go test ./...` after
regeneration.

## Running tests

Tests run on the host (no WASM runtime required):

```sh
go test ./...
```

## Building Preview 2 components

Declare a `go-component` build unit. KDK selects the WIT world from the typed
executable output role: `app-component` uses `app`, `auth-type` uses
`auth-type`, and `local-mcp-service` uses `local-mcp-service`. It materializes
the exact WIT package embedded in that KDK release; a project must not supply
an ambient WIT directory.

Its declared compiler requirement must name TinyGo's Preview 2 target and the
source inputs must include the vendored module closure:

```yaml
build:
  units:
    - id: behavior
      driver: go-component
      inputs: [go.mod, go.sum, vendor, main.go]
      toolchains:
        - {name: tinygo, target: wasip2}
        - {name: wasm-opt}
        - {name: wasm-tools}
      outputs:
        - {path: dist/module.wasm, package_path: module.wasm, role: app-component, executable: true}
```

The qualified Go builder invokes TinyGo with that restricted WIT package and
disables debug information. TinyGo debug metadata currently embeds build-specific
paths, so `-no-debug` is part of the reproducibility contract:

```sh
tinygo build -no-debug \
  -target=wasip2 \
  -wit-package=<KDK-isolated-WIT-package> \
  -wit-world=<role-selected-world> \
  -o dist/module.wasm .
```

The old `kdk build` connector-adapter inference emitted a Preview 1 core WASM
module. It is deliberately rejected for Go projects rather than producing a
module that cannot satisfy the component mod contract.

The retained source candidate must contain a complete module dependency closure
(for example, a populated `vendor/` directory). Verified compilation runs with
the network disabled and `GOFLAGS=-mod=vendor`; it never resolves a newer module
from the ambient network.
