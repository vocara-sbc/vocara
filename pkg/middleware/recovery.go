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

// RecoveryConfig 恢复中间件配置
type RecoveryConfig struct {
	StackSize int  // 堆栈大小（KB）
	PrintStack bool // 是否打印堆栈
}

// DefaultRecoveryConfig 默认配置
var DefaultRecoveryConfig = RecoveryConfig{
	StackSize:  4096, // 4KB
	PrintStack: true,
}

// Recovery panic 恢复中间件
func Recovery() app.HandlerFunc {
	return RecoveryWithConfig(DefaultRecoveryConfig)
}

// RecoveryWithConfig 使用配置创建恢复中间件
func RecoveryWithConfig(cfg RecoveryConfig) app.HandlerFunc {
	if cfg.StackSize <= 0 {
		cfg.StackSize = DefaultRecoveryConfig.StackSize
	}

	return func(c context.Context, ctx *app.RequestContext) {
		defer func() {
			if r := recover(); r != nil {
				// 记录错误
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}

				// 获取堆栈信息
				stack := debug.Stack()

				// 记录错误日志
				logger.ErrorCtx(c, "panic recovered",
					zap.String("error", err.Error()),
					zap.String("stack", string(stack)),
				)

				// 返回 500 错误
				response.Fail(ctx, errors.ErrSystemError)
				ctx.SetStatusCode(consts.StatusInternalServerError)
				ctx.Abort()
			}
		}()

		ctx.Next(c)
	}
}
