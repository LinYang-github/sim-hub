package core

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/internal/model"
	"github.com/liny/sim-hub/pkg/logger"
	"github.com/liny/sim-hub/pkg/storage"
)

const (
	ActionProcess = "PROCESS" // 全流程处理 (执行 Processor + 写 Sidecar)
	ActionRefresh = "REFRESH" // 仅刷新元数据 (重新生成 Sidecar)
	ActionExport  = "EXPORT"  // 异步打包导出
)

type ProcessJob struct {
	Action    string `json:"action"`
	TypeKey   string `json:"type_key"`
	ObjectKey string `json:"object_key"`
	VersionID string `json:"version_id"`
	TraceID   string `json:"trace_id"`
}

// Scheduler 负责任务的调度与分发 (Dispatcher 角色)
type Scheduler struct {
	data    *data.Data
	store   storage.MultipartBlobStore
	nats    *data.NATSClient
	bucket  string
	role    string          // "api", "worker", "combined"
	jobChan chan ProcessJob // 用于本地模式的任务分发

	// Worker 组件 (仅在 worker/combined 模式下有效)
	worker *Worker
}

func NewScheduler(d *data.Data, store storage.MultipartBlobStore, nats *data.NATSClient, bucket string, role string, worker *Worker) *Scheduler {
	s := &Scheduler{
		data:    d,
		store:   store,
		nats:    nats,
		bucket:  bucket,
		role:    role,
		jobChan: make(chan ProcessJob, 1000),
		worker:  worker,
	}

	// 启动消费者或本地 Worker 轮询
	if role == "worker" || role == "combined" {
		if nats != nil && nats.Config.Enabled {
			go s.startNATSSubscriber()
		} else {
			// 本地模式启动多个并发处理单元
			for i := 0; i < 4; i++ {
				go s.startLocalWorker(i)
			}
		}
	}

	return s
}

// Dispatch 发送任务
func (s *Scheduler) Dispatch(ctx context.Context, job ProcessJob) {
	// 注入 TraceID
	if job.TraceID == "" {
		job.TraceID = logger.GetTraceID(ctx)
	}

	// ActionRefresh 需要数据库访问，通常在 API 节点(本地)执行
	if job.Action == ActionRefresh {
		go s.syncSidecarInternal(ctx, job.ObjectKey, job.VersionID)
		return
	}

	if s.nats != nil && s.nats.Config.Enabled {
		if err := s.nats.Publish(&job); err != nil {
			slog.ErrorContext(ctx, "发送 NATS 消息失败，回退到本地队列", "error", err)
			s.jobChan <- job
		}
		return
	}
	s.jobChan <- job
}

func (s *Scheduler) startNATSSubscriber() {
	slog.InfoContext(context.Background(), "NATS 任务订阅者已启动", "subject", s.nats.Config.Subject)
	_, err := s.nats.Encoded.Subscribe(s.nats.Config.Subject, func(job *ProcessJob) {
		s.handleJob(context.Background(), *job)
	})
	if err != nil {
		slog.ErrorContext(context.Background(), "NATS 订阅失败", "error", err)
	}
}

func (s *Scheduler) startLocalWorker(id int) {
	for job := range s.jobChan {
		s.handleJob(context.Background(), job)
	}
}

func (s *Scheduler) handleJob(ctx context.Context, job ProcessJob) {
	// 恢复 TraceID 到 context 中，以便后续 slog 使用
	if job.TraceID != "" {
		ctx = logger.WithTraceID(ctx, job.TraceID)
	}

	if job.Action == ActionRefresh {
		s.syncSidecarInternal(ctx, job.ObjectKey, job.VersionID)
		return
	}

	if s.worker != nil {
		s.worker.HandleJob(ctx, job)
	} else {
		slog.Log(ctx, slog.LevelWarn, "接收到处理任务但本地未配置 Worker 实例", "action", job.Action, "version", job.VersionID)
	}
}

// syncSidecarInternal 仅由 API 节点(或具备 DB 访问权限的节点)执行
// 它负责将 DB 中的最新元数据打包成 sidecar 文件同步到对象存储
func (s *Scheduler) syncSidecarInternal(ctx context.Context, objectKey, versionID string) {
	if s.data == nil {
		slog.ErrorContext(ctx, "尝试运行 syncSidecar 但 DB 实例不可用")
		return
	}

	var ver model.ResourceVersion
	if err := s.data.DB.Preload("Resource").First(&ver, "id = ?", versionID).Error; err != nil {
		slog.ErrorContext(ctx, "同步 Sidecar 时找不到版本记录", "id", versionID, "error", err)
		return
	}

	sidecarData := map[string]any{
		"resource_id":   ver.Resource.ID,
		"resource_name": ver.Resource.Name,
		"tags":          ver.Resource.Tags,
		"version_id":    ver.ID,
		"type_key":      ver.Resource.TypeKey,
		"metadata":      ver.MetaData,
		"synced_at":     time.Now().Format(time.RFC3339),
	}

	if sidecarBytes, err := json.Marshal(sidecarData); err == nil {
		sidecarKey := objectKey + ".json"
		if err := s.store.Put(ctx, s.bucket, sidecarKey, bytes.NewReader(sidecarBytes), int64(len(sidecarBytes)), "application/json"); err != nil {
			slog.ErrorContext(ctx, "更新 Sidecar 失败", "key", sidecarKey, "error", err)
		} else {
			slog.DebugContext(ctx, "Sidecar 刷新成功", "key", sidecarKey)
		}
	}
}
