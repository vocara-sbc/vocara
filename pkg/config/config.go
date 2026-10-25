// Package config 提供基于 Viper 的配置管理功能
// 支持 YAML 配置文件和环境变量覆盖，使用结构体映射的方式访问配置项
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用总配置
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

// AppConfig 应用基本配置
type AppConfig struct {
	Name      string `mapstructure:"name" json:"name"`           // 应用名称
	Version   string `mapstructure:"version" json:"version"`     // 应用版本
	Env       string `mapstructure:"env" json:"env"`             // 运行环境: dev/test/prod
	SecretKey string `mapstructure:"secret_key" json:"-"`        // 密钥
	Timezone  string `mapstructure:"timezone" json:"timezone"`   // 时区
}

// ServerConfig HTTP 服务配置
type ServerConfig struct {
	Host            string `mapstructure:"host" json:"host"`                         // 监听地址
	Port            int    `mapstructure:"port" json:"port"`                         // 监听端口
	Mode            string `mapstructure:"mode" json:"mode"`                         // 运行模式: debug/release
	ReadTimeout     int    `mapstructure:"read_timeout" json:"read_timeout"`         // 读取超时(秒)
	WriteTimeout    int    `mapstructure:"write_timeout" json:"write_timeout"`       // 写入超时(秒)
	IdleTimeout     int    `mapstructure:"idle_timeout" json:"idle_timeout"`         // 空闲超时(秒)
	MaxRequestBody  int    `mapstructure:"max_request_body" json:"max_request_body"` // 请求体最大字节
	ShutdownTimeout int    `mapstructure:"shutdown_timeout" json:"shutdown_timeout"` // 优雅关闭超时(秒)
}

// MySQLConfig MySQL 数据库配置
type MySQLConfig struct {
	Host            string `mapstructure:"host" json:"host"`                          // 主机地址
	Port            int    `mapstructure:"port" json:"port"`                          // 端口
	User            string `mapstructure:"user" json:"user"`                          // 用户名
	Password        string `mapstructure:"password" json:"-"`                         // 密码
	Database        string `mapstructure:"database" json:"database"`                  // 数据库名
	Charset         string `mapstructure:"charset" json:"charset"`                    // 字符集
	Collation       string `mapstructure:"collation" json:"collation"`                // 排序规则
	ParseTime       bool   `mapstructure:"parse_time" json:"parse_time"`              // 解析时间
	Loc             string `mapstructure:"loc" json:"loc"`                            // 时区
	MaxIdleConns    int    `mapstructure:"max_idle_conns" json:"max_idle_conns"`      // 最大空闲连接数
	MaxOpenConns    int    `mapstructure:"max_open_conns" json:"max_open_conns"`      // 最大打开连接数
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime" json:"conn_max_lifetime"` // 连接最大生命周期(分钟)
	ConnMaxIdleTime int    `mapstructure:"conn_max_idle_time" json:"conn_max_idle_time"` // 连接最大空闲时间(分钟)
	AutoMigrate     bool   `mapstructure:"auto_migrate" json:"auto_migrate"`          // 自动迁移
	SlowThreshold   int    `mapstructure:"slow_threshold" json:"slow_threshold"`      // 慢查询阈值(毫秒)
	LogLevel        string `mapstructure:"log_level" json:"log_level"`                // 日志级别: silent/error/warn/info
}

// DSN 返回 MySQL 连接字符串
func (m *MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		m.User, m.Password, m.Host, m.Port, m.Database,
		m.Charset, m.ParseTime, m.Loc)
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Mode          string `mapstructure:"mode" json:"mode"`                    // 模式: single/cluster/sentinel
	Addrs         string `mapstructure:"addrs" json:"addrs"`                  // 地址列表，逗号分隔
	Username      string `mapstructure:"username" json:"username"`            // 用户名
	Password      string `mapstructure:"password" json:"-"`                   // 密码
	DB            int    `mapstructure:"db" json:"db"`                        // 数据库编号
	MasterName    string `mapstructure:"master_name" json:"master_name"`      // 哨兵主节点名称
	PoolSize      int    `mapstructure:"pool_size" json:"pool_size"`          // 连接池大小
	MinIdleConns  int    `mapstructure:"min_idle_conns" json:"min_idle_conns"` // 最小空闲连接数
	MaxRetries    int    `mapstructure:"max_retries" json:"max_retries"`      // 最大重试次数
	DialTimeout   int    `mapstructure:"dial_timeout" json:"dial_timeout"`    // 连接超时(毫秒)
	ReadTimeout   int    `mapstructure:"read_timeout" json:"read_timeout"`    // 读取超时(毫秒)
	WriteTimeout  int    `mapstructure:"write_timeout" json:"write_timeout"`  // 写入超时(毫秒)
	PoolTimeout   int    `mapstructure:"pool_timeout" json:"pool_timeout"`    // 连接池超时(毫秒)
	IdleTimeout   int    `mapstructure:"idle_timeout" json:"idle_timeout"`    // 空闲超时(秒)
	IdleCheckFreq int    `mapstructure:"idle_check_freq" json:"idle_check_freq"` // 空闲检查频率(秒)
}

// AddrList 返回地址列表
func (r *RedisConfig) AddrList() []string {
	if r.Addrs == "" {
		return []string{"127.0.0.1:6379"}
	}
	return strings.Split(r.Addrs, ",")
}

// NATSConfig NATS 消息队列配置
type NATSConfig struct {
	URLs          string `mapstructure:"urls" json:"urls"`                      // NATS 服务器地址列表，逗号分隔
	User          string `mapstructure:"user" json:"user"`                      // 用户名
	Password      string `mapstructure:"password" json:"-"`                     // 密码
	Token         string `mapstructure:"token" json:"-"`                        // Token
	ReconnectWait int    `mapstructure:"reconnect_wait" json:"reconnect_wait"`  // 重连等待(秒)
	MaxReconnect  int    `mapstructure:"max_reconnect" json:"max_reconnect"`    // 最大重连次数
	ReconnectBuf  int    `mapstructure:"reconnect_buf" json:"reconnect_buf"`    // 重连缓冲区大小(字节)
	Timeout       int    `mapstructure:"timeout" json:"timeout"`                // 连接超时(秒)
	FlushTimeout  int    `mapstructure:"flush_timeout" json:"flush_timeout"`    // Flush 超时(秒)
	ClusterID     string `mapstructure:"cluster_id" json:"cluster_id"`          // 集群ID
	ClientID      string `mapstructure:"client_id" json:"client_id"`            // 客户端ID
}

// URLList 返回 NATS 地址列表
func (n *NATSConfig) URLList() []string {
	if n.URLs == "" {
		return []string{"nats://127.0.0.1:4222"}
	}
	return strings.Split(n.URLs, ",")
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level" json:"level"`            // 日志级别: debug/info/warn/error/fatal/panic
	Format     string `mapstructure:"format" json:"format"`          // 输出格式: json/console
	Output     string `mapstructure:"output" json:"output"`          // 输出方式: stdout/file/both
	FilePath   string `mapstructure:"file_path" json:"file_path"`    // 日志文件路径
	MaxSize    int    `mapstructure:"max_size" json:"max_size"`      // 单个文件最大大小(MB)
	MaxAge     int    `mapstructure:"max_age" json:"max_age"`        // 文件保留天数
	MaxBackups int    `mapstructure:"max_backups" json:"max_backups"` // 最大备份文件数
	Compress   bool   `mapstructure:"compress" json:"compress"`      // 是否压缩备份
	Caller     bool   `mapstructure:"caller" json:"caller"`            // 是否显示调用位置
	CallerSkip int    `mapstructure:"caller_skip" json:"caller_skip"` // 调用栈跳过层数
}

// InboundRoute 呼入接入号路由配置。
type InboundRoute struct {
	DID          string `mapstructure:"did" json:"did"`
	QueueID      int64  `mapstructure:"queue_id" json:"queue_id"`
	SkillGroupID int64  `mapstructure:"skill_group_id" json:"skill_group_id"`
}

// CallCtrlConfig 呼叫控制服务专属配置
type CallCtrlConfig struct {
	FSInstance              string          `mapstructure:"fs_instance" json:"fs_instance"`
	ACDBaseURL              string          `mapstructure:"acd_base_url" json:"acd_base_url"`
	// AgentCenterBaseURL 座席中心地址（振铃失败时释放座席回就绪）
	AgentCenterBaseURL      string          `mapstructure:"agentcenter_base_url" json:"agentcenter_base_url"`
	// AdminBaseURL 管理后台地址（呼入 DID 路由动态拉取；不可达时回落 InboundRoutes 静态配置）
	AdminBaseURL            string          `mapstructure:"admin_base_url" json:"admin_base_url"`
	// RouteRefreshInterval 呼入路由缓存刷新间隔（秒），默认 60
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

// FSDialplanConfig FreeSWITCH 拨号计划动态下发配置。
// 拨号规则由 admin 经 mod_xml_curl 实时下发，仅环境相关的节点地址与网关名走本配置。
type FSDialplanConfig struct {
	// SBCPeer fs-agent 转发出局号码/未注册分机到 fs-sbc 的 to-sbc profile 入口（host:port）
	SBCPeer string `mapstructure:"sbc_peer" json:"sbc_peer"`
	// AgentExternal fs-sbc 转发来话到 fs-agent 的 external profile 入口（host:port）
	AgentExternal string `mapstructure:"agent_external" json:"agent_external"`
	// Gateway fs-sbc 外呼出局使用的运营商网关名（定义在 fs-sbc external profile）
	Gateway string `mapstructure:"gateway" json:"gateway"`
}

// PushConfig 推送服务专属配置
type PushConfig struct {
	// AgentCenterBaseURL 座席中心地址（座席命令网关调用 internal 接口）
	AgentCenterBaseURL string `mapstructure:"agentcenter_base_url" json:"agentcenter_base_url"`
}

// LicenseConfig License 授权配置
type LicenseConfig struct {
	// File License 文件路径
	File string `mapstructure:"file" json:"file"`
	// CheckInterval 运行期复校间隔（秒）
	CheckInterval int `mapstructure:"check_interval" json:"check_interval"`
	// FailOpen 签名 / 指纹等异常时是否放行业务（仅排障临时使用）
	FailOpen bool `mapstructure:"fail_open" json:"fail_open"`
	// InstallID install 模式的集群安装标识；集群各节点配置同一值，admin 可经系统表提供
	InstallID string `mapstructure:"install_id" json:"install_id"`
}

// ACDConfig ACD 排队服务专属配置
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

// ESLConfig FreeSWITCH ESL 配置
type ESLConfig struct {
	Host          string `mapstructure:"host" json:"host"`                // FreeSWITCH 主机
	Port          int    `mapstructure:"port" json:"port"`                // ESL 端口
	Password      string `mapstructure:"password" json:"-"`               // ESL 密码
	Timeout       int    `mapstructure:"timeout" json:"timeout"`          // 连接超时(秒)
	Reconnect     bool   `mapstructure:"reconnect" json:"reconnect"`      // 是否自动重连
	ReconnectWait int    `mapstructure:"reconnect_wait" json:"reconnect_wait"` // 重连等待(秒)
	MaxReconnect  int    `mapstructure:"max_reconnect" json:"max_reconnect"`   // 最大重连次数
	EventFilter   string `mapstructure:"event_filter" json:"event_filter"`     // 事件过滤，逗号分隔
	BufferSize    int    `mapstructure:"buffer_size" json:"buffer_size"`  // 事件缓冲区大小
}

// setDefaults 设置配置默认值
func setDefaults(v *viper.Viper) {
	// App 默认值
	v.SetDefault("app.name", "vocara")
	v.SetDefault("app.version", "1.0.0")
	v.SetDefault("app.env", "dev")
	v.SetDefault("app.timezone", "Asia/Shanghai")
	v.SetDefault("app.secret_key", "default-secret-key-change-me")

	// Server 默认值
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.read_timeout", 30)
	v.SetDefault("server.write_timeout", 30)
	v.SetDefault("server.idle_timeout", 60)
	v.SetDefault("server.max_request_body", 10485760) // 10MB
	v.SetDefault("server.shutdown_timeout", 30)

	// MySQL 默认值
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
	v.SetDefault("mysql.conn_max_lifetime", 60)  // 60分钟
	v.SetDefault("mysql.conn_max_idle_time", 10) // 10分钟
	v.SetDefault("mysql.auto_migrate", false)
	v.SetDefault("mysql.slow_threshold", 1000)   // 1秒
	v.SetDefault("mysql.log_level", "warn")

	// Redis 默认值
	v.SetDefault("redis.mode", "single")
	v.SetDefault("redis.addrs", "127.0.0.1:6379")
	v.SetDefault("redis.username", "")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.master_name", "")
	v.SetDefault("redis.pool_size", 50)
	v.SetDefault("redis.min_idle_conns", 10)
	v.SetDefault("redis.max_retries", 3)
	v.SetDefault("redis.dial_timeout", 5000)  // 5秒
	v.SetDefault("redis.read_timeout", 3000)  // 3秒
	v.SetDefault("redis.write_timeout", 3000) // 3秒
	v.SetDefault("redis.pool_timeout", 4000)  // 4秒
	v.SetDefault("redis.idle_timeout", 300)   // 5分钟
	v.SetDefault("redis.idle_check_freq", 60) // 1分钟

	// NATS 默认值
	v.SetDefault("nats.urls", "nats://127.0.0.1:4222")
	v.SetDefault("nats.user", "")
	v.SetDefault("nats.password", "")
	v.SetDefault("nats.token", "")
	v.SetDefault("nats.reconnect_wait", 2)
	v.SetDefault("nats.max_reconnect", -1) // 无限重连
	v.SetDefault("nats.reconnect_buf", 8388608) // 8MB
	v.SetDefault("nats.timeout", 5)
	v.SetDefault("nats.flush_timeout", 5)
	v.SetDefault("nats.cluster_id", "vocara")
	v.SetDefault("nats.client_id", "")

	// Log 默认值
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "both")
	v.SetDefault("log.file_path", "./logs/app.log")
	v.SetDefault("log.max_size", 100)   // 100MB
	v.SetDefault("log.max_age", 30)     // 30天
	v.SetDefault("log.max_backups", 10) // 10个备份
	v.SetDefault("log.compress", true)
	v.SetDefault("log.caller", true)
	v.SetDefault("log.caller_skip", 1)

	// ESL 默认值
	v.SetDefault("esl.host", "127.0.0.1")
	v.SetDefault("esl.port", 8021)
	v.SetDefault("esl.password", "ClueCon")
	v.SetDefault("esl.timeout", 10)
	v.SetDefault("esl.reconnect", true)
	v.SetDefault("esl.reconnect_wait", 5)
	v.SetDefault("esl.max_reconnect", -1) // 无限重连
	v.SetDefault("esl.event_filter", "")
	v.SetDefault("esl.buffer_size", 1000)

	// CallCtrl 默认值
	v.SetDefault("callctrl.fs_instance", "fs-01")
	v.SetDefault("callctrl.default_gateway", "default")
	v.SetDefault("callctrl.record_dir", "/data/records")
	v.SetDefault("callctrl.record_format", "wav")
	v.SetDefault("callctrl.max_call_duration", 0)
	v.SetDefault("callctrl.default_originate_timeout", 30)
	v.SetDefault("callctrl.event_worker_num", 4)
	v.SetDefault("callctrl.event_buffer_size", 1000)
	v.SetDefault("callctrl.call_event_nats_topic", "vocara.call.event")

	// FS 拨号计划动态下发默认值
	v.SetDefault("fs_dialplan.sbc_peer", "127.0.0.1:5061")
	v.SetDefault("fs_dialplan.agent_external", "127.0.0.1:5080")
	v.SetDefault("fs_dialplan.gateway", "carrier")

	// ACD 默认值
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

	// License 默认值
	v.SetDefault("license.file", "./license/vocara.lic")
	v.SetDefault("license.check_interval", 3600)
	v.SetDefault("license.fail_open", false)
}

// Load 从配置文件加载配置
// path 为配置文件路径（包含扩展名），支持 YAML/JSON/TOML 等 Viper 支持的格式
// 环境变量会覆盖配置文件中的值，环境变量前缀为 VOCARA_
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	// 设置默认值
	setDefaults(v)

	// 自动使用环境变量覆盖
	v.AutomaticEnv()
	v.SetEnvPrefix("VOCARA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		// 配置文件不存在时返回默认配置。
		// 兼容不同 viper 版本：可能返回 ConfigFileNotFoundError，也可能直接返回 os.PathError
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

// LoadFromEnv 从环境变量加载配置
// 环境变量前缀为 VOCARA_，支持嵌套配置用下划线分隔
// 例如: VOCARA_SERVER_PORT=8080 对应 server.port
func LoadFromEnv() *Config {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	// 仅使用环境变量
	v.AutomaticEnv()
	v.SetEnvPrefix("VOCARA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		// 如果解析失败，返回默认配置
		return defaultConfig()
	}

	return &cfg
}

// defaultConfig 返回默认配置
func defaultConfig() *Config {
	v := viper.New()
	setDefaults(v)

	var cfg Config
	_ = v.Unmarshal(&cfg)
	return &cfg
}
