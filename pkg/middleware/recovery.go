package middleware

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"go.uber.org/zap"

	"vocara/pkg/errors"
	"vocara/pkg/logger"
	"vocara/pkg/response"
)

// RecoveryConfig contains recovery middleware configuration.
type RecoveryConfig struct {
	StackSize int  // Stack size in KB
	PrintStack bool // Whether to print the stack
}

// DefaultRecoveryConfig is the default configuration.
var DefaultRecoveryConfig = RecoveryConfig{
	StackSize:  4096, // 4KB
	PrintStack: true,
}

// Recovery returns panic recovery middleware.
func Recovery() app.HandlerFunc {
	return RecoveryWithConfig(DefaultRecoveryConfig)
}

// RecoveryWithConfig creates recovery middleware with the given configuration.
func RecoveryWithConfig(cfg RecoveryConfig) app.HandlerFunc {
	if cfg.StackSize <= 0 {
		cfg.StackSize = DefaultRecoveryConfig.StackSize
	}

	return func(c context.Context, ctx *app.RequestContext) {
		defer func() {
			if r := recover(); r != nil {
				// Record the error
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}

				// Get stack information
				stack := debug.Stack()

				// Log the error
				logger.ErrorCtx(c, "panic recovered",
					zap.String("error", err.Error()),
					zap.String("stack", string(stack)),
				)

				// Return a 500 error
				response.Fail(ctx, errors.ErrSystemError)
				ctx.SetStatusCode(consts.StatusInternalServerError)
				ctx.Abort()
			}
		}()

		ctx.Next(c)
	}
}
