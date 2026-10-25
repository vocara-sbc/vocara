// Package version 提供构建期注入的版本信息。
package version

// 以下变量由构建期 -ldflags "-X" 注入，未注入时取开发默认值。
var (
	// Version 程序语义版本号。
	Version = "dev"
	// BuildTime 构建时间（UTC，RFC3339）。
	BuildTime = "unknown"
	// GitCommit 构建对应的提交哈希。
	GitCommit = "unknown"
)
