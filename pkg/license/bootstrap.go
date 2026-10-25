package license

import "context"

// Bootstrap 创建管理器，同步完成首次加载，并在后台按复校间隔周期复校。
// 首次加载失败不阻断调用方启动，当前状态可经 Result 获取，License 文件修复后于下次复校自动恢复。
func Bootstrap(ctx context.Context, cfg Config) *Manager {
	manager := NewManager(cfg)
	_ = manager.Load()
	go manager.Start(ctx)
	return manager
}
