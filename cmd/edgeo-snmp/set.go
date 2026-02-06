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

package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/edgeo-scada/snmp/snmp"
	"github.com/spf13/cobra"
)

var setCmd = &cobra.Command{
	Use:   "set OID TYPE VALUE [OID TYPE VALUE...]",
	Short: "Perform SNMP SET request",
	Long: `Perform an SNMP SET request to modify the value of one or more OIDs.

Type specifiers:
  i - INTEGER
  u - Unsigned INTEGER (Gauge32)
  c - Counter32
  s - OCTET STRING (text)
  x - OCTET STRING (hex bytes, e.g., "DE AD BE EF")
  d - OCTET STRING (decimal bytes, e.g., "10.0.1.1")
  n - NULL
  o - OBJECT IDENTIFIER
  t - TimeTicks
  a - IP Address

Examples:
  # Set system contact (string)
  edgeo-snmp set -t 192.168.1.1 1.3.6.1.2.1.1.4.0 s "admin@example.com"

  # Set system name
  edgeo-snmp set -t 192.168.1.1 1.3.6.1.2.1.1.5.0 s "switch01"

  # Set an integer value
  edgeo-snmp set -t 192.168.1.1 1.3.6.1.4.1.9.2.1.55.0 i 5

  # Set multiple values
  edgeo-snmp set -t 192.168.1.1 \
    1.3.6.1.2.1.1.4.0 s "admin@example.com" \
    1.3.6.1.2.1.1.5.0 s "switch01"`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 3 {
			return fmt.Errorf("requires at least 3 arguments: OID TYPE VALUE")
		}
		if len(args)%3 != 0 {
			return fmt.Errorf("arguments must be in groups of 3: OID TYPE VALUE")
		}
		return nil
	},
	RunE: runSet,
}

func init() {
	rootCmd.AddCommand(setCmd)
}

func runSet(cmd *cobra.Command, args []string) error {
	if err := checkTarget(); err != nil {
		return err
	}

	// Parse variable bindings
	variables, err := parseSetVariables(args)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	client, err := createClient(ctx)
	if err != nil {
		return err
	}
	defer disconnectClient(client)

	printVerbose("Sending SET request for %d variable(s)...", len(variables))
	start := time.Now()

	result, err := client.Set(ctx, variables...)
	if err != nil {
		return fmt.Errorf("SET failed: %w", err)
	}

	printVerbose("Response received in %s", formatDuration(time.Since(start)))

	formatter := NewFormatter(outputFormat)
	formatter.FormatVariables(result)

	return nil
}

func parseSetVariables(args []string) ([]snmp.Variable, error) {
	var variables []snmp.Variable

	for i := 0; i < len(args); i += 3 {
		oid, err := snmp.ParseOID(args[i])
		if err != nil {
			return nil, fmt.Errorf("invalid OID '%s': %w", args[i], err)
		}

		typeSpec := strings.ToLower(args[i+1])
		valueStr := args[i+2]

		v, err := parseValue(oid, typeSpec, valueStr)
		if err != nil {
			return nil, fmt.Errorf("invalid value for OID %s: %w", oid, err)
		}

		variables = append(variables, *v)
	}

	return variables, nil
}

func parseValue(oid snmp.OID, typeSpec, valueStr string) (*snmp.Variable, error) {
	v := &snmp.Variable{OID: oid}

	switch typeSpec {
	case "i": // INTEGER
		val, err := strconv.ParseInt(valueStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid integer: %w", err)
		}
		v.Type = snmp.TypeInteger
		v.Value = int(val)

	case "u": // Unsigned INTEGER (Gauge32)
		val, err := strconv.ParseUint(valueStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid unsigned integer: %w", err)
		}
		v.Type = snmp.TypeGauge32
		v.Value = uint32(val)

	case "c": // Counter32
		val, err := strconv.ParseUint(valueStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid counter: %w", err)
		}
		v.Type = snmp.TypeCounter32
		v.Value = uint32(val)

	case "s": // OCTET STRING (text)
		v.Type = snmp.TypeOctetString
		v.Value = []byte(valueStr)

	case "x": // OCTET STRING (hex)
		bytes, err := parseHexString(valueStr)
		if err != nil {
			return nil, fmt.Errorf("invalid hex string: %w", err)
		}
		v.Type = snmp.TypeOctetString
		v.Value = bytes

	case "d": // OCTET STRING (decimal/dotted)
		bytes, err := parseDottedDecimal(valueStr)
		if err != nil {
			return nil, fmt.Errorf("invalid decimal string: %w", err)
		}
		v.Type = snmp.TypeOctetString
		v.Value = bytes

	case "n": // NULL
		v.Type = snmp.TypeNull
		v.Value = nil

	case "o": // OBJECT IDENTIFIER
		oidVal, err := snmp.ParseOID(valueStr)
		if err != nil {
			return nil, fmt.Errorf("invalid OID value: %w", err)
		}
		v.Type = snmp.TypeObjectIdentifier
		v.Value = oidVal

	case "t": // TimeTicks
		val, err := strconv.ParseUint(valueStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid timeticks: %w", err)
		}
		v.Type = snmp.TypeTimeTicks
		v.Value = uint32(val)

	case "a": // IP Address
		ip := net.ParseIP(valueStr)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP address: %s", valueStr)
		}
		ip4 := ip.To4()
		if ip4 == nil {
			return nil, fmt.Errorf("not an IPv4 address: %s", valueStr)
		}
		v.Type = snmp.TypeIPAddress
		v.Value = ip4

	default:
		return nil, fmt.Errorf("unknown type specifier: %s (use i, u, c, s, x, d, n, o, t, or a)", typeSpec)
	}

	return v, nil
}

func parseHexString(s string) ([]byte, error) {
	// Remove common separators and whitespace
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "0x", "")
	s = strings.ReplaceAll(s, "0X", "")

	if len(s)%2 != 0 {
		return nil, fmt.Errorf("odd number of hex characters")
	}

	bytes := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		val, err := strconv.ParseUint(s[i:i+2], 16, 8)
		if err != nil {
			return nil, err
		}
		bytes[i/2] = byte(val)
	}

	return bytes, nil
}

func parseDottedDecimal(s string) ([]byte, error) {
	parts := strings.Split(s, ".")
	bytes := make([]byte, len(parts))

	for i, part := range parts {
		val, err := strconv.ParseUint(part, 10, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid byte value: %s", part)
		}
		bytes[i] = byte(val)
	}

	return bytes, nil
}
