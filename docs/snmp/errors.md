---
slug: /snmp/errors
---

# Gestion des erreurs

La librairie SNMP fournit une gestion complète des erreurs avec des types spécifiques pour différencier les erreurs protocole des erreurs réseau.

## Types d'erreurs

### SNMPError

L'erreur principale retournée lors d'erreurs protocole SNMP.

```go
type SNMPError struct {
    Status     ErrorStatus // Code d'erreur SNMP
    Index      int         // Index de la variable en erreur
    Message    string      // Message descriptif
    RequestID  uint32      // ID de la requête
}

func (e *SNMPError) Error() string
```

### Erreurs standard

```go
var (
    ErrTimeout          = errors.New("snmp: request timeout")
    ErrNotConnected     = errors.New("snmp: client not connected")
    ErrInvalidOID       = errors.New("snmp: invalid OID")
    ErrInvalidResponse  = errors.New("snmp: invalid response")
    ErrVersionMismatch  = errors.New("snmp: version mismatch")
    ErrAuthFailure      = errors.New("snmp: authentication failure")
    ErrPrivFailure      = errors.New("snmp: privacy/encryption failure")
    ErrEndOfMibView     = errors.New("snmp: end of MIB view")
    ErrNoSuchObject     = errors.New("snmp: no such object")
    ErrNoSuchInstance   = errors.New("snmp: no such instance")
)
```

## Codes d'erreur SNMP

| Code | Constante | Description |
|------|-----------|-------------|
| 0 | `ErrNoError` | Pas d'erreur |
| 1 | `ErrTooBig` | Réponse trop grande pour un seul message |
| 2 | `ErrNoSuchName` | OID non trouvé (v1) |
| 3 | `ErrBadValue` | Valeur invalide pour SET |
| 4 | `ErrReadOnly` | Tentative d'écriture sur OID en lecture seule |
| 5 | `ErrGenErr` | Erreur générale |
| 6 | `ErrNoAccess` | Accès refusé |
| 7 | `ErrWrongType` | Type incorrect pour SET |
| 8 | `ErrWrongLength` | Longueur incorrecte |
| 9 | `ErrWrongEncoding` | Encodage incorrect |
| 10 | `ErrWrongValue` | Valeur hors plage |
| 11 | `ErrNoCreation` | Création non supportée |
| 12 | `ErrInconsistentValue` | Valeur incohérente |
| 13 | `ErrResourceUnavailable` | Ressource non disponible |
| 14 | `ErrCommitFailed` | Échec du commit |
| 15 | `ErrUndoFailed` | Échec de l'annulation |
| 16 | `ErrAuthorizationError` | Erreur d'autorisation |
| 17 | `ErrNotWritable` | OID non modifiable |
| 18 | `ErrInconsistentName` | Nom incohérent |

## Gestion des erreurs

### Vérification du type d'erreur

```go
vars, err := client.Get(ctx, oid)
if err != nil {
    var snmpErr *snmp.SNMPError
    if errors.As(err, &snmpErr) {
        // Erreur protocole SNMP
        fmt.Printf("Erreur SNMP: %s (status=%d, index=%d)\n",
            snmpErr.Message, snmpErr.Status, snmpErr.Index)
    } else if errors.Is(err, snmp.ErrTimeout) {
        // Timeout réseau
        fmt.Println("La requête a expiré")
    } else if errors.Is(err, snmp.ErrAuthFailure) {
        // Échec d'authentification
        fmt.Println("Authentification échouée")
    } else {
        // Autre erreur (réseau, etc.)
        fmt.Printf("Erreur: %v\n", err)
    }
    return err
}
```

### Switch sur le code d'erreur

```go
vars, err := client.Get(ctx, oid)
if err != nil {
    var snmpErr *snmp.SNMPError
    if errors.As(err, &snmpErr) {
        switch snmpErr.Status {
        case snmp.ErrNoSuchName:
            fmt.Printf("OID %s non trouvé\n", oid)
        case snmp.ErrTooBig:
            fmt.Println("Réponse trop grande, réduire le nombre d'OIDs")
        case snmp.ErrNoAccess:
            fmt.Println("Accès refusé, vérifier les permissions")
        case snmp.ErrReadOnly:
            fmt.Println("L'OID est en lecture seule")
        case snmp.ErrBadValue:
            fmt.Println("La valeur fournie est invalide")
        case snmp.ErrGenErr:
            fmt.Println("Erreur générale sur l'agent")
        default:
            fmt.Printf("Erreur SNMP: %s\n", snmpErr)
        }
    }
    return err
}
```

### Gestion des erreurs de walk

```go
err := client.WalkFunc(ctx, rootOID, func(v snmp.Variable) error {
    // Vérifier les valeurs d'exception
    switch v.Type {
    case snmp.TypeNoSuchObject:
        log.Printf("OID %s: objet non existant", v.OID)
        return nil // Continuer le walk
    case snmp.TypeNoSuchInstance:
        log.Printf("OID %s: instance non existante", v.OID)
        return nil
    case snmp.TypeEndOfMibView:
        return snmp.ErrEndOfMibView // Arrêter le walk
    }

    // Traiter la valeur normale
    fmt.Printf("%s = %v\n", v.OID, v.Value)
    return nil
})

if err != nil && !errors.Is(err, snmp.ErrEndOfMibView) {
    return err
}
```

### Gestion des erreurs de SET

```go
result, err := client.Set(ctx, variable)
if err != nil {
    var snmpErr *snmp.SNMPError
    if errors.As(err, &snmpErr) {
        switch snmpErr.Status {
        case snmp.ErrReadOnly:
            return fmt.Errorf("impossible de modifier %s: lecture seule", variable.OID)
        case snmp.ErrBadValue:
            return fmt.Errorf("valeur invalide pour %s", variable.OID)
        case snmp.ErrWrongType:
            return fmt.Errorf("type incorrect pour %s", variable.OID)
        case snmp.ErrNoAccess:
            return fmt.Errorf("accès refusé pour %s", variable.OID)
        case snmp.ErrNotWritable:
            return fmt.Errorf("OID %s non modifiable", variable.OID)
        default:
            return fmt.Errorf("SET échoué: %s", snmpErr)
        }
    }
    return err
}
```

## Erreurs SNMPv3

### Authentification

```go
client, err := snmp.NewClient(ctx,
    snmp.WithVersion(snmp.Version3),
    snmp.WithSecurityName("admin"),
    snmp.WithAuthProtocol(snmp.AuthSHA),
    snmp.WithAuthPassword("wrongpassword"),
)
if err != nil {
    if errors.Is(err, snmp.ErrAuthFailure) {
        fmt.Println("Mot de passe d'authentification incorrect")
    }
    return err
}
```

### Chiffrement

```go
vars, err := client.Get(ctx, oid)
if err != nil {
    if errors.Is(err, snmp.ErrPrivFailure) {
        fmt.Println("Échec du déchiffrement - vérifier le mot de passe de chiffrement")
    }
    return err
}
```

## Erreurs réseau

### Timeout

```go
vars, err := client.Get(ctx, oid)
if err != nil {
    if errors.Is(err, snmp.ErrTimeout) {
        // Tenter une reconnexion ou augmenter le timeout
        fmt.Println("Timeout - l'agent ne répond pas")
    }
    return err
}
```

### Connexion

```go
vars, err := client.Get(ctx, oid)
if err != nil {
    if errors.Is(err, snmp.ErrNotConnected) {
        // Le client n'est pas connecté
        fmt.Println("Client non connecté")
    }
    return err
}
```

## Bonnes pratiques

### Logging des erreurs

```go
func logSNMPError(err error, operation string, oid snmp.OID) {
    var snmpErr *snmp.SNMPError
    if errors.As(err, &snmpErr) {
        log.Printf("[SNMP] %s %s failed: status=%d (%s), index=%d, request_id=%d",
            operation, oid, snmpErr.Status, snmpErr.Message,
            snmpErr.Index, snmpErr.RequestID)
    } else {
        log.Printf("[SNMP] %s %s failed: %v", operation, oid, err)
    }
}
```

### Retry avec backoff

```go
func getWithRetry(ctx context.Context, client *snmp.Client, oids ...snmp.OID) ([]snmp.Variable, error) {
    var lastErr error
    backoff := 100 * time.Millisecond

    for attempt := 0; attempt < 3; attempt++ {
        vars, err := client.Get(ctx, oids...)
        if err == nil {
            return vars, nil
        }

        lastErr = err

        // Ne pas réessayer pour certaines erreurs
        var snmpErr *snmp.SNMPError
        if errors.As(err, &snmpErr) {
            switch snmpErr.Status {
            case snmp.ErrNoSuchName, snmp.ErrNoAccess, snmp.ErrReadOnly:
                return nil, err // Pas de retry
            }
        }

        // Attendre avant de réessayer
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-time.After(backoff):
            backoff *= 2
        }
    }

    return nil, lastErr
}
```

### Wrapper d'erreur avec contexte

```go
func getSystemInfo(ctx context.Context, client *snmp.Client) (*SystemInfo, error) {
    vars, err := client.Get(ctx, snmp.OIDSysDescr, snmp.OIDSysName)
    if err != nil {
        return nil, fmt.Errorf("failed to get system info: %w", err)
    }

    // Vérifier les valeurs d'exception
    for _, v := range vars {
        if v.Type == snmp.TypeNoSuchObject || v.Type == snmp.TypeNoSuchInstance {
            return nil, fmt.Errorf("system MIB not available on target")
        }
    }

    return &SystemInfo{
        Description: string(vars[0].Value.([]byte)),
        Name:        string(vars[1].Value.([]byte)),
    }, nil
}
```

## Voir aussi

- [Client SNMP](client.md)
- [Options de configuration](options.md)
