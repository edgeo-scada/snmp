---
slug: /snmp/client
---

# Client SNMP

Le client SNMP est le composant principal pour interagir avec les agents SNMP.

## Création du client

### Client SNMPv1

```go
client, err := snmp.NewClient(ctx,
    snmp.WithTarget("192.168.1.1:161"),
    snmp.WithVersion(snmp.Version1),
    snmp.WithCommunity("public"),
)
```

### Client SNMPv2c

```go
client, err := snmp.NewClient(ctx,
    snmp.WithTarget("192.168.1.1:161"),
    snmp.WithVersion(snmp.Version2c),
    snmp.WithCommunity("public"),
    snmp.WithTimeout(5*time.Second),
    snmp.WithRetries(3),
)
```

### Client SNMPv3

```go
client, err := snmp.NewClient(ctx,
    snmp.WithTarget("192.168.1.1:161"),
    snmp.WithVersion(snmp.Version3),
    snmp.WithSecurityName("admin"),
    snmp.WithAuthProtocol(snmp.AuthSHA),
    snmp.WithAuthPassword("authpass"),
    snmp.WithPrivProtocol(snmp.PrivAES),
    snmp.WithPrivPassword("privpass"),
    snmp.WithSecurityLevel(snmp.AuthPriv),
)
```

## Opérations SNMP

### Get

Récupère la valeur d'un ou plusieurs OIDs spécifiques.

```go
// GET simple
vars, err := client.Get(ctx, snmp.OIDSysDescr)
if err != nil {
    return err
}
fmt.Printf("Description: %s\n", vars[0].Value)

// GET multiple
vars, err := client.Get(ctx,
    snmp.OIDSysDescr,
    snmp.OIDSysName,
    snmp.OIDSysUpTime,
    snmp.OIDSysContact,
)
if err != nil {
    return err
}

for _, v := range vars {
    fmt.Printf("%s = %v\n", v.OID, v.Value)
}
```

### GetNext

Récupère l'OID suivant dans l'arbre MIB.

```go
// GET-NEXT simple
vars, err := client.GetNext(ctx, snmp.OIDSysDescr)
if err != nil {
    return err
}
// Retourne sysObjectID (1.3.6.1.2.1.1.2.0)

// GET-NEXT multiple
vars, err := client.GetNext(ctx, oid1, oid2, oid3)
```

### GetBulk (v2c/v3 uniquement)

Récupère plusieurs OIDs en une seule requête, plus efficace que des GET-NEXT répétés.

```go
// Paramètres:
// - nonRepeaters: nombre d'OIDs à traiter comme des GET-NEXT simples
// - maxRepetitions: nombre maximum de répétitions pour les OIDs restants

vars, err := client.GetBulk(ctx, 0, 25, rootOID)
if err != nil {
    return err
}

for _, v := range vars {
    fmt.Printf("%s = %v\n", v.OID, v.Value)
}
```

### Set

Modifie la valeur d'un ou plusieurs OIDs.

```go
// SET simple
result, err := client.Set(ctx, snmp.Variable{
    OID:   snmp.OIDSysContact,
    Type:  snmp.TypeOctetString,
    Value: []byte("admin@example.com"),
})
if err != nil {
    return err
}

// SET multiple
result, err := client.Set(ctx,
    snmp.Variable{
        OID:   snmp.OIDSysContact,
        Type:  snmp.TypeOctetString,
        Value: []byte("admin@example.com"),
    },
    snmp.Variable{
        OID:   snmp.OIDSysName,
        Type:  snmp.TypeOctetString,
        Value: []byte("switch-01"),
    },
)
```

### Walk

Parcourt un sous-arbre de la MIB.

```go
// Walk avec fonction callback
rootOID, _ := snmp.ParseOID("1.3.6.1.2.1.2.2") // ifTable

err := client.WalkFunc(ctx, rootOID, func(v snmp.Variable) error {
    fmt.Printf("%s = %v\n", v.OID, v.Value)
    return nil // retourner une erreur arrête le walk
})
if err != nil {
    return err
}
```

Pour v2c/v3, le walk utilise automatiquement GET-BULK pour de meilleures performances.

## Types de données SNMP

### Types supportés

| Type | Constante | Description |
|------|-----------|-------------|
| INTEGER | `TypeInteger` | Entier signé 32 bits |
| OCTET STRING | `TypeOctetString` | Chaîne d'octets |
| NULL | `TypeNull` | Valeur nulle |
| OBJECT IDENTIFIER | `TypeObjectIdentifier` | OID |
| IP Address | `TypeIPAddress` | Adresse IPv4 |
| Counter32 | `TypeCounter32` | Compteur 32 bits (non signé, wrap à 2^32) |
| Gauge32 | `TypeGauge32` | Jauge 32 bits (non signé, max 2^32-1) |
| TimeTicks | `TypeTimeTicks` | Centièmes de seconde |
| Counter64 | `TypeCounter64` | Compteur 64 bits (v2c/v3 uniquement) |
| NoSuchObject | `TypeNoSuchObject` | OID existe mais pas d'instance |
| NoSuchInstance | `TypeNoSuchInstance` | Instance n'existe pas |
| EndOfMibView | `TypeEndOfMibView` | Fin de la vue MIB |

### Manipulation des valeurs

```go
// INTEGER
if val, ok := v.Value.(int); ok {
    fmt.Printf("Integer: %d\n", val)
}

// OCTET STRING
if val, ok := v.Value.([]byte); ok {
    fmt.Printf("String: %s\n", string(val))
}

// IP Address
if val, ok := v.Value.(net.IP); ok {
    fmt.Printf("IP: %s\n", val.String())
}

// Counter32/Gauge32/TimeTicks
if val, ok := v.Value.(uint32); ok {
    fmt.Printf("Counter: %d\n", val)
}

// Counter64
if val, ok := v.Value.(uint64); ok {
    fmt.Printf("Counter64: %d\n", val)
}

// OBJECT IDENTIFIER
if val, ok := v.Value.(snmp.OID); ok {
    fmt.Printf("OID: %s\n", val.String())
}
```

### Conversion TimeTicks

```go
// Convertir TimeTicks en chaîne lisible
if ticks, ok := v.Value.(uint32); ok {
    str := snmp.TimeTicksToString(ticks)
    fmt.Printf("Uptime: %s\n", str) // "5 days, 12:34:56.78"
}
```

## OIDs prédéfinis

### System MIB (1.3.6.1.2.1.1)

```go
snmp.OIDSysDescr     // 1.3.6.1.2.1.1.1.0 - Description système
snmp.OIDSysObjectID  // 1.3.6.1.2.1.1.2.0 - Object ID système
snmp.OIDSysUpTime    // 1.3.6.1.2.1.1.3.0 - Uptime (TimeTicks)
snmp.OIDSysContact   // 1.3.6.1.2.1.1.4.0 - Contact administrateur
snmp.OIDSysName      // 1.3.6.1.2.1.1.5.0 - Nom système
snmp.OIDSysLocation  // 1.3.6.1.2.1.1.6.0 - Localisation physique
snmp.OIDSysServices  // 1.3.6.1.2.1.1.7.0 - Services disponibles
```

### Parsing d'OIDs

```go
// Parser un OID depuis une chaîne
oid, err := snmp.ParseOID("1.3.6.1.4.1.9.2.1.55.0")
if err != nil {
    return err
}

// Vérifier si un OID est préfixe d'un autre
if rootOID.IsPrefix(childOID) {
    fmt.Println("childOID est sous rootOID")
}

// Comparer des OIDs
if oid1.Equal(oid2) {
    fmt.Println("OIDs identiques")
}

// Convertir en chaîne
str := oid.String() // "1.3.6.1.2.1.1.1.0"
```

## Options du client

### Réseau

```go
snmp.WithTarget("192.168.1.1:161")  // Adresse cible
snmp.WithTimeout(5*time.Second)     // Timeout par requête
snmp.WithRetries(3)                 // Nombre de tentatives
```

### Authentification v1/v2c

```go
snmp.WithVersion(snmp.Version2c)    // Version SNMP
snmp.WithCommunity("public")        // Community string
```

### Authentification v3

```go
snmp.WithSecurityName("admin")           // Nom d'utilisateur USM
snmp.WithSecurityLevel(snmp.AuthPriv)    // Niveau de sécurité
snmp.WithAuthProtocol(snmp.AuthSHA)      // Protocole d'authentification
snmp.WithAuthPassword("authpass")        // Mot de passe d'authentification
snmp.WithPrivProtocol(snmp.PrivAES)      // Protocole de chiffrement
snmp.WithPrivPassword("privpass")        // Mot de passe de chiffrement
snmp.WithContextName("context")          // Nom de contexte (optionnel)
snmp.WithContextEngineID("engineid")     // Engine ID de contexte (optionnel)
```

### Performance

```go
snmp.WithMaxRepetitions(25)    // Max repetitions pour GET-BULK
snmp.WithMaxOIDsPerRequest(10) // Max OIDs par requête
```

## Gestion des connexions

### Fermeture du client

```go
client, err := snmp.NewClient(ctx, opts...)
if err != nil {
    return err
}
defer client.Close() // Toujours fermer le client

// Utilisation...
```

### Vérification de l'état

```go
// Vérifier si le client est connecté
if client.IsConnected() {
    // Client prêt
}

// Obtenir les options actuelles
opts := client.Options()
fmt.Printf("Target: %s\n", opts.Target)
fmt.Printf("Version: %s\n", opts.Version)
```

## Bonnes pratiques

### Réutilisation du client

```go
// ✅ Bon - réutiliser le client
client, _ := snmp.NewClient(ctx, opts...)
defer client.Close()

for _, device := range devices {
    vars, _ := client.Get(ctx, oid)
    // ...
}

// ❌ Mauvais - créer un client par requête
for _, device := range devices {
    client, _ := snmp.NewClient(ctx, opts...)
    vars, _ := client.Get(ctx, oid)
    client.Close()
}
```

### Gestion du contexte

```go
// Timeout global
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

client, _ := snmp.NewClient(ctx, opts...)
vars, err := client.Get(ctx, oids...)
```

### Walk de grandes tables

```go
// Pour de grandes tables, utiliser WalkFunc avec traitement streaming
err := client.WalkFunc(ctx, rootOID, func(v snmp.Variable) error {
    // Traiter chaque variable immédiatement
    if err := processVariable(v); err != nil {
        return err // Arrêter le walk en cas d'erreur
    }
    return nil
})
```

## Voir aussi

- [Options de configuration](options.md)
- [Gestion des erreurs](errors.md)
- [Pool de connexions](pool.md)
