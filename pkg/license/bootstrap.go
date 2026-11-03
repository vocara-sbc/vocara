package license

import "context"

// Bootstrap creates a manager, synchronously performs the initial load, and periodically verifies the License in the background at the configured interval.
// An initial load failure does not block the caller's startup. The current state is available through Result, and subsequent periodic checks reassess the License file.
func Bootstrap(ctx context.Context, cfg Config) *Manager {
	manager := NewManager(cfg)
	_ = manager.Load()
	go manager.Start(ctx)
	return manager
}
