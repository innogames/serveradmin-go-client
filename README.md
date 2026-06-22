# Serveradmin Go Client

A Go client library and CLI tool for interacting with the [InnoGames Serveradmin](https://github.com/innogames/serveradmin) configuration management database system.

## Overview

Serveradmin is a central server database management system used by InnoGames. This Go client provides a convenient way to:

- Query server objects using Serveradmin's query language
- Retrieve server attributes and metadata
- Authenticate using SSH keys or security tokens
- Use as both a library and command-line tool
- Create, modify, and delete server objects with change tracking

## Installation

```bash
go get github.com/innogames/serveradmin-go
```

## Configuration

The client requires configuration to connect to your Serveradmin instance. Create a configuration file or set environment variables:

### Environment Variables

```bash
export SERVERADMIN_BASE_URL="https://your-serveradmin-instance.com"
export SERVERADMIN_TOKEN="your-auth-token"
# or set SERVERADMIN_KEY_PATH to an SSH private key, or have SSH_AUTH_SOCK available
```

These variables are read only by the deprecated package-level functions and by
`adminapi.NewClientFromEnv()`. The recommended `NewClient(Config{...})` path
reads no environment variables.

## Usage

### As a Go Library (recommended: explicit Client)

The recommended entry point is an explicit, per-instance `Client` built with
`NewClient(Config{...})`. A `Client` reads no environment variables, holds its
own `*http.Client`, and is safe for concurrent use — so a single process can
serve several targets with different URLs/credentials at once. Every network
call takes a `context.Context`, giving the caller control over cancellation and
timeouts.

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/innogames/serveradmin-go-client/adminapi"
)

func main() {
    // Construct a client with explicit configuration — no env reads, no globals.
    client, err := adminapi.NewClient(adminapi.Config{
        BaseURL: "https://your-serveradmin-instance.com",
        Token:   "your-token", // or: SSHSigner / KeyPath for SSH auth
        Timeout: 10 * time.Second,
    })
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    // Build a query bound to this client.
    query, err := client.FromQuery("hostname=web*")
    if err != nil {
        panic(err)
    }
    query.SetAttributes("hostname", "intern_ip", "environment")

    // Execute the query.
    servers, err := query.All(ctx)
    if err != nil {
        panic(err)
    }

    for _, server := range servers {
        fmt.Printf("Server: %s (%s)\n", server.GetString("hostname"), server.GetString("intern_ip"))
    }
}
```

Authentication is selected **explicitly** from `Config`, in the order
`SSHSigner` → `KeyPath` → `Token`. Unlike the legacy environment path, an
ambient `SSH_AUTH_SOCK` can never silently override an explicitly configured
token.

For deployments that are still configured entirely through environment
variables (for example the CLI), `adminapi.NewClientFromEnv()` builds a `Client`
from the `SERVERADMIN_*` variables.

#### Typed attribute getters

`Get` returns `any` and converts JSON numbers to `int` (lossy). When you need to
preserve numeric type, use the typed getters: `GetInt`, `GetFloat`, `GetBool`
(alongside the existing `GetString` and `GetMulti`).

### Deprecated: package-level functions and the global env client

The package-level `adminapi.FromQuery` / `NewQuery` / `CallAPI` / `NewObject`
still work: they lazily build a single process-global client from the
`SERVERADMIN_*` environment variables (the historical behavior). They are
**deprecated** in favor of an explicit `Client`, because the global config is
frozen after the first request and cannot serve multiple targets. Note that the
execution methods (`All`, `One`, `Count`, `Commit`) now require a
`context.Context` regardless of which path you use.

### As a CLI Tool

```bash
# Query servers with hostname starting with "web"
./serveradmin-go "hostname=web*" -a "hostname,ip,environment"

# Get exactly one server (fails if multiple matches)
./serveradmin-go "hostname=webserver01" -a "hostname,ip" -one

# Order results by specific attribute
./serveradmin-go "environment=production" -a "hostname,ip" -order "hostname"
```

## Query Language

The client supports Serveradmin's query language for filtering servers:

- **Exact match**: `hostname=webserver01`
- **Pattern matching**: `hostname=web*`
- **Multiple conditions**: `environment=production AND datacenter=fra1`
- **Attribute comparison**: `memory>8192`

## Authentication

### SSH Key Authentication (Recommended)

```go
// Explicit client: provide a pre-built signer or a key file path.
client, _ := adminapi.NewClient(adminapi.Config{
    BaseURL: "https://your-serveradmin-instance.com",
    KeyPath: "/path/to/id_ed25519", // or SSHSigner: <ssh.Signer>
})

// Env path (deprecated): SERVERADMIN_KEY_PATH, or a running SSH agent via SSH_AUTH_SOCK.
```

### Security Token Authentication

```go
// Explicit client.
client, _ := adminapi.NewClient(adminapi.Config{
    BaseURL: "https://your-serveradmin-instance.com",
    Token:   "your-token",
})

// Env path (deprecated): set SERVERADMIN_TOKEN.
```

## Examples

### Creating a New Server

```go
// NewObject fetches defaults, applies attributes, commits, and re-queries to
// populate object_id — all bound to the client and the provided context.
newServer, err := client.NewObject(ctx, "vm", adminapi.Attributes{
    "hostname":    "newwebserver",
    "environment": "staging",
})
if err != nil {
    panic(err)
}
fmt.Printf("Created %s (object_id %d)\n", newServer.GetString("hostname"), newServer.ObjectID())
```

### Modifying Existing Servers

```go
// Find and modify a server.
query, _ := client.FromQuery("hostname=webserver01")
server, err := query.One(ctx)
if err != nil {
    panic(err)
}

// Update attributes.
server.Set("backup_disabled", true)

// Commit changes.
if _, err := server.Commit(ctx); err != nil {
    panic(err)
}
```

### Calling API Functions

```go
// Call a remote API function by group and function name.
result, err := client.CallAPI(ctx, "ip", "get_free", map[string]any{"network": "internal"})
if err != nil {
    panic(err)
}
fmt.Printf("Free IP: %s\n", result)
```

## Building

```bash
# Build the CLI tool
make build

# Run tests
make test

# Run tests with coverage
make coverage
```

## Requirements

- Go 1.24 or later
- Access to a Serveradmin instance
- SSH private key or security token for authentication

## Related Links

- [InnoGames Serveradmin](https://github.com/innogames/serveradmin) - The main Serveradmin system
- [Serveradmin Documentation](https://serveradmin.readthedocs.io/) - Official documentation
- [FOSDEM 19 Talk](https://fosdem.org/2019/schedule/event/serveradmin/) - Deep dive into how InnoGames works with Serveradmin
