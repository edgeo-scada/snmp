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
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgeo-scada/snmp/snmp"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get basic system information",
	Long: `Get basic system information from an SNMP agent.

Retrieves common system MIB objects:
  - sysDescr (1.3.6.1.2.1.1.1.0) - System description
  - sysObjectID (1.3.6.1.2.1.1.2.0) - System object identifier
  - sysUpTime (1.3.6.1.2.1.1.3.0) - Time since last reboot
  - sysContact (1.3.6.1.2.1.1.4.0) - Contact person
  - sysName (1.3.6.1.2.1.1.5.0) - System name
  - sysLocation (1.3.6.1.2.1.1.6.0) - Physical location

Examples:
  # Get system info
  edgeo-snmp info -t 192.168.1.1

  # Get info with SNMPv3
  edgeo-snmp info -t 192.168.1.1 -V 3 -u admin -a SHA -A authpass`,
	RunE: runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) error {
	if err := checkTarget(); err != nil {
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

	oids := []snmp.OID{
		snmp.OIDSysDescr,
		snmp.OIDSysObjectID,
		snmp.OIDSysUpTime,
		snmp.OIDSysContact,
		snmp.OIDSysName,
		snmp.OIDSysLocation,
	}

	printVerbose("Retrieving system information...")
	start := time.Now()

	vars, err := client.Get(ctx, oids...)
	if err != nil {
		return fmt.Errorf("failed to get system info: %w", err)
	}

	printVerbose("Response received in %s", formatDuration(time.Since(start)))

	if outputFormat == "json" {
		formatter := NewFormatter(outputFormat)
		formatter.FormatVariables(vars)
		return nil
	}

	// Pretty print system info
	fmt.Println()
	fmt.Println(colorize("System Information", ColorBold))
	fmt.Println(colorize("==================", ColorBold))

	for _, v := range vars {
		name := getOIDName(v.OID)
		value := formatValue(v)

		// Special handling for uptime
		if v.OID.Equal(snmp.OIDSysUpTime) {
			if ticks, ok := v.Value.(uint32); ok {
				value = snmp.TimeTicksToString(ticks)
			}
		}

		fmt.Printf("  %-15s %s\n", colorize(name+":", ColorCyan), value)
	}

	fmt.Println()
	return nil
}

func getOIDName(oid snmp.OID) string {
	switch {
	case oid.Equal(snmp.OIDSysDescr):
		return "Description"
	case oid.Equal(snmp.OIDSysObjectID):
		return "Object ID"
	case oid.Equal(snmp.OIDSysUpTime):
		return "Uptime"
	case oid.Equal(snmp.OIDSysContact):
		return "Contact"
	case oid.Equal(snmp.OIDSysName):
		return "Name"
	case oid.Equal(snmp.OIDSysLocation):
		return "Location"
	case oid.Equal(snmp.OIDSysServices):
		return "Services"
	default:
		return oid.String()
	}
}
