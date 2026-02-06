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
