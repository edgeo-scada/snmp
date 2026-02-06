# SNMP Driver

[![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)](./changelog)
[![Go](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](https://github.com/edgeo-scada/snmp/blob/main/LICENSE)

A complete implementation of the SNMP (Simple Network Management Protocol) protocol in Go, supporting versions v1, v2c, and v3.

## Installation

```bash
go get github.com/edgeo-scada/snmp
```

## Features

- **Multi-version support**: SNMPv1, SNMPv2c, and SNMPv3
- **Complete operations**: GET, GET-NEXT, GET-BULK, SET, WALK
- **Trap reception**: Listener for SNMP notifications
- **Connection pool**: Optimized connection management
- **SNMPv3 Security**: AuthNoPriv and AuthPriv (MD5/SHA, DES/AES)
- **Built-in metrics**: Counters, gauges, and latency histograms
- **BER encoding**: Complete ASN.1/BER implementation
- **Full CLI**: `edgeo-snmp` command-line tool

## Quick Example

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

    // Retrieve system information
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

## Package Structure

```
snmp/
├── snmp/
│   ├── client.go      # Main SNMP client
│   ├── pool.go        # Connection pool
│   ├── trap.go        # Trap listener
│   ├── protocol.go    # BER encoding/decoding
│   ├── packets.go     # PDU structures and messages
│   ├── types.go       # Types and OIDs
│   ├── options.go     # Configuration options
│   ├── errors.go      # Error handling
│   ├── metrics.go     # Metrics
│   └── version.go     # SNMP versions
└── cmd/
    └── edgeo-snmp/    # CLI
```

## Documentation

- [Getting Started](getting-started.md)
- [SNMP Client](client.md)
- [Configuration Options](options.md)
- [Error Handling](errors.md)
- [Metrics](metrics.md)
- [Connection Pool](pool.md)
- [Trap Listener](trap-listener.md)
- [Changelog](changelog.md)

## Examples

- [Basic Client](examples/basic-client.md)
- [Trap Listener](examples/trap-listener.md)

## CLI

The `edgeo-snmp` tool provides a full command-line interface:

```bash
# System information
edgeo-snmp info -t 192.168.1.1

# Simple GET
edgeo-snmp get -t 192.168.1.1 1.3.6.1.2.1.1.1.0

# Walk a subtree
edgeo-snmp walk -t 192.168.1.1 1.3.6.1.2.1.2.2

# SET a value
edgeo-snmp set -t 192.168.1.1 1.3.6.1.2.1.1.4.0 s "admin@example.com"

# Listen for traps
edgeo-snmp trap-listen --listen ":1162"
```

## Supported SNMP Versions

| Version | Authentication | Encryption | Bulk Operations |
|---------|---------------|------------|-----------------|
| v1      | Community     | No         | No              |
| v2c     | Community     | No         | Yes             |
| v3      | USM (MD5/SHA) | DES/AES   | Yes             |

## License

Apache License 2.0 - see [LICENSE](https://github.com/edgeo-scada/snmp/blob/main/LICENSE) for details.
