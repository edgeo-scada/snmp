# Exemple: Listener de traps

Cet exemple montre comment créer un listener pour recevoir et traiter les notifications SNMP (traps).

## Code complet

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"

    "github.com/edgeo-scada/snmp/snmp"
)

func main() {
    // Configuration
    listenAddr := ":1162"
    community := ""  // Vide = accepte tout

    // Créer le contexte avec annulation
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Créer le processeur de traps
    processor := NewTrapProcessor()

    // Créer le listener
    listener := snmp.NewTrapListener(processor.Handle,
        snmp.WithListenAddress(listenAddr),
        snmp.WithTrapCommunity(community),
    )

    // Démarrer le listener
    fmt.Printf("Démarrage du listener sur %s\n", listenAddr)
    if err := listener.Start(ctx); err != nil {
        log.Fatalf("Erreur démarrage: %v", err)
    }

    fmt.Println("En attente de traps... (Ctrl+C pour arrêter)")
    fmt.Println()

    // Attendre l'interruption
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh

    // Arrêt gracieux
    fmt.Println("\nArrêt en cours...")
    cancel()

    if err := listener.Stop(); err != nil {
        log.Printf("Erreur arrêt listener: %v", err)
    }

    // Afficher les statistiques
    processor.PrintStats()

    // Attendre la fin du traitement
    processor.Close()
}

// TrapProcessor traite les traps reçus
type TrapProcessor struct {
    mu       sync.Mutex
    received int
    byType   map[int]int
    byAgent  map[string]int
    wg       sync.WaitGroup
    traps    chan *snmp.TrapPDU
}

func NewTrapProcessor() *TrapProcessor {
    tp := &TrapProcessor{
        byType:  make(map[int]int),
        byAgent: make(map[string]int),
        traps:   make(chan *snmp.TrapPDU, 100),
    }

    // Worker pour traitement asynchrone
    tp.wg.Add(1)
    go tp.worker()

    return tp
}

func (tp *TrapProcessor) Handle(trap *snmp.TrapPDU) {
    select {
    case tp.traps <- trap:
    default:
        log.Println("Buffer plein, trap ignoré")
    }
}

func (tp *TrapProcessor) worker() {
    defer tp.wg.Done()

    for trap := range tp.traps {
        tp.process(trap)
    }
}

func (tp *TrapProcessor) process(trap *snmp.TrapPDU) {
    tp.mu.Lock()
    tp.received++
    tp.byType[trap.GenericTrap]++
    tp.byAgent[trap.AgentAddress.String()]++
    tp.mu.Unlock()

    // Afficher le trap
    tp.printTrap(trap)

    // Optionnel: sauvegarder en JSON
    // tp.saveToFile(trap)
}

func (tp *TrapProcessor) printTrap(trap *snmp.TrapPDU) {
    fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
    fmt.Printf("║ TRAP REÇU                                                     ║\n")
    fmt.Printf("╠══════════════════════════════════════════════════════════════╣\n")
    fmt.Printf("║ Timestamp:  %s\n", trap.Timestamp.Format(time.RFC3339))
    fmt.Printf("║ Agent:      %s\n", trap.AgentAddress)
    fmt.Printf("║ Version:    %s\n", trap.Version)
    fmt.Printf("║ Community:  %s\n", trap.Community)
    fmt.Printf("║ Type:       %s (%d)\n", trapTypeName(trap.GenericTrap), trap.GenericTrap)

    if trap.GenericTrap == 6 { // Enterprise specific
        fmt.Printf("║ Enterprise: %s\n", trap.Enterprise)
        fmt.Printf("║ Specific:   %d\n", trap.SpecificTrap)
    }

    fmt.Printf("╠══════════════════════════════════════════════════════════════╣\n")
    fmt.Printf("║ Variables (%d):\n", len(trap.Variables))

    for i, v := range trap.Variables {
        fmt.Printf("║   [%d] %s\n", i+1, v.OID)
        fmt.Printf("║       Type:  %s\n", v.Type)
        fmt.Printf("║       Value: %s\n", formatTrapValue(v))
    }

    fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n")
    fmt.Println()
}

func (tp *TrapProcessor) PrintStats() {
    tp.mu.Lock()
    defer tp.mu.Unlock()

    fmt.Println("\n=== Statistiques ===")
    fmt.Printf("Total traps reçus: %d\n", tp.received)

    fmt.Println("\nPar type:")
    for t, count := range tp.byType {
        fmt.Printf("  %s: %d\n", trapTypeName(t), count)
    }

    fmt.Println("\nPar agent:")
    for agent, count := range tp.byAgent {
        fmt.Printf("  %s: %d\n", agent, count)
    }
}

func (tp *TrapProcessor) Close() {
    close(tp.traps)
    tp.wg.Wait()
}

func (tp *TrapProcessor) saveToFile(trap *snmp.TrapPDU) {
    data, _ := json.MarshalIndent(trap, "", "  ")

    filename := fmt.Sprintf("trap_%s_%d.json",
        trap.Timestamp.Format("20060102_150405"),
        trap.RequestID)

    os.WriteFile(filename, data, 0644)
}

func trapTypeName(t int) string {
    names := map[int]string{
        0: "coldStart",
        1: "warmStart",
        2: "linkDown",
        3: "linkUp",
        4: "authenticationFailure",
        5: "egpNeighborLoss",
        6: "enterpriseSpecific",
    }
    if name, ok := names[t]; ok {
        return name
    }
    return fmt.Sprintf("unknown(%d)", t)
}

func formatTrapValue(v snmp.Variable) string {
    switch val := v.Value.(type) {
    case []byte:
        // Essayer d'afficher comme string, sinon en hex
        if isPrintable(val) {
            return fmt.Sprintf("\"%s\"", string(val))
        }
        return fmt.Sprintf("0x%X", val)
    case uint32:
        if v.Type == snmp.TypeTimeTicks {
            return snmp.TimeTicksToString(val)
        }
        return fmt.Sprintf("%d", val)
    case int:
        return fmt.Sprintf("%d", val)
    case net.IP:
        return val.String()
    case snmp.OID:
        return val.String()
    default:
        return fmt.Sprintf("%v", val)
    }
}

func isPrintable(data []byte) bool {
    for _, b := range data {
        if b < 32 || b > 126 {
            return false
        }
    }
    return true
}
```

## Exécution

```bash
# Compiler
go build -o trap-listener main.go

# Exécuter (port 1162 ne nécessite pas root)
./trap-listener

# Pour le port standard 162 (nécessite root)
sudo ./trap-listener
```

## Tester avec snmptrap

```bash
# Envoyer un trap v2c
snmptrap -v 2c -c public localhost:1162 '' \
    1.3.6.1.4.1.8072.2.3.0.1 \
    1.3.6.1.4.1.8072.2.3.2.1 s "Test message"

# Envoyer un trap linkDown
snmptrap -v 2c -c public localhost:1162 '' \
    1.3.6.1.6.3.1.1.5.3 \
    1.3.6.1.2.1.2.2.1.1.1 i 1 \
    1.3.6.1.2.1.2.2.1.7.1 i 2 \
    1.3.6.1.2.1.2.2.1.8.1 i 2

# Envoyer un trap coldStart
snmptrap -v 2c -c public localhost:1162 '' \
    1.3.6.1.6.3.1.1.5.1
```

## Sortie attendue

```
Démarrage du listener sur :1162
En attente de traps... (Ctrl+C pour arrêter)

╔══════════════════════════════════════════════════════════════╗
║ TRAP REÇU                                                     ║
╠══════════════════════════════════════════════════════════════╣
║ Timestamp:  2026-02-02T10:30:45+01:00
║ Agent:      127.0.0.1
║ Version:    v2c
║ Community:  public
║ Type:       linkDown (2)
╠══════════════════════════════════════════════════════════════╣
║ Variables (3):
║   [1] 1.3.6.1.2.1.1.3.0
║       Type:  TimeTicks
║       Value: 5 days, 12:34:56.78
║   [2] 1.3.6.1.6.3.1.1.4.1.0
║       Type:  ObjectIdentifier
║       Value: 1.3.6.1.6.3.1.1.5.3
║   [3] 1.3.6.1.2.1.2.2.1.1.1
║       Type:  Integer
║       Value: 1
╚══════════════════════════════════════════════════════════════╝

^C
Arrêt en cours...

=== Statistiques ===
Total traps reçus: 3

Par type:
  coldStart: 1
  linkDown: 2

Par agent:
  127.0.0.1: 3
```

## Variantes

### Filtrage par community

```go
listener := snmp.NewTrapListener(handler,
    snmp.WithListenAddress(":1162"),
    snmp.WithTrapCommunity("mytraps"),  // Accepte uniquement "mytraps"
)
```

### Handler avec alertes

```go
func alertHandler(trap *snmp.TrapPDU) {
    // Envoyer une alerte pour certains types de traps
    switch trap.GenericTrap {
    case 2: // linkDown
        sendSlackAlert(fmt.Sprintf("Interface down sur %s", trap.AgentAddress))
    case 4: // authenticationFailure
        sendSecurityAlert(fmt.Sprintf("Auth failure de %s", trap.AgentAddress))
    }
}
```

### Persistance en base de données

```go
func dbHandler(db *sql.DB) func(*snmp.TrapPDU) {
    return func(trap *snmp.TrapPDU) {
        varsJSON, _ := json.Marshal(trap.Variables)

        _, err := db.Exec(`
            INSERT INTO traps
            (timestamp, agent, version, community, trap_type, variables)
            VALUES (?, ?, ?, ?, ?, ?)`,
            trap.Timestamp,
            trap.AgentAddress.String(),
            trap.Version.String(),
            trap.Community,
            trap.GenericTrap,
            varsJSON,
        )

        if err != nil {
            log.Printf("DB error: %v", err)
        }
    }
}
```

## Voir aussi

- [Trap Listener](../trap-listener.md)
- [Options de configuration](../options.md)
