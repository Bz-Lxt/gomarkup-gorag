// Package timeutil 统一使用 GMT+8 北京时间，禁止直接调用 time.Now/UTC。
package timeutil

import "time"

// Beijing 为 GMT+8 固定时区（不依赖宿主机 TZ 数据文件）。
var Beijing = time.FixedZone("CST", 8*3600)

// Now 返回当前北京时间（带时区）。
func Now() time.Time {
	return time.Now().In(Beijing)
}

// NowNaive 返回去掉时区信息的北京墙钟时间，供 JSON / 存储使用。
func NowNaive() time.Time {
	return Now().Truncate(time.Second)
}

// Format 以 yyyy-MM-dd HH:mm:ss 输出北京时间。
func Format(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(Beijing).Format("2006-01-02 15:04:05")
}

// UnixMilli 返回当前北京时间对应的 Unix 毫秒（绝对时刻，与时区无关）。
func UnixMilli() int64 {
	return time.Now().UnixMilli()
}

// FromUnixMilli 将毫秒时间戳转为北京时间。
func FromUnixMilli(ms int64) time.Time {
	return time.UnixMilli(ms).In(Beijing)
}
