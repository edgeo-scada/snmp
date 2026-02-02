package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Global flags
	cfgFile   string
	target    string
	port      int
	community string
	version   string
	timeout   time.Duration
	retries   int

	// SNMPv3 flags
	securityLevel  string
	securityName   string
	authProtocol   string
	authPassphrase string
	privProtocol   string
	privPassphrase string
	contextName    string

	// Output flags
	outputFormat string
	verbose      bool
	noColor      bool
	numeric      bool
)

var rootCmd = &cobra.Command{
	Use:   "edgeo-snmp",
	Short: "SNMP command-line client",
	Long: `edgeo-snmp is a complete SNMP v1/v2c/v3 command-line client for testing,
debugging, monitoring, and managing network devices.

Supports:
  - SNMPv1, SNMPv2c, and SNMPv3
  - GET, GET-NEXT, GET-BULK, SET operations
  - WALK and BULK-WALK
  - Trap receiving

Examples:
  # Get system description
  edgeo-snmp get -t 192.168.1.1 1.3.6.1.2.1.1.1.0

  # Walk interface table
  edgeo-snmp walk -t 192.168.1.1 1.3.6.1.2.1.2.2

  # Set a value
  edgeo-snmp set -t 192.168.1.1 1.3.6.1.2.1.1.4.0 s "admin@example.com"

  # Listen for traps
  edgeo-snmp trap-listen`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	cobra.OnInitialize(initConfig)

	// Connection flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "", "", "config file (default is $HOME/.edgeo-snmp.yaml)")
	rootCmd.PersistentFlags().StringVarP(&target, "target", "t", "", "SNMP agent address (required)")
	rootCmd.PersistentFlags().IntVarP(&port, "port", "p", 161, "SNMP agent port")
	rootCmd.PersistentFlags().StringVarP(&community, "community", "c", "public", "community string (v1/v2c)")
	rootCmd.PersistentFlags().StringVarP(&version, "version", "V", "2c", "SNMP version (1, 2c, 3)")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 5*time.Second, "request timeout")
	rootCmd.PersistentFlags().IntVarP(&retries, "retries", "r", 3, "number of retries")

	// SNMPv3 flags
	rootCmd.PersistentFlags().StringVar(&securityLevel, "security-level", "noAuthNoPriv", "security level (noAuthNoPriv, authNoPriv, authPriv)")
	rootCmd.PersistentFlags().StringVarP(&securityName, "security-name", "u", "", "security name (username)")
	rootCmd.PersistentFlags().StringVarP(&authProtocol, "auth-protocol", "a", "", "auth protocol (MD5, SHA, SHA-224, SHA-256, SHA-384, SHA-512)")
	rootCmd.PersistentFlags().StringVarP(&authPassphrase, "auth-passphrase", "A", "", "auth passphrase")
	rootCmd.PersistentFlags().StringVarP(&privProtocol, "priv-protocol", "x", "", "privacy protocol (DES, AES, AES-192, AES-256)")
	rootCmd.PersistentFlags().StringVarP(&privPassphrase, "priv-passphrase", "X", "", "privacy passphrase")
	rootCmd.PersistentFlags().StringVarP(&contextName, "context", "n", "", "context name")

	// Output flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "output format: table, json, csv, raw")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().BoolVar(&numeric, "numeric", false, "print OIDs numerically")

	// Bind flags to viper
	viper.BindPFlag("target", rootCmd.PersistentFlags().Lookup("target"))
	viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("community", rootCmd.PersistentFlags().Lookup("community"))
	viper.BindPFlag("version", rootCmd.PersistentFlags().Lookup("version"))
	viper.BindPFlag("timeout", rootCmd.PersistentFlags().Lookup("timeout"))
	viper.BindPFlag("retries", rootCmd.PersistentFlags().Lookup("retries"))
	viper.BindPFlag("security-level", rootCmd.PersistentFlags().Lookup("security-level"))
	viper.BindPFlag("security-name", rootCmd.PersistentFlags().Lookup("security-name"))
	viper.BindPFlag("auth-protocol", rootCmd.PersistentFlags().Lookup("auth-protocol"))
	viper.BindPFlag("auth-passphrase", rootCmd.PersistentFlags().Lookup("auth-passphrase"))
	viper.BindPFlag("priv-protocol", rootCmd.PersistentFlags().Lookup("priv-protocol"))
	viper.BindPFlag("priv-passphrase", rootCmd.PersistentFlags().Lookup("priv-passphrase"))
	viper.BindPFlag("context", rootCmd.PersistentFlags().Lookup("context"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("no-color", rootCmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("numeric", rootCmd.PersistentFlags().Lookup("numeric"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		viper.AddConfigPath(home)
		viper.AddConfigPath(filepath.Join(home, ".config"))
		viper.SetConfigName(".edgeo-snmp")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("EDGEO_SNMP")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}

	// Apply viper values to flags
	target = viper.GetString("target")
	port = viper.GetInt("port")
	community = viper.GetString("community")
	version = viper.GetString("version")
	timeout = viper.GetDuration("timeout")
	retries = viper.GetInt("retries")
	securityLevel = viper.GetString("security-level")
	securityName = viper.GetString("security-name")
	authProtocol = viper.GetString("auth-protocol")
	authPassphrase = viper.GetString("auth-passphrase")
	privProtocol = viper.GetString("priv-protocol")
	privPassphrase = viper.GetString("priv-passphrase")
	contextName = viper.GetString("context")
	outputFormat = viper.GetString("output")
	verbose = viper.GetBool("verbose")
	noColor = viper.GetBool("no-color")
	numeric = viper.GetBool("numeric")
}
