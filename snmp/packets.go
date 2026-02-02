package snmp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// PDU represents an SNMP Protocol Data Unit.
type PDU struct {
	Type        PDUType
	RequestID   int32
	ErrorStatus ErrorStatus
	ErrorIndex  int
	Variables   []Variable

	// GetBulk specific
	NonRepeaters   int
	MaxRepetitions int
}

// Encode encodes the PDU to bytes.
func (p *PDU) Encode() ([]byte, error) {
	var buf bytes.Buffer

	// Request ID
	requestIDBytes := encodeInteger(int64(p.RequestID))
	buf.Write(encodeTLV(TypeInteger, requestIDBytes))

	if p.Type == PDUType(TypeGetBulkRequest) {
		// GetBulk uses non-repeaters and max-repetitions instead of error-status/index
		buf.Write(encodeTLV(TypeInteger, encodeInteger(int64(p.NonRepeaters))))
		buf.Write(encodeTLV(TypeInteger, encodeInteger(int64(p.MaxRepetitions))))
	} else {
		// Error status
		buf.Write(encodeTLV(TypeInteger, encodeInteger(int64(p.ErrorStatus))))
		// Error index
		buf.Write(encodeTLV(TypeInteger, encodeInteger(int64(p.ErrorIndex))))
	}

	// Variable bindings
	varbinds, err := encodeVariableBindings(p.Variables)
	if err != nil {
		return nil, err
	}
	buf.Write(varbinds)

	// Wrap in PDU type
	return encodeTLV(BERType(p.Type), buf.Bytes()), nil
}

// DecodePDU decodes a PDU from BER data.
func DecodePDU(data []byte) (*PDU, error) {
	r := bytes.NewReader(data)
	return decodePDU(r)
}

func decodePDU(r io.Reader) (*PDU, error) {
	// Read PDU type and length
	pduType, pduData, err := decodeTLV(r)
	if err != nil {
		return nil, err
	}

	pdu := &PDU{
		Type: PDUType(pduType),
	}

	pduReader := bytes.NewReader(pduData)

	// Request ID
	_, requestIDData, err := decodeTLV(pduReader)
	if err != nil {
		return nil, err
	}
	pdu.RequestID = int32(decodeInteger(requestIDData))

	// Error status / non-repeaters
	_, errStatusData, err := decodeTLV(pduReader)
	if err != nil {
		return nil, err
	}
	if pduType == TypeGetBulkRequest {
		pdu.NonRepeaters = int(decodeInteger(errStatusData))
	} else {
		pdu.ErrorStatus = ErrorStatus(decodeInteger(errStatusData))
	}

	// Error index / max-repetitions
	_, errIndexData, err := decodeTLV(pduReader)
	if err != nil {
		return nil, err
	}
	if pduType == TypeGetBulkRequest {
		pdu.MaxRepetitions = int(decodeInteger(errIndexData))
	} else {
		pdu.ErrorIndex = int(decodeInteger(errIndexData))
	}

	// Variable bindings
	remaining := make([]byte, pduReader.Len())
	if _, err := io.ReadFull(pduReader, remaining); err != nil {
		return nil, err
	}
	pdu.Variables, err = decodeVariables(remaining)
	if err != nil {
		return nil, err
	}

	return pdu, nil
}

// Message represents a complete SNMP message.
type Message struct {
	Version   SNMPVersion
	Community string
	PDU       *PDU
}

// Encode encodes the SNMP message to bytes.
func (m *Message) Encode() ([]byte, error) {
	var buf bytes.Buffer

	// Version
	buf.Write(encodeTLV(TypeInteger, encodeInteger(int64(m.Version))))

	// Community
	buf.Write(encodeTLV(TypeOctetString, []byte(m.Community)))

	// PDU
	pduBytes, err := m.PDU.Encode()
	if err != nil {
		return nil, err
	}
	buf.Write(pduBytes)

	// Wrap in sequence
	return encodeTLV(TypeSequence, buf.Bytes()), nil
}

// DecodeMessage decodes an SNMP message from bytes.
func DecodeMessage(data []byte) (*Message, error) {
	r := bytes.NewReader(data)

	// Read outer sequence
	seqType, seqData, err := decodeTLV(r)
	if err != nil {
		return nil, err
	}
	if seqType != TypeSequence {
		return nil, NewParseError(fmt.Sprintf("expected sequence, got %s", seqType), -1)
	}

	seqReader := bytes.NewReader(seqData)
	msg := &Message{}

	// Version
	_, versionData, err := decodeTLV(seqReader)
	if err != nil {
		return nil, err
	}
	msg.Version = SNMPVersion(decodeInteger(versionData))

	// Community
	_, communityData, err := decodeTLV(seqReader)
	if err != nil {
		return nil, err
	}
	msg.Community = string(communityData)

	// PDU
	msg.PDU, err = decodePDU(seqReader)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// TrapV1PDU represents an SNMPv1 Trap PDU.
type TrapV1PDU struct {
	Enterprise   OID
	AgentAddress []byte
	GenericTrap  int
	SpecificTrap int
	Timestamp    uint32
	Variables    []Variable
}

// Encode encodes the v1 trap PDU to bytes.
func (t *TrapV1PDU) Encode() ([]byte, error) {
	var buf bytes.Buffer

	// Enterprise OID
	buf.Write(encodeTLV(TypeObjectIdentifier, encodeOID(t.Enterprise)))

	// Agent address (IP)
	buf.Write(encodeTLV(TypeIPAddress, t.AgentAddress))

	// Generic trap
	buf.Write(encodeTLV(TypeInteger, encodeInteger(int64(t.GenericTrap))))

	// Specific trap
	buf.Write(encodeTLV(TypeInteger, encodeInteger(int64(t.SpecificTrap))))

	// Timestamp
	buf.Write(encodeTLV(TypeTimeTicks, encodeUnsignedInteger(uint64(t.Timestamp))))

	// Variable bindings
	varbinds, err := encodeVariableBindings(t.Variables)
	if err != nil {
		return nil, err
	}
	buf.Write(varbinds)

	return encodeTLV(TypeTrapV1, buf.Bytes()), nil
}

// DecodeTrapV1PDU decodes an SNMPv1 trap PDU from bytes.
func DecodeTrapV1PDU(data []byte) (*TrapV1PDU, error) {
	r := bytes.NewReader(data)

	// Read trap type
	trapType, trapData, err := decodeTLV(r)
	if err != nil {
		return nil, err
	}
	if trapType != TypeTrapV1 {
		return nil, NewParseError(fmt.Sprintf("expected trap PDU, got %s", trapType), -1)
	}

	trapReader := bytes.NewReader(trapData)
	trap := &TrapV1PDU{}

	// Enterprise OID
	_, oidData, err := decodeTLV(trapReader)
	if err != nil {
		return nil, err
	}
	trap.Enterprise, err = decodeOID(oidData)
	if err != nil {
		return nil, err
	}

	// Agent address
	_, addrData, err := decodeTLV(trapReader)
	if err != nil {
		return nil, err
	}
	trap.AgentAddress = addrData

	// Generic trap
	_, genData, err := decodeTLV(trapReader)
	if err != nil {
		return nil, err
	}
	trap.GenericTrap = int(decodeInteger(genData))

	// Specific trap
	_, specData, err := decodeTLV(trapReader)
	if err != nil {
		return nil, err
	}
	trap.SpecificTrap = int(decodeInteger(specData))

	// Timestamp
	_, tsData, err := decodeTLV(trapReader)
	if err != nil {
		return nil, err
	}
	trap.Timestamp = uint32(decodeUnsignedInteger(tsData))

	// Variable bindings
	remaining := make([]byte, trapReader.Len())
	if _, err := io.ReadFull(trapReader, remaining); err != nil {
		return nil, err
	}
	trap.Variables, err = decodeVariables(remaining)
	if err != nil {
		return nil, err
	}

	return trap, nil
}

// TrapV1Message represents a complete SNMPv1 trap message.
type TrapV1Message struct {
	Version   SNMPVersion
	Community string
	PDU       *TrapV1PDU
}

// Encode encodes the v1 trap message to bytes.
func (m *TrapV1Message) Encode() ([]byte, error) {
	var buf bytes.Buffer

	// Version
	buf.Write(encodeTLV(TypeInteger, encodeInteger(int64(m.Version))))

	// Community
	buf.Write(encodeTLV(TypeOctetString, []byte(m.Community)))

	// Trap PDU
	pduBytes, err := m.PDU.Encode()
	if err != nil {
		return nil, err
	}
	buf.Write(pduBytes)

	return encodeTLV(TypeSequence, buf.Bytes()), nil
}

// DecodeTrapV1Message decodes an SNMPv1 trap message from bytes.
func DecodeTrapV1Message(data []byte) (*TrapV1Message, error) {
	r := bytes.NewReader(data)

	// Read outer sequence
	seqType, seqData, err := decodeTLV(r)
	if err != nil {
		return nil, err
	}
	if seqType != TypeSequence {
		return nil, NewParseError(fmt.Sprintf("expected sequence, got %s", seqType), -1)
	}

	seqReader := bytes.NewReader(seqData)
	msg := &TrapV1Message{}

	// Version
	_, versionData, err := decodeTLV(seqReader)
	if err != nil {
		return nil, err
	}
	msg.Version = SNMPVersion(decodeInteger(versionData))

	// Community
	_, communityData, err := decodeTLV(seqReader)
	if err != nil {
		return nil, err
	}
	msg.Community = string(communityData)

	// Trap PDU
	remaining := make([]byte, seqReader.Len())
	if _, err := io.ReadFull(seqReader, remaining); err != nil {
		return nil, err
	}
	msg.PDU, err = DecodeTrapV1PDU(remaining)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// NewGetRequest creates a new GET request PDU.
func NewGetRequest(requestID int32, oids ...OID) *PDU {
	variables := make([]Variable, len(oids))
	for i, oid := range oids {
		variables[i] = Variable{
			OID:   oid,
			Type:  TypeNull,
			Value: nil,
		}
	}
	return &PDU{
		Type:      PDUGetRequest,
		RequestID: requestID,
		Variables: variables,
	}
}

// NewGetNextRequest creates a new GET-NEXT request PDU.
func NewGetNextRequest(requestID int32, oids ...OID) *PDU {
	variables := make([]Variable, len(oids))
	for i, oid := range oids {
		variables[i] = Variable{
			OID:   oid,
			Type:  TypeNull,
			Value: nil,
		}
	}
	return &PDU{
		Type:      PDUGetNextRequest,
		RequestID: requestID,
		Variables: variables,
	}
}

// NewGetBulkRequest creates a new GET-BULK request PDU (v2c/v3 only).
func NewGetBulkRequest(requestID int32, nonRepeaters, maxRepetitions int, oids ...OID) *PDU {
	variables := make([]Variable, len(oids))
	for i, oid := range oids {
		variables[i] = Variable{
			OID:   oid,
			Type:  TypeNull,
			Value: nil,
		}
	}
	return &PDU{
		Type:           PDUGetBulkRequest,
		RequestID:      requestID,
		NonRepeaters:   nonRepeaters,
		MaxRepetitions: maxRepetitions,
		Variables:      variables,
	}
}

// NewSetRequest creates a new SET request PDU.
func NewSetRequest(requestID int32, variables ...Variable) *PDU {
	return &PDU{
		Type:      PDUSetRequest,
		RequestID: requestID,
		Variables: variables,
	}
}

// NewTrapV2 creates a new SNMPv2c trap PDU.
func NewTrapV2(requestID int32, sysUpTime uint32, trapOID OID, variables ...Variable) *PDU {
	// SNMPv2c traps include sysUpTime and snmpTrapOID as first two varbinds
	allVars := make([]Variable, 0, len(variables)+2)
	allVars = append(allVars, Variable{
		OID:   OIDSysUpTime,
		Type:  TypeTimeTicks,
		Value: sysUpTime,
	})
	allVars = append(allVars, Variable{
		OID:   OIDSnmpTrapOID,
		Type:  TypeObjectIdentifier,
		Value: trapOID,
	})
	allVars = append(allVars, variables...)

	return &PDU{
		Type:      PDUTrapV2,
		RequestID: requestID,
		Variables: allVars,
	}
}

// Helper to create a packet with request ID as big-endian bytes
func writeInt32(buf *bytes.Buffer, value int32) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(value))
	buf.Write(b)
}
