---
slug: /snmp/getting-started
---

# Démarrage rapide

Ce guide vous permet de démarrer rapidement avec la librairie SNMP.

## Installation

```bash
go get github.com/edgeo-scada/snmp
```

## Concepts de base

### OID (Object Identifier)

Les OIDs identifient les objets dans la MIB (Management Information Base) :

```go
// OIDs prédéfinis pour le système
snmp.OIDSysDescr     // 1.3.6.1.2.1.1.1.0 - Description système
snmp.OIDSysObjectID  // 1.3.6.1.2.1.1.2.0 - Object ID système
snmp.OIDSysUpTime    // 1.3.6.1.2.1.1.3.0 - Uptime système
snmp.OIDSysContact   // 1.3.6.1.2.1.1.4.0 - Contact
snmp.OIDSysName      // 1.3.6.1.2.1.1.5.0 - Nom système
snmp.OIDSysLocation  // 1.3.6.1.2.1.1.6.0 - Localisation

// Parser un OID personnalisé
oid, err := snmp.ParseOID("1.3.6.1.4.1.9.2.1.55.0")
```

### Versions SNMP

```go
snmp.Version1   // SNMPv1 - authentification par community
snmp.Version2c  // SNMPv2c - community + bulk operations
snmp.Version3   // SNMPv3 - authentification USM + chiffrement
```

## Client SNMPv1/v2c

### Connexion simple

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

    // Créer un client SNMPv2c
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

    // Utiliser le client...
}
```

### Opération GET

```go
// GET simple
vars, err := client.Get(ctx, snmp.OIDSysDescr)
if err != nil {
    log.Fatal(err)
}

for _, v := range vars {
    fmt.Printf("%s = %v\n", v.OID, v.Value)
}

// GET multiple
vars, err := client.Get(ctx,
    snmp.OIDSysDescr,
    snmp.OIDSysName,
    snmp.OIDSysUpTime,
)
```

### Opération GET-NEXT

```go
// GET-NEXT retourne l'OID suivant dans la MIB
vars, err := client.GetNext(ctx, snmp.OIDSysDescr)
if err != nil {
    log.Fatal(err)
}

// vars[0].OID sera 1.3.6.1.2.1.1.2.0 (sysObjectID)
```

### Opération GET-BULK (v2c/v3 uniquement)

```go
// GET-BULK pour récupérer plusieurs valeurs efficacement
vars, err := client.GetBulk(ctx, 0, 10, oid)
if err != nil {
    log.Fatal(err)
}
```

### Opération WALK

```go
// Walk d'un sous-arbre
rootOID, _ := snmp.ParseOID("1.3.6.1.2.1.2.2") // ifTable

err := client.WalkFunc(ctx, rootOID, func(v snmp.Variable) error {
    fmt.Printf("%s = %v\n", v.OID, v.Value)
    return nil
})
```

### Opération SET

```go
// SET pour modifier une valeur
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

## Client SNMPv3

### Authentification AuthNoPriv

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

### Authentification AuthPriv

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

## Réception de traps

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

    // Gestionnaire de traps
    handler := func(trap *snmp.TrapPDU) {
        fmt.Printf("Trap reçu de %s\n", trap.AgentAddress)
        fmt.Printf("  Type: %s\n", trap.Type)
        for _, v := range trap.Variables {
            fmt.Printf("  %s = %v\n", v.OID, v.Value)
        }
    }

    // Créer et démarrer le listener
    listener := snmp.NewTrapListener(handler,
        snmp.WithListenAddress(":1162"),
        snmp.WithTrapCommunity("public"),
    )

    if err := listener.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // Attendre l'interruption
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh

    listener.Stop()
}
```

## Pool de connexions

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

// Acquérir un client du pool
client, err := pool.Acquire(ctx)
if err != nil {
    log.Fatal(err)
}
defer pool.Release(client)

// Utiliser le client...
vars, err := client.Get(ctx, snmp.OIDSysDescr)
```

## Gestion des erreurs

```go
vars, err := client.Get(ctx, oid)
if err != nil {
    var snmpErr *snmp.SNMPError
    if errors.As(err, &snmpErr) {
        switch snmpErr.Status {
        case snmp.ErrNoSuchName:
            fmt.Println("OID non trouvé")
        case snmp.ErrTooBig:
            fmt.Println("Réponse trop grande")
        case snmp.ErrGenErr:
            fmt.Println("Erreur générale")
        default:
            fmt.Printf("Erreur SNMP: %s\n", snmpErr)
        }
    } else {
        fmt.Printf("Erreur réseau: %v\n", err)
    }
}
```

## Prochaines étapes

- [Client SNMP](client.md) - Documentation complète du client
- [Options de configuration](options.md) - Toutes les options disponibles
- [Pool de connexions](pool.md) - Gestion avancée des connexions
- [Listener de traps](trap-listener.md) - Réception de notifications
