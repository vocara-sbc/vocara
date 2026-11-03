# utils - Utility Function Package

Provides common utility functions, including string utilities, UUIDs, time utilities, number utilities, IP utilities, encryption utilities, struct copying, and more.

## Features

- String utilities (random strings, format conversion, checks, and more)
- UUID generation (v4)
- Time utilities (formatting, calculation, and conversion)
- Number utilities (conversion, rounding, and comparison)
- IP utilities (local IP, client IP, and private network checks)
- Encryption utilities (MD5 and SHA256)
- Struct copying (based on jinzhu/copier)

## Usage

### String Utilities

```go
import "vocara/pkg/utils"

// Random strings
s := utils.RandString(10)          // Random letters
s = utils.RandNumString(6)         // Random digits
s = utils.RandAlphaNum(16)         // Random alphanumeric characters
s = utils.RandHex(8)               // Random hexadecimal characters

// String operations
utils.Substring("hello world", 0, 5)  // "hello"
utils.Reverse("hello")                 // "olleh"
utils.IsBlank("  ")                    // true
utils.IsEmpty("")                      // true

// Format conversion
utils.CamelToSnake("CamelCase")        // "camel_case"
utils.SnakeToCamel("snake_case")      // "SnakeCase"

// Padding
utils.PadLeft("123", 5, '0')          // "00123"
utils.PadRight("abc", 5, '*')         // "abc**"

// First character
utils.FirstUpper("hello")              // "Hello"
utils.FirstLower("Hello")              // "hello"

// Checks
utils.Contains([]string{"a","b"}, "a") // true
utils.Truncate("long text", 5)         // "long ..."
```

### UUID

```go
import "vocara/pkg/utils"

// Generate a UUID v4
uuid := utils.UUID()
// "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

// Generate a short UUID (without hyphens)
short := utils.UUIDShort()
// "6ba7b8109dad11d180b400c04fd430c8"

// Validate a UUID
utils.IsUUID("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // true
```

### Time Utilities

```go
import "vocara/pkg/utils"

// Current time
now := utils.Now()
unix := utils.NowUnix()           // Seconds
milli := utils.NowUnixMilli()     // Milliseconds
micro := utils.NowUnixMicro()     // Microseconds
nano := utils.NowUnixNano()       // Nanoseconds

// Formatting
utils.FormatDate(now)             // "2024-01-15"
utils.FormatTime(now)             // "2024-01-15 10:30:00"
utils.FormatTimeMilli(now)        // "2024-01-15 10:30:00.000"
utils.FormatISO8601(now)          // "2024-01-15T10:30:00+08:00"

// Parsing
t, _ := utils.ParseDate("2024-01-15")
t, _ := utils.ParseTime("2024-01-15 10:30:00")
t, _ := utils.ParseISO8601("2024-01-15T10:30:00Z")

// Timestamp conversion
t = utils.UnixToTime(1700000000)
t = utils.UnixMilliToTime(1700000000000)

// Time calculations
utils.StartOfDay(t)               // Start of the current day
utils.EndOfDay(t)                 // End of the current day
utils.StartOfWeek(t)              // Start of the current week on Monday
utils.EndOfWeek(t)                // End of the current week on Sunday
utils.StartOfMonth(t)             // Start of the current month
utils.EndOfMonth(t)               // End of the current month

utils.IsSameDay(t1, t2)           // Whether both times are on the same day
utils.IsToday(t)                  // Whether the time is today
utils.DaysBetween(t1, t2)         // Number of days between
utils.AddDays(t, 5)               // Add 5 days
utils.AddHours(t, 3)              // Add 3 hours
```

### Number Utilities

```go
import "vocara/pkg/utils"

// Conversion
s := utils.IntToString(123)                // "123"
s = utils.Int64ToString(123456789)         // "123456789"
i := utils.StringToIntDefault("abc", -1)   // -1
f := utils.StringToFloat64Default("3.14", 0)

// Rounding
utils.Round(3.14159, 2)     // 3.14
utils.Ceil(3.1, 0)          // 4
utils.Floor(3.9, 0)         // 3

// Comparison
utils.MaxInt(1, 2)          // 2
utils.MinInt(1, 2)          // 1
utils.MaxInt64(100, 200)    // 200

// Absolute value
utils.AbsInt(-5)            // 5
utils.AbsInt64(-100)        // 100
```

### IP Utilities

```go
import "vocara/pkg/utils"

// Local IP
ip := utils.GetLocalIP()
ips := utils.GetLocalIPs()

// Validation
utils.IsValidIP("192.168.1.1")    // true
utils.IsInternalIP("127.0.0.1")    // true
utils.IsInternalIP("10.0.0.1")     // true
utils.IsInternalIP("8.8.8.8")      // false

// Get the client IP from an HTTP request
ip = utils.GetClientIP(r)
```

### Encryption Utilities

```go
import "vocara/pkg/utils"

// MD5
md5 := utils.MD5("hello")
// "5d41402abc4b2a76b9719d911017c592"

// SHA256
sha256 := utils.SHA256("hello")
// "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

// Byte versions
md5 = utils.MD5Bytes([]byte("hello"))
sha256 = utils.SHA256Bytes([]byte("hello"))
```

### Struct Copying

```go
import "vocara/pkg/utils"

type User struct {
    Name string
    Age  int
}

type UserVO struct {
    Name string
    Age  int
}

src := User{Name: "John Doe", Age: 25}
var dst UserVO

// Basic copy
err := utils.Copy(&dst, &src)

// Copy with options
utils.CopyWithOption(&dst, &src, copier.Option{
    IgnoreEmpty: true,
})

// MustCopy (panics on failure)
utils.MustCopy(&dst, &src)
```
