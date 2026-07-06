# dev-connect API Documentation

Status: Draft for Phase 10 review

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
dev-connect help
```

## Global Flags

```text
--config <path>
--context <name>
--cluster <name>
--gateway <name>
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
  "status": "Connected",
  "server": "dev01",
  "sessionId": "example-session-id",
  "localPort": 55221
}
```

## Stability Policy

- `apiVersion` is required.
- Additive fields may be introduced within the same version.
- Breaking changes require a new API version.
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
- optional `vscode`.

Unsupported schema versions shall be rejected before external processes are started.

## Session State API

Session state is JSON and machine-generated.

Session state is not a public stable API unless explicitly documented in a later phase.

Session state shall not contain credentials.
