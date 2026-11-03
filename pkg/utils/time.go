package utils

import (
	"time"
)

// Common time formats
const (
	DateFormat      = "2006-01-02"
	TimeFormat      = "2006-01-02 15:04:05"
	TimeFormatMilli = "2006-01-02 15:04:05.000"
	ISO8601Format   = "2006-01-02T15:04:05Z07:00"
	// ISO8601Milli is the ISO8601 format with milliseconds.
	ISO8601Milli = "2006-01-02T15:04:05.000Z07:00"
)

var (
	// CST is China Standard Time.
	CST = time.FixedZone("CST", 8*3600)
)

// Now returns the current time.
func Now() time.Time {
	return time.Now()
}

// NowIn returns the current time in the specified time zone.
func NowIn(loc *time.Location) time.Time {
	return time.Now().In(loc)
}

// NowUnix returns the current timestamp in seconds.
func NowUnix() int64 {
	return time.Now().Unix()
}

// NowUnixMilli returns the current timestamp in milliseconds.
func NowUnixMilli() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// NowUnixMicro returns the current timestamp in microseconds.
func NowUnixMicro() int64 {
	return time.Now().UnixNano() / int64(time.Microsecond)
}

// NowUnixNano returns the current timestamp in nanoseconds.
func NowUnixNano() int64 {
	return time.Now().UnixNano()
}

// FormatDate formats a date as YYYY-MM-DD.
func FormatDate(t time.Time) string {
	return t.Format(DateFormat)
}

// FormatTime formats a time as YYYY-MM-DD HH:MM:SS.
func FormatTime(t time.Time) string {
	return t.Format(TimeFormat)
}

// FormatTimeMilli formats a time with milliseconds.
func FormatTimeMilli(t time.Time) string {
	return t.Format(TimeFormatMilli)
}

// FormatISO8601 formats a time as ISO8601.
func FormatISO8601(t time.Time) string {
	return t.Format(ISO8601Format)
}

// ParseDate parses a date in YYYY-MM-DD format.
func ParseDate(s string) (time.Time, error) {
	return time.ParseInLocation(DateFormat, s, time.Local)
}

// ParseTime parses a time in YYYY-MM-DD HH:MM:SS format.
func ParseTime(s string) (time.Time, error) {
	return time.ParseInLocation(TimeFormat, s, time.Local)
}

// ParseISO8601 parses ISO8601 format.
func ParseISO8601(s string) (time.Time, error) {
	return time.Parse(ISO8601Format, s)
}

// UnixToTime converts a timestamp in seconds to a time.
func UnixToTime(sec int64) time.Time {
	return time.Unix(sec, 0)
}

// UnixMilliToTime converts a millisecond timestamp to a time.
func UnixMilliToTime(msec int64) time.Time {
	return time.Unix(msec/1000, (msec%1000)*int64(time.Millisecond))
}

// StartOfDay returns the start of the day.
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay returns the end of the day.
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), t.Location())
}

// StartOfWeek returns the start of the week on Monday.
func StartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return StartOfDay(t.AddDate(0, 0, -weekday+1))
}

// EndOfWeek returns the end of the week on Sunday.
func EndOfWeek(t time.Time) time.Time {
	return EndOfDay(StartOfWeek(t).AddDate(0, 0, 6))
}

// StartOfMonth returns the start of the month.
func StartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth returns the end of the month.
func EndOfMonth(t time.Time) time.Time {
	nextMonth := StartOfMonth(t.AddDate(0, 1, 0))
	return nextMonth.Add(-time.Nanosecond)
}

// IsSameDay reports whether two times are on the same day.
func IsSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// IsToday reports whether a time is today.
func IsToday(t time.Time) bool {
	return IsSameDay(t, time.Now())
}

// DaysBetween calculates the number of days between two dates.
func DaysBetween(t1, t2 time.Time) int {
	t1 = StartOfDay(t1)
	t2 = StartOfDay(t2)
	return int(t2.Sub(t1).Hours() / 24)
}

// AddDays adds days.
func AddDays(t time.Time, days int) time.Time {
	return t.AddDate(0, 0, days)
}

// AddHours adds hours.
func AddHours(t time.Time, hours int) time.Time {
	return t.Add(time.Duration(hours) * time.Hour)
}

// AddMinutes adds minutes.
func AddMinutes(t time.Time, minutes int) time.Time {
	return t.Add(time.Duration(minutes) * time.Minute)
}

// AddSeconds adds seconds.
func AddSeconds(t time.Time, seconds int) time.Time {
	return t.Add(time.Duration(seconds) * time.Second)
}

// DiffSeconds returns the difference between two times in seconds.
func DiffSeconds(t1, t2 time.Time) float64 {
	return t2.Sub(t1).Seconds()
}

// DiffMinutes returns the difference between two times in minutes.
func DiffMinutes(t1, t2 time.Time) float64 {
	return t2.Sub(t1).Minutes()
}

// DiffHours returns the difference between two times in hours.
func DiffHours(t1, t2 time.Time) float64 {
	return t2.Sub(t1).Hours()
}
