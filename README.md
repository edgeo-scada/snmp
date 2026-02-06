# SNMP Driver

A comprehensive SNMP (Simple Network Management Protocol) client library and CLI tool written in Go, supporting SNMPv1, SNMPv2c, and SNMPv3.

## Features

### Library (`snmp/`)

- Full SNMP protocol implementation (v1, v2c, v3)
- All standard operations: GET, GET-NEXT, GET-BULK, SET, WALK
- SNMPv3 security: USM with AuthNoPriv and AuthPriv (MD5/SHA, DES/AES)
- Trap listener for receiving SNMP notifications
- Connection pooling for high-throughput applications
- Complete ASN.1/BER encoding and decoding
- Metrics collection and monitoring
- Structured logging with Go's `slog` package

### CLI (`edgeo-snmp`)

- GET, SET, WALK, and BULK operations
- Trap listener mode
- SNMPv1/v2c/v3 support
- Multiple output formats (table, JSON, CSV, raw)
- Device information retrieval
- Configuration file support

## Installation

### CLI Tool

```bash
go install github.com/edgeo-scada/snmp/cmd/edgeo-snmp@latest
```

### Library

```bash
go get github.com/edgeo-scada/snmp
```

## Quick Start

### CLI Examples

```bash
# Get system information
edgeo-snmp info -t 192.168.1.1

# GET a single OID
edgeo-snmp get -t 192.168.1.1 1.3.6.1.2.1.1.1.0

# Walk a subtree
edgeo-snmp walk -t 192.168.1.1 1.3.6.1.2.1.2.2

# SET a value
edgeo-snmp set -t 192.168.1.1 1.3.6.1.2.1.1.4.0 s "admin@example.com"

# Listen for traps
edgeo-snmp trap-listen --listen ":1162"

# SNMPv3 GET with authentication
edgeo-snmp get -t 192.168.1.1 -V 3 -u admin -a SHA -A "authpass" -x AES -X "privpass" --security-level authPriv 1.3.6.1.2.1.1.1.0

# JSON output
edgeo-snmp get -t 192.168.1.1 -o json 1.3.6.1.2.1.1.1.0
```

### Library Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/edgeo-scada/snmp/snmp"
)

func main() {
    ctx := context.Background()

    // Create an SNMPv2c client
    client, err := snmp.NewClient(ctx,
        snmp.WithTarget("192.168.1.1:161"),
        snmp.WithVersion(snmp.Version2c),
        snmp.WithCommunity("public"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Get system information
    vars, err := client.Get(ctx,
        snmp.OIDSysDescr,
        snmp.OIDSysName,
        snmp.OIDSysUpTime,
    )
    if err != nil {
        log.Fatal(err)
    }

    for _, v := range vars {
        fmt.Printf("%s = %v\n", v.OID, v.Value)
    }
}
```

## CLI Reference

### Global Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--target` | `-t` | SNMP agent address (required) | |
| `--port` | `-p` | SNMP agent port | `161` |
| `--community` | `-c` | Community string (v1/v2c) | `public` |
| `--version` | `-V` | SNMP version (1, 2c, 3) | `2c` |
| `--timeout` | | Request timeout | `5s` |
| `--retries` | `-r` | Number of retries | `3` |
| `--output` | `-o` | Output format: table, json, csv, raw | `table` |
| `--verbose` | `-v` | Verbose output | `false` |
| `--no-color` | | Disable colored output | `false` |
| `--numeric` | | Print OIDs numerically | `false` |
| `--config` | | Config file path | `$HOME/.edgeo-snmp.yaml` |

### SNMPv3 Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--security-level` | | Security level (noAuthNoPriv, authNoPriv, authPriv) | `noAuthNoPriv` |
| `--security-name` | `-u` | Security name (username) | |
| `--auth-protocol` | `-a` | Auth protocol (MD5, SHA, SHA-224, SHA-256, SHA-384, SHA-512) | |
| `--auth-passphrase` | `-A` | Auth passphrase | |
| `--priv-protocol` | `-x` | Privacy protocol (DES, AES, AES-192, AES-256) | |
| `--priv-passphrase` | `-X` | Privacy passphrase | |
| `--context` | `-n` | Context name | |

### Commands

#### GET Command

```bash
# GET a single OID
edgeo-snmp get -t 192.168.1.1 1.3.6.1.2.1.1.1.0

# GET multiple OIDs
edgeo-snmp get -t 192.168.1.1 1.3.6.1.2.1.1.1.0 1.3.6.1.2.1.1.3.0 1.3.6.1.2.1.1.5.0
```

#### SET Command

```bash
# Set a string value
edgeo-snmp set -t 192.168.1.1 1.3.6.1.2.1.1.4.0 s "admin@example.com"

# Set an integer value
edgeo-snmp set -t 192.168.1.1 1.3.6.1.2.1.1.7.0 i 72
```

#### WALK Command

```bash
# Walk interface table
edgeo-snmp walk -t 192.168.1.1 1.3.6.1.2.1.2.2

# Walk with bulk requests
edgeo-snmp walk -t 192.168.1.1 --bulk 1.3.6.1.2.1.2.2
```

#### Trap Listener

```bash
# Listen for traps on default port
edgeo-snmp trap-listen

# Listen on a specific port
edgeo-snmp trap-listen --listen ":1162"
```

#### Info Command

```bash
# Get device system information
edgeo-snmp info -t 192.168.1.1

# JSON output
edgeo-snmp info -t 192.168.1.1 -o json
```

#### Version Command

```bash
edgeo-snmp version
```

## Supported SNMP Versions

| Version | Authentication | Encryption | Bulk Operations |
|---------|---------------|------------|-----------------|
| v1      | Community     | No         | No              |
| v2c     | Community     | No         | Yes             |
| v3      | USM (MD5/SHA) | DES/AES    | Yes             |

## Configuration

Create `~/.edgeo-snmp.yaml`:

```yaml
target: 192.168.1.1
port: 161
community: public
version: 2c
timeout: 5s
retries: 3
output: table
verbose: false
no-color: false
numeric: false

# SNMPv3 settings
security-level: authPriv
security-name: admin
auth-protocol: SHA
auth-passphrase: authpass
priv-protocol: AES
priv-passphrase: privpass
```

## Project Structure

```
snmp/
├── cmd/
│   └── edgeo-snmp/        # CLI application
│       ├── main.go
│       ├── root.go         # Root command and global flags
│       ├── get.go          # GET command
│       ├── set.go          # SET command
│       ├── walk.go         # WALK command
│       ├── trap.go         # Trap listener command
│       ├── info.go         # Device information
│       ├── output.go       # Output formatting
│       ├── common.go       # Shared utilities
│       └── version.go      # Version command
├── snmp/                   # SNMP library (importable)
│   ├── client.go           # Main client implementation
│   ├── pool.go             # Connection pooling
│   ├── trap.go             # Trap listener
│   ├── protocol.go         # BER encoding/decoding
│   ├── packets.go          # PDU structures and messages
│   ├── types.go            # Types and OIDs
│   ├── options.go          # Client options
│   ├── errors.go           # Error types
│   ├── metrics.go          # Metrics collection
│   └── version.go          # Version information
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Building from Source

```bash
# Clone the repository
git clone https://github.com/edgeo-scada/snmp.git
cd snmp

# Build the CLI
make build

# Build for all platforms
make build-all

# Run tests
make test
```

## Documentation

- [Getting Started](docs/snmp/getting-started.md) - Installation and first steps
- [Client API](docs/snmp/client.md) - Client operations reference
- [Configuration Options](docs/snmp/options.md) - All configuration options
- [Connection Pool](docs/snmp/pool.md) - Connection pooling
- [Trap Listener](docs/snmp/trap-listener.md) - Receiving SNMP notifications
- [Error Handling](docs/snmp/errors.md) - Error types and handling patterns
- [Metrics](docs/snmp/metrics.md) - Metrics collection and monitoring
- [CLI Reference](docs/snmp/cli.md) - Command-line tool documentation
- [Changelog](docs/snmp/changelog.md) - Version history

## License

MIT License
