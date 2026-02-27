# Serveradmin Go Client

A Go client library and CLI tool for interacting with the [InnoGames Serveradmin](https://github.com/innogames/serveradmin) configuration management database system.

## Overview

Serveradmin is a central server database management system used by InnoGames. This Go client provides a convenient way to:

- Query server objects using Serveradmin's query language
- Retrieve server attributes and metadata
- Authenticate using SSH keys or security tokens
- Use as both a library and command-line tool
- Soon: Create and modify server objects

## Installation

```bash
go get github.com/innogames/serveradmin-go
```

## Configuration

The client requires configuration to connect to your Serveradmin instance. Create a configuration file or set environment variables:

### Environment Variables

```bash
export SERVERADMIN_BASE_URL="https://your-serveradmin-instance.com"
export SERVERADMIN_AUTH_TOKEN="your-auth-token"
or have a SSH_AUTH_SOCKET available
```

## Usage

### As a Go Library

```go
package main

import (
    "fmt"
    "github.com/innogames/serveradmin-go-client/adminapi"
)

func main() {
    // Create a query
    query, err := adminapi.FromQuery("hostname=web*")
    if err != nil {
        panic(err)
    }

    // Set attributes to retrieve
    query.SetAttributes([]string{"hostname", "ip", "environment"})

    // Execute query
    servers, err := query.All()
    if err != nil {
        panic(err)
    }

    // Process results
    for _, server := range servers {
        hostname := server.Get("hostname")
        ip := server.Get("ip")
        fmt.Printf("Server: %s (%s)\n", hostname, ip)
    }
}
```

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
// The client will automatically use SSH keys from:
// - SSH agent
// - ~/.ssh/id_rsa (or other default keys)
// - Path specified in SERVERADMIN_SSH_KEY_PATH
```

### Security Token Authentication

```go
// Set SERVERADMIN_AUTH_TOKEN environment variable
// or configure in your config file
```

## Examples

### Creating a New Server

```go
// Create a new VM server
newServer, err := adminapi.NewServer("vm")
if err != nil {
    panic(err)
}

// Set attributes
newServer.Set("hostname", "newwebserver")
newServer.Set("environment", "staging")
newServer.Set("ip", "192.168.1.100")

// Commit to Serveradmin
err = newServer.Commit()
```

### Modifying Existing Servers

```go
// Find and modify a server
query, _ := adminapi.FromQuery("hostname=webserver01")
server := query.One()

// Update attributes
server.Set("backup_disabled", "true")
server.Set("maintenance_mode", "true")

// Commit changes
server.Commit()
```

### Calling API Functions

```go
// Call a remote API function by group and function name
result, err := adminapi.CallAPI("ip", "get_free", map[string]any{"network": "internal"})
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
