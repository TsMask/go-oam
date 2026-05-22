package date

import (
	"time"
)

// ParseStrToDate 根据时间字符串解析时间 时间格式看time.DateTime
func ParseStrToDate(dateStr, formatStr string) time.Time {
	if dateStr == "" || dateStr == "<nil>" {
		return time.Time{}
	}
	if formatStr == "" {
		formatStr = time.DateTime
	}
	t, err := time.Parse(formatStr, dateStr)
	if err != nil {
		return time.Time{}
	}
	return t
}

// ParseNumberToDate 根据时间数字时间戳解析时间
func ParseNumberToDate(dateV int64) time.Time {
	t := time.Time{}
	if dateV == 0 {
		return t
	}
	if dateV > 1e15 {
		t = time.UnixMicro(dateV)
	} else if dateV > 1e12 {
		t = time.UnixMilli(dateV)
	} else if dateV > 1e9 {
		t = time.Unix(dateV, 0)
	} else {
		return t
	}
	return t
}
