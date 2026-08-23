// Package logger 提供分级日志。生产环境（LOG_ENV=production）自动屏蔽 debug。
package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/xavskye/gorag/pkg/timeutil"
)

var (
	mu     sync.RWMutex
	global *slog.Logger
	level  slog.Level
)

func init() {
	Init(os.Getenv("LOG_LEVEL"), os.Getenv("LOG_ENV"))
}

// Init 根据环境初始化全局 logger。level: debug|info|warn|error。
func Init(levelName, env string) {
	lv := parseLevel(levelName)
	if strings.EqualFold(env, "production") && lv < slog.LevelInfo {
		lv = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lv,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String(slog.TimeKey, timeutil.Format(timeutil.Now()))
			}
			return a
		},
	})
	mu.Lock()
	defer mu.Unlock()
	level = lv
	global = slog.New(h)
}

// SetOutput 仅供测试替换输出。
func SetOutput(w io.Writer, lv slog.Level) {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lv})
	mu.Lock()
	defer mu.Unlock()
	level = lv
	global = slog.New(h)
}

func L() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	if global == nil {
		return slog.Default()
	}
	return global
}

func Debug(msg string, args ...any) { L().Debug(msg, args...) }
func Info(msg string, args ...any)  { L().Info(msg, args...) }
func Warn(msg string, args ...any)  { L().Warn(msg, args...) }
func Error(msg string, args ...any) { L().Error(msg, args...) }

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
