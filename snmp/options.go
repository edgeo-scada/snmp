package snmp

import (
	"log/slog"
	"time"
)

// ClientOptions contains configuration options for the SNMP client.
type ClientOptions struct {
	// Target is the SNMP agent address (host:port).
	Target string
	// Port is the SNMP agent port (default 161).
	Port int
	// Version is the SNMP version to use.
	Version SNMPVersion
	// Community is the community string (v1/v2c).
	Community string
	// Timeout is the request timeout.
	Timeout time.Duration
	// Retries is the number of retries on timeout.
	Retries int
	// MaxOids is the maximum OIDs per request.
	MaxOids int
	// MaxRepetitions is the max-repetitions for GetBulk (v2c/v3).
	MaxRepetitions int
	// NonRepeaters is the non-repeaters for GetBulk.
	NonRepeaters int

	// SNMPv3 Security
	SecurityLevel    SecurityLevel
	SecurityName     string
	AuthProtocol     AuthProtocol
	AuthPassphrase   string
	PrivProtocol     PrivProtocol
	PrivPassphrase   string
	ContextName      string
	ContextEngineID  string

	// Connection
	AutoReconnect        bool
	MaxReconnectInterval time.Duration
	ConnectRetryInterval time.Duration
	MaxRetries           int

	// Callbacks
	OnConnect        OnConnectHandler
	OnConnectionLost ConnectionLostHandler
	OnReconnecting   ReconnectHandler

	// Logger
	Logger *slog.Logger
}

// SecurityLevel represents SNMPv3 security levels.
type SecurityLevel int

const (
	// NoAuthNoPriv - No authentication, no privacy.
	NoAuthNoPriv SecurityLevel = iota
	// AuthNoPriv - Authentication, no privacy.
	AuthNoPriv
	// AuthPriv - Authentication and privacy.
	AuthPriv
)

// String returns the string representation of the security level.
func (s SecurityLevel) String() string {
	switch s {
	case NoAuthNoPriv:
		return "noAuthNoPriv"
	case AuthNoPriv:
		return "authNoPriv"
	case AuthPriv:
		return "authPriv"
	default:
		return "unknown"
	}
}

// AuthProtocol represents SNMPv3 authentication protocols.
type AuthProtocol int

const (
	NoAuth AuthProtocol = iota
	MD5
	SHA
	SHA224
	SHA256
	SHA384
	SHA512
)

// String returns the string representation of the auth protocol.
func (a AuthProtocol) String() string {
	switch a {
	case NoAuth:
		return "NoAuth"
	case MD5:
		return "MD5"
	case SHA:
		return "SHA"
	case SHA224:
		return "SHA-224"
	case SHA256:
		return "SHA-256"
	case SHA384:
		return "SHA-384"
	case SHA512:
		return "SHA-512"
	default:
		return "unknown"
	}
}

// PrivProtocol represents SNMPv3 privacy protocols.
type PrivProtocol int

const (
	NoPriv PrivProtocol = iota
	DES
	AES
	AES192
	AES256
	AES192C
	AES256C
)

// String returns the string representation of the privacy protocol.
func (p PrivProtocol) String() string {
	switch p {
	case NoPriv:
		return "NoPriv"
	case DES:
		return "DES"
	case AES:
		return "AES"
	case AES192:
		return "AES-192"
	case AES256:
		return "AES-256"
	case AES192C:
		return "AES-192-C"
	case AES256C:
		return "AES-256-C"
	default:
		return "unknown"
	}
}

// NewClientOptions creates ClientOptions with default values.
func NewClientOptions() *ClientOptions {
	return &ClientOptions{
		Port:                 DefaultPort,
		Version:              Version2c,
		Community:            DefaultCommunity,
		Timeout:              DefaultTimeout,
		Retries:              DefaultRetries,
		MaxOids:              DefaultMaxOids,
		MaxRepetitions:       DefaultMaxRepetitions,
		NonRepeaters:         DefaultNonRepeaters,
		AutoReconnect:        true,
		MaxReconnectInterval: 2 * time.Minute,
		ConnectRetryInterval: time.Second,
		MaxRetries:           0,
		SecurityLevel:        NoAuthNoPriv,
	}
}

// Option is a functional option for configuring the client.
type Option func(*ClientOptions)

// WithTarget sets the target address.
func WithTarget(target string) Option {
	return func(o *ClientOptions) {
		o.Target = target
	}
}

// WithPort sets the target port.
func WithPort(port int) Option {
	return func(o *ClientOptions) {
		o.Port = port
	}
}

// WithVersion sets the SNMP version.
func WithVersion(version SNMPVersion) Option {
	return func(o *ClientOptions) {
		o.Version = version
	}
}

// WithCommunity sets the community string.
func WithCommunity(community string) Option {
	return func(o *ClientOptions) {
		o.Community = community
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(d time.Duration) Option {
	return func(o *ClientOptions) {
		o.Timeout = d
	}
}

// WithRetries sets the number of retries.
func WithRetries(n int) Option {
	return func(o *ClientOptions) {
		o.Retries = n
	}
}

// WithMaxOids sets the maximum OIDs per request.
func WithMaxOids(n int) Option {
	return func(o *ClientOptions) {
		o.MaxOids = n
	}
}

// WithMaxRepetitions sets the max-repetitions for GetBulk.
func WithMaxRepetitions(n int) Option {
	return func(o *ClientOptions) {
		o.MaxRepetitions = n
	}
}

// WithNonRepeaters sets the non-repeaters for GetBulk.
func WithNonRepeaters(n int) Option {
	return func(o *ClientOptions) {
		o.NonRepeaters = n
	}
}

// WithSecurityLevel sets the SNMPv3 security level.
func WithSecurityLevel(level SecurityLevel) Option {
	return func(o *ClientOptions) {
		o.SecurityLevel = level
	}
}

// WithSecurityName sets the SNMPv3 security name (username).
func WithSecurityName(name string) Option {
	return func(o *ClientOptions) {
		o.SecurityName = name
	}
}

// WithAuth sets the SNMPv3 authentication parameters.
func WithAuth(protocol AuthProtocol, passphrase string) Option {
	return func(o *ClientOptions) {
		o.AuthProtocol = protocol
		o.AuthPassphrase = passphrase
	}
}

// WithPrivacy sets the SNMPv3 privacy parameters.
func WithPrivacy(protocol PrivProtocol, passphrase string) Option {
	return func(o *ClientOptions) {
		o.PrivProtocol = protocol
		o.PrivPassphrase = passphrase
	}
}

// WithContextName sets the SNMPv3 context name.
func WithContextName(name string) Option {
	return func(o *ClientOptions) {
		o.ContextName = name
	}
}

// WithContextEngineID sets the SNMPv3 context engine ID.
func WithContextEngineID(id string) Option {
	return func(o *ClientOptions) {
		o.ContextEngineID = id
	}
}

// WithAutoReconnect enables or disables automatic reconnection.
func WithAutoReconnect(enabled bool) Option {
	return func(o *ClientOptions) {
		o.AutoReconnect = enabled
	}
}

// WithMaxReconnectInterval sets the maximum reconnection interval.
func WithMaxReconnectInterval(d time.Duration) Option {
	return func(o *ClientOptions) {
		o.MaxReconnectInterval = d
	}
}

// WithConnectRetryInterval sets the initial reconnection interval.
func WithConnectRetryInterval(d time.Duration) Option {
	return func(o *ClientOptions) {
		o.ConnectRetryInterval = d
	}
}

// WithMaxConnectRetries sets the maximum number of reconnection attempts.
func WithMaxConnectRetries(n int) Option {
	return func(o *ClientOptions) {
		o.MaxRetries = n
	}
}

// WithOnConnect sets the connection callback.
func WithOnConnect(handler OnConnectHandler) Option {
	return func(o *ClientOptions) {
		o.OnConnect = handler
	}
}

// WithOnConnectionLost sets the connection lost callback.
func WithOnConnectionLost(handler ConnectionLostHandler) Option {
	return func(o *ClientOptions) {
		o.OnConnectionLost = handler
	}
}

// WithOnReconnecting sets the reconnecting callback.
func WithOnReconnecting(handler ReconnectHandler) Option {
	return func(o *ClientOptions) {
		o.OnReconnecting = handler
	}
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) Option {
	return func(o *ClientOptions) {
		o.Logger = logger
	}
}

// PoolOptions contains configuration options for the connection pool.
type PoolOptions struct {
	// Size is the number of connections in the pool.
	Size int
	// MaxIdleTime is the maximum time a connection can be idle.
	MaxIdleTime time.Duration
	// HealthCheckInterval is the interval between health checks.
	HealthCheckInterval time.Duration
	// ClientOptions are the options for each client in the pool.
	ClientOptions []Option
}

// NewPoolOptions creates PoolOptions with default values.
func NewPoolOptions() *PoolOptions {
	return &PoolOptions{
		Size:                3,
		MaxIdleTime:         5 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
	}
}

// PoolOption is a functional option for configuring the pool.
type PoolOption func(*PoolOptions)

// WithPoolSize sets the pool size.
func WithPoolSize(size int) PoolOption {
	return func(o *PoolOptions) {
		o.Size = size
	}
}

// WithPoolMaxIdleTime sets the maximum idle time.
func WithPoolMaxIdleTime(d time.Duration) PoolOption {
	return func(o *PoolOptions) {
		o.MaxIdleTime = d
	}
}

// WithPoolHealthCheckInterval sets the health check interval.
func WithPoolHealthCheckInterval(d time.Duration) PoolOption {
	return func(o *PoolOptions) {
		o.HealthCheckInterval = d
	}
}

// WithPoolClientOptions sets client options for pool connections.
func WithPoolClientOptions(opts ...Option) PoolOption {
	return func(o *PoolOptions) {
		o.ClientOptions = opts
	}
}

// TrapListenerOptions contains configuration for the trap listener.
type TrapListenerOptions struct {
	// Address is the listen address (default ":162").
	Address string
	// Community is the expected community string (empty = accept all).
	Community string
	// Logger is the logger.
	Logger *slog.Logger
}

// NewTrapListenerOptions creates TrapListenerOptions with default values.
func NewTrapListenerOptions() *TrapListenerOptions {
	return &TrapListenerOptions{
		Address: ":162",
	}
}

// TrapListenerOption is a functional option for configuring the trap listener.
type TrapListenerOption func(*TrapListenerOptions)

// WithListenAddress sets the listen address.
func WithListenAddress(addr string) TrapListenerOption {
	return func(o *TrapListenerOptions) {
		o.Address = addr
	}
}

// WithTrapCommunity sets the expected community string.
func WithTrapCommunity(community string) TrapListenerOption {
	return func(o *TrapListenerOptions) {
		o.Community = community
	}
}

// WithTrapLogger sets the logger for the trap listener.
func WithTrapLogger(logger *slog.Logger) TrapListenerOption {
	return func(o *TrapListenerOptions) {
		o.Logger = logger
	}
}
