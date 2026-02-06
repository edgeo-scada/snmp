# Exemple: Client SNMP basique

Cet exemple montre comment utiliser le client SNMP pour récupérer des informations depuis un agent SNMP.

## Code complet

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/edgeo-scada/snmp/snmp"
)

func main() {
    ctx := context.Background()

    // Créer un client SNMPv2c
    client, err := snmp.NewClient(ctx,
        snmp.WithTarget("192.168.1.1:161"),
        snmp.WithVersion(snmp.Version2c),
        snmp.WithCommunity("public"),
        snmp.WithTimeout(5*time.Second),
        snmp.WithRetries(3),
    )
    if err != nil {
        log.Fatalf("Erreur création client: %v", err)
    }
    defer client.Close()

    // 1. Récupérer les informations système
    fmt.Println("=== Informations système ===")
    if err := getSystemInfo(ctx, client); err != nil {
        log.Printf("Erreur: %v", err)
    }

    // 2. Walk de la table des interfaces
    fmt.Println("\n=== Table des interfaces ===")
    if err := walkInterfaces(ctx, client); err != nil {
        log.Printf("Erreur: %v", err)
    }

    // 3. Modifier un paramètre (si autorisé)
    fmt.Println("\n=== Modification de sysContact ===")
    if err := setContact(ctx, client); err != nil {
        log.Printf("Erreur: %v", err)
    }
}

func getSystemInfo(ctx context.Context, client *snmp.Client) error {
    // Récupérer plusieurs OIDs en une requête
    vars, err := client.Get(ctx,
        snmp.OIDSysDescr,
        snmp.OIDSysObjectID,
        snmp.OIDSysUpTime,
        snmp.OIDSysContact,
        snmp.OIDSysName,
        snmp.OIDSysLocation,
    )
    if err != nil {
        return err
    }

    // Afficher les résultats
    for _, v := range vars {
        name := getOIDName(v.OID)
        value := formatValue(v)
        fmt.Printf("  %-15s %s\n", name+":", value)
    }

    return nil
}

func walkInterfaces(ctx context.Context, client *snmp.Client) error {
    // OID de base pour ifTable
    rootOID, _ := snmp.ParseOID("1.3.6.1.2.1.2.2.1")

    count := 0
    err := client.WalkFunc(ctx, rootOID, func(v snmp.Variable) error {
        fmt.Printf("  %s = %v\n", v.OID, formatValue(v))
        count++
        return nil
    })

    if err != nil {
        return err
    }

    fmt.Printf("\n  Total: %d entrées\n", count)
    return nil
}

func setContact(ctx context.Context, client *snmp.Client) error {
    // Note: Nécessite community "private" ou équivalent avec droits d'écriture
    variable := snmp.Variable{
        OID:   snmp.OIDSysContact,
        Type:  snmp.TypeOctetString,
        Value: []byte("admin@example.com"),
    }

    result, err := client.Set(ctx, variable)
    if err != nil {
        return fmt.Errorf("SET échoué: %w", err)
    }

    fmt.Printf("  Nouvelle valeur: %s\n", result[0].Value)
    return nil
}

func getOIDName(oid snmp.OID) string {
    switch {
    case oid.Equal(snmp.OIDSysDescr):
        return "Description"
    case oid.Equal(snmp.OIDSysObjectID):
        return "Object ID"
    case oid.Equal(snmp.OIDSysUpTime):
        return "Uptime"
    case oid.Equal(snmp.OIDSysContact):
        return "Contact"
    case oid.Equal(snmp.OIDSysName):
        return "Name"
    case oid.Equal(snmp.OIDSysLocation):
        return "Location"
    default:
        return oid.String()
    }
}

func formatValue(v snmp.Variable) string {
    switch val := v.Value.(type) {
    case []byte:
        return string(val)
    case uint32:
        if v.Type == snmp.TypeTimeTicks {
            return snmp.TimeTicksToString(val)
        }
        return fmt.Sprintf("%d", val)
    case snmp.OID:
        return val.String()
    default:
        return fmt.Sprintf("%v", val)
    }
}
```

## Exécution

```bash
# Compiler
go build -o snmp-example main.go

# Exécuter
./snmp-example
```

## Sortie attendue

```
=== Informations système ===
  Description:    Linux server 5.4.0-42-generic #46-Ubuntu SMP x86_64
  Object ID:      1.3.6.1.4.1.8072.3.2.10
  Uptime:         5 days, 12:34:56.78
  Contact:        admin@localhost
  Name:           server01
  Location:       Data Center Rack 12

=== Table des interfaces ===
  1.3.6.1.2.1.2.2.1.1.1 = 1
  1.3.6.1.2.1.2.2.1.2.1 = lo
  1.3.6.1.2.1.2.2.1.1.2 = 2
  1.3.6.1.2.1.2.2.1.2.2 = eth0
  ...

  Total: 42 entrées

=== Modification de sysContact ===
  Nouvelle valeur: admin@example.com
```

## Variantes

### Client SNMPv1

```go
client, err := snmp.NewClient(ctx,
    snmp.WithTarget("192.168.1.1:161"),
    snmp.WithVersion(snmp.Version1),
    snmp.WithCommunity("public"),
)
```

### Client SNMPv3

```go
client, err := snmp.NewClient(ctx,
    snmp.WithTarget("192.168.1.1:161"),
    snmp.WithVersion(snmp.Version3),
    snmp.WithSecurityName("admin"),
    snmp.WithSecurityLevel(snmp.AuthPriv),
    snmp.WithAuthProtocol(snmp.AuthSHA),
    snmp.WithAuthPassword("authpassword"),
    snmp.WithPrivProtocol(snmp.PrivAES),
    snmp.WithPrivPassword("privpassword"),
)
```

### Avec pool de connexions

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

// Acquérir un client
client, err := pool.Acquire(ctx)
if err != nil {
    log.Fatal(err)
}
defer pool.Release(client)

// Utiliser le client...
```

## Voir aussi

- [Client SNMP](../client.md)
- [Options de configuration](../options.md)
- [Gestion des erreurs](../errors.md)
