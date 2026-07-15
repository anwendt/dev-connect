# dev-connect API Documentation

Status: Approved

## Scope

The first public API surface is the CLI command interface and JSON command output schema.

The Go package API is not public for the initial implementation.

## CLI Commands

```text
dev-connect connect <target>
dev-connect disconnect
dev-connect status
dev-connect list
dev-connect version
dev-connect config location
dev-connect config validate
dev-connect help
```

## Global Flags

```text
--config <path>
--context <name>
--cluster <name>
--gateway <name>
--kubectl-path <path>
--log-level <error|warn|info|debug>
--log-format <text|json>
--output <text|json>
--no-code
--no-reconnect
--timeout <duration>
```

`--output` controls user-visible command responses.

`--log-format` controls internal log formatting.

## JSON Output Contract

JSON output is a public and versioned API.

Every JSON response shall include:

```json
{
  "apiVersion": "v1"
}
```

Example status response:

```json
{
  "apiVersion": "v1",
  "status": "connected",
  "server": "dev01",
  "sessionId": "example-session-id",
  "localPort": 55221,
  "kubernetesContext": "rancher-dev",
  "namespace": "dev-connect",
  "gateway": "dev01",
  "reconnect": false,
  "uptime": "1m0s"
}
```

Example list response:

```json
{
  "apiVersion": "v1",
  "status": "ok",
  "targets": [
    {
      "name": "dev01",
      "gateway": "dev01",
      "user": "developer"
    }
  ],
  "clusters": [
    {
      "name": "dev",
      "kubernetesContext": "rancher-dev"
    }
  ],
  "gateways": [
    {
      "name": "dev01",
      "namespace": "dev-connect",
      "service": "dev-connect-gateway-dev01",
      "port": 22
    }
  ],
  "defaultContext": "platform-dev",
  "defaultGateway": "dev01"
}
```

Example version response:

```json
{
  "apiVersion": "v1",
  "status": "ok",
  "version": "dev",
  "commit": "unknown",
  "buildDate": "unknown",
  "goVersion": "go1.26",
  "os": "windows",
  "arch": "amd64"
}
```

## Stability Policy

- `apiVersion` is required.
- Additive fields may be introduced within the same version.
- Status values are lowercase identifiers such as `prepared`, `connected`, `stale`, `disconnected`, `valid`, and `ok`.
- Automation should ignore unknown fields.

## JSON Schemas

JSON schemas shall be generated automatically from the released implementation.

Schemas shall be versioned, for example:

```text
schemas/v1/
```

Changes to existing schemas shall follow semantic versioning. Breaking schema changes require a new major schema version.

## Configuration API

Configuration files use YAML.

Expected top-level fields:

- `apiVersion`,
- `kind`,
- `contexts`,
- `clusters`,
- `gateways`,
- `targets`,
- `hostKeys`,
- optional `ssh`,
- optional `vscode`.

Unsupported schema versions shall be rejected before external processes are started.

## Session State API

Session state is JSON and machine-generated.

Session state is not a public stable API unless explicitly documented in a later phase.

Session state shall not contain credentials.
