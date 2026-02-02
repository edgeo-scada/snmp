package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get OID [OID...]",
	Short: "Perform SNMP GET request",
	Long: `Perform an SNMP GET request to retrieve the value of one or more OIDs.

Examples:
  # Get system description
  edgeo-snmp get -t 192.168.1.1 1.3.6.1.2.1.1.1.0

  # Get multiple OIDs
  edgeo-snmp get -t 192.168.1.1 1.3.6.1.2.1.1.1.0 1.3.6.1.2.1.1.3.0 1.3.6.1.2.1.1.5.0

  # Using SNMPv3
  edgeo-snmp get -t 192.168.1.1 -V 3 -u admin -a SHA -A authpass -x AES -X privpass 1.3.6.1.2.1.1.1.0`,
	Args: cobra.MinimumNArgs(1),
	RunE: runGet,
}

var getNextCmd = &cobra.Command{
	Use:   "getnext OID [OID...]",
	Short: "Perform SNMP GET-NEXT request",
	Long: `Perform an SNMP GET-NEXT request to retrieve the next OID in the MIB tree.

Examples:
  # Get next OID after sysDescr
  edgeo-snmp getnext -t 192.168.1.1 1.3.6.1.2.1.1.1

  # Get next for multiple OIDs
  edgeo-snmp getnext -t 192.168.1.1 1.3.6.1.2.1.1.1 1.3.6.1.2.1.1.3`,
	Args: cobra.MinimumNArgs(1),
	RunE: runGetNext,
}

var getBulkCmd = &cobra.Command{
	Use:   "getbulk OID [OID...]",
	Short: "Perform SNMP GET-BULK request (v2c/v3)",
	Long: `Perform an SNMP GET-BULK request to efficiently retrieve multiple OIDs.
Only available for SNMPv2c and SNMPv3.

Examples:
  # Get bulk with default repetitions
  edgeo-snmp getbulk -t 192.168.1.1 1.3.6.1.2.1.2.2.1

  # Get bulk with custom repetitions
  edgeo-snmp getbulk -t 192.168.1.1 --max-repetitions 25 1.3.6.1.2.1.2.2.1`,
	Args: cobra.MinimumNArgs(1),
	RunE: runGetBulk,
}

var (
	maxRepetitions int
	nonRepeaters   int
)

func init() {
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(getNextCmd)
	rootCmd.AddCommand(getBulkCmd)

	getBulkCmd.Flags().IntVar(&maxRepetitions, "max-repetitions", 10, "max-repetitions value")
	getBulkCmd.Flags().IntVar(&nonRepeaters, "non-repeaters", 0, "non-repeaters value")
}

func runGet(cmd *cobra.Command, args []string) error {
	if err := checkTarget(); err != nil {
		return err
	}

	oids, err := parseOIDs(args)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt
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

	printVerbose("Sending GET request for %d OID(s)...", len(oids))
	start := time.Now()

	vars, err := client.Get(ctx, oids...)
	if err != nil {
		return fmt.Errorf("GET failed: %w", err)
	}

	printVerbose("Response received in %s", formatDuration(time.Since(start)))

	formatter := NewFormatter(outputFormat)
	formatter.FormatVariables(vars)

	return nil
}

func runGetNext(cmd *cobra.Command, args []string) error {
	if err := checkTarget(); err != nil {
		return err
	}

	oids, err := parseOIDs(args)
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

	printVerbose("Sending GET-NEXT request for %d OID(s)...", len(oids))
	start := time.Now()

	vars, err := client.GetNext(ctx, oids...)
	if err != nil {
		return fmt.Errorf("GET-NEXT failed: %w", err)
	}

	printVerbose("Response received in %s", formatDuration(time.Since(start)))

	formatter := NewFormatter(outputFormat)
	formatter.FormatVariables(vars)

	return nil
}

func runGetBulk(cmd *cobra.Command, args []string) error {
	if err := checkTarget(); err != nil {
		return err
	}

	if version == "1" || version == "v1" {
		return fmt.Errorf("GET-BULK is not available in SNMPv1")
	}

	oids, err := parseOIDs(args)
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

	printVerbose("Sending GET-BULK request (non-repeaters=%d, max-repetitions=%d)...",
		nonRepeaters, maxRepetitions)
	start := time.Now()

	vars, err := client.GetBulk(ctx, nonRepeaters, maxRepetitions, oids...)
	if err != nil {
		return fmt.Errorf("GET-BULK failed: %w", err)
	}

	printVerbose("Response received in %s (%d variables)", formatDuration(time.Since(start)), len(vars))

	formatter := NewFormatter(outputFormat)
	formatter.FormatVariables(vars)

	return nil
}
