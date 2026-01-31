package core

import (
	"context"
	"log/slog"

	"sim-hub/internal/data"
)

// ResultWatcher 负责监听 Worker 上报的处理结果并更新数据库
type ResultWatcher struct {
	nats          *data.NATSClient
	resultHandler func(ctx context.Context, versionID string, req ProcessResultRequest) error
}

func NewResultWatcher(nats *data.NATSClient, resultHandler func(ctx context.Context, versionID string, req ProcessResultRequest) error) *ResultWatcher {
	return &ResultWatcher{
		nats:          nats,
		resultHandler: resultHandler,
	}
}

func (w *ResultWatcher) Start() {
	if w.nats == nil || !w.nats.Config.Enabled {
		return
	}

	subject := "simhub.results.resource"
	slog.Info("ResultWatcher 正在监听处理结果", "subject", subject)

	_, err := w.nats.Encoded.Subscribe(subject, func(payload *struct {
		VersionID string               `json:"version_id"`
		Result    ProcessResultRequest `json:"result"`
	}) {
		slog.Info("接收到 Worker 处理结果", "version", payload.VersionID, "state", payload.Result.State)
		if err := w.resultHandler(context.Background(), payload.VersionID, payload.Result); err != nil {
			slog.Error("处理 Worker 结果失败", "version", payload.VersionID, "error", err)
		}
	})

	if err != nil {
		slog.Error("ResultWatcher 订阅 NATS 失败", "error", err)
	}
}

// LocalResultEmitter 用于单机模式，直接通过内存调用
type LocalResultEmitter struct {
	ResultHandler func(ctx context.Context, versionID string, req ProcessResultRequest) error
}

func (e *LocalResultEmitter) EmitResult(ctx context.Context, versionID string, result ProcessResultRequest) error {
	if e.ResultHandler != nil {
		return e.ResultHandler(ctx, versionID, result)
	}
	return nil
}
