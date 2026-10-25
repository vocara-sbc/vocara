package utils

import (
	"time"
)

// 常用时间格式
const (
	DateFormat      = "2006-01-02"
	TimeFormat      = "2006-01-02 15:04:05"
	TimeFormatMilli = "2006-01-02 15:04:05.000"
	ISO8601Format   = "2006-01-02T15:04:05Z07:00"
	// ISO8601Milli 带毫秒的 ISO8601 格式
	ISO8601Milli = "2006-01-02T15:04:05.000Z07:00"
)

var (
	// CST 中国标准时间
	CST = time.FixedZone("CST", 8*3600)
)

// Now 获取当前时间
func Now() time.Time {
	return time.Now()
}

// NowIn 获取指定时区的当前时间
func NowIn(loc *time.Location) time.Time {
	return time.Now().In(loc)
}

// NowUnix 当前时间戳（秒）
func NowUnix() int64 {
	return time.Now().Unix()
}

// NowUnixMilli 当前时间戳（毫秒）
func NowUnixMilli() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// NowUnixMicro 当前时间戳（微秒）
func NowUnixMicro() int64 {
	return time.Now().UnixNano() / int64(time.Microsecond)
}

// NowUnixNano 当前时间戳（纳秒）
func NowUnixNano() int64 {
	return time.Now().UnixNano()
}

// FormatDate 格式化日期 YYYY-MM-DD
func FormatDate(t time.Time) string {
	return t.Format(DateFormat)
}

// FormatTime 格式化时间 YYYY-MM-DD HH:MM:SS
func FormatTime(t time.Time) string {
	return t.Format(TimeFormat)
}

// FormatTimeMilli 格式化时间（带毫秒）
func FormatTimeMilli(t time.Time) string {
	return t.Format(TimeFormatMilli)
}

// FormatISO8601 格式化为 ISO8601
func FormatISO8601(t time.Time) string {
	return t.Format(ISO8601Format)
}

// ParseDate 解析日期 YYYY-MM-DD
func ParseDate(s string) (time.Time, error) {
	return time.ParseInLocation(DateFormat, s, time.Local)
}

// ParseTime 解析时间 YYYY-MM-DD HH:MM:SS
func ParseTime(s string) (time.Time, error) {
	return time.ParseInLocation(TimeFormat, s, time.Local)
}

// ParseISO8601 解析 ISO8601 格式
func ParseISO8601(s string) (time.Time, error) {
	return time.Parse(ISO8601Format, s)
}

// UnixToTime 时间戳转时间（秒）
func UnixToTime(sec int64) time.Time {
	return time.Unix(sec, 0)
}

// UnixMilliToTime 毫秒时间戳转时间
func UnixMilliToTime(msec int64) time.Time {
	return time.Unix(msec/1000, (msec%1000)*int64(time.Millisecond))
}

// StartOfDay 获取当天开始时间
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay 获取当天结束时间
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), t.Location())
}

// StartOfWeek 获取本周开始时间（周一）
func StartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return StartOfDay(t.AddDate(0, 0, -weekday+1))
}

// EndOfWeek 获取本周结束时间（周日）
func EndOfWeek(t time.Time) time.Time {
	return EndOfDay(StartOfWeek(t).AddDate(0, 0, 6))
}

// StartOfMonth 获取本月开始时间
func StartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth 获取本月结束时间
func EndOfMonth(t time.Time) time.Time {
	nextMonth := StartOfMonth(t.AddDate(0, 1, 0))
	return nextMonth.Add(-time.Nanosecond)
}

// IsSameDay 判断是否同一天
func IsSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// IsToday 判断是否是今天
func IsToday(t time.Time) bool {
	return IsSameDay(t, time.Now())
}

// DaysBetween 计算两个日期之间的天数差
func DaysBetween(t1, t2 time.Time) int {
	t1 = StartOfDay(t1)
	t2 = StartOfDay(t2)
	return int(t2.Sub(t1).Hours() / 24)
}

// AddDays 增加天数
func AddDays(t time.Time, days int) time.Time {
	return t.AddDate(0, 0, days)
}

// AddHours 增加小时
func AddHours(t time.Time, hours int) time.Time {
	return t.Add(time.Duration(hours) * time.Hour)
}

// AddMinutes 增加分钟
func AddMinutes(t time.Time, minutes int) time.Time {
	return t.Add(time.Duration(minutes) * time.Minute)
}

// AddSeconds 增加秒数
func AddSeconds(t time.Time, seconds int) time.Time {
	return t.Add(time.Duration(seconds) * time.Second)
}

// DiffSeconds 两个时间相差的秒数
func DiffSeconds(t1, t2 time.Time) float64 {
	return t2.Sub(t1).Seconds()
}

// DiffMinutes 两个时间相差的分钟数
func DiffMinutes(t1, t2 time.Time) float64 {
	return t2.Sub(t1).Minutes()
}

// DiffHours 两个时间相差的小时数
func DiffHours(t1, t2 time.Time) float64 {
	return t2.Sub(t1).Hours()
}
