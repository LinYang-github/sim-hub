package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"

	"sim-hub/internal/conf"
	"gopkg.in/natefinch/lumberjack.v2"
)

type contextKey string

const (
	TraceIDKey contextKey = "trace_id"
)

// WithTraceID 为 context 注入 TraceID
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID 从 context 获取 TraceID
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(TraceIDKey).(string); ok {
		return v
	}
	return ""
}

// InitLogger 初始化全局默认 Logger
func InitLogger(c *conf.Log) {
	var w io.Writer

	// 1. 配置日志输出目标
	if c.Filename != "" {
		w = &lumberjack.Logger{
			Filename:   c.Filename,
			MaxSize:    c.MaxSize,
			MaxBackups: c.MaxBackups,
			MaxAge:     c.MaxAge,
			Compress:   c.Compress,
		}
	} else {
		w = os.Stdout
	}

	// 2. 配置日志级别
	var level slog.Level
	switch strings.ToLower(c.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	// 3. 配置 Handler
	var handler slog.Handler
	if strings.ToLower(c.Format) == "json" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		// 默认为自定义 Web 风格 Text 格式
		handler = NewSimHubHandler(w, opts)
	}

	// 4. 设置为全局默认 Logger
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// SimHubHandler 自定义文本日志处理器
type SimHubHandler struct {
	w     io.Writer
	opts  *slog.HandlerOptions
	mu    sync.Mutex
	attrs []slog.Attr // 预与其属性 (WithAttrs)
	group string      // 当前组名 (WithGroup - 简化实现)
}

func NewSimHubHandler(w io.Writer, opts *slog.HandlerOptions) *SimHubHandler {
	return &SimHubHandler{
		w:    w,
		opts: opts,
	}
}

func (h *SimHubHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *SimHubHandler) Handle(ctx context.Context, r slog.Record) error {
	// 格式: 2026-01-13 11:28:47.21 [INFO] Message key=value
	buf := make([]byte, 0, 1024)

	// 1. Time
	buf = fmt.Appendf(buf, "%s ", r.Time.Format("2006-01-02 15:04:05.000"))

	// 2. Level
	level := r.Level.String()
	buf = fmt.Appendf(buf, "[%s] ", level)

	// 3. TraceID (if exists in context)
	if traceID := GetTraceID(ctx); traceID != "" {
		buf = fmt.Appendf(buf, "[%s] ", traceID)
	}

	// 4. Message
	buf = fmt.Appendf(buf, "%s", r.Message)

	// 5. Attributes (合并预设属性和记录属性)
	// 先处理 WithAttrs 添加的属性
	for _, a := range h.attrs {
		buf = h.appendAttr(buf, a)
	}
	// 再处理当前日志记录的属性
	r.Attrs(func(a slog.Attr) bool {
		buf = h.appendAttr(buf, a)
		return true
	})

	buf = append(buf, '\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(buf)
	return err
}

func (h *SimHubHandler) appendAttr(buf []byte, a slog.Attr) []byte {
	// 简单 resolve (不处理 group 嵌套的复杂情况以保持高性能)
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return buf
	}
	// key=value 格式
	return fmt.Appendf(buf, " %s=%v", a.Key, a.Value.Any())
}

func (h *SimHubHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// 创建新副本
	return &SimHubHandler{
		w:     h.w,
		opts:  h.opts,
		attrs: append(h.attrs, attrs...),
		group: h.group,
	}
}

func (h *SimHubHandler) WithGroup(name string) slog.Handler {
	// 简化版：仅作为前缀或暂不支持复杂 Group
	// 为了这种简单的行日志格式，通常忽略 group 或将其拼接到 key 中
	return h
}
