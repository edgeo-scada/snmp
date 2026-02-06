# Métriques

La librairie SNMP inclut un système de métriques intégré pour le monitoring et l'observabilité des opérations SNMP.

## Vue d'ensemble

Le système de métriques permet de suivre :

- **Compteurs** : Nombre total de requêtes, erreurs, etc.
- **Jauges** : Valeurs instantanées (connexions actives, etc.)
- **Histogrammes de latence** : Distribution des temps de réponse

## Types de métriques

### Counter

Compteur monotone croissant, idéal pour compter des événements.

```go
type Counter struct {
    value uint64
}

func (c *Counter) Inc()              // Incrémenter de 1
func (c *Counter) Add(delta uint64)  // Ajouter une valeur
func (c *Counter) Value() uint64     // Obtenir la valeur
func (c *Counter) Reset()            // Réinitialiser à 0
```

### Gauge

Valeur qui peut augmenter ou diminuer.

```go
type Gauge struct {
    value int64
}

func (g *Gauge) Set(value int64)     // Définir une valeur
func (g *Gauge) Inc()                // Incrémenter de 1
func (g *Gauge) Dec()                // Décrémenter de 1
func (g *Gauge) Add(delta int64)     // Ajouter (positif ou négatif)
func (g *Gauge) Value() int64        // Obtenir la valeur
```

### LatencyHistogram

Histogramme pour mesurer la distribution des latences.

```go
type LatencyHistogram struct {
    buckets []uint64    // Compteurs par bucket
    bounds  []float64   // Limites des buckets (ms)
    sum     float64     // Somme totale
    count   uint64      // Nombre total d'observations
}

func (h *LatencyHistogram) Observe(latency time.Duration)
func (h *LatencyHistogram) Percentile(p float64) float64
func (h *LatencyHistogram) Mean() float64
func (h *LatencyHistogram) Count() uint64
func (h *LatencyHistogram) Sum() float64
```

## Structure des métriques

```go
type Metrics struct {
    // Compteurs de requêtes
    RequestsTotal     *Counter  // Total des requêtes
    RequestsSuccess   *Counter  // Requêtes réussies
    RequestsError     *Counter  // Requêtes en erreur
    RequestsTimeout   *Counter  // Timeouts

    // Compteurs par type d'opération
    GetRequests       *Counter  // Requêtes GET
    GetNextRequests   *Counter  // Requêtes GET-NEXT
    GetBulkRequests   *Counter  // Requêtes GET-BULK
    SetRequests       *Counter  // Requêtes SET
    WalkRequests      *Counter  // Opérations WALK

    // Compteurs de traps
    TrapsReceived     *Counter  // Traps reçus
    TrapsProcessed    *Counter  // Traps traités
    TrapsDropped      *Counter  // Traps rejetés

    // Jauges
    ActiveConnections *Gauge    // Connexions actives
    PoolSize          *Gauge    // Taille du pool
    PoolAvailable     *Gauge    // Connexions disponibles

    // Latences
    RequestLatency    *LatencyHistogram  // Latence des requêtes
    WalkLatency       *LatencyHistogram  // Latence des walks
}
```

## Utilisation

### Accès aux métriques du client

```go
client, _ := snmp.NewClient(ctx, opts...)

// Après quelques opérations...
vars, _ := client.Get(ctx, oids...)

// Accéder aux métriques
metrics := client.Metrics()

fmt.Printf("Total requêtes: %d\n", metrics.RequestsTotal.Value())
fmt.Printf("Succès: %d\n", metrics.RequestsSuccess.Value())
fmt.Printf("Erreurs: %d\n", metrics.RequestsError.Value())
fmt.Printf("Latence moyenne: %.2fms\n", metrics.RequestLatency.Mean())
fmt.Printf("P99 latence: %.2fms\n", metrics.RequestLatency.Percentile(99))
```

### Accès aux métriques du pool

```go
pool, _ := snmp.NewPool(ctx, opts...)

metrics := pool.Metrics()

fmt.Printf("Connexions actives: %d\n", metrics.ActiveConnections.Value())
fmt.Printf("Pool size: %d\n", metrics.PoolSize.Value())
fmt.Printf("Disponibles: %d\n", metrics.PoolAvailable.Value())
```

### Accès aux métriques du TrapListener

```go
listener := snmp.NewTrapListener(handler, opts...)

metrics := listener.Metrics()

fmt.Printf("Traps reçus: %d\n", metrics.TrapsReceived.Value())
fmt.Printf("Traps traités: %d\n", metrics.TrapsProcessed.Value())
fmt.Printf("Traps rejetés: %d\n", metrics.TrapsDropped.Value())
```

## Histogramme de latence

### Buckets par défaut

Les buckets par défaut couvrent une plage de 0.1ms à 10s :

```go
[]float64{0.1, 0.5, 1, 2.5, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000}
```

### Percentiles

```go
latency := metrics.RequestLatency

fmt.Printf("P50 (médiane): %.2fms\n", latency.Percentile(50))
fmt.Printf("P90: %.2fms\n", latency.Percentile(90))
fmt.Printf("P95: %.2fms\n", latency.Percentile(95))
fmt.Printf("P99: %.2fms\n", latency.Percentile(99))
fmt.Printf("P99.9: %.2fms\n", latency.Percentile(99.9))
```

### Statistiques

```go
latency := metrics.RequestLatency

fmt.Printf("Observations: %d\n", latency.Count())
fmt.Printf("Somme: %.2fms\n", latency.Sum())
fmt.Printf("Moyenne: %.2fms\n", latency.Mean())
```

## Intégration Prometheus

### Export des métriques

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// Créer les métriques Prometheus
var (
    snmpRequests = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "snmp_requests_total",
            Help: "Total number of SNMP requests",
        },
        []string{"operation", "status"},
    )

    snmpLatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "snmp_request_duration_seconds",
            Help:    "SNMP request latency in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"operation"},
    )

    snmpConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "snmp_active_connections",
            Help: "Number of active SNMP connections",
        },
    )
)

func init() {
    prometheus.MustRegister(snmpRequests)
    prometheus.MustRegister(snmpLatency)
    prometheus.MustRegister(snmpConnections)
}

// Exporter périodiquement
func exportMetrics(client *snmp.Client) {
    metrics := client.Metrics()

    snmpRequests.WithLabelValues("get", "success").Add(
        float64(metrics.GetRequests.Value()))
    snmpRequests.WithLabelValues("total", "error").Add(
        float64(metrics.RequestsError.Value()))

    snmpConnections.Set(float64(metrics.ActiveConnections.Value()))
}

// Handler HTTP
http.Handle("/metrics", promhttp.Handler())
```

## Intégration OpenTelemetry

```go
import (
    "go.opentelemetry.io/otel/metric"
)

func setupOTelMetrics(meter metric.Meter) {
    requestCounter, _ := meter.Int64Counter(
        "snmp.requests.total",
        metric.WithDescription("Total SNMP requests"),
    )

    latencyHistogram, _ := meter.Float64Histogram(
        "snmp.request.duration",
        metric.WithDescription("Request duration in milliseconds"),
        metric.WithUnit("ms"),
    )

    // Utiliser dans le code
    requestCounter.Add(ctx, 1)
    latencyHistogram.Record(ctx, latencyMs)
}
```

## Monitoring en temps réel

### Dashboard simple

```go
func printMetricsDashboard(client *snmp.Client) {
    metrics := client.Metrics()

    fmt.Println("\n=== SNMP Metrics Dashboard ===")
    fmt.Println()

    // Requêtes
    fmt.Println("Requests:")
    fmt.Printf("  Total:    %d\n", metrics.RequestsTotal.Value())
    fmt.Printf("  Success:  %d\n", metrics.RequestsSuccess.Value())
    fmt.Printf("  Errors:   %d\n", metrics.RequestsError.Value())
    fmt.Printf("  Timeouts: %d\n", metrics.RequestsTimeout.Value())
    fmt.Println()

    // Par opération
    fmt.Println("By Operation:")
    fmt.Printf("  GET:      %d\n", metrics.GetRequests.Value())
    fmt.Printf("  GET-NEXT: %d\n", metrics.GetNextRequests.Value())
    fmt.Printf("  GET-BULK: %d\n", metrics.GetBulkRequests.Value())
    fmt.Printf("  SET:      %d\n", metrics.SetRequests.Value())
    fmt.Printf("  WALK:     %d\n", metrics.WalkRequests.Value())
    fmt.Println()

    // Latences
    fmt.Println("Latency (ms):")
    fmt.Printf("  Mean:  %.2f\n", metrics.RequestLatency.Mean())
    fmt.Printf("  P50:   %.2f\n", metrics.RequestLatency.Percentile(50))
    fmt.Printf("  P95:   %.2f\n", metrics.RequestLatency.Percentile(95))
    fmt.Printf("  P99:   %.2f\n", metrics.RequestLatency.Percentile(99))
    fmt.Println()

    // Taux d'erreur
    total := metrics.RequestsTotal.Value()
    if total > 0 {
        errorRate := float64(metrics.RequestsError.Value()) / float64(total) * 100
        fmt.Printf("Error Rate: %.2f%%\n", errorRate)
    }
}
```

### Monitoring périodique

```go
func startMetricsReporter(client *snmp.Client, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    var lastTotal uint64

    for range ticker.C {
        metrics := client.Metrics()
        currentTotal := metrics.RequestsTotal.Value()

        // Calculer le débit
        requestsPerSecond := float64(currentTotal-lastTotal) / interval.Seconds()
        lastTotal = currentTotal

        log.Printf("[Metrics] RPS: %.2f, Active: %d, P99: %.2fms",
            requestsPerSecond,
            metrics.ActiveConnections.Value(),
            metrics.RequestLatency.Percentile(99),
        )
    }
}
```

## Réinitialisation des métriques

```go
// Réinitialiser un compteur
metrics.RequestsTotal.Reset()

// Réinitialiser toutes les métriques
func resetAllMetrics(metrics *snmp.Metrics) {
    metrics.RequestsTotal.Reset()
    metrics.RequestsSuccess.Reset()
    metrics.RequestsError.Reset()
    metrics.RequestsTimeout.Reset()
    metrics.GetRequests.Reset()
    metrics.GetNextRequests.Reset()
    metrics.GetBulkRequests.Reset()
    metrics.SetRequests.Reset()
    metrics.WalkRequests.Reset()
}
```

## Thread Safety

Toutes les métriques sont thread-safe et utilisent des opérations atomiques :

```go
// Sûr d'être appelé depuis plusieurs goroutines
go func() {
    metrics.RequestsTotal.Inc()
}()

go func() {
    metrics.RequestLatency.Observe(latency)
}()
```

## Voir aussi

- [Client SNMP](client.md)
- [Pool de connexions](pool.md)
