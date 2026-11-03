// Package scripts embeds the Lua scripts used for ACD queueing.
// Embed directives cannot reference files outside the package directory, including parent directories,
// so this package resides with the scripts and embeds them for the repository layer.
package scripts

import _ "embed"

// The scripts are embedded in the binary at compile time, so deployment does not require separate .lua files.
var (
	// Enqueue is the enqueue script.
	//go:embed enqueue.lua
	Enqueue string

	// Allocate is the allocation script with peek semantics that does not remove calls from the queue.
	//go:embed allocate.lua
	Allocate string

	// Pause is the pause script.
	//go:embed pause.lua
	Pause string

	// Resume is the resume script.
	//go:embed resume.lua
	Resume string

	// Dequeue is the dequeue script.
	//go:embed dequeue.lua
	Dequeue string
)
