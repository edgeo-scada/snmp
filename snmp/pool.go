// Copyright 2025 Edgeo SCADA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package snmp

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// Pool manages a pool of SNMP client connections.
type Pool struct {
	opts       *PoolOptions
	clients    []*poolClient
	clientOpts []Option
	mu         sync.RWMutex
	robin      uint64
	done       chan struct{}
	wg         sync.WaitGroup
	metrics    *PoolMetrics
}

type poolClient struct {
	client   *Client
	lastUsed time.Time
	inFlight int64
	mu       sync.Mutex
}

// NewPool creates a new connection pool.
func NewPool(opts ...PoolOption) *Pool {
	options := NewPoolOptions()
	for _, opt := range opts {
		opt(options)
	}

	p := &Pool{
		opts:       options,
		clients:    make([]*poolClient, options.Size),
		clientOpts: options.ClientOptions,
		done:       make(chan struct{}),
		metrics:    &PoolMetrics{},
	}

	return p
}

// Connect initializes all connections in the pool.
func (p *Pool) Connect(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var firstErr error
	successCount := 0

	for i := 0; i < p.opts.Size; i++ {
		client := NewClient(p.clientOpts...)
		if err := client.Connect(ctx); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		p.clients[i] = &poolClient{
			client:   client,
			lastUsed: time.Now(),
		}
		successCount++
	}

	p.metrics.TotalClients.Set(int64(successCount))
	p.metrics.HealthyClients.Set(int64(successCount))

	if successCount == 0 {
		return firstErr
	}

	// Start health checker
	p.wg.Add(1)
	go p.healthChecker()

	return nil
}

// Close closes all connections in the pool.
func (p *Pool) Close() error {
	close(p.done)
	p.wg.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for _, pc := range p.clients {
		if pc != nil && pc.client != nil {
			if err := pc.client.Disconnect(context.Background()); err != nil {
				lastErr = err
			}
		}
	}

	p.clients = nil
	p.metrics.TotalClients.Set(0)
	p.metrics.HealthyClients.Set(0)

	return lastErr
}

// Get returns a client from the pool using round-robin selection.
func (p *Pool) Get() (*Client, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.clients) == 0 {
		return nil, errors.New("snmp: pool is empty")
	}

	p.metrics.TotalRequests.Add(1)

	// Round-robin with fallback to first healthy
	start := atomic.AddUint64(&p.robin, 1) % uint64(len(p.clients))

	for i := 0; i < len(p.clients); i++ {
		idx := (int(start) + i) % len(p.clients)
		pc := p.clients[idx]
		if pc != nil && pc.client != nil && pc.client.IsConnected() {
			pc.mu.Lock()
			pc.lastUsed = time.Now()
			atomic.AddInt64(&pc.inFlight, 1)
			pc.mu.Unlock()
			return pc.client, nil
		}
	}

	p.metrics.FailedRequests.Add(1)
	return nil, errors.New("snmp: no healthy connections available")
}

// Release returns a client to the pool (decrements in-flight counter).
func (p *Pool) Release(client *Client) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, pc := range p.clients {
		if pc != nil && pc.client == client {
			atomic.AddInt64(&pc.inFlight, -1)
			return
		}
	}
}

// Get performs a GET using a pooled connection.
func (p *Pool) GetOIDs(ctx context.Context, oids ...OID) ([]Variable, error) {
	client, err := p.Get()
	if err != nil {
		return nil, err
	}
	defer p.Release(client)

	return client.Get(ctx, oids...)
}

// GetNext performs a GET-NEXT using a pooled connection.
func (p *Pool) GetNext(ctx context.Context, oids ...OID) ([]Variable, error) {
	client, err := p.Get()
	if err != nil {
		return nil, err
	}
	defer p.Release(client)

	return client.GetNext(ctx, oids...)
}

// GetBulk performs a GET-BULK using a pooled connection.
func (p *Pool) GetBulk(ctx context.Context, nonRepeaters, maxRepetitions int, oids ...OID) ([]Variable, error) {
	client, err := p.Get()
	if err != nil {
		return nil, err
	}
	defer p.Release(client)

	return client.GetBulk(ctx, nonRepeaters, maxRepetitions, oids...)
}

// Set performs a SET using a pooled connection.
func (p *Pool) Set(ctx context.Context, variables ...Variable) ([]Variable, error) {
	client, err := p.Get()
	if err != nil {
		return nil, err
	}
	defer p.Release(client)

	return client.Set(ctx, variables...)
}

// Walk performs a walk using a pooled connection.
func (p *Pool) Walk(ctx context.Context, rootOID OID) ([]Variable, error) {
	client, err := p.Get()
	if err != nil {
		return nil, err
	}
	defer p.Release(client)

	return client.Walk(ctx, rootOID)
}

func (p *Pool) healthChecker() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.opts.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			p.checkHealth()
		}
	}
}

func (p *Pool) checkHealth() {
	p.mu.Lock()
	defer p.mu.Unlock()

	healthy := int64(0)
	for i, pc := range p.clients {
		if pc == nil || pc.client == nil {
			// Try to create a new connection
			client := NewClient(p.clientOpts...)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := client.Connect(ctx); err == nil {
				p.clients[i] = &poolClient{
					client:   client,
					lastUsed: time.Now(),
				}
				healthy++
			}
			cancel()
			continue
		}

		if !pc.client.IsConnected() {
			// Try to reconnect
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := pc.client.Connect(ctx); err != nil {
				// Replace with new client
				pc.client = nil
				client := NewClient(p.clientOpts...)
				if err := client.Connect(ctx); err == nil {
					p.clients[i] = &poolClient{
						client:   client,
						lastUsed: time.Now(),
					}
					healthy++
				}
			} else {
				healthy++
			}
			cancel()
			continue
		}

		// Check idle timeout
		pc.mu.Lock()
		idle := time.Since(pc.lastUsed)
		inFlight := atomic.LoadInt64(&pc.inFlight)
		pc.mu.Unlock()

		if idle > p.opts.MaxIdleTime && inFlight == 0 {
			// Close idle connection but keep slot for later
			pc.client.Disconnect(context.Background())
			continue
		}

		healthy++
	}

	p.metrics.HealthyClients.Set(healthy)
}

// Metrics returns the pool metrics.
func (p *Pool) Metrics() *PoolMetrics {
	return p.metrics
}

// Size returns the pool size.
func (p *Pool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.clients)
}

// HealthyCount returns the number of healthy connections.
func (p *Pool) HealthyCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, pc := range p.clients {
		if pc != nil && pc.client != nil && pc.client.IsConnected() {
			count++
		}
	}
	return count
}
