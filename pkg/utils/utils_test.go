package utils

import (
	"testing"
	"time"
)

func TestUUID(t *testing.T) {
	uuid := UUID()
	if len(uuid) != 36 {
		t.Errorf("UUID length = %d, want 36", len(uuid))
	}
	if !IsUUID(uuid) {
		t.Error("UUID should be valid")
	}
}

func TestUUIDShort(t *testing.T) {
	uuid := UUIDShort()
	if len(uuid) != 32 {
		t.Errorf("UUIDShort length = %d, want 32", len(uuid))
	}
}

func TestIsUUID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"6ba7b810-9dad-11d1-80b4-00c04fd430c8", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsUUID(tt.input); got != tt.want {
				t.Errorf("IsUUID(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMD5(t *testing.T) {
	result := MD5("hello")
	expected := "5d41402abc4b2a76b9719d911017c592"
	if result != expected {
		t.Errorf("MD5('hello') = %s, want %s", result, expected)
	}
}

func TestSHA256(t *testing.T) {
	result := SHA256("hello")
	expected := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if result != expected {
		t.Errorf("SHA256('hello') = %s, want %s", result, expected)
	}
}

func TestMD5Bytes(t *testing.T) {
	result := MD5Bytes([]byte("hello"))
	expected := "5d41402abc4b2a76b9719d911017c592"
	if result != expected {
		t.Errorf("MD5Bytes('hello') = %s, want %s", result, expected)
	}
}

func TestSHA256Bytes(t *testing.T) {
	result := SHA256Bytes([]byte("hello"))
	expected := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if result != expected {
		t.Errorf("SHA256Bytes('hello') = %s, want %s", result, expected)
	}
}

func TestGetLocalIP(t *testing.T) {
	ip := GetLocalIP()
	if ip == "" {
		t.Error("GetLocalIP should not return empty")
	}
	t.Logf("Local IP: %s", ip)
}

func TestGetLocalIPs(t *testing.T) {
	ips := GetLocalIPs()
	t.Logf("Local IPs: %v", ips)
}

func TestIsValidIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},
		{"192.168.1.1", true},
		{"255.255.255.255", true},
		{"::1", true},
		{"256.1.1.1", false},
		{"abc", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			if got := IsValidIP(tt.ip); got != tt.want {
				t.Errorf("IsValidIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestIsInternalIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"8.8.8.8", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			if got := IsInternalIP(tt.ip); got != tt.want {
				t.Errorf("IsInternalIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestFormatDate(t *testing.T) {
	tm := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	result := FormatDate(tm)
	expected := "2024-01-15"
	if result != expected {
		t.Errorf("FormatDate = %s, want %s", result, expected)
	}
}

func TestFormatTime(t *testing.T) {
	tm := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	result := FormatTime(tm)
	expected := "2024-01-15 10:30:00"
	if result != expected {
		t.Errorf("FormatTime = %s, want %s", result, expected)
	}
}

func TestParseDate(t *testing.T) {
	tm, err := ParseDate("2024-01-15")
	if err != nil {
		t.Fatalf("ParseDate failed: %v", err)
	}
	if tm.Year() != 2024 || tm.Month() != 1 || tm.Day() != 15 {
		t.Errorf("ParseDate result incorrect: %v", tm)
	}
}

func TestParseTime(t *testing.T) {
	tm, err := ParseTime("2024-01-15 10:30:00")
	if err != nil {
		t.Fatalf("ParseTime failed: %v", err)
	}
	if tm.Hour() != 10 || tm.Minute() != 30 {
		t.Errorf("ParseTime result incorrect: %v", tm)
	}
}

func TestStartOfDay(t *testing.T) {
	tm := time.Date(2024, 1, 15, 10, 30, 45, 0, time.Local)
	start := StartOfDay(tm)
	if start.Hour() != 0 || start.Minute() != 0 || start.Second() != 0 {
		t.Errorf("StartOfDay incorrect: %v", start)
	}
}

func TestEndOfDay(t *testing.T) {
	tm := time.Date(2024, 1, 15, 10, 30, 45, 0, time.Local)
	end := EndOfDay(tm)
	if end.Hour() != 23 || end.Minute() != 59 || end.Second() != 59 {
		t.Errorf("EndOfDay incorrect: %v", end)
	}
}

func TestStartOfMonth(t *testing.T) {
	tm := time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local)
	start := StartOfMonth(tm)
	if start.Day() != 1 {
		t.Errorf("StartOfMonth day = %d, want 1", start.Day())
	}
}

func TestEndOfMonth(t *testing.T) {
	tm := time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local)
	end := EndOfMonth(tm)
	if end.Month() != 1 {
		t.Errorf("EndOfMonth month = %v, want January", end.Month())
	}
}

func TestIsSameDay(t *testing.T) {
	t1 := time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local)
	t2 := time.Date(2024, 1, 15, 20, 0, 0, 0, time.Local)
	t3 := time.Date(2024, 1, 16, 10, 30, 0, 0, time.Local)

	if !IsSameDay(t1, t2) {
		t.Error("t1 and t2 should be same day")
	}
	if IsSameDay(t1, t3) {
		t.Error("t1 and t3 should not be same day")
	}
}

func TestDaysBetween(t *testing.T) {
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)
	t2 := time.Date(2024, 1, 10, 0, 0, 0, 0, time.Local)

	days := DaysBetween(t1, t2)
	if days != 9 {
		t.Errorf("DaysBetween = %d, want 9", days)
	}
}

func TestNowUnix(t *testing.T) {
	unix := NowUnix()
	if unix <= 0 {
		t.Error("NowUnix should be positive")
	}
}

func TestNowUnixMilli(t *testing.T) {
	milli := NowUnixMilli()
	if milli <= 0 {
		t.Error("NowUnixMilli should be positive")
	}
}

func TestUnixToTime(t *testing.T) {
	tm := UnixToTime(1700000000)
	if tm.IsZero() {
		t.Error("UnixToTime should not be zero")
	}
}

func TestAddDays(t *testing.T) {
	tm := time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)
	result := AddDays(tm, 5)
	if result.Day() != 6 {
		t.Errorf("AddDays day = %d, want 6", result.Day())
	}
}

func TestIntToString(t *testing.T) {
	if IntToString(123) != "123" {
		t.Error("IntToString failed")
	}
	if IntToString(0) != "0" {
		t.Error("IntToString failed for 0")
	}
}

func TestStringToInt(t *testing.T) {
	v, err := StringToInt("123")
	if err != nil || v != 123 {
		t.Error("StringToInt failed")
	}

	v = StringToIntDefault("abc", -1)
	if v != -1 {
		t.Error("StringToIntDefault failed")
	}
}

func TestInt64ToString(t *testing.T) {
	if Int64ToString(123456789) != "123456789" {
		t.Error("Int64ToString failed")
	}
}

func TestFloat64ToString(t *testing.T) {
	s := Float64ToString(3.14)
	t.Logf("Float64ToString(3.14) = %s", s)
}

func TestRound(t *testing.T) {
	tests := []struct {
		f    float64
		n    int
		want float64
	}{
		{3.14159, 2, 3.14},
		{3.145, 2, 3.15},
		{3.1, 0, 3},
	}

	for _, tt := range tests {
		got := Round(tt.f, tt.n)
		if got != tt.want {
			t.Errorf("Round(%f, %d) = %f, want %f", tt.f, tt.n, got, tt.want)
		}
	}
}

func TestMaxMinInt(t *testing.T) {
	if MaxInt(1, 2) != 2 {
		t.Error("MaxInt failed")
	}
	if MinInt(1, 2) != 1 {
		t.Error("MinInt failed")
	}
}

func TestAbsInt(t *testing.T) {
	if AbsInt(-5) != 5 {
		t.Error("AbsInt failed for -5")
	}
	if AbsInt(5) != 5 {
		t.Error("AbsInt failed for 5")
	}
}

func TestCopy(t *testing.T) {
	type Src struct {
		Name string
		Age  int
	}
	type Dst struct {
		Name string
		Age  int
	}

	src := Src{Name: "test", Age: 18}
	var dst Dst

	err := Copy(&dst, &src)
	if err != nil {
		t.Fatalf("Copy failed: %v", err)
	}
	if dst.Name != src.Name || dst.Age != src.Age {
		t.Errorf("Copy result incorrect: %+v", dst)
	}
}
