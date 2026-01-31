package core

import (
	"log/slog"
	"time"

	"sim-hub/internal/data"
)

// 生命周期事件类型
const (
	EventVersionActivated = "version.activated" // 版本处理完成，转为可用
	EventResourceCreated  = "resource.created"  // 新资源创建
	EventResourceDeleted  = "resource.deleted"  // 资源被删除
	EventResourceUpdated  = "resource.updated"  // 资源属性（更名、移动）变更
	EventVersionDeleted   = "version.deleted"   // 某个版本被物理或逻辑删除
)

// LifecycleEvent 生命周期事件标准载荷
type LifecycleEvent struct {
	Type       string         `json:"type"` // 事件类型
	ResourceID string         `json:"resource_id"`
	VersionID  string         `json:"version_id,omitempty"`
	TypeKey    string         `json:"type_key"`
	Timestamp  time.Time      `json:"timestamp"`
	Data       map[string]any `json:"data,omitempty"` // 附加信息
}

// EventEmitter 生命周期事件发射器
type EventEmitter struct {
	nats *data.NATSClient
}

func NewEventEmitter(nats *data.NATSClient) *EventEmitter {
	return &EventEmitter{nats: nats}
}

// Emit 发送一个生命周期事件
func (e *EventEmitter) Emit(event LifecycleEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	slog.Info("发送生命周期事件", "type", event.Type, "resource", event.ResourceID)

	// 如果 NATS 启用，发送到 NATS
	if e.nats != nil && e.nats.Config.Enabled {
		subject := "simhub.events.resource"
		if err := e.nats.Encoded.Publish(subject, &event); err != nil {
			slog.Error("NATS 事件发布失败", "error", err)
		}
	}

	// 未来可以扩展：在此处增加 Webhook 调用逻辑
}
