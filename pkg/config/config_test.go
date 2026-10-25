package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// 创建临时配置文件
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `
app:
  name: test-app
  version: 1.0.0
  env: test
server:
  host: 127.0.0.1
  port: 9090
  mode: release
mysql:
  host: 192.168.1.100
  port: 3307
  user: testuser
  database: testdb
redis:
  addrs: 127.0.0.1:6379,127.0.0.1:6380
  db: 1
log:
  level: debug
  format: console
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("write config file failed: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// 验证 App 配置
	if cfg.App.Name != "test-app" {
		t.Errorf("expected app.name = test-app, got %s", cfg.App.Name)
	}
	if cfg.App.Version != "1.0.0" {
		t.Errorf("expected app.version = 1.0.0, got %s", cfg.App.Version)
	}
	if cfg.App.Env != "test" {
		t.Errorf("expected app.env = test, got %s", cfg.App.Env)
	}

	// 验证 Server 配置
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("expected server.host = 127.0.0.1, got %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("expected server.port = 9090, got %d", cfg.Server.Port)
	}
	if cfg.Server.Mode != "release" {
		t.Errorf("expected server.mode = release, got %s", cfg.Server.Mode)
	}

	// 验证 MySQL 配置
	if cfg.MySQL.Host != "192.168.1.100" {
		t.Errorf("expected mysql.host = 192.168.1.100, got %s", cfg.MySQL.Host)
	}
	if cfg.MySQL.Port != 3307 {
		t.Errorf("expected mysql.port = 3307, got %d", cfg.MySQL.Port)
	}

	// 验证 Redis 配置
	if cfg.Redis.DB != 1 {
		t.Errorf("expected redis.db = 1, got %d", cfg.Redis.DB)
	}
	addrs := cfg.Redis.AddrList()
	if len(addrs) != 2 {
		t.Errorf("expected 2 redis addrs, got %d", len(addrs))
	}

	// 验证 Log 配置
	if cfg.Log.Level != "debug" {
		t.Errorf("expected log.level = debug, got %s", cfg.Log.Level)
	}
	if cfg.Log.Format != "console" {
		t.Errorf("expected log.format = console, got %s", cfg.Log.Format)
	}
}

func TestLoad_Defaults(t *testing.T) {
	// 测试默认值 - 加载不存在的文件应返回默认配置
	dir := t.TempDir()
	configPath := filepath.Join(dir, "nonexistent.yaml")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load with nonexistent file failed: %v", err)
	}

	// 验证默认值
	if cfg.App.Name != "vocara" {
		t.Errorf("expected default app.name = vocara, got %s", cfg.App.Name)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default server.port = 8080, got %d", cfg.Server.Port)
	}
	if cfg.MySQL.Port != 3306 {
		t.Errorf("expected default mysql.port = 3306, got %d", cfg.MySQL.Port)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("expected default log.level = info, got %s", cfg.Log.Level)
	}
}

func TestLoadFromEnv(t *testing.T) {
	// 设置环境变量
	os.Setenv("VOCARA_APP_NAME", "env-app")
	os.Setenv("VOCARA_SERVER_PORT", "7070")
	os.Setenv("VOCARA_MYSQL_HOST", "env-mysql-host")
	os.Setenv("VOCARA_LOG_LEVEL", "warn")
	defer func() {
		os.Unsetenv("VOCARA_APP_NAME")
		os.Unsetenv("VOCARA_SERVER_PORT")
		os.Unsetenv("VOCARA_MYSQL_HOST")
		os.Unsetenv("VOCARA_LOG_LEVEL")
	}()

	cfg := LoadFromEnv()

	if cfg.App.Name != "env-app" {
		t.Errorf("expected app.name = env-app, got %s", cfg.App.Name)
	}
	if cfg.Server.Port != 7070 {
		t.Errorf("expected server.port = 7070, got %d", cfg.Server.Port)
	}
	if cfg.MySQL.Host != "env-mysql-host" {
		t.Errorf("expected mysql.host = env-mysql-host, got %s", cfg.MySQL.Host)
	}
	if cfg.Log.Level != "warn" {
		t.Errorf("expected log.level = warn, got %s", cfg.Log.Level)
	}
}

func TestMySQLConfig_DSN(t *testing.T) {
	cfg := &MySQLConfig{
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "pass",
		Database: "test",
		Charset:  "utf8mb4",
		ParseTime: true,
		Loc:      "Local",
	}

	dsn := cfg.DSN()
	expected := "root:pass@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=true&loc=Local"
	if dsn != expected {
		t.Errorf("expected DSN = %s, got %s", expected, dsn)
	}
}

func TestRedisConfig_AddrList(t *testing.T) {
	// 测试空地址
	cfg := &RedisConfig{Addrs: ""}
	addrs := cfg.AddrList()
	if len(addrs) != 1 || addrs[0] != "127.0.0.1:6379" {
		t.Errorf("expected default addr, got %v", addrs)
	}

	// 测试多个地址
	cfg.Addrs = "host1:6379,host2:6379,host3:6379"
	addrs = cfg.AddrList()
	if len(addrs) != 3 {
		t.Errorf("expected 3 addrs, got %d", len(addrs))
	}
	if addrs[0] != "host1:6379" || addrs[2] != "host3:6379" {
		t.Errorf("unexpected addr list: %v", addrs)
	}
}

func TestNATSConfig_URLList(t *testing.T) {
	cfg := &NATSConfig{URLs: ""}
	urls := cfg.URLList()
	if len(urls) != 1 {
		t.Errorf("expected 1 url, got %d", len(urls))
	}

	cfg.URLs = "nats://h1:4222,nats://h2:4222"
	urls = cfg.URLList()
	if len(urls) != 2 {
		t.Errorf("expected 2 urls, got %d", len(urls))
	}
}
