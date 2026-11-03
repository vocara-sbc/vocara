package license

import (
	"fmt"
	"time"
)

// State is the authorization state.
type State string

const (
	// StateValid indicates a valid authorization.
	StateValid State = "valid"
	// StateExpiring indicates expiration within 30 days.
	StateExpiring State = "expiring"
	// StateGrace indicates the grace period after expiration.
	StateGrace State = "grace"
	// StateExpired indicates an expired authorization.
	StateExpired State = "expired"
	// StateInvalid indicates an invalid authorization due to not being active, missing claims, or signature or fingerprint errors.
	StateInvalid State = "invalid"
)

// Result is the authorization evaluation result.
type Result struct {
	// State is the authorization state.
	State State `json:"state"`
	// Claims contains the authorization claims used for evaluation.
	Claims *LicenseClaims `json:"-"`
	// DaysLeft is the number of whole days until expiration; -1 for perpetual authorization and 0 after expiration.
	DaysLeft int `json:"days_left"`
	// Message is the user-facing status description.
	Message string `json:"message"`
}

// InvalidResult constructs an invalid authorization result for scenarios without valid claims, such as a missing file or signature verification failure.
func InvalidResult(message string) Result {
	return Result{State: StateInvalid, Message: message}
}

// Evaluate determines the authorization state from the claims and current time.
// The caller constructs invalid results directly for prerequisite failures such as file reading or signature verification; those failures are not handled here.
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

// ceilDays rounds a duration up to whole days.
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
