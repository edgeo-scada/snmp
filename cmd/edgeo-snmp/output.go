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
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/edgeo-scada/snmp/snmp"
)

// OutputFormat represents the output format type.
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"
	FormatRaw   OutputFormat = "raw"
)

// VariableOutput represents a variable for output.
type VariableOutput struct {
	OID   string      `json:"oid"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// Formatter handles output formatting.
type Formatter struct {
	format    OutputFormat
	writer    io.Writer
	csvWriter *csv.Writer
	first     bool
}

// NewFormatter creates a new formatter.
func NewFormatter(format string) *Formatter {
	f := &Formatter{
		format: OutputFormat(format),
		writer: os.Stdout,
		first:  true,
	}
	if f.format == FormatCSV {
		f.csvWriter = csv.NewWriter(os.Stdout)
	}
	return f
}

// FormatVariable formats and prints a variable.
func (f *Formatter) FormatVariable(v snmp.Variable) {
	switch f.format {
	case FormatJSON:
		f.formatJSON(v)
	case FormatCSV:
		f.formatCSV(v)
	case FormatRaw:
		f.formatRaw(v)
	default:
		f.formatTable(v)
	}
}

// FormatVariables formats and prints multiple variables.
func (f *Formatter) FormatVariables(vars []snmp.Variable) {
	for _, v := range vars {
		f.FormatVariable(v)
	}
}

func (f *Formatter) formatTable(v snmp.Variable) {
	var sb strings.Builder

	// OID
	sb.WriteString(colorize(v.OID.String(), ColorCyan))
	sb.WriteString(" = ")

	// Type
	sb.WriteString(colorize(v.Type.String(), ColorYellow))
	sb.WriteString(": ")

	// Value
	sb.WriteString(formatValue(v))

	fmt.Fprintln(f.writer, sb.String())
}

func (f *Formatter) formatJSON(v snmp.Variable) {
	output := VariableOutput{
		OID:   v.OID.String(),
		Type:  v.Type.String(),
		Value: convertValue(v),
	}
	data, _ := json.Marshal(output)
	fmt.Fprintln(f.writer, string(data))
}

func (f *Formatter) formatCSV(v snmp.Variable) {
	if f.first {
		f.csvWriter.Write([]string{"oid", "type", "value"})
		f.first = false
	}

	f.csvWriter.Write([]string{
		v.OID.String(),
		v.Type.String(),
		formatValue(v),
	})
	f.csvWriter.Flush()
}

func (f *Formatter) formatRaw(v snmp.Variable) {
	fmt.Fprintln(f.writer, formatValue(v))
}

// formatValue formats a variable value for display.
func formatValue(v snmp.Variable) string {
	switch v.Type {
	case snmp.TypeNull:
		return "NULL"

	case snmp.TypeInteger:
		return fmt.Sprintf("%d", v.Value)

	case snmp.TypeOctetString:
		switch val := v.Value.(type) {
		case []byte:
			// Try to print as string if printable
			if isPrintable(val) {
				return fmt.Sprintf("\"%s\"", string(val))
			}
			// Otherwise print as hex
			return formatHex(val)
		case string:
			return fmt.Sprintf("\"%s\"", val)
		default:
			return fmt.Sprintf("%v", v.Value)
		}

	case snmp.TypeObjectIdentifier:
		if oid, ok := v.Value.(snmp.OID); ok {
			return oid.String()
		}
		return fmt.Sprintf("%v", v.Value)

	case snmp.TypeIPAddress:
		if ip, ok := v.Value.(net.IP); ok {
			return ip.String()
		}
		if data, ok := v.Value.([]byte); ok && len(data) == 4 {
			return net.IP(data).String()
		}
		return fmt.Sprintf("%v", v.Value)

	case snmp.TypeCounter32, snmp.TypeGauge32, snmp.TypeUInteger32:
		return fmt.Sprintf("%d", v.Value)

	case snmp.TypeTimeTicks:
		if ticks, ok := v.Value.(uint32); ok {
			return fmt.Sprintf("%d (%s)", ticks, snmp.TimeTicksToString(ticks))
		}
		return fmt.Sprintf("%v", v.Value)

	case snmp.TypeCounter64:
		return fmt.Sprintf("%d", v.Value)

	case snmp.TypeOpaque:
		if data, ok := v.Value.([]byte); ok {
			return formatHex(data)
		}
		return fmt.Sprintf("%v", v.Value)

	case snmp.TypeNoSuchObject:
		return "No Such Object"

	case snmp.TypeNoSuchInstance:
		return "No Such Instance"

	case snmp.TypeEndOfMibView:
		return "End of MIB View"

	default:
		return fmt.Sprintf("%v", v.Value)
	}
}

// convertValue converts a variable value for JSON output.
func convertValue(v snmp.Variable) interface{} {
	switch v.Type {
	case snmp.TypeNull:
		return nil

	case snmp.TypeOctetString:
		switch val := v.Value.(type) {
		case []byte:
			if isPrintable(val) {
				return string(val)
			}
			return formatHex(val)
		default:
			return v.Value
		}

	case snmp.TypeObjectIdentifier:
		if oid, ok := v.Value.(snmp.OID); ok {
			return oid.String()
		}
		return v.Value

	case snmp.TypeIPAddress:
		if ip, ok := v.Value.(net.IP); ok {
			return ip.String()
		}
		if data, ok := v.Value.([]byte); ok && len(data) == 4 {
			return net.IP(data).String()
		}
		return v.Value

	case snmp.TypeTimeTicks:
		if ticks, ok := v.Value.(uint32); ok {
			return map[string]interface{}{
				"ticks":   ticks,
				"seconds": float64(ticks) / 100,
				"human":   snmp.TimeTicksToString(ticks),
			}
		}
		return v.Value

	default:
		return v.Value
	}
}

// isPrintable checks if bytes are printable ASCII.
func isPrintable(data []byte) bool {
	for _, b := range data {
		if b < 32 || b > 126 {
			return false
		}
	}
	return true
}

// formatHex formats bytes as hex string.
func formatHex(data []byte) string {
	var parts []string
	for _, b := range data {
		parts = append(parts, fmt.Sprintf("%02X", b))
	}
	return strings.Join(parts, " ")
}

// Color codes for terminal output.
const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorGray    = "\033[90m"
	ColorBold    = "\033[1m"
)

// colorize wraps text with color codes.
func colorize(text, color string) string {
	if noColor {
		return text
	}
	return color + text + ColorReset
}

// TableWriter writes formatted tables.
type TableWriter struct {
	headers []string
	rows    [][]string
	widths  []int
}

// NewTableWriter creates a new table writer.
func NewTableWriter(headers ...string) *TableWriter {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	return &TableWriter{
		headers: headers,
		widths:  widths,
	}
}

// AddRow adds a row to the table.
func (t *TableWriter) AddRow(values ...string) {
	for i, v := range values {
		if i < len(t.widths) && len(v) > t.widths[i] {
			t.widths[i] = len(v)
		}
	}
	t.rows = append(t.rows, values)
}

// Render renders the table to stdout.
func (t *TableWriter) Render() {
	// Print header
	for i, h := range t.headers {
		fmt.Printf("%-*s  ", t.widths[i], colorize(h, ColorBold))
	}
	fmt.Println()

	// Print separator
	for i := range t.headers {
		fmt.Print(strings.Repeat("-", t.widths[i]) + "  ")
	}
	fmt.Println()

	// Print rows
	for _, row := range t.rows {
		for i, v := range row {
			if i < len(t.widths) {
				fmt.Printf("%-*s  ", t.widths[i], v)
			}
		}
		fmt.Println()
	}
}

// PrintKeyValue prints a key-value pair formatted nicely.
func PrintKeyValue(key, value string) {
	fmt.Printf("  %-20s %s\n", colorize(key+":", ColorCyan), value)
}

// PrintSection prints a section header.
func PrintSection(title string) {
	fmt.Printf("\n%s\n", colorize(title, ColorBold))
	fmt.Println(strings.Repeat("-", len(title)))
}

// TrapOutput represents a trap for output.
type TrapOutput struct {
	Timestamp     time.Time        `json:"timestamp"`
	Version       string           `json:"version"`
	Community     string           `json:"community,omitempty"`
	SourceAddress string           `json:"source_address"`
	Enterprise    string           `json:"enterprise,omitempty"`
	AgentAddress  string           `json:"agent_address,omitempty"`
	GenericTrap   int              `json:"generic_trap,omitempty"`
	SpecificTrap  int              `json:"specific_trap,omitempty"`
	Uptime        string           `json:"uptime,omitempty"`
	Variables     []VariableOutput `json:"variables"`
}

// FormatTrap formats a trap for output.
func (f *Formatter) FormatTrap(trap *snmp.TrapPDU) {
	switch f.format {
	case FormatJSON:
		f.formatTrapJSON(trap)
	default:
		f.formatTrapTable(trap)
	}
}

func (f *Formatter) formatTrapTable(trap *snmp.TrapPDU) {
	fmt.Println()
	fmt.Println(colorize("=== TRAP RECEIVED ===", ColorBold))
	fmt.Printf("  %s: %s\n", colorize("Time", ColorCyan), time.Now().Format(time.RFC3339))
	fmt.Printf("  %s: %s\n", colorize("Source", ColorCyan), trap.SourceAddress)
	fmt.Printf("  %s: %s\n", colorize("Version", ColorCyan), trap.Version)
	fmt.Printf("  %s: %s\n", colorize("Community", ColorCyan), trap.Community)

	if trap.Version == snmp.Version1 {
		fmt.Printf("  %s: %s\n", colorize("Enterprise", ColorCyan), trap.Enterprise)
		fmt.Printf("  %s: %s\n", colorize("Agent Address", ColorCyan), trap.AgentAddress)
		fmt.Printf("  %s: %d\n", colorize("Generic Trap", ColorCyan), trap.GenericTrap)
		fmt.Printf("  %s: %d\n", colorize("Specific Trap", ColorCyan), trap.SpecificTrap)
	}

	fmt.Printf("  %s: %s\n", colorize("Uptime", ColorCyan), snmp.TimeTicksToString(trap.Timestamp))

	if len(trap.Variables) > 0 {
		fmt.Println()
		fmt.Println(colorize("Variables:", ColorBold))
		for _, v := range trap.Variables {
			fmt.Printf("    %s = %s: %s\n",
				colorize(v.OID.String(), ColorCyan),
				colorize(v.Type.String(), ColorYellow),
				formatValue(v))
		}
	}
	fmt.Println()
}

func (f *Formatter) formatTrapJSON(trap *snmp.TrapPDU) {
	output := TrapOutput{
		Timestamp:     time.Now(),
		Version:       trap.Version.String(),
		Community:     trap.Community,
		SourceAddress: trap.SourceAddress,
		Uptime:        snmp.TimeTicksToString(trap.Timestamp),
	}

	if trap.Version == snmp.Version1 {
		output.Enterprise = trap.Enterprise.String()
		output.AgentAddress = trap.AgentAddress
		output.GenericTrap = trap.GenericTrap
		output.SpecificTrap = trap.SpecificTrap
	}

	for _, v := range trap.Variables {
		output.Variables = append(output.Variables, VariableOutput{
			OID:   v.OID.String(),
			Type:  v.Type.String(),
			Value: convertValue(v),
		})
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Fprintln(f.writer, string(data))
}
