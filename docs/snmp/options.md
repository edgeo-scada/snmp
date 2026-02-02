---
slug: /snmp/options
---

# Options de configuration

La librairie SNMP utilise le pattern des options fonctionnelles pour une configuration flexible et extensible.

## Options du Client

### Options de connexion

#### WithTarget

Définit l'adresse de l'agent SNMP cible.

```go
snmp.WithTarget("192.168.1.1:161")
snmp.WithTarget("switch.example.com:161")
```

**Valeur par défaut:** `"127.0.0.1:161"`

#### WithTimeout

Définit le timeout pour chaque requête SNMP.

```go
snmp.WithTimeout(5 * time.Second)
snmp.WithTimeout(10 * time.Second)
```

**Valeur par défaut:** `5 * time.Second`

#### WithRetries

Définit le nombre de tentatives en cas d'échec.

```go
snmp.WithRetries(3)
snmp.WithRetries(5)
```

**Valeur par défaut:** `3`

### Options de version

#### WithVersion

Définit la version SNMP à utiliser.

```go
snmp.WithVersion(snmp.Version1)   // SNMPv1
snmp.WithVersion(snmp.Version2c)  // SNMPv2c
snmp.WithVersion(snmp.Version3)   // SNMPv3
```

**Valeur par défaut:** `snmp.Version2c`

### Options d'authentification v1/v2c

#### WithCommunity

Définit la community string pour SNMPv1/v2c.

```go
snmp.WithCommunity("public")     // Lecture seule
snmp.WithCommunity("private")    // Lecture/écriture
```

**Valeur par défaut:** `"public"`

### Options d'authentification v3

#### WithSecurityName

Définit le nom d'utilisateur USM (User-based Security Model).

```go
snmp.WithSecurityName("admin")
snmp.WithSecurityName("snmpuser")
```

**Valeur par défaut:** `""`

#### WithSecurityLevel

Définit le niveau de sécurité SNMPv3.

```go
snmp.WithSecurityLevel(snmp.NoAuthNoPriv)  // Pas d'auth, pas de chiffrement
snmp.WithSecurityLevel(snmp.AuthNoPriv)    // Auth sans chiffrement
snmp.WithSecurityLevel(snmp.AuthPriv)      // Auth avec chiffrement
```

**Valeur par défaut:** `snmp.NoAuthNoPriv`

#### WithAuthProtocol

Définit le protocole d'authentification.

```go
snmp.WithAuthProtocol(snmp.AuthMD5)  // MD5 (128 bits)
snmp.WithAuthProtocol(snmp.AuthSHA)  // SHA-1 (160 bits)
```

**Valeur par défaut:** `snmp.AuthMD5`

#### WithAuthPassword

Définit le mot de passe d'authentification.

```go
snmp.WithAuthPassword("myauthpassword")
```

**Note:** Le mot de passe doit avoir au minimum 8 caractères.

#### WithPrivProtocol

Définit le protocole de chiffrement.

```go
snmp.WithPrivProtocol(snmp.PrivDES)  // DES (56 bits)
snmp.WithPrivProtocol(snmp.PrivAES)  // AES-128
```

**Valeur par défaut:** `snmp.PrivDES`

#### WithPrivPassword

Définit le mot de passe de chiffrement.

```go
snmp.WithPrivPassword("myprivpassword")
```

**Note:** Le mot de passe doit avoir au minimum 8 caractères.

#### WithContextName

Définit le nom de contexte SNMPv3.

```go
snmp.WithContextName("mycontext")
```

**Valeur par défaut:** `""`

#### WithContextEngineID

Définit l'Engine ID de contexte SNMPv3.

```go
snmp.WithContextEngineID("8000000001020304")
```

**Valeur par défaut:** Dérivé automatiquement

### Options de performance

#### WithMaxRepetitions

Définit le nombre maximum de répétitions pour GET-BULK.

```go
snmp.WithMaxRepetitions(10)
snmp.WithMaxRepetitions(50)
```

**Valeur par défaut:** `10`

#### WithMaxOIDsPerRequest

Définit le nombre maximum d'OIDs par requête.

```go
snmp.WithMaxOIDsPerRequest(10)
snmp.WithMaxOIDsPerRequest(25)
```

**Valeur par défaut:** `10`

## Options du Pool

### WithPoolSize

Définit le nombre de connexions dans le pool.

```go
snmp.WithPoolSize(10)
snmp.WithPoolSize(50)
```

**Valeur par défaut:** `10`

### WithPoolTarget

Définit l'adresse cible pour toutes les connexions du pool.

```go
snmp.WithPoolTarget("192.168.1.1:161")
```

### WithPoolVersion

Définit la version SNMP pour le pool.

```go
snmp.WithPoolVersion(snmp.Version2c)
```

### WithPoolCommunity

Définit la community pour le pool (v1/v2c).

```go
snmp.WithPoolCommunity("public")
```

### WithPoolHealthCheck

Active la vérification périodique de la santé des connexions.

```go
snmp.WithPoolHealthCheck(30 * time.Second)
```

**Valeur par défaut:** Désactivé

### WithPoolMaxIdleTime

Définit la durée maximale d'inactivité avant fermeture.

```go
snmp.WithPoolMaxIdleTime(5 * time.Minute)
```

**Valeur par défaut:** `5 * time.Minute`

## Options du TrapListener

### WithListenAddress

Définit l'adresse d'écoute pour les traps.

```go
snmp.WithListenAddress(":162")        // Port standard (nécessite root)
snmp.WithListenAddress(":1162")       // Port alternatif
snmp.WithListenAddress("0.0.0.0:162") // Toutes les interfaces
```

**Valeur par défaut:** `":162"`

### WithTrapCommunity

Filtre les traps par community string.

```go
snmp.WithTrapCommunity("public")    // Accepte uniquement "public"
snmp.WithTrapCommunity("")          // Accepte toutes les communities
```

**Valeur par défaut:** `""` (accepte tout)

### WithTrapHandler

Définit le gestionnaire de traps personnalisé.

```go
snmp.WithTrapHandler(func(trap *snmp.TrapPDU) {
    log.Printf("Trap reçu: %v", trap)
})
```

## Exemples de configuration

### Client SNMPv2c minimal

```go
client, err := snmp.NewClient(ctx,
    snmp.WithTarget("192.168.1.1:161"),
)
```

### Client SNMPv2c complet

```go
client, err := snmp.NewClient(ctx,
    snmp.WithTarget("192.168.1.1:161"),
    snmp.WithVersion(snmp.Version2c),
    snmp.WithCommunity("public"),
    snmp.WithTimeout(10*time.Second),
    snmp.WithRetries(3),
    snmp.WithMaxRepetitions(25),
)
```

### Client SNMPv3 AuthPriv

```go
client, err := snmp.NewClient(ctx,
    snmp.WithTarget("192.168.1.1:161"),
    snmp.WithVersion(snmp.Version3),
    snmp.WithSecurityName("admin"),
    snmp.WithSecurityLevel(snmp.AuthPriv),
    snmp.WithAuthProtocol(snmp.AuthSHA),
    snmp.WithAuthPassword("authpassword123"),
    snmp.WithPrivProtocol(snmp.PrivAES),
    snmp.WithPrivPassword("privpassword123"),
    snmp.WithTimeout(5*time.Second),
)
```

### Pool de connexions

```go
pool, err := snmp.NewPool(ctx,
    snmp.WithPoolSize(20),
    snmp.WithPoolTarget("192.168.1.1:161"),
    snmp.WithPoolVersion(snmp.Version2c),
    snmp.WithPoolCommunity("public"),
    snmp.WithPoolHealthCheck(30*time.Second),
    snmp.WithPoolMaxIdleTime(5*time.Minute),
)
```

### Trap Listener

```go
listener := snmp.NewTrapListener(handler,
    snmp.WithListenAddress(":1162"),
    snmp.WithTrapCommunity("traps"),
)
```

## Tableau récapitulatif

### Options Client

| Option | Type | Défaut | Description |
|--------|------|--------|-------------|
| `WithTarget` | `string` | `"127.0.0.1:161"` | Adresse de l'agent |
| `WithVersion` | `SNMPVersion` | `Version2c` | Version SNMP |
| `WithCommunity` | `string` | `"public"` | Community string |
| `WithTimeout` | `time.Duration` | `5s` | Timeout par requête |
| `WithRetries` | `int` | `3` | Tentatives |
| `WithSecurityName` | `string` | `""` | Utilisateur v3 |
| `WithSecurityLevel` | `SecurityLevel` | `NoAuthNoPriv` | Niveau sécurité v3 |
| `WithAuthProtocol` | `AuthProtocol` | `AuthMD5` | Protocole auth |
| `WithAuthPassword` | `string` | `""` | Mot de passe auth |
| `WithPrivProtocol` | `PrivProtocol` | `PrivDES` | Protocole chiffrement |
| `WithPrivPassword` | `string` | `""` | Mot de passe chiffrement |
| `WithMaxRepetitions` | `int` | `10` | Max rep. GET-BULK |

### Options Pool

| Option | Type | Défaut | Description |
|--------|------|--------|-------------|
| `WithPoolSize` | `int` | `10` | Taille du pool |
| `WithPoolTarget` | `string` | - | Adresse cible |
| `WithPoolHealthCheck` | `time.Duration` | `0` | Intervalle health check |
| `WithPoolMaxIdleTime` | `time.Duration` | `5m` | Durée max inactivité |

### Options TrapListener

| Option | Type | Défaut | Description |
|--------|------|--------|-------------|
| `WithListenAddress` | `string` | `":162"` | Adresse d'écoute |
| `WithTrapCommunity` | `string` | `""` | Filtre community |

## Voir aussi

- [Client SNMP](client.md)
- [Pool de connexions](pool.md)
- [Listener de traps](trap-listener.md)
