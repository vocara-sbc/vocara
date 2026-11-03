# config - Configuration Management Package

A configuration management package based on Viper, supporting YAML configuration files and environment variable overrides, with configuration values accessed through struct mapping.

## Features

- Supports multiple configuration file formats, including YAML, JSON, and TOML
- Supports environment variable overrides (prefix `VOCARA_`)
- Comprehensive default configuration values
- Type-safe struct mapping
- Supports configuration hot reloading (native Viper support)

## Configuration Structure

### Overall Config

```go
type Config struct {
    App    AppConfig    // Basic application configuration
    Server ServerConfig // HTTP server configuration
    MySQL  MySQLConfig  // MySQL database configuration
    Redis  RedisConfig  // Redis configuration
    NATS   NATSConfig   // NATS message queue configuration
    Log    LogConfig    // Logging configuration
    ESL    ESLConfig    // FreeSWITCH ESL configuration
}
```

## Usage

### Load Configuration from a File

```go
import "vocara/pkg/config"

cfg, err := config.Load("./config.yaml")
if err != nil {
    log.Fatal(err)
}

fmt.Println(cfg.App.Name)
fmt.Println(cfg.Server.Port)
```

### Load Configuration from Environment Variables

```go
import "vocara/pkg/config"

cfg := config.LoadFromEnv()
fmt.Println(cfg.App.Name)
```

### Environment Variable Naming Rules

Environment variables use the `VOCARA_` prefix, and underscores `_` separate configuration levels:

| Configuration Item | Environment Variable |
|--------|----------|
| app.name | VOCARA_APP_NAME |
| server.port | VOCARA_SERVER_PORT |
| mysql.host | VOCARA_MYSQL_HOST |
| redis.password | VOCARA_REDIS_PASSWORD |
| log.level | VOCARA_LOG_LEVEL |

### Configuration File Example

```yaml
app:
  name: vocara
  version: 1.0.0
  env: dev
  secret_key: your-secret-key
  timezone: Asia/Shanghai

server:
  host: 0.0.0.0
  port: 8080
  mode: debug
  read_timeout: 30
  write_timeout: 30

mysql:
  host: 127.0.0.1
  port: 3306
  user: root
  password: password
  database: vocara
  charset: utf8mb4
  max_idle_conns: 10
  max_open_conns: 100

redis:
  mode: single
  addrs: 127.0.0.1:6379
  password: ""
  db: 0
  pool_size: 50

nats:
  urls: nats://127.0.0.1:4222
  cluster_id: vocara

log:
  level: info
  format: json
  output: both
  file_path: ./logs/app.log
  max_size: 100
  max_age: 30

esl:
  host: 127.0.0.1
  port: 8021
  password: ClueCon
```

## Utility Methods

### MySQL DSN

```go
dsn := cfg.MySQL.DSN()
// root:password@tcp(127.0.0.1:3306)/vocara?charset=utf8mb4&parseTime=true&loc=Local
```

### Redis Address List

```go
addrs := cfg.Redis.AddrList()
// ["127.0.0.1:6379"]
```

### NATS URL List

```go
urls := cfg.NATS.URLList()
// ["nats://127.0.0.1:4222"]
```
