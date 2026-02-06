# Getting Started

This guide will help you get started quickly with the SNMP library.

## Installation

```bash
go get github.com/edgeo-scada/snmp
```

## Basic Concepts

### OID (Object Identifier)

OIDs identify objects in the MIB (Management Information Base):

```go
// Predefined OIDs for system information
snmp.OIDSysDescr     // 1.3.6.1.2.1.1.1.0 - System description
snmp.OIDSysObjectID  // 1.3.6.1.2.1.1.2.0 - System object ID
snmp.OIDSysUpTime    // 1.3.6.1.2.1.1.3.0 - System uptime
snmp.OIDSysContact   // 1.3.6.1.2.1.1.4.0 - Contact
snmp.OIDSysName      // 1.3.6.1.2.1.1.5.0 - System name
snmp.OIDSysLocation  // 1.3.6.1.2.1.1.6.0 - Location

// Parse a custom OID
oid, err := snmp.ParseOID("1.3.6.1.4.1.9.2.1.55.0")
```

### SNMP Versions

```go
snmp.Version1   // SNMPv1 - community-based authentication
snmp.Version2c  // SNMPv2c - community + bulk operations
snmp.Version3   // SNMPv3 - USM authentication + encryption
```

## SNMPv1/v2c Client

### Simple Connection

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
        snmp.WithTimeout(5*time.Second),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Use the client...
}
```

### GET Operation

```go
// Simple GET
vars, err := client.Get(ctx, snmp.OIDSysDescr)
if err != nil {
    log.Fatal(err)
}

for _, v := range vars {
    fmt.Printf("%s = %v\n", v.OID, v.Value)
}

// Multiple GET
vars, err := client.Get(ctx,
    snmp.OIDSysDescr,
    snmp.OIDSysName,
    snmp.OIDSysUpTime,
)
```

### GET-NEXT Operation

```go
// GET-NEXT returns the next OID in the MIB
vars, err := client.GetNext(ctx, snmp.OIDSysDescr)
if err != nil {
    log.Fatal(err)
}

// vars[0].OID will be 1.3.6.1.2.1.1.2.0 (sysObjectID)
```

### GET-BULK Operation (v2c/v3 only)

```go
// GET-BULK to retrieve multiple values efficiently
vars, err := client.GetBulk(ctx, 0, 10, oid)
if err != nil {
    log.Fatal(err)
}
```

### WALK Operation

```go
// Walk a subtree
rootOID, _ := snmp.ParseOID("1.3.6.1.2.1.2.2") // ifTable

err := client.WalkFunc(ctx, rootOID, func(v snmp.Variable) error {
    fmt.Printf("%s = %v\n", v.OID, v.Value)
    return nil
})
```

### SET Operation

```go
// SET to modify a value
vars := []snmp.Variable{
    {
        OID:   snmp.OIDSysContact,
        Type:  snmp.TypeOctetString,
        Value: []byte("admin@example.com"),
    },
}

result, err := client.Set(ctx, vars...)
if err != nil {
    log.Fatal(err)
}
```

## SNMPv3 Client

### AuthNoPriv Authentication

```go
client, err := snmp.NewClient(ctx,
    snmp.WithTarget("192.168.1.1:161"),
    snmp.WithVersion(snmp.Version3),
    snmp.WithSecurityName("admin"),
    snmp.WithAuthProtocol(snmp.AuthSHA),
    snmp.WithAuthPassword("authpassword"),
    snmp.WithSecurityLevel(snmp.AuthNoPriv),
)
```

### AuthPriv Authentication

```go
client, err := snmp.NewClient(ctx,
    snmp.WithTarget("192.168.1.1:161"),
    snmp.WithVersion(snmp.Version3),
    snmp.WithSecurityName("admin"),
    snmp.WithAuthProtocol(snmp.AuthSHA),
    snmp.WithAuthPassword("authpassword"),
    snmp.WithPrivProtocol(snmp.PrivAES),
    snmp.WithPrivPassword("privpassword"),
    snmp.WithSecurityLevel(snmp.AuthPriv),
)
```

## Trap Reception

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/edgeo-scada/snmp/snmp"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Trap handler
    handler := func(trap *snmp.TrapPDU) {
        fmt.Printf("Trap received from %s\n", trap.AgentAddress)
        fmt.Printf("  Type: %s\n", trap.Type)
        for _, v := range trap.Variables {
            fmt.Printf("  %s = %v\n", v.OID, v.Value)
        }
    }

    // Create and start the listener
    listener := snmp.NewTrapListener(handler,
        snmp.WithListenAddress(":1162"),
        snmp.WithTrapCommunity("public"),
    )

    if err := listener.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // Wait for interrupt
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh

    listener.Stop()
}
```

## Connection Pool

```go
pool, err := snmp.NewPool(ctx,
    snmp.WithPoolSize(10),
    snmp.WithPoolTarget("192.168.1.1:161"),
    snmp.WithPoolVersion(snmp.Version2c),
    snmp.WithPoolCommunity("public"),
)
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

// Acquire a client from the pool
client, err := pool.Acquire(ctx)
if err != nil {
    log.Fatal(err)
}
defer pool.Release(client)

// Use the client...
vars, err := client.Get(ctx, snmp.OIDSysDescr)
```

## Error Handling

```go
vars, err := client.Get(ctx, oid)
if err != nil {
    var snmpErr *snmp.SNMPError
    if errors.As(err, &snmpErr) {
        switch snmpErr.Status {
        case snmp.ErrNoSuchName:
            fmt.Println("OID not found")
        case snmp.ErrTooBig:
            fmt.Println("Response too large")
        case snmp.ErrGenErr:
            fmt.Println("General error")
        default:
            fmt.Printf("SNMP error: %s\n", snmpErr)
        }
    } else {
        fmt.Printf("Network error: %v\n", err)
    }
}
```

## Next Steps

- [SNMP Client](client.md) - Complete client documentation
- [Configuration Options](options.md) - All available options
- [Connection Pool](pool.md) - Advanced connection management
- [Trap Listener](trap-listener.md) - Receiving notifications
