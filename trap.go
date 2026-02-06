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
	"log/slog"
	"net"
	"sync"
)

// TrapListener listens for SNMP traps.
type TrapListener struct {
	opts    *TrapListenerOptions
	conn    *net.UDPConn
	handler TrapHandler
	logger  *slog.Logger
	done    chan struct{}
	wg      sync.WaitGroup
	metrics *Metrics
}

// NewTrapListener creates a new trap listener.
func NewTrapListener(handler TrapHandler, opts ...TrapListenerOption) *TrapListener {
	options := NewTrapListenerOptions()
	for _, opt := range opts {
		opt(options)
	}

	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &TrapListener{
		opts:    options,
		handler: handler,
		logger:  logger,
		done:    make(chan struct{}),
		metrics: NewMetrics(),
	}
}

// Start starts listening for traps.
func (l *TrapListener) Start(ctx context.Context) error {
	addr, err := net.ResolveUDPAddr("udp", l.opts.Address)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	l.conn = conn
	l.logger.Info("trap listener started", "address", l.opts.Address)

	l.wg.Add(1)
	go l.listen()

	return nil
}

// Stop stops the trap listener.
func (l *TrapListener) Stop() error {
	close(l.done)
	if l.conn != nil {
		l.conn.Close()
	}
	l.wg.Wait()
	l.logger.Info("trap listener stopped")
	return nil
}

func (l *TrapListener) listen() {
	defer l.wg.Done()

	buf := make([]byte, 65535)
	for {
		select {
		case <-l.done:
			return
		default:
		}

		n, remoteAddr, err := l.conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-l.done:
				return
			default:
				l.logger.Warn("error reading trap", "error", err)
				continue
			}
		}

		l.metrics.TrapsReceived.Add(1)

		// Try to decode the trap
		trap, err := l.decodeTrap(buf[:n], remoteAddr)
		if err != nil {
			l.logger.Warn("failed to decode trap", "error", err, "source", remoteAddr)
			l.metrics.Errors.Add(1)
			continue
		}

		// Check community if specified
		if l.opts.Community != "" && trap.Community != l.opts.Community {
			l.logger.Warn("trap community mismatch",
				"expected", l.opts.Community,
				"received", trap.Community,
				"source", remoteAddr)
			continue
		}

		// Call handler
		if l.handler != nil {
			go l.handler(trap)
		}
	}
}

func (l *TrapListener) decodeTrap(data []byte, remoteAddr *net.UDPAddr) (*TrapPDU, error) {
	// First, try to decode as a regular SNMP message (v2c trap)
	msg, err := DecodeMessage(data)
	if err != nil {
		// Try v1 trap format
		return l.decodeV1Trap(data, remoteAddr)
	}

	trap := &TrapPDU{
		Version:       msg.Version,
		Community:     msg.Community,
		SourceAddress: remoteAddr.String(),
	}

	if msg.PDU.Type == PDUTrapV2 || msg.PDU.Type == PDUInformRequest {
		trap.Variables = msg.PDU.Variables

		// Extract sysUpTime and snmpTrapOID from varbinds
		for _, v := range msg.PDU.Variables {
			if v.OID.Equal(OIDSysUpTime) {
				if val, ok := v.Value.(uint32); ok {
					trap.Timestamp = val
				}
			}
		}
	}

	return trap, nil
}

func (l *TrapListener) decodeV1Trap(data []byte, remoteAddr *net.UDPAddr) (*TrapPDU, error) {
	msg, err := DecodeTrapV1Message(data)
	if err != nil {
		return nil, err
	}

	// Convert agent address
	var agentAddr string
	if len(msg.PDU.AgentAddress) == 4 {
		agentAddr = net.IP(msg.PDU.AgentAddress).String()
	}

	return &TrapPDU{
		Version:       msg.Version,
		Community:     msg.Community,
		Enterprise:    msg.PDU.Enterprise,
		AgentAddress:  agentAddr,
		GenericTrap:   msg.PDU.GenericTrap,
		SpecificTrap:  msg.PDU.SpecificTrap,
		Timestamp:     msg.PDU.Timestamp,
		Variables:     msg.PDU.Variables,
		SourceAddress: remoteAddr.String(),
	}, nil
}

// Metrics returns the listener metrics.
func (l *TrapListener) Metrics() *Metrics {
	return l.metrics
}

// Address returns the listen address.
func (l *TrapListener) Address() string {
	if l.conn != nil {
		return l.conn.LocalAddr().String()
	}
	return l.opts.Address
}
