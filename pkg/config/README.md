# config - 配置管理包

基于 Viper 的配置管理包，支持 YAML 配置文件和环境变量覆盖，使用结构体映射的方式访问配置项。

## 功能特性

- 支持 YAML/JSON/TOML 等多种配置文件格式
- 支持环境变量覆盖配置（前缀 `VOCARA_`）
- 完整的默认值配置
- 结构体映射，类型安全
- 支持配置项热加载（Viper 原生支持）

## 配置结构

### Config 总配置

```go
type Config struct {
    App    AppConfig    // 应用基本配置
    Server ServerConfig // HTTP 服务配置
    MySQL  MySQLConfig  // MySQL 数据库配置
    Redis  RedisConfig  // Redis 配置
    NATS   NATSConfig   // NATS 消息队列配置
    Log    LogConfig    // 日志配置
    ESL    ESLConfig    // FreeSWITCH ESL 配置
}
```

## 使用方法

### 从文件加载配置

```go
import "vocara/pkg/config"

cfg, err := config.Load("./config.yaml")
if err != nil {
    log.Fatal(err)
}

fmt.Println(cfg.App.Name)
fmt.Println(cfg.Server.Port)
```

### 从环境变量加载配置

```go
import "vocara/pkg/config"

cfg := config.LoadFromEnv()
fmt.Println(cfg.App.Name)
```

### 环境变量命名规则

环境变量前缀为 `VOCARA_`，多级配置用下划线 `_` 分隔：

| 配置项 | 环境变量 |
|--------|----------|
| app.name | VOCARA_APP_NAME |
| server.port | VOCARA_SERVER_PORT |
| mysql.host | VOCARA_MYSQL_HOST |
| redis.password | VOCARA_REDIS_PASSWORD |
| log.level | VOCARA_LOG_LEVEL |

### 配置文件示例

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

## 工具方法

### MySQL DSN

```go
dsn := cfg.MySQL.DSN()
// root:password@tcp(127.0.0.1:3306)/vocara?charset=utf8mb4&parseTime=true&loc=Local
```

### Redis 地址列表

```go
addrs := cfg.Redis.AddrList()
// ["127.0.0.1:6379"]
```

### NATS 地址列表

```go
urls := cfg.NATS.URLList()
// ["nats://127.0.0.1:4222"]
```
