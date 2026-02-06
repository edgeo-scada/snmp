# SNMPv3 Example

This example demonstrates secure SNMPv3 communication with authentication and encryption.

## SNMPv3 Security Levels

| Level | Authentication | Encryption | Use Case |
|-------|---------------|------------|----------|
| NoAuthNoPriv | No | No | Legacy compatibility |
| AuthNoPriv | Yes | No | Authentication only |
| AuthPriv | Yes | Yes | Full security |

## AuthPriv Example (Recommended)

Full security with authentication and encryption:

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/edgeo/drivers/snmp/snmp"
)

func main() {
    client := snmp.NewClient(
        snmp.WithTarget("192.168.1.1"),
        snmp.WithVersion(snmp.Version3),
        snmp.WithSecurityLevel(snmp.AuthPriv),
        snmp.WithSecurityName("admin"),
        snmp.WithAuth(snmp.AuthSHA256, "myauthpassword"),
        snmp.WithPrivacy(snmp.PrivAES256, "myprivpassword"),
        snmp.WithTimeout(5*time.Second),
    )

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatalf("Connection failed: %v", err)
    }
    defer client.Close()

    log.Println("Connected with SNMPv3 AuthPriv")

    // Get system info
    result, err := client.Get(ctx,
        snmp.OIDSysDescr,
        snmp.OIDSysName,
        snmp.OIDSysUpTime,
    )
    if err != nil {
        log.Fatalf("Get error: %v", err)
    }

    log.Printf("System: %s", result[0].Value)
    log.Printf("Name: %s", result[1].Value)
    log.Printf("Uptime: %v ticks", result[2].Value)
}
```

## AuthNoPriv Example

Authentication only (no encryption):

```go
client := snmp.NewClient(
    snmp.WithTarget("192.168.1.1"),
    snmp.WithVersion(snmp.Version3),
    snmp.WithSecurityLevel(snmp.AuthNoPriv),
    snmp.WithSecurityName("monitor"),
    snmp.WithAuth(snmp.AuthSHA, "monitorpassword"),
)
```

## NoAuthNoPriv Example

No security (not recommended):

```go
client := snmp.NewClient(
    snmp.WithTarget("192.168.1.1"),
    snmp.WithVersion(snmp.Version3),
    snmp.WithSecurityLevel(snmp.NoAuthNoPriv),
    snmp.WithSecurityName("public"),
)
```

## Authentication Protocols

| Protocol | Security | Performance | Recommendation |
|----------|----------|-------------|----------------|
| MD5 | Weak | Fast | Not recommended |
| SHA | Good | Fast | Acceptable |
| SHA-224 | Good | Medium | Good |
| SHA-256 | Strong | Medium | Recommended |
| SHA-384 | Strong | Slower | High security |
| SHA-512 | Strongest | Slowest | Maximum security |

## Privacy Protocols

| Protocol | Security | Performance | Recommendation |
|----------|----------|-------------|----------------|
| DES | Weak | Fast | Not recommended |
| AES-128 | Good | Fast | Acceptable |
| AES-192 | Strong | Medium | Good |
| AES-256 | Strongest | Medium | Recommended |

## Context Configuration

For devices with multiple contexts:

```go
client := snmp.NewClient(
    snmp.WithTarget("192.168.1.1"),
    snmp.WithVersion(snmp.Version3),
    snmp.WithSecurityLevel(snmp.AuthPriv),
    snmp.WithSecurityName("admin"),
    snmp.WithAuth(snmp.AuthSHA256, "authpass"),
    snmp.WithPrivacy(snmp.PrivAES256, "privpass"),
    snmp.WithContextName("vrf-management"),
)
```

## Error Handling for SNMPv3

```go
result, err := client.Get(ctx, oid)
if err != nil {
    switch {
    case errors.Is(err, snmp.ErrAuthenticationFailure):
        log.Println("Authentication failed")
        log.Println("- Check username is correct")
        log.Println("- Verify auth password")
        log.Println("- Confirm auth protocol matches device")

    case errors.Is(err, snmp.ErrDecryptionError):
        log.Println("Decryption failed")
        log.Println("- Check privacy password")
        log.Println("- Verify privacy protocol matches device")

    case errors.Is(err, snmp.ErrUnknownSecurityName):
        log.Println("Unknown security name")
        log.Println("- User not configured on device")

    case errors.Is(err, snmp.ErrNotInTimeWindow):
        log.Println("Message outside time window")
        log.Println("- Check clock synchronization")

    default:
        log.Printf("Error: %v", err)
    }
}
```

## Complete Secure Monitoring Example

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

type DeviceStats struct {
    Timestamp   time.Time `json:"timestamp"`
    Target      string    `json:"target"`
    SysName     string    `json:"sys_name"`
    Uptime      uint32    `json:"uptime_ticks"`
    IfInOctets  uint64    `json:"if_in_octets"`
    IfOutOctets uint64    `json:"if_out_octets"`
}

func main() {
    // Secure SNMPv3 client
    client := snmp.NewClient(
        snmp.WithTarget("192.168.1.1"),
        snmp.WithVersion(snmp.Version3),
        snmp.WithSecurityLevel(snmp.AuthPriv),
        snmp.WithSecurityName("monitor"),
        snmp.WithAuth(snmp.AuthSHA256, os.Getenv("SNMP_AUTH_PASS")),
        snmp.WithPrivacy(snmp.PrivAES256, os.Getenv("SNMP_PRIV_PASS")),
        snmp.WithTimeout(5*time.Second),
        snmp.WithRetries(2),
    )

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    log.Println("Started secure monitoring")

    // Poll every 30 seconds
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)

    for {
        select {
        case <-sigCh:
            log.Println("Shutting down")
            return

        case <-ticker.C:
            stats, err := collectStats(ctx, client)
            if err != nil {
                log.Printf("Error: %v", err)
                continue
            }

            data, _ := json.Marshal(stats)
            log.Println(string(data))
        }
    }
}

func collectStats(ctx context.Context, client *snmp.Client) (*DeviceStats, error) {
    result, err := client.Get(ctx,
        snmp.OIDSysName,
        snmp.OIDSysUpTime,
        "1.3.6.1.2.1.2.2.1.10.1", // ifInOctets.1
        "1.3.6.1.2.1.2.2.1.16.1", // ifOutOctets.1
    )
    if err != nil {
        return nil, err
    }

    stats := &DeviceStats{
        Timestamp:   time.Now(),
        Target:      "192.168.1.1",
        SysName:     result[0].Value.(string),
        Uptime:      result[1].Value.(uint32),
        IfInOctets:  uint64(result[2].Value.(uint32)),
        IfOutOctets: uint64(result[3].Value.(uint32)),
    }

    return stats, nil
}
```

## Device Configuration

### Cisco IOS SNMPv3 Configuration

```
snmp-server group v3group v3 priv
snmp-server user admin v3group v3 auth sha256 authpassword priv aes 256 privpassword
```

### Linux net-snmp Configuration

`/etc/snmp/snmpd.conf`:
```
createUser admin SHA-256 authpassword AES256 privpassword
rouser admin priv
```

## Best Practices

1. **Always use AuthPriv** - Full security for production
2. **Use SHA-256 or higher** - Avoid MD5 and SHA-1
3. **Use AES-256** - Avoid DES
4. **Store credentials securely** - Use environment variables or secrets manager
5. **Rotate passwords regularly** - Follow security policies
6. **Use separate users** - Different users for different access levels
