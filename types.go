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
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ASN.1 BER type tags used in SNMP.
type BERType byte

const (
	// Primitive types
	TypeInteger          BERType = 0x02
	TypeBitString        BERType = 0x03
	TypeOctetString      BERType = 0x04
	TypeNull             BERType = 0x05
	TypeObjectIdentifier BERType = 0x06

	// Application types (SNMP-specific)
	TypeIPAddress   BERType = 0x40
	TypeCounter32   BERType = 0x41
	TypeGauge32     BERType = 0x42
	TypeTimeTicks   BERType = 0x43
	TypeOpaque      BERType = 0x44
	TypeNsapAddress BERType = 0x45
	TypeCounter64   BERType = 0x46
	TypeUInteger32  BERType = 0x47

	// Sequence type
	TypeSequence BERType = 0x30

	// Context-specific types (PDU types)
	TypeGetRequest     BERType = 0xA0
	TypeGetNextRequest BERType = 0xA1
	TypeGetResponse    BERType = 0xA2
	TypeSetRequest     BERType = 0xA3
	TypeTrapV1         BERType = 0xA4 // SNMPv1 Trap
	TypeGetBulkRequest BERType = 0xA5
	TypeInformRequest  BERType = 0xA6
	TypeTrapV2         BERType = 0xA7 // SNMPv2c Trap

	// Exception types (SNMPv2c)
	TypeNoSuchObject   BERType = 0x80
	TypeNoSuchInstance BERType = 0x81
	TypeEndOfMibView   BERType = 0x82
)

// String returns the string representation of the BER type.
func (t BERType) String() string {
	switch t {
	case TypeInteger:
		return "INTEGER"
	case TypeBitString:
		return "BIT STRING"
	case TypeOctetString:
		return "OCTET STRING"
	case TypeNull:
		return "NULL"
	case TypeObjectIdentifier:
		return "OBJECT IDENTIFIER"
	case TypeIPAddress:
		return "IpAddress"
	case TypeCounter32:
		return "Counter32"
	case TypeGauge32:
		return "Gauge32"
	case TypeTimeTicks:
		return "TimeTicks"
	case TypeOpaque:
		return "Opaque"
	case TypeCounter64:
		return "Counter64"
	case TypeUInteger32:
		return "UInteger32"
	case TypeSequence:
		return "SEQUENCE"
	case TypeGetRequest:
		return "GetRequest-PDU"
	case TypeGetNextRequest:
		return "GetNextRequest-PDU"
	case TypeGetResponse:
		return "GetResponse-PDU"
	case TypeSetRequest:
		return "SetRequest-PDU"
	case TypeTrapV1:
		return "Trap-PDU (v1)"
	case TypeGetBulkRequest:
		return "GetBulkRequest-PDU"
	case TypeInformRequest:
		return "InformRequest-PDU"
	case TypeTrapV2:
		return "SNMPv2-Trap-PDU"
	case TypeNoSuchObject:
		return "noSuchObject"
	case TypeNoSuchInstance:
		return "noSuchInstance"
	case TypeEndOfMibView:
		return "endOfMibView"
	default:
		return fmt.Sprintf("Unknown(0x%02X)", byte(t))
	}
}

// PDUType represents SNMP PDU types.
type PDUType byte

const (
	PDUGetRequest     PDUType = 0xA0
	PDUGetNextRequest PDUType = 0xA1
	PDUGetResponse    PDUType = 0xA2
	PDUSetRequest     PDUType = 0xA3
	PDUTrapV1         PDUType = 0xA4
	PDUGetBulkRequest PDUType = 0xA5
	PDUInformRequest  PDUType = 0xA6
	PDUTrapV2         PDUType = 0xA7
)

// String returns the string representation of the PDU type.
func (p PDUType) String() string {
	return BERType(p).String()
}

// ErrorStatus represents SNMP error status codes.
type ErrorStatus int

const (
	NoError             ErrorStatus = 0
	TooBig              ErrorStatus = 1
	NoSuchName          ErrorStatus = 2
	BadValue            ErrorStatus = 3
	ReadOnly            ErrorStatus = 4
	GenErr              ErrorStatus = 5
	NoAccess            ErrorStatus = 6
	WrongType           ErrorStatus = 7
	WrongLength         ErrorStatus = 8
	WrongEncoding       ErrorStatus = 9
	WrongValue          ErrorStatus = 10
	NoCreation          ErrorStatus = 11
	InconsistentValue   ErrorStatus = 12
	ResourceUnavailable ErrorStatus = 13
	CommitFailed        ErrorStatus = 14
	UndoFailed          ErrorStatus = 15
	AuthorizationError  ErrorStatus = 16
	NotWritable         ErrorStatus = 17
	InconsistentName    ErrorStatus = 18
)

// String returns the string representation of the error status.
func (e ErrorStatus) String() string {
	switch e {
	case NoError:
		return "noError"
	case TooBig:
		return "tooBig"
	case NoSuchName:
		return "noSuchName"
	case BadValue:
		return "badValue"
	case ReadOnly:
		return "readOnly"
	case GenErr:
		return "genErr"
	case NoAccess:
		return "noAccess"
	case WrongType:
		return "wrongType"
	case WrongLength:
		return "wrongLength"
	case WrongEncoding:
		return "wrongEncoding"
	case WrongValue:
		return "wrongValue"
	case NoCreation:
		return "noCreation"
	case InconsistentValue:
		return "inconsistentValue"
	case ResourceUnavailable:
		return "resourceUnavailable"
	case CommitFailed:
		return "commitFailed"
	case UndoFailed:
		return "undoFailed"
	case AuthorizationError:
		return "authorizationError"
	case NotWritable:
		return "notWritable"
	case InconsistentName:
		return "inconsistentName"
	default:
		return fmt.Sprintf("unknown(%d)", e)
	}
}

// OID represents an SNMP Object Identifier.
type OID []int

// String returns the dotted-decimal string representation.
func (o OID) String() string {
	if len(o) == 0 {
		return ""
	}
	parts := make([]string, len(o))
	for i, n := range o {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ".")
}

// ParseOID parses a dotted-decimal OID string.
func ParseOID(s string) (OID, error) {
	if s == "" {
		return nil, ErrInvalidOID
	}

	// Remove leading dot if present
	s = strings.TrimPrefix(s, ".")

	parts := strings.Split(s, ".")
	oid := make(OID, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid OID component '%s': %w", p, err)
		}
		if n < 0 {
			return nil, fmt.Errorf("negative OID component: %d", n)
		}
		oid[i] = n
	}

	return oid, nil
}

// MustParseOID parses an OID string and panics on error.
func MustParseOID(s string) OID {
	oid, err := ParseOID(s)
	if err != nil {
		panic(err)
	}
	return oid
}

// Equal checks if two OIDs are equal.
func (o OID) Equal(other OID) bool {
	if len(o) != len(other) {
		return false
	}
	for i, n := range o {
		if n != other[i] {
			return false
		}
	}
	return true
}

// HasPrefix checks if the OID starts with the given prefix.
func (o OID) HasPrefix(prefix OID) bool {
	if len(prefix) > len(o) {
		return false
	}
	for i, n := range prefix {
		if n != o[i] {
			return false
		}
	}
	return true
}

// Copy returns a copy of the OID.
func (o OID) Copy() OID {
	c := make(OID, len(o))
	copy(c, o)
	return c
}

// Variable represents an SNMP variable binding.
type Variable struct {
	OID   OID
	Type  BERType
	Value interface{}
}

// String returns a string representation of the variable.
func (v *Variable) String() string {
	return fmt.Sprintf("%s = %s: %v", v.OID, v.Type, v.Value)
}

// AsInt returns the value as an integer.
func (v *Variable) AsInt() (int64, bool) {
	switch val := v.Value.(type) {
	case int:
		return int64(val), true
	case int32:
		return int64(val), true
	case int64:
		return val, true
	case uint32:
		return int64(val), true
	case uint64:
		return int64(val), true
	default:
		return 0, false
	}
}

// AsUint returns the value as an unsigned integer.
func (v *Variable) AsUint() (uint64, bool) {
	switch val := v.Value.(type) {
	case int:
		return uint64(val), true
	case int32:
		return uint64(val), true
	case int64:
		return uint64(val), true
	case uint32:
		return uint64(val), true
	case uint64:
		return val, true
	default:
		return 0, false
	}
}

// AsString returns the value as a string.
func (v *Variable) AsString() string {
	switch val := v.Value.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", v.Value)
	}
}

// AsBytes returns the value as bytes.
func (v *Variable) AsBytes() []byte {
	switch val := v.Value.(type) {
	case []byte:
		return val
	case string:
		return []byte(val)
	default:
		return nil
	}
}

// ConnectionState represents the state of a client connection.
type ConnectionState int

const (
	// StateDisconnected indicates the client is not connected.
	StateDisconnected ConnectionState = iota
	// StateConnecting indicates the client is attempting to connect.
	StateConnecting
	// StateConnected indicates the client is connected and ready.
	StateConnected
	// StateDisconnecting indicates the client is gracefully disconnecting.
	StateDisconnecting
)

// String returns the string representation of the connection state.
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "Disconnected"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
		return "Connected"
	case StateDisconnecting:
		return "Disconnecting"
	default:
		return "Unknown"
	}
}

// Token represents an asynchronous operation result.
type Token interface {
	// Wait blocks until the operation completes.
	Wait() error
	// WaitTimeout blocks until completion or timeout.
	WaitTimeout(timeout time.Duration) error
	// Done returns a channel that closes when complete.
	Done() <-chan struct{}
	// Error returns the error, if any.
	Error() error
}

// token implements the Token interface.
type token struct {
	done chan struct{}
	err  error
	mu   sync.Mutex
}

// newToken creates a new token.
func newToken() *token {
	return &token{
		done: make(chan struct{}),
	}
}

// Wait blocks until the operation completes.
func (t *token) Wait() error {
	<-t.done
	return t.err
}

// WaitTimeout blocks until completion or timeout.
func (t *token) WaitTimeout(timeout time.Duration) error {
	select {
	case <-t.done:
		return t.err
	case <-time.After(timeout):
		return ErrTimeout
	}
}

// Done returns a channel that closes when complete.
func (t *token) Done() <-chan struct{} {
	return t.done
}

// Error returns the error, if any.
func (t *token) Error() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.err
}

// complete marks the token as complete.
func (t *token) complete(err error) {
	t.mu.Lock()
	t.err = err
	t.mu.Unlock()
	close(t.done)
}

// GetToken is returned from Get operations.
type GetToken struct {
	*token
	Variables []Variable
}

// newGetToken creates a new get token.
func newGetToken() *GetToken {
	return &GetToken{
		token: newToken(),
	}
}

// WalkToken is returned from Walk operations.
type WalkToken struct {
	*token
	Variables []Variable
}

// newWalkToken creates a new walk token.
func newWalkToken() *WalkToken {
	return &WalkToken{
		token: newToken(),
	}
}

// SetToken is returned from Set operations.
type SetToken struct {
	*token
	Variables []Variable
}

// newSetToken creates a new set token.
func newSetToken() *SetToken {
	return &SetToken{
		token: newToken(),
	}
}

// ResponseHandler is a callback for received responses.
type ResponseHandler func(variables []Variable)

// TrapHandler is a callback for received traps.
type TrapHandler func(trap *TrapPDU)

// ConnectionLostHandler is a callback for connection loss.
type ConnectionLostHandler func(client *Client, err error)

// OnConnectHandler is a callback for successful connection.
type OnConnectHandler func(client *Client)

// ReconnectHandler is a callback for reconnection attempts.
type ReconnectHandler func(client *Client, opts *ClientOptions)

// TrapPDU represents an SNMP trap.
type TrapPDU struct {
	Version       SNMPVersion
	Community     string
	Enterprise    OID       // v1 only
	AgentAddress  string    // v1 only
	GenericTrap   int       // v1 only
	SpecificTrap  int       // v1 only
	Timestamp     uint32    // v1: TimeTicks, v2: sysUpTime
	Variables     []Variable
	SourceAddress string    // Source address of the trap
}

// Common OIDs
var (
	OIDSysDescr    = MustParseOID("1.3.6.1.2.1.1.1.0")
	OIDSysObjectID = MustParseOID("1.3.6.1.2.1.1.2.0")
	OIDSysUpTime   = MustParseOID("1.3.6.1.2.1.1.3.0")
	OIDSysContact  = MustParseOID("1.3.6.1.2.1.1.4.0")
	OIDSysName     = MustParseOID("1.3.6.1.2.1.1.5.0")
	OIDSysLocation = MustParseOID("1.3.6.1.2.1.1.6.0")
	OIDSysServices = MustParseOID("1.3.6.1.2.1.1.7.0")

	// Interface table
	OIDIfNumber = MustParseOID("1.3.6.1.2.1.2.1.0")
	OIDIfTable  = MustParseOID("1.3.6.1.2.1.2.2")

	// SNMPv2-MIB trap OIDs
	OIDSnmpTrapOID     = MustParseOID("1.3.6.1.6.3.1.1.4.1.0")
	OIDSnmpTrapEnterprise = MustParseOID("1.3.6.1.6.3.1.1.4.3.0")
)

// Default values.
const (
	DefaultTimeout         = 5 * time.Second
	DefaultRetries         = 3
	DefaultPort            = 161
	DefaultTrapPort        = 162
	DefaultCommunity       = "public"
	DefaultMaxOids         = 60
	DefaultMaxRepetitions  = 10
	DefaultNonRepeaters    = 0
)
