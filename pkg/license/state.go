package license

import (
	"fmt"
	"time"
)

// State 授权状态。
type State string

const (
	// StateValid 授权正常
	StateValid State = "valid"
	// StateExpiring 即将到期（到期前 30 天内）
	StateExpiring State = "expiring"
	// StateGrace 到期宽限期
	StateGrace State = "grace"
	// StateExpired 授权已失效
	StateExpired State = "expired"
	// StateInvalid 授权无效（未生效、载荷缺失、签名或指纹异常等）
	StateInvalid State = "invalid"
)

// Result 授权评估结果。
type Result struct {
	// State 授权状态
	State State `json:"state"`
	// Claims 评估所依据的授权载荷
	Claims *LicenseClaims `json:"-"`
	// DaysLeft 距到期整天数；永久授权为 -1，已过期为 0
	DaysLeft int `json:"days_left"`
	// Message 面向用户的状态说明
	Message string `json:"message"`
}

// InvalidResult 构造无效授权结果，用于文件缺失、签名失败等无有效载荷的场景。
func InvalidResult(message string) Result {
	return Result{State: StateInvalid, Message: message}
}

// Evaluate 依据授权载荷与当前时间判定授权状态。
// 文件读取、签名验签等前置失败由调用方直接构造 invalid 结果，不在此处理。
func Evaluate(claims *LicenseClaims, now time.Time) Result {
	if claims == nil {
		return InvalidResult("授权载荷缺失")
	}
	if now.Before(claims.NotBefore) {
		return Result{State: StateInvalid, Claims: claims, Message: "授权尚未生效"}
	}
	if claims.IsPerpetual() {
		return Result{State: StateValid, Claims: claims, DaysLeft: -1, Message: "永久授权"}
	}

	expire := claims.ExpireAt
	if !now.After(*expire) {
		daysLeft := ceilDays(expire.Sub(now))
		if expire.Sub(now) <= ExpiringWindow {
			return Result{
				State:    StateExpiring,
				Claims:   claims,
				DaysLeft: daysLeft,
				Message:  fmt.Sprintf("授权将于 %d 天后到期", daysLeft),
			}
		}
		return Result{State: StateValid, Claims: claims, DaysLeft: daysLeft, Message: "授权正常"}
	}

	graceEnd := expire.Add(time.Duration(claims.Grace()) * 24 * time.Hour)
	if !now.After(graceEnd) {
		return Result{
			State:   StateGrace,
			Claims:  claims,
			Message: fmt.Sprintf("授权已到期 %d 天，处于宽限期", ceilDays(now.Sub(*expire))),
		}
	}
	return Result{State: StateExpired, Claims: claims, Message: "授权已到期"}
}

// ceilDays 将时长向上归整为整天数。
func ceilDays(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	days := d / (24 * time.Hour)
	if d%(24*time.Hour) > 0 {
		days++
	}
	return int(days)
}
