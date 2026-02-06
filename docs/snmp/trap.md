# Trap Listener

The SNMP library includes a trap listener for receiving SNMP trap notifications.

## Overview

SNMP traps are asynchronous notifications sent by devices to alert management systems of events such as:
- Interface up/down
- Authentication failures
- Temperature thresholds
- Device reboots
- Custom application events

## Creating a Trap Listener

```go
package main

import (
    "context"
    "log"

    "github.com/edgeo/drivers/snmp/snmp"
)

func main() {
    // Define trap handler
    handler := func(trap *snmp.TrapPDU) {
        log.Printf("Trap from %s:", trap.AgentAddress)
        log.Printf("  Type: %s", trap.Type)
        log.Printf("  Enterprise: %s", trap.Enterprise)
        log.Printf("  Uptime: %d", trap.Uptime)

        for _, vb := range trap.VarBinds {
            log.Printf("  %s = %v", vb.OID, vb.Value)
        }
    }

    // Create listener
    listener := snmp.NewTrapListener(handler,
        snmp.WithListenAddress(":162"),
    )

    // Start listening
    ctx := context.Background()
    if err := listener.Start(ctx); err != nil {
        log.Fatal(err)
    }

    log.Println("Trap listener started on :162")

    // Wait for interrupt
    select {}
}
```

## Trap Listener Options

### WithListenAddress

Sets the address to listen on.

```go
snmp.WithListenAddress(":162")     // All interfaces, port 162
snmp.WithListenAddress("0.0.0.0:1162") // Non-privileged port
```

**Note:** Port 162 requires root/administrator privileges.

### WithTrapCommunity

Filters traps by community string (SNMPv1/v2c).

```go
snmp.WithTrapCommunity("public") // Only accept traps with this community
```

Empty string (default) accepts all communities.

### WithTrapLogger

Sets a custom logger.

```go
snmp.WithTrapLogger(logger)
```

## Trap PDU Structure

```go
type TrapPDU struct {
    // Source information
    AgentAddress net.IP
    SourcePort   int

    // Trap identification
    Type       TrapType
    Enterprise string    // SNMPv1: Enterprise OID
    Generic    int       // SNMPv1: Generic trap type
    Specific   int       // SNMPv1: Specific trap code
    Uptime     uint32    // System uptime

    // SNMP version info
    Version   Version
    Community string    // v1/v2c community

    // Variable bindings
    VarBinds []VarBind
}
```

## Trap Types

### SNMPv1 Generic Traps

| Type | Value | Description |
|------|-------|-------------|
| `TrapColdStart` | 0 | Device restarted (cold) |
| `TrapWarmStart` | 1 | Device restarted (warm) |
| `TrapLinkDown` | 2 | Interface went down |
| `TrapLinkUp` | 3 | Interface came up |
| `TrapAuthFailure` | 4 | Authentication failure |
| `TrapEGPNeighborLoss` | 5 | EGP neighbor lost |
| `TrapEnterpriseSpecific` | 6 | Enterprise-specific trap |

### SNMPv2c/v3 Notifications

SNMPv2c uses notification OIDs:

| OID | Description |
|-----|-------------|
| 1.3.6.1.6.3.1.1.5.1 | coldStart |
| 1.3.6.1.6.3.1.1.5.2 | warmStart |
| 1.3.6.1.6.3.1.1.5.3 | linkDown |
| 1.3.6.1.6.3.1.1.5.4 | linkUp |
| 1.3.6.1.6.3.1.1.5.5 | authenticationFailure |

## Processing Traps

### Basic Processing

```go
handler := func(trap *snmp.TrapPDU) {
    switch trap.Type {
    case snmp.TrapLinkDown:
        log.Printf("ALERT: Interface down on %s", trap.AgentAddress)
    case snmp.TrapLinkUp:
        log.Printf("INFO: Interface up on %s", trap.AgentAddress)
    case snmp.TrapAuthFailure:
        log.Printf("SECURITY: Auth failure from %s", trap.AgentAddress)
    default:
        log.Printf("Trap: %s from %s", trap.Type, trap.AgentAddress)
    }
}
```

### Extract Variable Bindings

```go
handler := func(trap *snmp.TrapPDU) {
    // Find specific OID
    for _, vb := range trap.VarBinds {
        switch vb.OID {
        case "1.3.6.1.2.1.2.2.1.1": // ifIndex
            log.Printf("Interface index: %v", vb.Value)
        case "1.3.6.1.2.1.2.2.1.2": // ifDescr
            log.Printf("Interface: %s", vb.Value)
        case "1.3.6.1.2.1.2.2.1.8": // ifOperStatus
            log.Printf("Status: %v", vb.Value)
        }
    }
}
```

### Forward to Channel

```go
trapChannel := make(chan *snmp.TrapPDU, 100)

handler := func(trap *snmp.TrapPDU) {
    select {
    case trapChannel <- trap:
    default:
        log.Println("Trap channel full, dropping trap")
    }
}

// Process traps in separate goroutine
go func() {
    for trap := range trapChannel {
        processTrap(trap)
    }
}()
```

## Complete Example

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "os/signal"
    "time"

    "github.com/edgeo/drivers/snmp/snmp"
)

type TrapEvent struct {
    Timestamp time.Time         `json:"timestamp"`
    Source    string            `json:"source"`
    Type      string            `json:"type"`
    VarBinds  map[string]string `json:"var_binds"`
}

func main() {
    handler := func(trap *snmp.TrapPDU) {
        event := TrapEvent{
            Timestamp: time.Now(),
            Source:    trap.AgentAddress.String(),
            Type:      trap.Type.String(),
            VarBinds:  make(map[string]string),
        }

        for _, vb := range trap.VarBinds {
            event.VarBinds[vb.OID] = fmt.Sprintf("%v", vb.Value)
        }

        data, _ := json.Marshal(event)
        log.Println(string(data))
    }

    listener := snmp.NewTrapListener(handler,
        snmp.WithListenAddress(":1162"), // Non-privileged port
    )

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    if err := listener.Start(ctx); err != nil {
        log.Fatal(err)
    }

    log.Println("Trap listener started on :1162")

    // Wait for interrupt
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh

    log.Println("Shutting down...")
    listener.Stop()
}
```

## Testing Trap Listener

Send test traps using `snmptrap`:

```bash
# SNMPv1 trap
snmptrap -v 1 -c public localhost:1162 \
    1.3.6.1.4.1.8072.2.3.0.1 \
    localhost 6 1 '' \
    1.3.6.1.4.1.8072.2.3.2.1 s "Test message"

# SNMPv2c notification
snmptrap -v 2c -c public localhost:1162 \
    '' 1.3.6.1.6.3.1.1.5.4 \
    1.3.6.1.2.1.2.2.1.1 i 1 \
    1.3.6.1.2.1.2.2.1.2 s "eth0"
```

## Best Practices

1. **Use non-privileged ports** - Port 162 requires root; use 1162 for testing
2. **Buffer traps** - Use channels to handle bursts
3. **Log all traps** - Store for debugging and analysis
4. **Filter by community** - Ignore unwanted sources
5. **Monitor listener health** - Watch for errors
6. **Handle graceful shutdown** - Stop listener properly
