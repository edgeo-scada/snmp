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
	"errors"
	"fmt"
)

// Standard errors.
var (
	ErrNotConnected     = errors.New("snmp: not connected")
	ErrAlreadyConnected = errors.New("snmp: already connected")
	ErrConnectionLost   = errors.New("snmp: connection lost")
	ErrTimeout          = errors.New("snmp: operation timed out")
	ErrInvalidOID       = errors.New("snmp: invalid OID")
	ErrInvalidPacket    = errors.New("snmp: invalid packet")
	ErrInvalidPDU       = errors.New("snmp: invalid PDU")
	ErrInvalidType      = errors.New("snmp: invalid type")
	ErrInvalidLength    = errors.New("snmp: invalid length")
	ErrInvalidValue     = errors.New("snmp: invalid value")
	ErrInvalidVersion   = errors.New("snmp: invalid SNMP version")
	ErrInvalidCommunity = errors.New("snmp: invalid community string")
	ErrPacketTooLarge   = errors.New("snmp: packet too large")
	ErrMalformedPacket  = errors.New("snmp: malformed packet")
	ErrNoResponse       = errors.New("snmp: no response received")
	ErrEndOfMIB         = errors.New("snmp: end of MIB view")
	ErrNoSuchObject     = errors.New("snmp: no such object")
	ErrNoSuchInstance   = errors.New("snmp: no such instance")
	ErrRequestIDMismatch = errors.New("snmp: request ID mismatch")
	ErrAuthFailure      = errors.New("snmp: authentication failure")
	ErrPrivFailure      = errors.New("snmp: privacy failure")
	ErrClientClosed     = errors.New("snmp: client closed")
)

// SNMPError represents an SNMP protocol error.
type SNMPError struct {
	Status      ErrorStatus
	Index       int
	Message     string
	RequestOID  OID
}

// Error implements the error interface.
func (e *SNMPError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("snmp: %s (index %d): %s", e.Status.String(), e.Index, e.Message)
	}
	if e.RequestOID != nil {
		return fmt.Sprintf("snmp: %s at index %d (OID: %s)", e.Status.String(), e.Index, e.RequestOID)
	}
	return fmt.Sprintf("snmp: %s at index %d", e.Status.String(), e.Index)
}

// NewSNMPError creates a new SNMP error.
func NewSNMPError(status ErrorStatus, index int, oid OID) *SNMPError {
	return &SNMPError{
		Status:     status,
		Index:      index,
		RequestOID: oid,
	}
}

// IsTimeout returns true if the error is a timeout error.
func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout)
}

// IsEndOfMIB returns true if the error indicates end of MIB view.
func IsEndOfMIB(err error) bool {
	return errors.Is(err, ErrEndOfMIB)
}

// IsNoSuchObject returns true if the error indicates no such object.
func IsNoSuchObject(err error) bool {
	return errors.Is(err, ErrNoSuchObject)
}

// IsNoSuchInstance returns true if the error indicates no such instance.
func IsNoSuchInstance(err error) bool {
	return errors.Is(err, ErrNoSuchInstance)
}

// ErrorStatusToError converts an error status to an error.
func ErrorStatusToError(status ErrorStatus, index int, oid OID) error {
	if status == NoError {
		return nil
	}
	return NewSNMPError(status, index, oid)
}

// ParseError represents a packet parsing error.
type ParseError struct {
	Message string
	Offset  int
	Data    []byte
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	if e.Offset >= 0 {
		return fmt.Sprintf("snmp: parse error at offset %d: %s", e.Offset, e.Message)
	}
	return fmt.Sprintf("snmp: parse error: %s", e.Message)
}

// NewParseError creates a new parse error.
func NewParseError(message string, offset int) *ParseError {
	return &ParseError{
		Message: message,
		Offset:  offset,
	}
}
