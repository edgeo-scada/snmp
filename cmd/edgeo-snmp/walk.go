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

var walkCmd = &cobra.Command{
	Use:   "walk OID",
	Short: "Walk an SNMP MIB subtree",
	Long: `Walk an SNMP MIB subtree starting from the given OID.

For SNMPv1, this uses GET-NEXT requests.
For SNMPv2c/v3, this uses GET-BULK requests for better performance.

Examples:
  # Walk the system group
  edgeo-snmp walk -t 192.168.1.1 1.3.6.1.2.1.1

  # Walk interface table
  edgeo-snmp walk -t 192.168.1.1 1.3.6.1.2.1.2.2

  # Walk entire MIB
  edgeo-snmp walk -t 192.168.1.1 1.3`,
	Args: cobra.ExactArgs(1),
	RunE: runWalk,
}

var bulkWalkCmd = &cobra.Command{
	Use:   "bulkwalk OID",
	Short: "Walk using GET-BULK (v2c/v3)",
	Long: `Walk an SNMP MIB subtree using GET-BULK requests.
Only available for SNMPv2c and SNMPv3.

This is more efficient than regular walk for large subtrees.

Examples:
  # Bulk walk interface table
  edgeo-snmp bulkwalk -t 192.168.1.1 1.3.6.1.2.1.2.2

  # Bulk walk with custom repetitions
  edgeo-snmp bulkwalk -t 192.168.1.1 --max-repetitions 50 1.3.6.1.2.1.2.2`,
	Args: cobra.ExactArgs(1),
	RunE: runBulkWalk,
}

var (
	walkMaxRepetitions int
	walkShowCount      bool
)

func init() {
	rootCmd.AddCommand(walkCmd)
	rootCmd.AddCommand(bulkWalkCmd)

	walkCmd.Flags().IntVar(&walkMaxRepetitions, "max-repetitions", 10, "max-repetitions for bulk operations")
	walkCmd.Flags().BoolVar(&walkShowCount, "count", false, "show count of variables at the end")

	bulkWalkCmd.Flags().IntVar(&walkMaxRepetitions, "max-repetitions", 10, "max-repetitions value")
	bulkWalkCmd.Flags().BoolVar(&walkShowCount, "count", false, "show count of variables at the end")
}

func runWalk(cmd *cobra.Command, args []string) error {
	if err := checkTarget(); err != nil {
		return err
	}

	rootOID, err := parseOID(args[0])
	if err != nil {
		return fmt.Errorf("invalid OID: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nInterrupted")
		cancel()
	}()

	client, err := createClient(ctx)
	if err != nil {
		return err
	}
	defer disconnectClient(client)

	// Override max-repetitions if specified
	if walkMaxRepetitions > 0 && client.Options().Version != snmp.Version1 {
		client.Options().MaxRepetitions = walkMaxRepetitions
	}

	printVerbose("Walking from %s...", rootOID)
	start := time.Now()

	formatter := NewFormatter(outputFormat)
	count := 0

	err = client.WalkFunc(ctx, rootOID, func(v snmp.Variable) error {
		formatter.FormatVariable(v)
		count++
		return nil
	})

	elapsed := time.Since(start)

	if err != nil && ctx.Err() == nil {
		return fmt.Errorf("walk failed: %w", err)
	}

	if walkShowCount || verbose {
		fmt.Fprintf(os.Stderr, "\n%d variables retrieved in %s\n", count, formatDuration(elapsed))
	}

	return nil
}

func runBulkWalk(cmd *cobra.Command, args []string) error {
	if err := checkTarget(); err != nil {
		return err
	}

	if version == "1" || version == "v1" {
		return fmt.Errorf("bulk walk is not available in SNMPv1, use 'walk' instead")
	}

	rootOID, err := parseOID(args[0])
	if err != nil {
		return fmt.Errorf("invalid OID: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nInterrupted")
		cancel()
	}()

	client, err := createClient(ctx)
	if err != nil {
		return err
	}
	defer disconnectClient(client)

	// Set max-repetitions
	client.Options().MaxRepetitions = walkMaxRepetitions

	printVerbose("Bulk walking from %s (max-repetitions=%d)...", rootOID, walkMaxRepetitions)
	start := time.Now()

	formatter := NewFormatter(outputFormat)
	count := 0

	err = client.WalkFunc(ctx, rootOID, func(v snmp.Variable) error {
		formatter.FormatVariable(v)
		count++
		return nil
	})

	elapsed := time.Since(start)

	if err != nil && ctx.Err() == nil {
		return fmt.Errorf("bulk walk failed: %w", err)
	}

	if walkShowCount || verbose {
		fmt.Fprintf(os.Stderr, "\n%d variables retrieved in %s\n", count, formatDuration(elapsed))
	}

	return nil
}
