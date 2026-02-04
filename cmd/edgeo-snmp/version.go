package main

import (
	"fmt"
	"runtime"

	"github.com/edgeo-scada/snmp/snmp"
	"github.com/spf13/cobra"
)

var (
	// Build information, set via ldflags
	cliVersion = "dev"
	commit     = "unknown"
	buildDate  = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print detailed version information including build metadata.`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersionInfo()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func printVersionInfo() {
	fmt.Printf("edgeo-snmp version %s\n", cliVersion)
	fmt.Printf("  SNMP Library:  %s\n", snmp.Version)
	fmt.Printf("  SNMP Protocol: %s\n", snmp.ProtocolName)
	fmt.Printf("  Go version:    %s\n", runtime.Version())
	fmt.Printf("  OS/Arch:       %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  Git commit:    %s\n", commit)
	fmt.Printf("  Build date:    %s\n", buildDate)
}
