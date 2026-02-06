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
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Client is an SNMP client.
type Client struct {
	opts    *ClientOptions
	conn    net.Conn
	state   atomic.Int32
	mu      sync.RWMutex
	wg      sync.WaitGroup
	done    chan struct{}
	metrics *Metrics
	logger  *slog.Logger

	// Request ID management
	requestID     int32
	requestIDLock sync.Mutex

	// Pending requests
	pending     map[int32]chan *PDU
	pendingLock sync.RWMutex
}

// NewClient creates a new SNMP client.
func NewClient(opts ...Option) *Client {
	options := NewClientOptions()
	for _, opt := range opts {
		opt(options)
	}

	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}

	c := &Client{
		opts:      options,
		done:      make(chan struct{}),
		metrics:   NewMetrics(),
		logger:    logger,
		pending:   make(map[int32]chan *PDU),
		requestID: rand.Int31(),
	}

	return c
}

// Connect establishes a connection to the SNMP agent.
func (c *Client) Connect(ctx context.Context) error {
	if !c.state.CompareAndSwap(int32(StateDisconnected), int32(StateConnecting)) {
		return ErrAlreadyConnected
	}

	if c.opts.Target == "" {
		c.state.Store(int32(StateDisconnected))
		return fmt.Errorf("snmp: no target configured")
	}

	c.metrics.ConnectionAttempts.Add(1)

	// Build address
	addr := fmt.Sprintf("%s:%d", c.opts.Target, c.opts.Port)

	// Connect with timeout
	dialer := net.Dialer{Timeout: c.opts.Timeout}
	conn, err := dialer.DialContext(ctx, "udp", addr)
	if err != nil {
		c.state.Store(int32(StateDisconnected))
		return fmt.Errorf("snmp: connection failed: %w", err)
	}

	c.conn = conn
	c.state.Store(int32(StateConnected))
	c.metrics.ActiveConnections.Add(1)

	// Reset channels
	c.done = make(chan struct{})

	// Start response reader
	c.wg.Add(1)
	go c.readLoop()

	// Call OnConnect callback
	if c.opts.OnConnect != nil {
		go c.opts.OnConnect(c)
	}

	c.logger.Info("connected to SNMP agent",
		"target", addr,
		"version", c.opts.Version)

	return nil
}

// Disconnect closes the connection.
func (c *Client) Disconnect(ctx context.Context) error {
	if !c.state.CompareAndSwap(int32(StateConnected), int32(StateDisconnecting)) {
		return ErrNotConnected
	}

	c.state.Store(int32(StateDisconnected))
	c.metrics.ActiveConnections.Add(-1)

	close(c.done)
	c.wg.Wait()

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	// Fail pending requests
	c.failPending(ErrClientClosed)

	c.logger.Info("disconnected from SNMP agent")
	return nil
}

func (c *Client) readLoop() {
	defer c.wg.Done()

	buf := make([]byte, 65535)
	for {
		select {
		case <-c.done:
			return
		default:
		}

		// Set read deadline
		c.conn.SetReadDeadline(time.Now().Add(c.opts.Timeout * 2))

		n, err := c.conn.Read(buf)
		if err != nil {
			select {
			case <-c.done:
				return
			default:
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				c.handleConnectionLost(err)
				return
			}
		}

		// Decode message
		msg, err := DecodeMessage(buf[:n])
		if err != nil {
			c.logger.Warn("failed to decode response", "error", err)
			c.metrics.Errors.Add(1)
			continue
		}

		c.metrics.ResponsesReceived.Add(1)
		c.metrics.VarbindsReceived.Add(int64(len(msg.PDU.Variables)))

		// Find pending request
		c.pendingLock.RLock()
		ch, ok := c.pending[msg.PDU.RequestID]
		c.pendingLock.RUnlock()

		if ok {
			select {
			case ch <- msg.PDU:
			default:
			}
		}
	}
}

func (c *Client) handleConnectionLost(err error) {
	if !c.state.CompareAndSwap(int32(StateConnected), int32(StateDisconnected)) {
		return
	}

	c.metrics.ActiveConnections.Add(-1)
	close(c.done)

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	c.logger.Info("connection lost", "error", err)

	if c.opts.OnConnectionLost != nil {
		go c.opts.OnConnectionLost(c, err)
	}

	c.failPending(err)

	if c.opts.AutoReconnect {
		go c.reconnect()
	}
}

func (c *Client) failPending(err error) {
	c.pendingLock.Lock()
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
	c.pendingLock.Unlock()
}

func (c *Client) reconnect() {
	backoff := c.opts.ConnectRetryInterval
	retries := 0

	for {
		if c.opts.OnReconnecting != nil {
			c.opts.OnReconnecting(c, c.opts)
		}

		c.metrics.ReconnectAttempts.Add(1)

		ctx, cancel := context.WithTimeout(context.Background(), c.opts.Timeout)
		err := c.Connect(ctx)
		cancel()

		if err == nil {
			return
		}

		c.logger.Warn("reconnection failed", "error", err, "retry_in", backoff)

		retries++
		if c.opts.MaxRetries > 0 && retries >= c.opts.MaxRetries {
			c.logger.Error("max reconnection attempts reached")
			return
		}

		time.Sleep(backoff)

		// Exponential backoff with jitter
		backoff = time.Duration(float64(backoff) * (1.5 + rand.Float64()*0.5))
		if backoff > c.opts.MaxReconnectInterval {
			backoff = c.opts.MaxReconnectInterval
		}
	}
}

func (c *Client) nextRequestID() int32 {
	c.requestIDLock.Lock()
	defer c.requestIDLock.Unlock()

	c.requestID++
	if c.requestID <= 0 {
		c.requestID = 1
	}
	return c.requestID
}

func (c *Client) sendRequest(ctx context.Context, pdu *PDU) (*PDU, error) {
	if c.State() != StateConnected {
		return nil, ErrNotConnected
	}

	// Create response channel
	respCh := make(chan *PDU, 1)
	c.pendingLock.Lock()
	c.pending[pdu.RequestID] = respCh
	c.pendingLock.Unlock()

	defer func() {
		c.pendingLock.Lock()
		delete(c.pending, pdu.RequestID)
		c.pendingLock.Unlock()
	}()

	// Encode message
	msg := &Message{
		Version:   c.opts.Version,
		Community: c.opts.Community,
		PDU:       pdu,
	}

	data, err := msg.Encode()
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}

	// Send with retries
	var lastErr error
	for retry := 0; retry <= c.opts.Retries; retry++ {
		if retry > 0 {
			c.metrics.Retries.Add(1)
			c.logger.Debug("retrying request", "retry", retry, "request_id", pdu.RequestID)
		}

		start := time.Now()

		// Set write deadline
		c.conn.SetWriteDeadline(time.Now().Add(c.opts.Timeout))
		_, err := c.conn.Write(data)
		if err != nil {
			lastErr = fmt.Errorf("write failed: %w", err)
			continue
		}

		c.metrics.RequestsSent.Add(1)
		c.metrics.VarbindsSent.Add(int64(len(pdu.Variables)))

		// Wait for response
		select {
		case resp, ok := <-respCh:
			if !ok {
				return nil, ErrClientClosed
			}
			c.metrics.RequestLatency.ObserveDuration(time.Since(start))

			// Check for errors
			if resp.ErrorStatus != NoError {
				var oid OID
				if resp.ErrorIndex > 0 && resp.ErrorIndex <= len(pdu.Variables) {
					oid = pdu.Variables[resp.ErrorIndex-1].OID
				}
				return resp, NewSNMPError(resp.ErrorStatus, resp.ErrorIndex, oid)
			}

			return resp, nil

		case <-time.After(c.opts.Timeout):
			lastErr = ErrTimeout
			c.metrics.Timeouts.Add(1)

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, lastErr
}

// Get performs an SNMP GET request.
func (c *Client) Get(ctx context.Context, oids ...OID) ([]Variable, error) {
	c.metrics.GetRequests.Add(1)

	pdu := NewGetRequest(c.nextRequestID(), oids...)
	resp, err := c.sendRequest(ctx, pdu)
	if err != nil {
		c.metrics.Errors.Add(1)
		return nil, err
	}

	return resp.Variables, nil
}

// GetNext performs an SNMP GET-NEXT request.
func (c *Client) GetNext(ctx context.Context, oids ...OID) ([]Variable, error) {
	c.metrics.GetNextRequests.Add(1)

	pdu := NewGetNextRequest(c.nextRequestID(), oids...)
	resp, err := c.sendRequest(ctx, pdu)
	if err != nil {
		c.metrics.Errors.Add(1)
		return nil, err
	}

	return resp.Variables, nil
}

// GetBulk performs an SNMP GET-BULK request (v2c/v3 only).
func (c *Client) GetBulk(ctx context.Context, nonRepeaters, maxRepetitions int, oids ...OID) ([]Variable, error) {
	if c.opts.Version == Version1 {
		return nil, fmt.Errorf("snmp: GetBulk not supported in SNMPv1")
	}

	c.metrics.GetBulkRequests.Add(1)

	pdu := NewGetBulkRequest(c.nextRequestID(), nonRepeaters, maxRepetitions, oids...)
	resp, err := c.sendRequest(ctx, pdu)
	if err != nil {
		c.metrics.Errors.Add(1)
		return nil, err
	}

	return resp.Variables, nil
}

// Set performs an SNMP SET request.
func (c *Client) Set(ctx context.Context, variables ...Variable) ([]Variable, error) {
	c.metrics.SetRequests.Add(1)

	pdu := NewSetRequest(c.nextRequestID(), variables...)
	resp, err := c.sendRequest(ctx, pdu)
	if err != nil {
		c.metrics.Errors.Add(1)
		return nil, err
	}

	return resp.Variables, nil
}

// Walk performs an SNMP walk starting from the given OID.
func (c *Client) Walk(ctx context.Context, rootOID OID) ([]Variable, error) {
	c.metrics.WalkRequests.Add(1)

	var results []Variable
	currentOID := rootOID.Copy()

	for {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		var vars []Variable
		var err error

		if c.opts.Version == Version1 {
			vars, err = c.GetNext(ctx, currentOID)
		} else {
			vars, err = c.GetBulk(ctx, c.opts.NonRepeaters, c.opts.MaxRepetitions, currentOID)
		}

		if err != nil {
			// Check if it's an expected end condition
			if IsEndOfMIB(err) || IsNoSuchObject(err) || IsNoSuchInstance(err) {
				break
			}
			c.metrics.Errors.Add(1)
			return results, err
		}

		if len(vars) == 0 {
			break
		}

		for _, v := range vars {
			// Check if we're still under the root OID
			if !v.OID.HasPrefix(rootOID) {
				return results, nil
			}

			// Check for end-of-mib markers
			if v.Type == TypeEndOfMibView || v.Type == TypeNoSuchObject || v.Type == TypeNoSuchInstance {
				return results, nil
			}

			results = append(results, v)
			currentOID = v.OID
		}

		// For v1, we only get one result per request
		if c.opts.Version == Version1 && len(vars) > 0 {
			currentOID = vars[0].OID
		} else if len(vars) > 0 {
			currentOID = vars[len(vars)-1].OID
		}
	}

	return results, nil
}

// WalkFunc walks the MIB tree and calls fn for each variable.
func (c *Client) WalkFunc(ctx context.Context, rootOID OID, fn func(Variable) error) error {
	c.metrics.WalkRequests.Add(1)

	currentOID := rootOID.Copy()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var vars []Variable
		var err error

		if c.opts.Version == Version1 {
			vars, err = c.GetNext(ctx, currentOID)
		} else {
			vars, err = c.GetBulk(ctx, c.opts.NonRepeaters, c.opts.MaxRepetitions, currentOID)
		}

		if err != nil {
			if IsEndOfMIB(err) || IsNoSuchObject(err) || IsNoSuchInstance(err) {
				return nil
			}
			c.metrics.Errors.Add(1)
			return err
		}

		if len(vars) == 0 {
			return nil
		}

		for _, v := range vars {
			if !v.OID.HasPrefix(rootOID) {
				return nil
			}

			if v.Type == TypeEndOfMibView || v.Type == TypeNoSuchObject || v.Type == TypeNoSuchInstance {
				return nil
			}

			if err := fn(v); err != nil {
				return err
			}

			currentOID = v.OID
		}

		if c.opts.Version == Version1 && len(vars) > 0 {
			currentOID = vars[0].OID
		} else if len(vars) > 0 {
			currentOID = vars[len(vars)-1].OID
		}
	}
}

// State returns the current connection state.
func (c *Client) State() ConnectionState {
	return ConnectionState(c.state.Load())
}

// IsConnected returns true if connected.
func (c *Client) IsConnected() bool {
	return c.State() == StateConnected
}

// Metrics returns the client metrics.
func (c *Client) Metrics() *Metrics {
	return c.metrics
}

// Options returns the client options.
func (c *Client) Options() *ClientOptions {
	return c.opts
}
