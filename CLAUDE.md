# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go client library and CLI tool for InnoGames Serveradmin, a configuration management database system. The library provides both string-based and programmatic query interfaces with support for complex filters, and handles create/update/delete operations with change tracking.

## Development Commands

```bash
# Build the CLI tool
make build

# Run all tests
make test

# Run tests with race detector
make test-race

# Generate test coverage report (creates coverage.html)
make test-coverage

# Run linter with auto-fix
make linter

# Run a specific test
go test -run TestParseQuery ./adminapi

# Run a specific benchmark
go test -bench BenchmarkParseQuery_Simple ./adminapi
```

## Architecture Overview

### Package Structure

**adminapi/** - Core library package containing all API client functionality:
- `query.go` - Query building (`Query`, `FromQuery`, `NewQuery`)
- `parse.go` - Query string parser converting "hostname=web*" to Filters
- `filters.go` - Filter functions (Regexp, Any, All, Not, Empty)
- `server_object.go` - ServerObject with change tracking and state management
- `commit.go` - Commit/rollback operations for objects
- `transport.go` - HTTP client with SSH and token authentication
- `config.go` - Configuration loading from environment variables

**examples/** - Standalone example programs demonstrating library usage

### Core Data Flow

1. **Query Creation** → 2. **Fetch** → 3. **Modify** → 4. **Commit**

```
FromQuery("hostname=web*") or NewQuery(Filters{...})
  ↓
Query.All() / Query.One()  [transport.go sends HTTP request]
  ↓
ServerObjects with attributes loaded
  ↓
ServerObject.Set(key, value)  [tracks oldValues internally]
  ↓
ServerObject.Commit() or ServerObjects.Commit()  [sends delta to API]
```

### Key Architectural Patterns

**Change Tracking**: `ServerObject` maintains an `oldValues` map that records original attribute values on first modification. The `serializeChanges()` method computes deltas, sending only modified fields to the API. This mimics the Python client's behavior.

**Multi-attributes**: Slice-valued attributes use set semantics during commit, computing `add` and `remove` sets rather than replacing the entire slice. See `sliceDiff()` in `server_object.go`.

**State Machine**: ServerObject has four states returned by `CommitState()`:
- `"created"` - object_id is nil (new object not yet committed)
- `"deleted"` - marked for deletion
- `"changed"` - has modifications in oldValues
- `"consistent"` - no pending changes

**Authentication**: The client supports two auth methods (checked in order):
1. SSH key signing (via `SERVERADMIN_KEY_PATH` or `SSH_AUTH_SOCK` agent)
2. Security token (via `SERVERADMIN_TOKEN` with HMAC-SHA1 signing)

Configuration is loaded once via `sync.OnceValues` in `config.go`.

**Filter System**: Two ways to build queries:
1. String-based: `FromQuery("hostname=regexp(web.*) environment=production")`
2. Programmatic: `NewQuery(Filters{"hostname": Regexp("web.*")})`

The parser (`parse.go`) handles nested parentheses and converts function names case-insensitively (e.g., "ReGEXP" → "Regexp").

## Important Implementation Details

### Query Interface

Both `FromQuery` and `NewQuery` return a `Query` struct. Key methods:
- `SetAttributes([]string)` or `SetAttributes(...string)` - specify which attributes to fetch
- `AddFilter(key, value)` - add filters incrementally to existing Query
- `All()` → `ServerObjects` - fetch all matching objects
- `One()` → `*ServerObject` - fetch exactly one (errors if 0 or >1 results)

### ServerObject Methods

- `Get(attr)` returns `any` (auto-converts JSON float64 to int)
- `GetString(attr)` returns `string`
- `Set(key, value)` tracks changes; returns error if attribute doesn't exist
- `Delete()` marks for deletion (doesn't actually delete until commit)
- `Rollback()` discards all local changes
- `Commit()` sends changes to API and clears oldValues on success

### Filter Functions

Implemented in `filters.go`:
- `Regexp(pattern string)` - regex matching
- `Not(value)` - negation (works with values or other filters)
- `Any(values...)` - OR semantics (match any of)
- `All(values...)` - AND semantics (match all of)
- `Empty()` - checks for empty/nil values

These can be nested: `Not(Any(Regexp("^test.*"), Regexp("^dev.*")))`

Additional filters exist in the parser's `allFilters` map (GreaterThan, LessThan, etc.) but lack Go helper functions. These can still be used via `FromQuery` string syntax.

### Testing Patterns

Tests use testify/assert and testify/require. The codebase has table-driven tests (see `parse_test.go`).

When writing tests:
- Use `require.NoError` for setup that must succeed
- Use `assert.Error` for expected failures with descriptive messages
- Table-driven tests should have descriptive `name` fields
- Go 1.25+ uses `b.Loop()` instead of `for i := 0; i < b.N; i++` in benchmarks

### Linting

The project uses golangci-lint with an extensive linter configuration (`.golangci.yml`). Key points:
- Formatters gci and gofumpt are enabled (imports grouped, strict formatting)
- Some linters (errcheck, perfsprint) are relaxed for `_test.go` files
- Examples directory has relaxed rules
- Run `make linter` to auto-fix issues before committing
- SHA1 usage is intentional (required by protocol) - use `//nolint:gosec` comments

## Configuration Requirements

The client requires these environment variables:

```bash
# Required
export SERVERADMIN_BASE_URL="https://serveradmin.example.com"

# One of these auth methods:
export SERVERADMIN_TOKEN="your-token"           # Token-based auth
# OR
export SERVERADMIN_KEY_PATH="/path/to/key"      # SSH key file
# OR
export SSH_AUTH_SOCK="/path/to/ssh-agent.sock"  # SSH agent (auto-detected)
```

The client fails fast if `SERVERADMIN_BASE_URL` or auth credentials are missing.

## Examples Directory

The `examples/` directory contains standalone programs demonstrating:
- `update_example.go` - Single/batch updates, create, delete, rollback
- `query_example.go` - Query patterns (string vs programmatic, simple vs nested filters)

These use a shorter import alias pattern: `import api "github.com/innogames/serveradmin-go-client/adminapi"`

Examples are excluded from strict linting rules and can ignore errors for brevity.

## Version Compatibility

- Requires Go 1.24+ (per README, using latest Go 1.25 features like `b.Loop()`)
- API version is hardcoded in `config.go` as `version = "4.9.0"`
- The client maintains compatibility with the Python Serveradmin client's behavior (change tracking, JSON comparison logic)
