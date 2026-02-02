// edgeo-snmp is a command-line SNMP client for testing, debugging, and monitoring.
package main

import (
	"os"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
