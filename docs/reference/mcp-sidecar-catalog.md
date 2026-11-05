# MCP sidecar catalog (reference)

Blessed **stdio** and streamable **HTTP** MCP sidecars that Chimera documents for operator and harness configuration. This reference is the operator-facing companion to the platform contract in [`gateway-mcp-tool-backends.md`](../features/gateway-mcp-tool-backends.md).

**Status:** v0.5 Phase 2b — one blessed no-network sidecar (`fake_echo`). Git, shell, and browser entries remain **documented templates only** (default **off**) until a later phase blesses a specific upstream CLI.

---

## Blessed: `fake_echo` (no-network echo)

| Field | Value |
|-------|--------|
| **Server id** | `fake_echo` |
| **Purpose** | End-to-end MCP transport and harness wiring without network or repo access |
| **Transport** | `stdio` |
| **OpenAI slug** | `fake_echo__echo` (tool `echo`) |
| **Tier** | Blessed for dev/test; safe for local stacks |

### Command

Run from the **repository root** (paths match [`config/chimera.example.yaml`](../../config/chimera.example.yaml)):

| Key | Value |
|-----|--------|
| `command` | `go` |
| `args` | `run`, `./chimera/chimera-gateway/internal/mcpclient/fake/cmd` |

Optional flag `-collision-tool` exposes an MCP tool named `read_file` for collision tests only — **do not** enable in production assistant bindings.

### Environment allowlist

Deny-by-default: only these keys are passed to the child (plus their current values from the gateway process environment):

`PATH`, `HOME`, `GOROOT`, `GOPATH`, `GOMOD`, `GOWORK`, `CGO_ENABLED`

Do **not** add broker API keys, cloud tokens, or `CHIMERA_*` secrets to the allowlist.

### Security notes

- **No network**, no filesystem tools beyond echoing the `message` argument in the MCP result.
- **No host command execution** — the binary is built in-tree; operators should not substitute an untrusted MCP CLI under this server id.
- **Deny-by-default tool bind:** the assistant must list `echo: true` under `mcp.assistants[].servers[].tools`; absent entries are not injected.
- **`auto_start`:** default `false` in examples; set `true` only when the gateway should spawn the child at startup (still requires explicit assistant tool allowlist).

### Example YAML

See the live snippet in [`config/chimera.example.yaml`](../../config/chimera.example.yaml) under `mcp.servers` and `mcp.assistants`.

---

## Templates (default off): git, shell, browser

These **Tier B** sidecars are **not** blessed in v0.5 Phase 2b. Keep them **disabled** in config until you select a documented upstream and enable per-tool allows on the bound assistant.

| Server id (template) | Domain | Default |
|------------------------|--------|---------|
| `git_local` | Git / `gh` read-only workflows | `disabled: true` |
| `shell_sandbox` | Sandboxed shell MCP | `disabled: true` |
| `browser_cdp` | Web / browser / CDP MCP | `disabled: true` |

Commented placeholders for all three appear in [`config/chimera.example.yaml`](../../config/chimera.example.yaml). When enabling shell or browser sidecars later, enforce workspace cwd ⊆ roots, **network off by default**, and a per-call **timeout** (see platform contract).

---

## In-repo test server (implementation)

- Package: `chimera/chimera-gateway/internal/mcpclient/fake/`
- Entrypoint: `chimera/chimera-gateway/internal/mcpclient/fake/cmd/main.go`
- Tools: `echo` (always); `read_file` (only with `-collision-tool`)

Integration tests spawn this server via `go run` on the entrypoint path; operators can use the same command in YAML.
