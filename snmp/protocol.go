package snmp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
)

// BER encoding/decoding functions for SNMP packets.

// encodeLength encodes a BER length.
func encodeLength(length int) []byte {
	if length < 128 {
		return []byte{byte(length)}
	}

	// Long form
	buf := make([]byte, 0, 5)
	temp := length
	for temp > 0 {
		buf = append([]byte{byte(temp & 0xff)}, buf...)
		temp >>= 8
	}
	return append([]byte{byte(0x80 | len(buf))}, buf...)
}

// decodeLength decodes a BER length from a reader.
func decodeLength(r io.Reader) (int, error) {
	b := make([]byte, 1)
	if _, err := io.ReadFull(r, b); err != nil {
		return 0, err
	}

	if b[0] < 128 {
		return int(b[0]), nil
	}

	// Long form
	numBytes := int(b[0] & 0x7f)
	if numBytes > 4 {
		return 0, NewParseError("length too large", -1)
	}

	lenBytes := make([]byte, numBytes)
	if _, err := io.ReadFull(r, lenBytes); err != nil {
		return 0, err
	}

	length := 0
	for _, lb := range lenBytes {
		length = (length << 8) | int(lb)
	}

	return length, nil
}

// encodeInteger encodes an integer using BER.
func encodeInteger(value int64) []byte {
	// Determine the minimum number of bytes needed
	var buf []byte

	if value == 0 {
		buf = []byte{0}
	} else if value > 0 {
		// Positive number
		temp := value
		for temp > 0 {
			buf = append([]byte{byte(temp & 0xff)}, buf...)
			temp >>= 8
		}
		// Add leading zero if high bit is set (to indicate positive)
		if buf[0]&0x80 != 0 {
			buf = append([]byte{0}, buf...)
		}
	} else {
		// Negative number (two's complement)
		temp := value
		for temp < -1 || (temp == -1 && len(buf) == 0) {
			buf = append([]byte{byte(temp & 0xff)}, buf...)
			temp >>= 8
		}
		// Ensure high bit is set (to indicate negative)
		if len(buf) > 0 && buf[0]&0x80 == 0 {
			buf = append([]byte{0xff}, buf...)
		}
	}

	return buf
}

// decodeInteger decodes a BER integer.
func decodeInteger(data []byte) int64 {
	if len(data) == 0 {
		return 0
	}

	var value int64
	if data[0]&0x80 != 0 {
		// Negative number
		value = -1
	}

	for _, b := range data {
		value = (value << 8) | int64(b)
	}

	return value
}

// encodeUnsignedInteger encodes an unsigned integer using BER.
func encodeUnsignedInteger(value uint64) []byte {
	if value == 0 {
		return []byte{0}
	}

	var buf []byte
	temp := value
	for temp > 0 {
		buf = append([]byte{byte(temp & 0xff)}, buf...)
		temp >>= 8
	}

	// Add leading zero if high bit is set (to prevent interpretation as negative)
	if buf[0]&0x80 != 0 {
		buf = append([]byte{0}, buf...)
	}

	return buf
}

// decodeUnsignedInteger decodes a BER unsigned integer.
func decodeUnsignedInteger(data []byte) uint64 {
	var value uint64
	for _, b := range data {
		value = (value << 8) | uint64(b)
	}
	return value
}

// encodeOID encodes an OID using BER.
func encodeOID(oid OID) []byte {
	if len(oid) < 2 {
		return nil
	}

	// First two components are combined: first*40 + second
	buf := []byte{byte(oid[0]*40 + oid[1])}

	for i := 2; i < len(oid); i++ {
		buf = append(buf, encodeOIDComponent(oid[i])...)
	}

	return buf
}

// encodeOIDComponent encodes a single OID component.
func encodeOIDComponent(value int) []byte {
	if value < 128 {
		return []byte{byte(value)}
	}

	var buf []byte
	temp := value
	for temp > 0 {
		buf = append([]byte{byte(temp & 0x7f)}, buf...)
		temp >>= 7
	}

	// Set high bit on all but last byte
	for i := 0; i < len(buf)-1; i++ {
		buf[i] |= 0x80
	}

	return buf
}

// decodeOID decodes a BER OID.
func decodeOID(data []byte) (OID, error) {
	if len(data) == 0 {
		return nil, NewParseError("empty OID", -1)
	}

	// First byte contains first two components
	oid := OID{int(data[0] / 40), int(data[0] % 40)}

	var current int
	for i := 1; i < len(data); i++ {
		current = (current << 7) | int(data[i]&0x7f)
		if data[i]&0x80 == 0 {
			oid = append(oid, current)
			current = 0
		}
	}

	return oid, nil
}

// encodeTLV encodes a Type-Length-Value structure.
func encodeTLV(berType BERType, value []byte) []byte {
	length := encodeLength(len(value))
	result := make([]byte, 1+len(length)+len(value))
	result[0] = byte(berType)
	copy(result[1:], length)
	copy(result[1+len(length):], value)
	return result
}

// decodeTLV decodes a Type-Length-Value structure.
func decodeTLV(r io.Reader) (BERType, []byte, error) {
	// Read type
	typeByte := make([]byte, 1)
	if _, err := io.ReadFull(r, typeByte); err != nil {
		return 0, nil, err
	}
	berType := BERType(typeByte[0])

	// Read length
	length, err := decodeLength(r)
	if err != nil {
		return 0, nil, err
	}

	// Read value
	value := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, value); err != nil {
			return 0, nil, err
		}
	}

	return berType, value, nil
}

// encodeVariable encodes a Variable to BER.
func encodeVariable(v *Variable) ([]byte, error) {
	var buf bytes.Buffer

	// Encode OID
	oidBytes := encodeOID(v.OID)
	buf.Write(encodeTLV(TypeObjectIdentifier, oidBytes))

	// Encode value based on type
	switch v.Type {
	case TypeNull:
		buf.Write(encodeTLV(TypeNull, nil))

	case TypeInteger:
		val, ok := v.AsInt()
		if !ok {
			return nil, fmt.Errorf("invalid integer value: %v", v.Value)
		}
		buf.Write(encodeTLV(TypeInteger, encodeInteger(val)))

	case TypeOctetString:
		var data []byte
		switch val := v.Value.(type) {
		case []byte:
			data = val
		case string:
			data = []byte(val)
		default:
			return nil, fmt.Errorf("invalid octet string value: %v", v.Value)
		}
		buf.Write(encodeTLV(TypeOctetString, data))

	case TypeObjectIdentifier:
		oid, ok := v.Value.(OID)
		if !ok {
			return nil, fmt.Errorf("invalid OID value: %v", v.Value)
		}
		buf.Write(encodeTLV(TypeObjectIdentifier, encodeOID(oid)))

	case TypeIPAddress:
		var ip net.IP
		switch val := v.Value.(type) {
		case net.IP:
			ip = val
		case string:
			ip = net.ParseIP(val)
		default:
			return nil, fmt.Errorf("invalid IP address value: %v", v.Value)
		}
		if ip == nil {
			return nil, fmt.Errorf("invalid IP address: %v", v.Value)
		}
		ip4 := ip.To4()
		if ip4 == nil {
			return nil, fmt.Errorf("not an IPv4 address: %v", v.Value)
		}
		buf.Write(encodeTLV(TypeIPAddress, ip4))

	case TypeCounter32, TypeGauge32, TypeTimeTicks, TypeUInteger32:
		val, ok := v.AsUint()
		if !ok {
			return nil, fmt.Errorf("invalid unsigned integer value: %v", v.Value)
		}
		buf.Write(encodeTLV(v.Type, encodeUnsignedInteger(val)))

	case TypeCounter64:
		val, ok := v.AsUint()
		if !ok {
			return nil, fmt.Errorf("invalid counter64 value: %v", v.Value)
		}
		buf.Write(encodeTLV(TypeCounter64, encodeUnsignedInteger(val)))

	case TypeOpaque:
		data, ok := v.Value.([]byte)
		if !ok {
			return nil, fmt.Errorf("invalid opaque value: %v", v.Value)
		}
		buf.Write(encodeTLV(TypeOpaque, data))

	default:
		return nil, fmt.Errorf("unsupported type: %s", v.Type)
	}

	// Wrap in sequence
	return encodeTLV(TypeSequence, buf.Bytes()), nil
}

// decodeVariable decodes a Variable from BER data.
func decodeVariable(data []byte) (*Variable, error) {
	r := bytes.NewReader(data)

	// Decode sequence
	seqType, seqData, err := decodeTLV(r)
	if err != nil {
		return nil, err
	}
	if seqType != TypeSequence {
		return nil, NewParseError(fmt.Sprintf("expected sequence, got %s", seqType), -1)
	}

	seqReader := bytes.NewReader(seqData)

	// Decode OID
	oidType, oidData, err := decodeTLV(seqReader)
	if err != nil {
		return nil, err
	}
	if oidType != TypeObjectIdentifier {
		return nil, NewParseError(fmt.Sprintf("expected OID, got %s", oidType), -1)
	}
	oid, err := decodeOID(oidData)
	if err != nil {
		return nil, err
	}

	// Decode value
	valType, valData, err := decodeTLV(seqReader)
	if err != nil {
		return nil, err
	}

	v := &Variable{
		OID:  oid,
		Type: valType,
	}

	switch valType {
	case TypeNull:
		v.Value = nil

	case TypeInteger:
		v.Value = int(decodeInteger(valData))

	case TypeOctetString:
		v.Value = valData

	case TypeObjectIdentifier:
		v.Value, err = decodeOID(valData)
		if err != nil {
			return nil, err
		}

	case TypeIPAddress:
		if len(valData) == 4 {
			v.Value = net.IP(valData)
		} else {
			v.Value = valData
		}

	case TypeCounter32, TypeGauge32, TypeTimeTicks, TypeUInteger32:
		v.Value = uint32(decodeUnsignedInteger(valData))

	case TypeCounter64:
		v.Value = decodeUnsignedInteger(valData)

	case TypeOpaque:
		v.Value = valData

	case TypeNoSuchObject, TypeNoSuchInstance, TypeEndOfMibView:
		v.Value = nil

	default:
		v.Value = valData
	}

	return v, nil
}

// decodeVariables decodes a list of variables from BER data.
func decodeVariables(data []byte) ([]Variable, error) {
	r := bytes.NewReader(data)

	// Decode sequence
	seqType, seqData, err := decodeTLV(r)
	if err != nil {
		return nil, err
	}
	if seqType != TypeSequence {
		return nil, NewParseError(fmt.Sprintf("expected sequence, got %s", seqType), -1)
	}

	var variables []Variable
	seqReader := bytes.NewReader(seqData)

	for seqReader.Len() > 0 {
		// Read variable binding sequence
		vbType, vbData, err := decodeTLV(seqReader)
		if err != nil {
			return nil, err
		}
		if vbType != TypeSequence {
			return nil, NewParseError(fmt.Sprintf("expected sequence, got %s", vbType), -1)
		}

		vbReader := bytes.NewReader(vbData)

		// Decode OID
		oidType, oidData, err := decodeTLV(vbReader)
		if err != nil {
			return nil, err
		}
		if oidType != TypeObjectIdentifier {
			return nil, NewParseError(fmt.Sprintf("expected OID, got %s", oidType), -1)
		}
		oid, err := decodeOID(oidData)
		if err != nil {
			return nil, err
		}

		// Decode value
		valType, valData, err := decodeTLV(vbReader)
		if err != nil {
			return nil, err
		}

		v := Variable{
			OID:  oid,
			Type: valType,
		}

		switch valType {
		case TypeNull:
			v.Value = nil

		case TypeInteger:
			v.Value = int(decodeInteger(valData))

		case TypeOctetString:
			v.Value = valData

		case TypeObjectIdentifier:
			v.Value, err = decodeOID(valData)
			if err != nil {
				return nil, err
			}

		case TypeIPAddress:
			if len(valData) == 4 {
				v.Value = net.IP(valData)
			} else {
				v.Value = valData
			}

		case TypeCounter32, TypeGauge32, TypeTimeTicks, TypeUInteger32:
			v.Value = uint32(decodeUnsignedInteger(valData))

		case TypeCounter64:
			v.Value = decodeUnsignedInteger(valData)

		case TypeOpaque:
			v.Value = valData

		case TypeNoSuchObject, TypeNoSuchInstance, TypeEndOfMibView:
			v.Value = nil

		default:
			v.Value = valData
		}

		variables = append(variables, v)
	}

	return variables, nil
}

// encodeVariableBindings encodes a list of variables to a varbind list.
func encodeVariableBindings(variables []Variable) ([]byte, error) {
	var buf bytes.Buffer

	for _, v := range variables {
		vbBytes, err := encodeVariable(&v)
		if err != nil {
			return nil, err
		}
		buf.Write(vbBytes)
	}

	return encodeTLV(TypeSequence, buf.Bytes()), nil
}

// Helper function to encode request ID as 4 bytes
func encodeRequestID(requestID int32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(requestID))
	// Remove leading zeros but keep at least one byte
	for len(buf) > 1 && buf[0] == 0 && buf[1]&0x80 == 0 {
		buf = buf[1:]
	}
	return buf
}

// Helper to convert float64 seconds to TimeTicks (centiseconds)
func SecondsToTimeTicks(seconds float64) uint32 {
	return uint32(seconds * 100)
}

// Helper to convert TimeTicks to seconds
func TimeTicksToSeconds(ticks uint32) float64 {
	return float64(ticks) / 100
}

// Helper to convert TimeTicks to human-readable string
func TimeTicksToString(ticks uint32) string {
	totalSeconds := ticks / 100
	days := totalSeconds / 86400
	hours := (totalSeconds % 86400) / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	centiseconds := ticks % 100

	if days > 0 {
		return fmt.Sprintf("%d days, %02d:%02d:%02d.%02d", days, hours, minutes, seconds, centiseconds)
	}
	return fmt.Sprintf("%02d:%02d:%02d.%02d", hours, minutes, seconds, centiseconds)
}

// MaxInt32 is the maximum value for int32
const MaxInt32 = math.MaxInt32
