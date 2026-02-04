package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/edgeo-scada/snmp/snmp"
)

// createClient creates and connects an SNMP client with the current configuration.
func createClient(ctx context.Context) (*snmp.Client, error) {
	opts := buildClientOptions()
	client := snmp.NewClient(opts...)

	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := client.Connect(connectCtx); err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}

	return client, nil
}

// buildClientOptions builds SNMP client options from the current configuration.
func buildClientOptions() []snmp.Option {
	opts := []snmp.Option{
		snmp.WithTarget(target),
		snmp.WithPort(port),
		snmp.WithCommunity(community),
		snmp.WithTimeout(timeout),
		snmp.WithRetries(retries),
		snmp.WithAutoReconnect(false),
	}

	// Parse SNMP version
	switch strings.ToLower(version) {
	case "1", "v1":
		opts = append(opts, snmp.WithVersion(snmp.Version1))
	case "2c", "v2c", "2":
		opts = append(opts, snmp.WithVersion(snmp.Version2c))
	case "3", "v3":
		opts = append(opts, snmp.WithVersion(snmp.Version3))
		opts = append(opts, buildV3Options()...)
	}

	if verbose {
		opts = append(opts, snmp.WithOnConnect(func(c *snmp.Client) {
			fmt.Fprintln(os.Stderr, "Connected to agent")
		}))
		opts = append(opts, snmp.WithOnConnectionLost(func(c *snmp.Client, err error) {
			fmt.Fprintf(os.Stderr, "Connection lost: %v\n", err)
		}))
		// Enable debug logging
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		opts = append(opts, snmp.WithLogger(logger))
	}

	return opts
}

// buildV3Options builds SNMPv3-specific options.
func buildV3Options() []snmp.Option {
	var opts []snmp.Option

	// Security level
	switch strings.ToLower(securityLevel) {
	case "noauthnopriv":
		opts = append(opts, snmp.WithSecurityLevel(snmp.NoAuthNoPriv))
	case "authnopriv":
		opts = append(opts, snmp.WithSecurityLevel(snmp.AuthNoPriv))
	case "authpriv":
		opts = append(opts, snmp.WithSecurityLevel(snmp.AuthPriv))
	}

	// Security name
	if securityName != "" {
		opts = append(opts, snmp.WithSecurityName(securityName))
	}

	// Auth protocol
	if authProtocol != "" {
		var proto snmp.AuthProtocol
		switch strings.ToUpper(authProtocol) {
		case "MD5":
			proto = snmp.MD5
		case "SHA", "SHA-1":
			proto = snmp.SHA
		case "SHA-224":
			proto = snmp.SHA224
		case "SHA-256":
			proto = snmp.SHA256
		case "SHA-384":
			proto = snmp.SHA384
		case "SHA-512":
			proto = snmp.SHA512
		}
		opts = append(opts, snmp.WithAuth(proto, authPassphrase))
	}

	// Privacy protocol
	if privProtocol != "" {
		var proto snmp.PrivProtocol
		switch strings.ToUpper(privProtocol) {
		case "DES":
			proto = snmp.DES
		case "AES", "AES-128":
			proto = snmp.AES
		case "AES-192":
			proto = snmp.AES192
		case "AES-256":
			proto = snmp.AES256
		}
		opts = append(opts, snmp.WithPrivacy(proto, privPassphrase))
	}

	// Context name
	if contextName != "" {
		opts = append(opts, snmp.WithContextName(contextName))
	}

	return opts
}

// disconnectClient gracefully disconnects the client.
func disconnectClient(client *snmp.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client.Disconnect(ctx)
}

// printVerbose prints a message if verbose mode is enabled.
func printVerbose(format string, args ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

// printError prints an error message to stderr.
func printError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

// parseOID parses an OID string.
func parseOID(s string) (snmp.OID, error) {
	return snmp.ParseOID(s)
}

// parseOIDs parses multiple OID strings.
func parseOIDs(args []string) ([]snmp.OID, error) {
	oids := make([]snmp.OID, len(args))
	for i, arg := range args {
		oid, err := snmp.ParseOID(arg)
		if err != nil {
			return nil, fmt.Errorf("invalid OID '%s': %w", arg, err)
		}
		oids[i] = oid
	}
	return oids, nil
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// formatBytes formats bytes for display.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// checkTarget verifies that a target is specified.
func checkTarget() error {
	if target == "" {
		return fmt.Errorf("target is required (use -t or --target)")
	}
	return nil
}
