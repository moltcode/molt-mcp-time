# molt-mcp-time

A tiny time & timezone [MCP](https://modelcontextprotocol.io) server written in
Go with zero dependencies. It is the reference **self-contained plugin** for the
Molt plugin system: prebuilt static binaries for every supported platform ship
inside the package, there are no post-install steps, and all configuration is
declared up front in `package.json`.

## Tools

| Tool | Description |
| --- | --- |
| `current_time` | Current date/time, optionally in a given IANA timezone |
| `unix_timestamp` | Current Unix timestamp (seconds and milliseconds) |
| `add_duration` | Add a Go-style duration (`2h45m`, `-30m`) to now or a given RFC3339 timestamp |

## Parameters

| Key | Kind | Description |
| --- | --- | --- |
| `MOLT_TIME_DEFAULT_TZ` | env | IANA timezone used when a tool call omits one |

## Package layout

The published tarball is npm-layout (`package/` root) and carries a `molt`
section in `package.json` describing what the plugin contributes — here a
single stdio MCP server whose `command` is resolved per platform:

```
package/
  package.json        # manifest, incl. molt.contributes.mcpServers
  dist/<platform>/molt-mcp-time[.exe]
```

Supported platforms: `darwin-arm64`, `darwin-x64`, `linux-x64`, `linux-arm64`,
`win32-x64`.

## Building

```bash
make dist   # cross-builds all platforms and produces molt-mcp-time-<version>.tgz
```

## Running manually

The server speaks newline-delimited JSON-RPC 2.0 on stdio:

```bash
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./dist/darwin-arm64/molt-mcp-time
```
