// Package config provides Viper-based configuration management.
// It supports YAML configuration files and environment variable overrides, with configuration items accessed through struct mappings.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config is the application's overall configuration.
type Config struct {
	App      AppConfig      `mapstructure:"app" json:"app"`
	Server   ServerConfig   `mapstructure:"server" json:"server"`
	MySQL    MySQLConfig    `mapstructure:"mysql" json:"mysql"`
	Redis    RedisConfig    `mapstructure:"redis" json:"redis"`
	NATS     NATSConfig     `mapstructure:"nats" json:"nats"`
	Log      LogConfig      `mapstructure:"log" json:"log"`
	ESL        ESLConfig        `mapstructure:"esl" json:"esl"`
	CallCtrl   CallCtrlConfig   `mapstructure:"callctrl" json:"callctrl"`
	FSDialplan FSDialplanConfig `mapstructure:"fs_dialplan" json:"fs_dialplan"`
	ACD        ACDConfig        `mapstructure:"acd" json:"acd"`
	Push     PushConfig     `mapstructure:"push" json:"push"`
	License  LicenseConfig  `mapstructure:"license" json:"license"`
}

// AppConfig contains basic application configuration.
type AppConfig struct {
	Name      string `mapstructure:"name" json:"name"`           // Application name
	Version   string `mapstructure:"version" json:"version"`     // Application version
	Env       string `mapstructure:"env" json:"env"`             // Runtime environment: dev/test/prod
	SecretKey string `mapstructure:"secret_key" json:"-"`        // Secret key
	Timezone  string `mapstructure:"timezone" json:"timezone"`   // Time zone
}

// ServerConfig contains HTTP service configuration.
type ServerConfig struct {
	Host            string `mapstructure:"host" json:"host"`                         // Listen address
	Port            int    `mapstructure:"port" json:"port"`                         // Listen port
	Mode            string `mapstructure:"mode" json:"mode"`                         // Runtime mode: debug/release
	ReadTimeout     int    `mapstructure:"read_timeout" json:"read_timeout"`         // Read timeout in seconds
	WriteTimeout    int    `mapstructure:"write_timeout" json:"write_timeout"`       // Write timeout in seconds
	IdleTimeout     int    `mapstructure:"idle_timeout" json:"idle_timeout"`         // Idle timeout in seconds
	MaxRequestBody  int    `mapstructure:"max_request_body" json:"max_request_body"` // Maximum request body size in bytes
	ShutdownTimeout int    `mapstructure:"shutdown_timeout" json:"shutdown_timeout"` // Graceful shutdown timeout in seconds
}

// MySQLConfig contains MySQL database configuration.
type MySQLConfig struct {
	Host            string `mapstructure:"host" json:"host"`                          // Host address
	Port            int    `mapstructure:"port" json:"port"`                          // Port
	User            string `mapstructure:"user" json:"user"`                          // Username
	Password        string `mapstructure:"password" json:"-"`                         // Password
	Database        string `mapstructure:"database" json:"database"`                  // Database name
	Charset         string `mapstructure:"charset" json:"charset"`                    // Character set
	Collation       string `mapstructure:"collation" json:"collation"`                // Collation
	ParseTime       bool   `mapstructure:"parse_time" json:"parse_time"`              // Parse time values
	Loc             string `mapstructure:"loc" json:"loc"`                            // Time zone
	MaxIdleConns    int    `mapstructure:"max_idle_conns" json:"max_idle_conns"`      // Maximum idle connections
	MaxOpenConns    int    `mapstructure:"max_open_conns" json:"max_open_conns"`      // Maximum open connections
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime" json:"conn_max_lifetime"` // Maximum connection lifetime in minutes
	ConnMaxIdleTime int    `mapstructure:"conn_max_idle_time" json:"conn_max_idle_time"` // Maximum connection idle time in minutes
	AutoMigrate     bool   `mapstructure:"auto_migrate" json:"auto_migrate"`          // Automatic migration
	SlowThreshold   int    `mapstructure:"slow_threshold" json:"slow_threshold"`      // Slow query threshold in milliseconds
	LogLevel        string `mapstructure:"log_level" json:"log_level"`                // Log level: silent/error/warn/info
}

// DSN returns the MySQL connection string.
func (m *MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		m.User, m.Password, m.Host, m.Port, m.Database,
		m.Charset, m.ParseTime, m.Loc)
}

// RedisConfig contains Redis configuration.
type RedisConfig struct {
	Mode          string `mapstructure:"mode" json:"mode"`                    // Mode: single/cluster/sentinel
	Addrs         string `mapstructure:"addrs" json:"addrs"`                  // Comma-separated address list
	Username      string `mapstructure:"username" json:"username"`            // Username
	Password      string `mapstructure:"password" json:"-"`                   // Password
	DB            int    `mapstructure:"db" json:"db"`                        // Database number
	MasterName    string `mapstructure:"master_name" json:"master_name"`      // Sentinel master node name
	PoolSize      int    `mapstructure:"pool_size" json:"pool_size"`          // Connection pool size
	MinIdleConns  int    `mapstructure:"min_idle_conns" json:"min_idle_conns"` // Minimum idle connections
	MaxRetries    int    `mapstructure:"max_retries" json:"max_retries"`      // Maximum retries
	DialTimeout   int    `mapstructure:"dial_timeout" json:"dial_timeout"`    // Connection timeout in milliseconds
	ReadTimeout   int    `mapstructure:"read_timeout" json:"read_timeout"`    // Read timeout in milliseconds
	WriteTimeout  int    `mapstructure:"write_timeout" json:"write_timeout"`  // Write timeout in milliseconds
	PoolTimeout   int    `mapstructure:"pool_timeout" json:"pool_timeout"`    // Connection pool timeout in milliseconds
	IdleTimeout   int    `mapstructure:"idle_timeout" json:"idle_timeout"`    // Idle timeout in seconds
	IdleCheckFreq int    `mapstructure:"idle_check_freq" json:"idle_check_freq"` // Idle check frequency in seconds
}

// AddrList returns the address list.
func (r *RedisConfig) AddrList() []string {
	if r.Addrs == "" {
		return []string{"127.0.0.1:6379"}
	}
	return strings.Split(r.Addrs, ",")
}

// NATSConfig contains NATS message queue configuration.
type NATSConfig struct {
	URLs          string `mapstructure:"urls" json:"urls"`                      // Comma-separated NATS server address list
	User          string `mapstructure:"user" json:"user"`                      // Username
	Password      string `mapstructure:"password" json:"-"`                     // Password
	Token         string `mapstructure:"token" json:"-"`                        // Token
	ReconnectWait int    `mapstructure:"reconnect_wait" json:"reconnect_wait"`  // Reconnect wait time in seconds
	MaxReconnect  int    `mapstructure:"max_reconnect" json:"max_reconnect"`    // Maximum reconnection attempts
	ReconnectBuf  int    `mapstructure:"reconnect_buf" json:"reconnect_buf"`    // Reconnect buffer size in bytes
	Timeout       int    `mapstructure:"timeout" json:"timeout"`                // Connection timeout in seconds
	FlushTimeout  int    `mapstructure:"flush_timeout" json:"flush_timeout"`    // Flush timeout in seconds
	ClusterID     string `mapstructure:"cluster_id" json:"cluster_id"`          // Cluster ID
	ClientID      string `mapstructure:"client_id" json:"client_id"`            // Client ID
}

// URLList returns the NATS address list.
func (n *NATSConfig) URLList() []string {
	if n.URLs == "" {
		return []string{"nats://127.0.0.1:4222"}
	}
	return strings.Split(n.URLs, ",")
}

// LogConfig contains logging configuration.
type LogConfig struct {
	Level      string `mapstructure:"level" json:"level"`            // Log level: debug/info/warn/error/fatal/panic
	Format     string `mapstructure:"format" json:"format"`          // Output format: json/console
	Output     string `mapstructure:"output" json:"output"`          // Output mode: stdout/file/both
	FilePath   string `mapstructure:"file_path" json:"file_path"`    // Log file path
	MaxSize    int    `mapstructure:"max_size" json:"max_size"`      // Maximum size of each file in MB
	MaxAge     int    `mapstructure:"max_age" json:"max_age"`        // File retention period in days
	MaxBackups int    `mapstructure:"max_backups" json:"max_backups"` // Maximum number of backup files
	Compress   bool   `mapstructure:"compress" json:"compress"`      // Whether to compress backups
	Caller     bool   `mapstructure:"caller" json:"caller"`            // Whether to show the caller location
	CallerSkip int    `mapstructure:"caller_skip" json:"caller_skip"` // Number of call stack frames to skip
}

// InboundRoute contains routing configuration for an inbound access number.
type InboundRoute struct {
	DID          string `mapstructure:"did" json:"did"`
	QueueID      int64  `mapstructure:"queue_id" json:"queue_id"`
	SkillGroupID int64  `mapstructure:"skill_group_id" json:"skill_group_id"`
}

// CallCtrlConfig contains configuration specific to the call control service.
type CallCtrlConfig struct {
	FSInstance              string          `mapstructure:"fs_instance" json:"fs_instance"`
	ACDBaseURL              string          `mapstructure:"acd_base_url" json:"acd_base_url"`
	// AgentCenterBaseURL is the agent center address used to release an agent back to ready when ringing fails.
	AgentCenterBaseURL      string          `mapstructure:"agentcenter_base_url" json:"agentcenter_base_url"`
	// AdminBaseURL is the admin backend address used to fetch inbound DID routes dynamically, with fallback to static InboundRoutes configuration when unreachable.
	AdminBaseURL            string          `mapstructure:"admin_base_url" json:"admin_base_url"`
	// RouteRefreshInterval is the inbound route cache refresh interval in seconds; the default is 60.
	RouteRefreshInterval    int             `mapstructure:"route_refresh_interval" json:"route_refresh_interval"`
	InboundRoutes           []InboundRoute  `mapstructure:"inbound_routes" json:"inbound_routes"`
	DefaultGateway          string          `mapstructure:"default_gateway" json:"default_gateway"`
	RecordDir               string          `mapstructure:"record_dir" json:"record_dir"`
	RecordFormat            string          `mapstructure:"record_format" json:"record_format"`
	MaxCallDuration         int             `mapstructure:"max_call_duration" json:"max_call_duration"`
	DefaultOriginateTimeout int             `mapstructure:"default_originate_timeout" json:"default_originate_timeout"`
	EventWorkerNum          int             `mapstructure:"event_worker_num" json:"event_worker_num"`
	EventBufferSize         int             `mapstructure:"event_buffer_size" json:"event_buffer_size"`
	CallEventNATSTopic      string          `mapstructure:"call_event_nats_topic" json:"call_event_nats_topic"`
}

// FSDialplanConfig contains configuration for dynamic FreeSWITCH dialplan delivery.
// Dialing rules are delivered in real time by admin through mod_xml_curl; only environment-specific node addresses and gateway names use this configuration.
type FSDialplanConfig struct {
	// SBCPeer is the to-sbc profile endpoint (host:port) where fs-agent forwards outbound numbers and unregistered extensions to fs-sbc.
	SBCPeer string `mapstructure:"sbc_peer" json:"sbc_peer"`
	// AgentExternal is the external profile endpoint (host:port) where fs-sbc forwards incoming calls to fs-agent.
	AgentExternal string `mapstructure:"agent_external" json:"agent_external"`
	// Gateway is the carrier gateway name used by fs-sbc for outbound calls, defined in the fs-sbc external profile.
	Gateway string `mapstructure:"gateway" json:"gateway"`
}

// PushConfig contains configuration specific to the push service.
type PushConfig struct {
	// AgentCenterBaseURL is the agent center address used by the agent command gateway to call internal APIs.
	AgentCenterBaseURL string `mapstructure:"agentcenter_base_url" json:"agentcenter_base_url"`
}

// LicenseConfig contains license authorization configuration.
type LicenseConfig struct {
	// File is the license file path.
	File string `mapstructure:"file" json:"file"`
	// CheckInterval is the runtime reverification interval in seconds.
	CheckInterval int `mapstructure:"check_interval" json:"check_interval"`
	// FailOpen determines whether business operations proceed when signature, fingerprint, or other checks fail; use only for temporary troubleshooting.
	FailOpen bool `mapstructure:"fail_open" json:"fail_open"`
	// InstallID is the cluster installation ID in install mode; all cluster nodes use the same value, which admin can provide through a system table.
	InstallID string `mapstructure:"install_id" json:"install_id"`
}

// ACDConfig contains configuration specific to the ACD queueing service.
type ACDConfig struct {
	MaxQueueLen         int    `mapstructure:"max_queue_len" json:"max_queue_len"`
	MaxPriority         int    `mapstructure:"max_priority" json:"max_priority"`
	PriorityAgingCycle  int    `mapstructure:"priority_aging_cycle" json:"priority_aging_cycle"`
	MaxWaitTime         int    `mapstructure:"max_wait_time" json:"max_wait_time"`
	ScheduleInterval    int    `mapstructure:"schedule_interval" json:"schedule_interval"`
	LockTimeout         int    `mapstructure:"lock_timeout" json:"lock_timeout"`
	AllocateMaxRetries  int    `mapstructure:"allocate_max_retries" json:"allocate_max_retries"`
	AgentCenterAddr     string `mapstructure:"agent_center_addr" json:"agent_center_addr"`
	AgentCenterToken    string `mapstructure:"agent_center_token" json:"agent_center_token"`
	UseMockAgentClient  bool   `mapstructure:"use_mock_agent_client" json:"use_mock_agent_client"`
}

// ESLConfig contains FreeSWITCH ESL configuration.
type ESLConfig struct {
	Host          string `mapstructure:"host" json:"host"`                // FreeSWITCH host
	Port          int    `mapstructure:"port" json:"port"`                // ESL port
	Password      string `mapstructure:"password" json:"-"`               // ESL password
	Timeout       int    `mapstructure:"timeout" json:"timeout"`          // Connection timeout in seconds
	Reconnect     bool   `mapstructure:"reconnect" json:"reconnect"`      // Whether to reconnect automatically
	ReconnectWait int    `mapstructure:"reconnect_wait" json:"reconnect_wait"` // Reconnect wait time in seconds
	MaxReconnect  int    `mapstructure:"max_reconnect" json:"max_reconnect"`   // Maximum reconnection attempts
	EventFilter   string `mapstructure:"event_filter" json:"event_filter"`     // Comma-separated event filter
	BufferSize    int    `mapstructure:"buffer_size" json:"buffer_size"`  // Event buffer size
}

// setDefaults sets configuration default values.
func setDefaults(v *viper.Viper) {
	// App default values
	v.SetDefault("app.name", "vocara")
	v.SetDefault("app.version", "1.0.0")
	v.SetDefault("app.env", "dev")
	v.SetDefault("app.timezone", "Asia/Shanghai")
	v.SetDefault("app.secret_key", "default-secret-key-change-me")

	// Server default values
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.read_timeout", 30)
	v.SetDefault("server.write_timeout", 30)
	v.SetDefault("server.idle_timeout", 60)
	v.SetDefault("server.max_request_body", 10485760) // 10MB
	v.SetDefault("server.shutdown_timeout", 30)

	// MySQL default values
	v.SetDefault("mysql.host", "127.0.0.1")
	v.SetDefault("mysql.port", 3306)
	v.SetDefault("mysql.user", "root")
	v.SetDefault("mysql.password", "")
	v.SetDefault("mysql.database", "vocara")
	v.SetDefault("mysql.charset", "utf8mb4")
	v.SetDefault("mysql.collation", "utf8mb4_unicode_ci")
	v.SetDefault("mysql.parse_time", true)
	v.SetDefault("mysql.loc", "Local")
	v.SetDefault("mysql.max_idle_conns", 10)
	v.SetDefault("mysql.max_open_conns", 100)
	v.SetDefault("mysql.conn_max_lifetime", 60)  // 60 minutes
	v.SetDefault("mysql.conn_max_idle_time", 10) // 10 minutes
	v.SetDefault("mysql.auto_migrate", false)
	v.SetDefault("mysql.slow_threshold", 1000)   // 1 second
	v.SetDefault("mysql.log_level", "warn")

	// Redis default values
	v.SetDefault("redis.mode", "single")
	v.SetDefault("redis.addrs", "127.0.0.1:6379")
	v.SetDefault("redis.username", "")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.master_name", "")
	v.SetDefault("redis.pool_size", 50)
	v.SetDefault("redis.min_idle_conns", 10)
	v.SetDefault("redis.max_retries", 3)
	v.SetDefault("redis.dial_timeout", 5000)  // 5 seconds
	v.SetDefault("redis.read_timeout", 3000)  // 3 seconds
	v.SetDefault("redis.write_timeout", 3000) // 3 seconds
	v.SetDefault("redis.pool_timeout", 4000)  // 4 seconds
	v.SetDefault("redis.idle_timeout", 300)   // 5 minutes
	v.SetDefault("redis.idle_check_freq", 60) // 1 minute

	// NATS default values
	v.SetDefault("nats.urls", "nats://127.0.0.1:4222")
	v.SetDefault("nats.user", "")
	v.SetDefault("nats.password", "")
	v.SetDefault("nats.token", "")
	v.SetDefault("nats.reconnect_wait", 2)
	v.SetDefault("nats.max_reconnect", -1) // Unlimited reconnections
	v.SetDefault("nats.reconnect_buf", 8388608) // 8MB
	v.SetDefault("nats.timeout", 5)
	v.SetDefault("nats.flush_timeout", 5)
	v.SetDefault("nats.cluster_id", "vocara")
	v.SetDefault("nats.client_id", "")

	// Log default values
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "both")
	v.SetDefault("log.file_path", "./logs/app.log")
	v.SetDefault("log.max_size", 100)   // 100MB
	v.SetDefault("log.max_age", 30)     // 30 days
	v.SetDefault("log.max_backups", 10) // 10 backups
	v.SetDefault("log.compress", true)
	v.SetDefault("log.caller", true)
	v.SetDefault("log.caller_skip", 1)

	// ESL default values
	v.SetDefault("esl.host", "127.0.0.1")
	v.SetDefault("esl.port", 8021)
	v.SetDefault("esl.password", "ClueCon")
	v.SetDefault("esl.timeout", 10)
	v.SetDefault("esl.reconnect", true)
	v.SetDefault("esl.reconnect_wait", 5)
	v.SetDefault("esl.max_reconnect", -1) // Unlimited reconnections
	v.SetDefault("esl.event_filter", "")
	v.SetDefault("esl.buffer_size", 1000)

	// CallCtrl default values
	v.SetDefault("callctrl.fs_instance", "fs-01")
	v.SetDefault("callctrl.default_gateway", "default")
	v.SetDefault("callctrl.record_dir", "/data/records")
	v.SetDefault("callctrl.record_format", "wav")
	v.SetDefault("callctrl.max_call_duration", 0)
	v.SetDefault("callctrl.default_originate_timeout", 30)
	v.SetDefault("callctrl.event_worker_num", 4)
	v.SetDefault("callctrl.event_buffer_size", 1000)
	v.SetDefault("callctrl.call_event_nats_topic", "vocara.call.event")

	// FS dynamic dialplan delivery default values
	v.SetDefault("fs_dialplan.sbc_peer", "127.0.0.1:5061")
	v.SetDefault("fs_dialplan.agent_external", "127.0.0.1:5080")
	v.SetDefault("fs_dialplan.gateway", "carrier")

	// ACD default values
	v.SetDefault("acd.max_queue_len", 1000)
	v.SetDefault("acd.max_priority", 255)
	v.SetDefault("acd.priority_aging_cycle", 60)
	v.SetDefault("acd.max_wait_time", 600)
	v.SetDefault("acd.schedule_interval", 5)
	v.SetDefault("acd.lock_timeout", 10)
	v.SetDefault("acd.allocate_max_retries", 3)
	v.SetDefault("acd.agent_center_addr", "http://127.0.0.1:8081")
	v.SetDefault("acd.agent_center_token", "vocara-agent-center-secret-key")
	v.SetDefault("acd.use_mock_agent_client", true)

	// License default values
	v.SetDefault("license.file", "./license/vocara.lic")
	v.SetDefault("license.check_interval", 3600)
	v.SetDefault("license.fail_open", false)
}

// Load loads configuration from a configuration file.
// path is the configuration file path including its extension. It supports YAML, JSON, TOML, and other formats supported by Viper.
// Environment variables override configuration file values and use the VOCARA_ prefix.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	// Set default values
	setDefaults(v)

	// Automatically apply environment variable overrides
	v.AutomaticEnv()
	v.SetEnvPrefix("VOCARA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// Read the configuration file
	if err := v.ReadInConfig(); err != nil {
		// Return the default configuration when the configuration file does not exist.
		// Handle different Viper versions: they may return ConfigFileNotFoundError or os.PathError directly.
		var nf viper.ConfigFileNotFoundError
		if errors.As(err, &nf) || os.IsNotExist(err) {
			var cfg Config
			if err := v.Unmarshal(&cfg); err != nil {
				return nil, fmt.Errorf("unmarshal config failed: %w", err)
			}
			return &cfg, nil
		}
		return nil, fmt.Errorf("read config file failed: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}

	return &cfg, nil
}

// LoadFromEnv loads configuration from environment variables.
// Environment variables use the VOCARA_ prefix, with underscores separating nested configuration keys.
// For example, VOCARA_SERVER_PORT=8080 corresponds to server.port.
func LoadFromEnv() *Config {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Use environment variables only
	v.AutomaticEnv()
	v.SetEnvPrefix("VOCARA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		// Return the default configuration if parsing fails
		return defaultConfig()
	}

	return &cfg
}

// defaultConfig returns the default configuration.
func defaultConfig() *Config {
	v := viper.New()
	setDefaults(v)

	var cfg Config
	_ = v.Unmarshal(&cfg)
	return &cfg
}
