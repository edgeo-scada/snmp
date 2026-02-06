# Listener de traps

Le TrapListener permet de recevoir et traiter les notifications SNMP (traps et informs) envoyées par les agents.

## Vue d'ensemble

Les traps SNMP sont des messages asynchrones envoyés par les agents pour notifier des événements :
- Changements d'état d'interfaces
- Alertes de seuils dépassés
- Erreurs matérielles
- Événements de sécurité

## Types de notifications

| Type | Version | Description |
|------|---------|-------------|
| Trap | v1 | Notification unidirectionnelle |
| Trap | v2c/v3 | Notification améliorée (SNMPv2-Trap-PDU) |
| Inform | v2c/v3 | Notification avec accusé de réception |

## Création du listener

### Configuration minimale

```go
handler := func(trap *snmp.TrapPDU) {
    fmt.Printf("Trap reçu de %s\n", trap.AgentAddress)
}

listener := snmp.NewTrapListener(handler,
    snmp.WithListenAddress(":1162"),
)
```

### Configuration complète

```go
handler := func(trap *snmp.TrapPDU) {
    log.Printf("[TRAP] From: %s, Type: %s", trap.AgentAddress, trap.Type)
    for _, v := range trap.Variables {
        log.Printf("  %s = %v", v.OID, v.Value)
    }
}

listener := snmp.NewTrapListener(handler,
    // Adresse d'écoute (port 162 nécessite root)
    snmp.WithListenAddress(":1162"),

    // Filtrer par community
    snmp.WithTrapCommunity("traps"),
)
```

## Structure TrapPDU

```go
type TrapPDU struct {
    // Informations de base
    Version      SNMPVersion  // Version SNMP du trap
    Type         PDUType      // Type de PDU (Trap, SNMPv2Trap, Inform)
    RequestID    uint32       // ID de la requête

    // Adresse de l'agent
    AgentAddress net.IP       // Adresse IP de l'agent

    // SNMPv1 spécifique
    Enterprise   OID          // OID de l'entreprise
    GenericTrap  int          // Type de trap générique (0-6)
    SpecificTrap int          // Code trap spécifique

    // Variables bindings
    Variables    []Variable   // Données du trap

    // Métadonnées
    Community    string       // Community string (v1/v2c)
    Timestamp    time.Time    // Horodatage de réception
}
```

## Types de traps génériques (v1)

| Code | Constante | Description |
|------|-----------|-------------|
| 0 | `TrapColdStart` | Réinitialisation à froid |
| 1 | `TrapWarmStart` | Réinitialisation à chaud |
| 2 | `TrapLinkDown` | Interface down |
| 3 | `TrapLinkUp` | Interface up |
| 4 | `TrapAuthFailure` | Échec d'authentification |
| 5 | `TrapEgpNeighborLoss` | Perte de voisin EGP |
| 6 | `TrapEnterpriseSpecific` | Trap spécifique entreprise |

## Démarrage et arrêt

### Démarrage

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

listener := snmp.NewTrapListener(handler, opts...)

if err := listener.Start(ctx); err != nil {
    log.Fatalf("Erreur démarrage listener: %v", err)
}

fmt.Println("Listener démarré sur", listener.Address())
```

### Arrêt gracieux

```go
// Arrêter le listener
if err := listener.Stop(); err != nil {
    log.Printf("Erreur arrêt listener: %v", err)
}
```

### Gestion des signaux

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())

    // Gestionnaire de traps
    handler := func(trap *snmp.TrapPDU) {
        processTrap(trap)
    }

    listener := snmp.NewTrapListener(handler,
        snmp.WithListenAddress(":1162"),
    )

    if err := listener.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // Attendre signal d'interruption
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh

    fmt.Println("Arrêt en cours...")
    cancel()
    listener.Stop()
}
```

## Traitement des traps

### Handler simple

```go
handler := func(trap *snmp.TrapPDU) {
    fmt.Printf("=== Trap reçu ===\n")
    fmt.Printf("De: %s\n", trap.AgentAddress)
    fmt.Printf("Version: %s\n", trap.Version)
    fmt.Printf("Community: %s\n", trap.Community)
    fmt.Printf("Timestamp: %s\n", trap.Timestamp.Format(time.RFC3339))

    for _, v := range trap.Variables {
        fmt.Printf("  %s = %v (%s)\n", v.OID, v.Value, v.Type)
    }
    fmt.Println()
}
```

### Handler avec routage

```go
handler := func(trap *snmp.TrapPDU) {
    // Router selon le type de trap
    switch {
    case trap.GenericTrap == snmp.TrapLinkDown:
        handleLinkDown(trap)
    case trap.GenericTrap == snmp.TrapLinkUp:
        handleLinkUp(trap)
    case trap.GenericTrap == snmp.TrapAuthFailure:
        handleAuthFailure(trap)
    case trap.GenericTrap == snmp.TrapEnterpriseSpecific:
        handleEnterpriseSpecific(trap)
    default:
        handleGenericTrap(trap)
    }
}

func handleLinkDown(trap *snmp.TrapPDU) {
    // Extraire l'index d'interface
    for _, v := range trap.Variables {
        if v.OID.HasPrefix(snmp.OIDIfIndex) {
            ifIndex := v.Value.(int)
            log.Printf("Interface %d down sur %s", ifIndex, trap.AgentAddress)
        }
    }
}
```

### Handler asynchrone

```go
type TrapProcessor struct {
    traps chan *snmp.TrapPDU
    wg    sync.WaitGroup
}

func NewTrapProcessor(workers int) *TrapProcessor {
    tp := &TrapProcessor{
        traps: make(chan *snmp.TrapPDU, 1000),
    }

    // Démarrer les workers
    for i := 0; i < workers; i++ {
        tp.wg.Add(1)
        go tp.worker()
    }

    return tp
}

func (tp *TrapProcessor) Handler() func(*snmp.TrapPDU) {
    return func(trap *snmp.TrapPDU) {
        select {
        case tp.traps <- trap:
            // Trap envoyé au worker
        default:
            log.Println("Buffer plein, trap abandonné")
        }
    }
}

func (tp *TrapProcessor) worker() {
    defer tp.wg.Done()
    for trap := range tp.traps {
        tp.process(trap)
    }
}

func (tp *TrapProcessor) process(trap *snmp.TrapPDU) {
    // Traitement du trap (peut être lent)
    saveToDB(trap)
    sendAlert(trap)
}

func (tp *TrapProcessor) Close() {
    close(tp.traps)
    tp.wg.Wait()
}
```

## Filtrage des traps

### Par community

```go
listener := snmp.NewTrapListener(handler,
    snmp.WithListenAddress(":1162"),
    snmp.WithTrapCommunity("traps"),  // Accepte uniquement "traps"
)
```

### Filtrage personnalisé dans le handler

```go
allowedHosts := map[string]bool{
    "192.168.1.1": true,
    "192.168.1.2": true,
}

handler := func(trap *snmp.TrapPDU) {
    // Filtrer par adresse source
    if !allowedHosts[trap.AgentAddress.String()] {
        log.Printf("Trap ignoré de %s (non autorisé)", trap.AgentAddress)
        return
    }

    // Filtrer par OID enterprise
    if !trap.Enterprise.HasPrefix(myEnterpriseOID) {
        return
    }

    // Traiter le trap
    processTrap(trap)
}
```

## Métriques

```go
listener := snmp.NewTrapListener(handler, opts...)

// Après réception de traps...
metrics := listener.Metrics()

fmt.Printf("Traps reçus: %d\n", metrics.TrapsReceived.Value())
fmt.Printf("Traps traités: %d\n", metrics.TrapsProcessed.Value())
fmt.Printf("Traps rejetés: %d\n", metrics.TrapsDropped.Value())
```

## Extraction des variables

### Variables communes

```go
handler := func(trap *snmp.TrapPDU) {
    for _, v := range trap.Variables {
        switch {
        // sysUpTime (1.3.6.1.2.1.1.3.0)
        case v.OID.Equal(snmp.OIDSysUpTime):
            if ticks, ok := v.Value.(uint32); ok {
                uptime := snmp.TimeTicksToString(ticks)
                fmt.Printf("Uptime: %s\n", uptime)
            }

        // snmpTrapOID (1.3.6.1.6.3.1.1.4.1.0)
        case v.OID.Equal(snmp.OIDSnmpTrapOID):
            if trapOID, ok := v.Value.(snmp.OID); ok {
                fmt.Printf("Trap OID: %s\n", trapOID)
            }

        // Autres variables
        default:
            fmt.Printf("%s = %v\n", v.OID, v.Value)
        }
    }
}
```

### Variables d'interface

```go
// OIDs ifTable courants
var (
    OIDIfIndex       = snmp.MustParseOID("1.3.6.1.2.1.2.2.1.1")
    OIDIfDescr       = snmp.MustParseOID("1.3.6.1.2.1.2.2.1.2")
    OIDIfOperStatus  = snmp.MustParseOID("1.3.6.1.2.1.2.2.1.8")
    OIDIfAdminStatus = snmp.MustParseOID("1.3.6.1.2.1.2.2.1.7")
)

func extractInterfaceInfo(trap *snmp.TrapPDU) {
    var ifIndex int
    var ifDescr string
    var ifOperStatus int

    for _, v := range trap.Variables {
        switch {
        case v.OID.HasPrefix(OIDIfIndex):
            ifIndex = v.Value.(int)
        case v.OID.HasPrefix(OIDIfDescr):
            ifDescr = string(v.Value.([]byte))
        case v.OID.HasPrefix(OIDIfOperStatus):
            ifOperStatus = v.Value.(int)
        }
    }

    statusStr := map[int]string{1: "up", 2: "down", 3: "testing"}
    fmt.Printf("Interface %d (%s) is now %s\n", ifIndex, ifDescr, statusStr[ifOperStatus])
}
```

## Persistance des traps

### Sauvegarde en fichier

```go
type TrapLogger struct {
    file   *os.File
    mu     sync.Mutex
}

func (tl *TrapLogger) Handler() func(*snmp.TrapPDU) {
    return func(trap *snmp.TrapPDU) {
        tl.mu.Lock()
        defer tl.mu.Unlock()

        entry := fmt.Sprintf("%s,%s,%s,%d\n",
            trap.Timestamp.Format(time.RFC3339),
            trap.AgentAddress,
            trap.Enterprise,
            len(trap.Variables),
        )

        tl.file.WriteString(entry)
    }
}
```

### Sauvegarde en base de données

```go
func saveTrapToDB(db *sql.DB) func(*snmp.TrapPDU) {
    return func(trap *snmp.TrapPDU) {
        // Sérialiser les variables
        varsJSON, _ := json.Marshal(trap.Variables)

        _, err := db.Exec(`
            INSERT INTO traps (timestamp, agent_address, enterprise, generic_trap, specific_trap, variables)
            VALUES (?, ?, ?, ?, ?, ?)
        `,
            trap.Timestamp,
            trap.AgentAddress.String(),
            trap.Enterprise.String(),
            trap.GenericTrap,
            trap.SpecificTrap,
            varsJSON,
        )

        if err != nil {
            log.Printf("Erreur sauvegarde trap: %v", err)
        }
    }
}
```

## Port privilégié

Le port SNMP trap standard (162) nécessite des privilèges root/administrateur.

### Option 1: Port alternatif

```go
listener := snmp.NewTrapListener(handler,
    snmp.WithListenAddress(":1162"),  // Port non privilégié
)
```

### Option 2: Capabilities Linux

```bash
# Donner la capability CAP_NET_BIND_SERVICE au binaire
sudo setcap 'cap_net_bind_service=+ep' /usr/local/bin/trap-listener
```

### Option 3: iptables redirect

```bash
# Rediriger le port 162 vers 1162
sudo iptables -t nat -A PREROUTING -p udp --dport 162 -j REDIRECT --to-port 1162
```

## Bonnes pratiques

### Handler non bloquant

```go
// ✅ Bon - traitement asynchrone
trapChan := make(chan *snmp.TrapPDU, 1000)

handler := func(trap *snmp.TrapPDU) {
    select {
    case trapChan <- trap:
    default:
        log.Println("Buffer plein")
    }
}

// Worker séparé
go func() {
    for trap := range trapChan {
        processSlowly(trap)
    }
}()

// ❌ Mauvais - traitement bloquant
handler := func(trap *snmp.TrapPDU) {
    saveToDatabase(trap)  // Peut être lent!
    sendEmail(trap)       // Peut être très lent!
}
```

### Logging structuré

```go
handler := func(trap *snmp.TrapPDU) {
    log.WithFields(log.Fields{
        "agent":    trap.AgentAddress,
        "version":  trap.Version,
        "type":     trap.Type,
        "generic":  trap.GenericTrap,
        "specific": trap.SpecificTrap,
        "vars":     len(trap.Variables),
    }).Info("Trap received")
}
```

## Voir aussi

- [Client SNMP](client.md)
- [Options de configuration](options.md)
- [Exemple: Trap Listener](examples/trap-listener.md)
