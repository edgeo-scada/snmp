---
slug: /snmp
---

# SNMP Driver

![Version](https://img.shields.io/badge/version-1.0.0-blue)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-green)

Une implémentation complète du protocole SNMP (Simple Network Management Protocol) en Go, supportant les versions v1, v2c et v3.

## Installation

```bash
go get github.com/edgeo/drivers/snmp
```

## Fonctionnalités

- ✅ **Support multi-version** : SNMPv1, SNMPv2c et SNMPv3
- ✅ **Opérations complètes** : GET, GET-NEXT, GET-BULK, SET, WALK
- ✅ **Réception de traps** : Listener pour notifications SNMP
- ✅ **Pool de connexions** : Gestion optimisée des connexions
- ✅ **SNMPv3 Security** : AuthNoPriv et AuthPriv (MD5/SHA, DES/AES)
- ✅ **Métriques intégrées** : Compteurs, jauges et histogrammes de latence
- ✅ **Encodage BER** : Implémentation complète ASN.1/BER
- ✅ **CLI complète** : Outil en ligne de commande `edgeo-snmp`

## Exemple rapide

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/edgeo/drivers/snmp/snmp"
)

func main() {
    ctx := context.Background()

    // Créer un client SNMPv2c
    client, err := snmp.NewClient(ctx,
        snmp.WithTarget("192.168.1.1:161"),
        snmp.WithVersion(snmp.Version2c),
        snmp.WithCommunity("public"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Récupérer les informations système
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

## Structure du package

```
snmp/
├── snmp/
│   ├── client.go      # Client SNMP principal
│   ├── pool.go        # Pool de connexions
│   ├── trap.go        # Listener de traps
│   ├── protocol.go    # Encodage/décodage BER
│   ├── packets.go     # Structures PDU et messages
│   ├── types.go       # Types et OIDs
│   ├── options.go     # Options de configuration
│   ├── errors.go      # Gestion des erreurs
│   ├── metrics.go     # Métriques
│   └── version.go     # Versions SNMP
└── cmd/
    └── edgeo-snmp/    # CLI
```

## Documentation

- [Démarrage rapide](getting-started.md)
- [Client SNMP](client.md)
- [Options de configuration](options.md)
- [Gestion des erreurs](errors.md)
- [Métriques](metrics.md)
- [Pool de connexions](pool.md)
- [Listener de traps](trap-listener.md)
- [Changelog](changelog.md)

## Exemples

- [Client basique](examples/basic-client.md)
- [Listener de traps](examples/trap-listener.md)

## CLI

L'outil `edgeo-snmp` fournit une interface en ligne de commande complète :

```bash
# Informations système
edgeo-snmp info -t 192.168.1.1

# GET simple
edgeo-snmp get -t 192.168.1.1 1.3.6.1.2.1.1.1.0

# Walk d'un sous-arbre
edgeo-snmp walk -t 192.168.1.1 1.3.6.1.2.1.2.2

# SET d'une valeur
edgeo-snmp set -t 192.168.1.1 1.3.6.1.2.1.1.4.0 s "admin@example.com"

# Écoute des traps
edgeo-snmp trap-listen --listen ":1162"
```

## Versions SNMP supportées

| Version | Authentification | Chiffrement | Bulk Operations |
|---------|-----------------|-------------|-----------------|
| v1      | Community       | Non         | Non             |
| v2c     | Community       | Non         | Oui             |
| v3      | USM (MD5/SHA)   | DES/AES     | Oui             |

## Licence

MIT License - voir [LICENSE](../../LICENSE) pour plus de détails.
