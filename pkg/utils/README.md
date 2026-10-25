# utils - 工具函数包

提供常用工具函数，包括字符串工具、UUID、时间工具、数字工具、IP 工具、加密工具、结构体拷贝等。

## 功能特性

- 字符串工具（随机字符串、格式转换、判断等）
- UUID 生成（v4）
- 时间工具（格式化、计算、转换）
- 数字工具（转换、四舍五入、比较）
- IP 工具（本机 IP、客户端 IP、内网判断）
- 加密工具（MD5、SHA256）
- 结构体拷贝（基于 jinzhu/copier）

## 使用方法

### 字符串工具

```go
import "vocara/pkg/utils"

// 随机字符串
s := utils.RandString(10)          // 随机字母
s = utils.RandNumString(6)         // 随机数字
s = utils.RandAlphaNum(16)         // 随机字母数字
s = utils.RandHex(8)               // 随机十六进制

// 字符串操作
utils.Substring("hello world", 0, 5)  // "hello"
utils.Reverse("hello")                 // "olleh"
utils.IsBlank("  ")                    // true
utils.IsEmpty("")                      // true

// 格式转换
utils.CamelToSnake("CamelCase")        // "camel_case"
utils.SnakeToCamel("snake_case")      // "SnakeCase"

// 填充
utils.PadLeft("123", 5, '0')          // "00123"
utils.PadRight("abc", 5, '*')         // "abc**"

// 首字母
utils.FirstUpper("hello")              // "Hello"
utils.FirstLower("Hello")              // "hello"

// 判断
utils.Contains([]string{"a","b"}, "a") // true
utils.Truncate("long text", 5)         // "long ..."
```

### UUID

```go
import "vocara/pkg/utils"

// 生成 UUID v4
uuid := utils.UUID()
// "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

// 生成短 UUID（无连字符）
short := utils.UUIDShort()
// "6ba7b8109dad11d180b400c04fd430c8"

// 验证 UUID
utils.IsUUID("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // true
```

### 时间工具

```go
import "vocara/pkg/utils"

// 当前时间
now := utils.Now()
unix := utils.NowUnix()           // 秒
milli := utils.NowUnixMilli()     // 毫秒
micro := utils.NowUnixMicro()     // 微秒
nano := utils.NowUnixNano()       // 纳秒

// 格式化
utils.FormatDate(now)             // "2024-01-15"
utils.FormatTime(now)             // "2024-01-15 10:30:00"
utils.FormatTimeMilli(now)        // "2024-01-15 10:30:00.000"
utils.FormatISO8601(now)          // "2024-01-15T10:30:00+08:00"

// 解析
t, _ := utils.ParseDate("2024-01-15")
t, _ := utils.ParseTime("2024-01-15 10:30:00")
t, _ := utils.ParseISO8601("2024-01-15T10:30:00Z")

// 时间戳转换
t = utils.UnixToTime(1700000000)
t = utils.UnixMilliToTime(1700000000000)

// 时间计算
utils.StartOfDay(t)               // 当天开始
utils.EndOfDay(t)                 // 当天结束
utils.StartOfWeek(t)              // 本周一开始
utils.EndOfWeek(t)                // 本周日结束
utils.StartOfMonth(t)             // 本月开始
utils.EndOfMonth(t)               // 本月结束

utils.IsSameDay(t1, t2)           // 是否同一天
utils.IsToday(t)                  // 是否今天
utils.DaysBetween(t1, t2)         // 相差天数
utils.AddDays(t, 5)               // 加5天
utils.AddHours(t, 3)              // 加3小时
```

### 数字工具

```go
import "vocara/pkg/utils"

// 转换
s := utils.IntToString(123)                // "123"
s = utils.Int64ToString(123456789)         // "123456789"
i := utils.StringToIntDefault("abc", -1)   // -1
f := utils.StringToFloat64Default("3.14", 0)

// 四舍五入
utils.Round(3.14159, 2)     // 3.14
utils.Ceil(3.1, 0)          // 4
utils.Floor(3.9, 0)         // 3

// 比较
utils.MaxInt(1, 2)          // 2
utils.MinInt(1, 2)          // 1
utils.MaxInt64(100, 200)    // 200

// 绝对值
utils.AbsInt(-5)            // 5
utils.AbsInt64(-100)        // 100
```

### IP 工具

```go
import "vocara/pkg/utils"

// 本机 IP
ip := utils.GetLocalIP()
ips := utils.GetLocalIPs()

// 验证
utils.IsValidIP("192.168.1.1")    // true
utils.IsInternalIP("127.0.0.1")    // true
utils.IsInternalIP("10.0.0.1")     // true
utils.IsInternalIP("8.8.8.8")      // false

// 从 HTTP 请求获取客户端 IP
ip = utils.GetClientIP(r)
```

### 加密工具

```go
import "vocara/pkg/utils"

// MD5
md5 := utils.MD5("hello")
// "5d41402abc4b2a76b9719d911017c592"

// SHA256
sha256 := utils.SHA256("hello")
// "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

// 字节版本
md5 = utils.MD5Bytes([]byte("hello"))
sha256 = utils.SHA256Bytes([]byte("hello"))
```

### 结构体拷贝

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

src := User{Name: "张三", Age: 25}
var dst UserVO

// 基本拷贝
err := utils.Copy(&dst, &src)

// 带选项拷贝
utils.CopyWithOption(&dst, &src, copier.Option{
    IgnoreEmpty: true,
})

// MustCopy（失败 panic）
utils.MustCopy(&dst, &src)
```
