# Pool de connexions

Le pool de connexions permet de gérer efficacement plusieurs connexions SNMP vers un même agent, optimisant les performances pour les applications à fort trafic.

## Vue d'ensemble

Le pool maintient un ensemble de connexions SNMP pré-établies, permettant :

- Réutilisation des connexions existantes
- Limitation du nombre de connexions simultanées
- Health checks périodiques
- Gestion automatique des connexions inactives

## Création du pool

### Configuration minimale

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
```

### Configuration complète

```go
pool, err := snmp.NewPool(ctx,
    // Taille du pool
    snmp.WithPoolSize(20),

    // Configuration cible
    snmp.WithPoolTarget("192.168.1.1:161"),
    snmp.WithPoolVersion(snmp.Version2c),
    snmp.WithPoolCommunity("public"),

    // Health checks
    snmp.WithPoolHealthCheck(30*time.Second),

    // Gestion des connexions inactives
    snmp.WithPoolMaxIdleTime(5*time.Minute),

    // Options client héritées
    snmp.WithTimeout(5*time.Second),
    snmp.WithRetries(3),
)
```

## Utilisation du pool

### Acquire et Release

```go
// Acquérir un client du pool
client, err := pool.Acquire(ctx)
if err != nil {
    return err
}
defer pool.Release(client)

// Utiliser le client
vars, err := client.Get(ctx, snmp.OIDSysDescr)
if err != nil {
    return err
}
```

### Pattern avec fonction

```go
// Helper pour automatiser acquire/release
func withClient(ctx context.Context, pool *snmp.Pool, fn func(*snmp.Client) error) error {
    client, err := pool.Acquire(ctx)
    if err != nil {
        return err
    }
    defer pool.Release(client)

    return fn(client)
}

// Utilisation
err := withClient(ctx, pool, func(client *snmp.Client) error {
    vars, err := client.Get(ctx, snmp.OIDSysDescr)
    if err != nil {
        return err
    }
    fmt.Printf("Description: %s\n", vars[0].Value)
    return nil
})
```

## Opérations sur le pool

### Acquire

Obtient un client disponible du pool. Bloque si aucun client n'est disponible.

```go
// Acquire avec timeout via context
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

client, err := pool.Acquire(ctx)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("Timeout en attendant un client disponible")
    }
    return err
}
defer pool.Release(client)
```

### Release

Retourne un client au pool pour réutilisation.

```go
client, _ := pool.Acquire(ctx)

// Utilisation...
vars, err := client.Get(ctx, oid)

// IMPORTANT: toujours release, même en cas d'erreur
pool.Release(client)
```

### Close

Ferme le pool et toutes ses connexions.

```go
pool, _ := snmp.NewPool(ctx, opts...)

// Utilisation...

// Fermer proprement
if err := pool.Close(); err != nil {
    log.Printf("Erreur lors de la fermeture du pool: %v", err)
}
```

## Health Checks

### Configuration

```go
pool, _ := snmp.NewPool(ctx,
    // Vérifier les connexions toutes les 30 secondes
    snmp.WithPoolHealthCheck(30*time.Second),
)
```

### Fonctionnement

Le health check :
1. Envoie une requête GET sur `sysUpTime` à chaque connexion
2. Marque les connexions défaillantes comme invalides
3. Remplace les connexions invalides par de nouvelles

### Health check manuel

```go
// Forcer un health check immédiat
healthy, unhealthy := pool.HealthCheck(ctx)
fmt.Printf("Connexions saines: %d, défaillantes: %d\n", healthy, unhealthy)
```

## Gestion des connexions inactives

### Configuration

```go
pool, _ := snmp.NewPool(ctx,
    // Fermer les connexions inactives depuis plus de 5 minutes
    snmp.WithPoolMaxIdleTime(5*time.Minute),
)
```

### Éviction automatique

Les connexions inactives sont automatiquement fermées et remplacées lors de l'acquisition suivante.

## Statistiques du pool

### Stats en temps réel

```go
stats := pool.Stats()

fmt.Printf("Taille totale: %d\n", stats.Size)
fmt.Printf("En utilisation: %d\n", stats.InUse)
fmt.Printf("Disponibles: %d\n", stats.Available)
fmt.Printf("Attente: %d\n", stats.Waiting)
fmt.Printf("Total acquis: %d\n", stats.TotalAcquired)
fmt.Printf("Total relâchés: %d\n", stats.TotalReleased)
```

### Métriques

```go
metrics := pool.Metrics()

fmt.Printf("Connexions actives: %d\n", metrics.ActiveConnections.Value())
fmt.Printf("Taille du pool: %d\n", metrics.PoolSize.Value())
fmt.Printf("Disponibles: %d\n", metrics.PoolAvailable.Value())
```

## Patterns avancés

### Pool avec plusieurs cibles

```go
// Créer un pool par cible
pools := make(map[string]*snmp.Pool)

for _, target := range targets {
    pool, err := snmp.NewPool(ctx,
        snmp.WithPoolSize(5),
        snmp.WithPoolTarget(target),
        snmp.WithPoolCommunity("public"),
    )
    if err != nil {
        return err
    }
    pools[target] = pool
}

// Cleanup
defer func() {
    for _, pool := range pools {
        pool.Close()
    }
}()
```

### Pool avec worker pattern

```go
type SNMPWorkerPool struct {
    pool    *snmp.Pool
    jobs    chan Job
    results chan Result
}

type Job struct {
    OID snmp.OID
}

type Result struct {
    OID   snmp.OID
    Value interface{}
    Error error
}

func (wp *SNMPWorkerPool) Start(ctx context.Context, workers int) {
    for i := 0; i < workers; i++ {
        go wp.worker(ctx)
    }
}

func (wp *SNMPWorkerPool) worker(ctx context.Context) {
    for job := range wp.jobs {
        client, err := wp.pool.Acquire(ctx)
        if err != nil {
            wp.results <- Result{OID: job.OID, Error: err}
            continue
        }

        vars, err := client.Get(ctx, job.OID)
        wp.pool.Release(client)

        if err != nil {
            wp.results <- Result{OID: job.OID, Error: err}
        } else {
            wp.results <- Result{OID: job.OID, Value: vars[0].Value}
        }
    }
}
```

### Pool avec retry

```go
func getWithPoolRetry(ctx context.Context, pool *snmp.Pool, oid snmp.OID) ([]snmp.Variable, error) {
    var lastErr error

    for attempt := 0; attempt < 3; attempt++ {
        client, err := pool.Acquire(ctx)
        if err != nil {
            return nil, err
        }

        vars, err := client.Get(ctx, oid)
        pool.Release(client)

        if err == nil {
            return vars, nil
        }

        lastErr = err

        // Attendre avant de réessayer
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-time.After(time.Duration(attempt+1) * 100 * time.Millisecond):
        }
    }

    return nil, fmt.Errorf("failed after 3 attempts: %w", lastErr)
}
```

## Bonnes pratiques

### Dimensionnement du pool

```go
// Règle générale: taille = nb_workers * 2
// Pour 10 workers concurrents:
pool, _ := snmp.NewPool(ctx,
    snmp.WithPoolSize(20),
)
```

### Toujours Release

```go
// ✅ Bon - defer immédiatement après acquire
client, err := pool.Acquire(ctx)
if err != nil {
    return err
}
defer pool.Release(client)

// ❌ Mauvais - oubli de release
client, err := pool.Acquire(ctx)
if err != nil {
    return err
}
vars, err := client.Get(ctx, oid)
if err != nil {
    return err // Fuite de connexion!
}
pool.Release(client)
```

### Timeout sur Acquire

```go
// ✅ Bon - timeout explicite
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

client, err := pool.Acquire(ctx)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        // Gérer le cas où le pool est saturé
    }
    return err
}
```

### Monitoring du pool

```go
// Surveiller la saturation du pool
go func() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        stats := pool.Stats()
        utilization := float64(stats.InUse) / float64(stats.Size) * 100

        if utilization > 80 {
            log.Printf("WARNING: Pool utilization at %.1f%%", utilization)
        }

        if stats.Waiting > 0 {
            log.Printf("WARNING: %d requests waiting for connection", stats.Waiting)
        }
    }
}()
```

## Configuration SNMPv3

```go
pool, err := snmp.NewPool(ctx,
    snmp.WithPoolSize(10),
    snmp.WithPoolTarget("192.168.1.1:161"),
    snmp.WithPoolVersion(snmp.Version3),
    snmp.WithSecurityName("admin"),
    snmp.WithSecurityLevel(snmp.AuthPriv),
    snmp.WithAuthProtocol(snmp.AuthSHA),
    snmp.WithAuthPassword("authpassword"),
    snmp.WithPrivProtocol(snmp.PrivAES),
    snmp.WithPrivPassword("privpassword"),
    snmp.WithPoolHealthCheck(30*time.Second),
)
```

## Voir aussi

- [Client SNMP](client.md)
- [Options de configuration](options.md)
- [Métriques](metrics.md)
