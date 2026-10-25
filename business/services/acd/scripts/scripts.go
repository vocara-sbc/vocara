// Package scripts 内嵌 ACD 排队使用的 Lua 脚本。
// 嵌入指令不允许引用包目录之外（含 ..）的文件，
// 因此在脚本所在目录建立本包统一嵌入，供 repository 层引用。
package scripts

import _ "embed"

// 以下脚本在编译时嵌入二进制，部署时无需单独携带 .lua 文件。
var (
	// Enqueue 入队脚本
	//go:embed enqueue.lua
	Enqueue string

	// Allocate 分配脚本（peek 语义，不从队列移除）
	//go:embed allocate.lua
	Allocate string

	// Pause 暂停脚本
	//go:embed pause.lua
	Pause string

	// Resume 恢复脚本
	//go:embed resume.lua
	Resume string

	// Dequeue 出队脚本
	//go:embed dequeue.lua
	Dequeue string
)
