# edgeo-snmp - SNMP Command Line Interface

A complete command-line tool for SNMP operations.

## Installation

```bash
go build -o edgeo-snmp ./cmd/edgeo-snmp
```

## Commands Overview

| Command | Description |
|---------|-------------|
| `get` | GET operation on OID(s) |
| `set` | SET operation to modify values |
| `walk` | Walk MIB subtree |
| `trap-listen` | Listen for trap notifications |
| `info` | Display device information |
| `version` | Print version information |

## Global Flags

### Connection

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--target` | `-t` | - | SNMP agent address (required) |
| `--port` | `-p` | `161` | Agent port |
| `--version` | `-V` | `2c` | SNMP version: 1, 2c, 3 |
| `--community` | `-c` | `public` | Community string |
| `--timeout` | | `5s` | Request timeout |
| `--retries` | `-r` | `3` | Number of retries |

### SNMPv3

| Flag | Description |
|------|-------------|
| `--security-level` | Security level: noAuthNoPriv, authNoPriv, authPriv |
| `--security-name` | SNMPv3 username |
| `--auth-protocol` | Auth protocol: MD5, SHA, SHA256, SHA384, SHA512 |
| `--auth-passphrase` | Authentication password |
| `--priv-protocol` | Privacy protocol: DES, AES, AES192, AES256 |
| `--priv-passphrase` | Encryption password |

### Output

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `table` | Format: table, json, csv, raw |
| `--verbose` | `-v` | `false` | Verbose output |
| `--numeric` | `-n` | `false` | Show numeric OIDs |

### Configuration

| Flag | Description |
|------|-------------|
| `--config` | Config file path (default: `~/.edgeo-snmp.yaml`) |

## Command: get

Retrieve OID values.

### Usage

```bash
edgeo-snmp get -t <target> <oid> [oid...]
```

### Examples

```bash
# Get single OID
edgeo-snmp get -t 192.168.1.1 1.3.6.1.2.1.1.1.0

# Get system info
edgeo-snmp get -t 192.168.1.1 sysDescr.0 sysName.0 sysUpTime.0

# Get multiple OIDs
edgeo-snmp get -t 192.168.1.1 \
    1.3.6.1.2.1.1.1.0 \
    1.3.6.1.2.1.1.3.0 \
    1.3.6.1.2.1.1.5.0

# SNMPv3 with auth and privacy
edgeo-snmp get -t 192.168.1.1 -V 3 \
    --security-level authPriv \
    --security-name admin \
    --auth-protocol SHA256 \
    --auth-passphrase myauthpass \
    --priv-protocol AES256 \
    --priv-passphrase myprivpass \
    sysDescr.0

# JSON output
edgeo-snmp get -t 192.168.1.1 -o json sysDescr.0
```

## Command: set

Modify OID values.

### Usage

```bash
edgeo-snmp set -t <target> <oid> <type> <value>
```

### Types

| Type | Flag | Description |
|------|------|-------------|
| Integer | `i` | 32-bit integer |
| String | `s` | Octet string |
| OID | `o` | Object identifier |
| IP Address | `a` | IP address |
| Counter32 | `c` | 32-bit counter |
| Gauge32 | `g` | 32-bit gauge |
| TimeTicks | `t` | Time ticks |

### Examples

```bash
# Set system contact
edgeo-snmp set -t 192.168.1.1 sysContact.0 s "admin@example.com"

# Set system location
edgeo-snmp set -t 192.168.1.1 sysLocation.0 s "Server Room"

# Set integer value
edgeo-snmp set -t 192.168.1.1 1.3.6.1.4.1.9.2.1.55.192.168.1.1 i 1

# With write community
edgeo-snmp set -t 192.168.1.1 -c private sysContact.0 s "admin@example.com"
```

## Command: walk

Walk a MIB subtree.

### Usage

```bash
edgeo-snmp walk -t <target> <oid>
```

### Flags

| Flag | Description |
|------|-------------|
| `--bulk` | Use GetBulk (v2c/v3 only) |
| `--max-repetitions` | Max repetitions for GetBulk (default: 10) |

### Examples

```bash
# Walk system subtree
edgeo-snmp walk -t 192.168.1.1 system

# Walk interfaces
edgeo-snmp walk -t 192.168.1.1 1.3.6.1.2.1.2.2

# Walk with GetBulk
edgeo-snmp walk -t 192.168.1.1 --bulk --max-repetitions 20 ifTable

# Numeric OIDs
edgeo-snmp walk -t 192.168.1.1 -n 1.3.6.1.2.1.2.2
```

## Command: trap-listen

Listen for SNMP trap notifications.

### Usage

```bash
edgeo-snmp trap-listen [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--listen` | Listen address (default: ":162") |
| `--community` | Filter by community (empty = all) |

### Examples

```bash
# Listen on default port (requires root)
sudo edgeo-snmp trap-listen

# Listen on non-privileged port
edgeo-snmp trap-listen --listen :1162

# Filter by community
edgeo-snmp trap-listen --listen :1162 --community public

# JSON output
edgeo-snmp trap-listen --listen :1162 -o json
```

## Command: info

Display device information.

### Usage

```bash
edgeo-snmp info -t <target>
```

### Examples

```bash
# Get device info
edgeo-snmp info -t 192.168.1.1

# JSON output
edgeo-snmp info -t 192.168.1.1 -o json
```

**Output:**

```
Device Information
------------------
System Description: Cisco IOS Software, C2960 Software
System OID:         1.3.6.1.4.1.9.1.1208
System Name:        switch-1
System Location:    Server Room
System Contact:     admin@example.com
System Uptime:      45 days 12:34:56
```

## OID Shortcuts

The CLI supports common OID shortcuts:

| Shortcut | OID |
|----------|-----|
| `sysDescr` | 1.3.6.1.2.1.1.1 |
| `sysObjectID` | 1.3.6.1.2.1.1.2 |
| `sysUpTime` | 1.3.6.1.2.1.1.3 |
| `sysContact` | 1.3.6.1.2.1.1.4 |
| `sysName` | 1.3.6.1.2.1.1.5 |
| `sysLocation` | 1.3.6.1.2.1.1.6 |
| `ifTable` | 1.3.6.1.2.1.2.2 |
| `ifDescr` | 1.3.6.1.2.1.2.2.1.2 |
| `ifOperStatus` | 1.3.6.1.2.1.2.2.1.8 |

## Configuration File

Create `~/.edgeo-snmp.yaml` for default settings:

```yaml
# Connection
target: 192.168.1.1
port: 161
version: "2c"
community: public

# Timeouts
timeout: 5s
retries: 3

# SNMPv3 (if version: "3")
security-level: authPriv
security-name: admin
auth-protocol: SHA256
auth-passphrase: myauthpass
priv-protocol: AES256
priv-passphrase: myprivpass

# Output
output: table
verbose: false
numeric: false
```

## Environment Variables

Environment variables use the `SNMP_` prefix:

```bash
export SNMP_TARGET=192.168.1.1
export SNMP_COMMUNITY=public
export SNMP_VERSION=2c
export SNMP_TIMEOUT=5s
```

## Output Formats

### Table (default)

```
OID                      TYPE           VALUE
1.3.6.1.2.1.1.1.0       OctetString    Cisco IOS Software...
1.3.6.1.2.1.1.3.0       TimeTicks      393456789
1.3.6.1.2.1.1.5.0       OctetString    switch-1
```

### JSON

```json
[
  {"oid": "1.3.6.1.2.1.1.1.0", "type": "OctetString", "value": "Cisco IOS Software..."},
  {"oid": "1.3.6.1.2.1.1.3.0", "type": "TimeTicks", "value": 393456789}
]
```

### CSV

```csv
oid,type,value
1.3.6.1.2.1.1.1.0,OctetString,Cisco IOS Software...
1.3.6.1.2.1.1.3.0,TimeTicks,393456789
```

### Raw

```
Cisco IOS Software...
```

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | Error (connection failed, OID not found, etc.) |

## See Also

- [Client Library Documentation](client.md)
- [Configuration Options](options.md)
- [Trap Listener](trap.md)
