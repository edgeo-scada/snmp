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

// Package snmp provides a pure Go SNMP v1/v2c/v3 client implementation.
package snmp

// Version information for the SNMP client library.
const (
	// Version is the current version of the library.
	Version = "1.0.0"

	// ProtocolName is the SNMP protocol name.
	ProtocolName = "SNMP"
)

// SNMPVersion represents the SNMP protocol version.
type SNMPVersion int

const (
	// Version1 is SNMP v1.
	Version1 SNMPVersion = 0
	// Version2c is SNMP v2c.
	Version2c SNMPVersion = 1
	// Version3 is SNMP v3.
	Version3 SNMPVersion = 3
)

// String returns the string representation of the SNMP version.
func (v SNMPVersion) String() string {
	switch v {
	case Version1:
		return "SNMPv1"
	case Version2c:
		return "SNMPv2c"
	case Version3:
		return "SNMPv3"
	default:
		return "Unknown"
	}
}

// BuildInfo contains build metadata.
type BuildInfo struct {
	Version   string
	GoVersion string
	OS        string
	Arch      string
	BuildTime string
}

// GetBuildInfo returns the current build information.
func GetBuildInfo() BuildInfo {
	return BuildInfo{
		Version: Version,
	}
}
