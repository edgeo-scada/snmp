package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/edgeo-scada/snmp/snmp"
	"github.com/spf13/cobra"
)

var trapListenCmd = &cobra.Command{
	Use:   "trap-listen",
	Short: "Listen for SNMP traps",
	Long: `Start a listener to receive SNMP traps and notifications.

By default, listens on port 162 (the standard SNMP trap port).
Note: Port 162 typically requires root/administrator privileges.

Examples:
  # Listen on default port (162)
  sudo edgeo-snmp trap-listen

  # Listen on alternate port
  edgeo-snmp trap-listen --listen ":1162"

  # Listen with community filter
  edgeo-snmp trap-listen --trap-community private`,
	RunE: runTrapListen,
}

var (
	listenAddress string
	trapCommunity string
)

func init() {
	rootCmd.AddCommand(trapListenCmd)

	trapListenCmd.Flags().StringVar(&listenAddress, "listen", ":162", "listen address (host:port)")
	trapListenCmd.Flags().StringVar(&trapCommunity, "trap-community", "", "filter by community string (empty = accept all)")
}

func runTrapListen(cmd *cobra.Command, args []string) error {
	fmt.Printf("Starting SNMP trap listener on %s\n", listenAddress)
	if trapCommunity != "" {
		fmt.Printf("Filtering by community: %s\n", trapCommunity)
	}
	fmt.Println("Press Ctrl+C to stop...")
	fmt.Println()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	formatter := NewFormatter(outputFormat)

	listener := snmp.NewTrapListener(
		func(trap *snmp.TrapPDU) {
			formatter.FormatTrap(trap)
		},
		snmp.WithListenAddress(listenAddress),
		snmp.WithTrapCommunity(trapCommunity),
	)

	if err := listener.Start(ctx); err != nil {
		return fmt.Errorf("failed to start trap listener: %w", err)
	}

	// Wait for interrupt
	<-sigCh
	fmt.Println("\nShutting down...")

	return listener.Stop()
}
