package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/internal/model"
	"github.com/liny/sim-hub/pkg/storage"
)

const (
	ActionProcess = "PROCESS" // 全流程处理 (执行 Processor + 写 Sidecar)
	ActionRefresh = "REFRESH" // 仅刷新元数据 (重新生成 Sidecar)
	ActionExport  = "EXPORT"  // 异步打包导出
)

type ProcessJob struct {
	Action    string
	TypeKey   string
	ObjectKey string
	VersionID string
}

type Scheduler struct {
	data       *data.Data
	store      storage.MultipartBlobStore
	nats       *data.NATSClient
	handlers   map[string]string // 资源类型与处理器的映射
	role       string            // "api", "worker", "combined"
	apiBaseURL string
	bucket     string
	jobChan    chan ProcessJob // 任务队列 (本地模式使用)

	// 为了使 Scheduler 能回调写库，我们需要一个 Writer 的引用或接口
	// 由于 Go 循环依赖限制，且 Scheduler 主要是内部工作，我们在此直接使用 DB 更新
	// 或者通过 UseCase 注入。这里再次使用 data 直接访问数据库。
	// 但 notifyResult 需要 ReportProcessResult 逻辑。
	resultHandler func(ctx context.Context, versionID string, req ProcessResultRequest) error
}

func NewScheduler(d *data.Data, store storage.MultipartBlobStore, nats *data.NATSClient, bucket string, role, apiBaseURL string, handlers map[string]string) *Scheduler {
	s := &Scheduler{
		data:       d,
		store:      store,
		nats:       nats,
		bucket:     bucket,
		role:       role,
		apiBaseURL: apiBaseURL,
		handlers:   handlers,
		jobChan:    make(chan ProcessJob, 1000),
	}

	// auto start
	if role == "worker" || role == "combined" {
		if nats != nil && nats.Config.Enabled {
			// 分布式模式：启动 NATS 订阅者
			go s.startNATSSubscriber()
		} else {
			// 本地模式：启动内部 Worker
			for i := 0; i < 4; i++ {
				go s.startWorker(i)
			}
		}
	} else {
		slog.Info("当前节点为 API 模式，不启动本地任务执行器")
	}

	return s
}

func (s *Scheduler) SetResultHandler(handler func(ctx context.Context, versionID string, req ProcessResultRequest) error) {
	s.resultHandler = handler
}

func (s *Scheduler) Dispatch(job ProcessJob) {
	// ActionRefresh 需要数据库访问，强制在本地执行 (API 节点有 DB)
	if job.Action == ActionRefresh {
		go s.handleJob(context.Background(), job)
		return
	}

	if s.nats != nil && s.nats.Config.Enabled {
		if err := s.nats.Encoded.Publish(s.nats.Config.Subject, &job); err != nil {
			slog.Error("发送 NATS 消息失败，回退到本地队列", "error", err)
			s.jobChan <- job
		}
		return
	}
	s.jobChan <- job
}

func (s *Scheduler) startNATSSubscriber() {
	slog.Info("NATS 订阅者已启动", "subject", s.nats.Config.Subject)
	_, err := s.nats.Encoded.Subscribe(s.nats.Config.Subject, func(job *ProcessJob) {
		slog.Debug("接收到 NATS 任务", "action", job.Action, "key", job.ObjectKey)
		s.handleJob(context.Background(), *job)
	})
	if err != nil {
		slog.Error("NATS 订阅失败", "error", err)
	}
}

func (s *Scheduler) startWorker(id int) {
	slog.Info("本地 Worker 启动", "worker_id", id)
	for job := range s.jobChan {
		s.handleJob(context.Background(), job)
	}
}

func (s *Scheduler) handleJob(ctx context.Context, job ProcessJob) {
	switch job.Action {
	case ActionProcess:
		s.processResourceInternal(ctx, job.TypeKey, job.ObjectKey, job.VersionID)
	case ActionRefresh:
		s.syncSidecarInternal(ctx, job.ObjectKey, job.VersionID)
	}
}

// processResourceInternal 异步处理资源逻辑 (由 Worker 调用)
func (s *Scheduler) processResourceInternal(ctx context.Context, typeKey, objectKey, versionID string) {
	slog.Debug("开始处理资源", "key", objectKey, "type", typeKey, "role", s.role)

	// 1. 查询本地是否存在对应的处理器
	processorCmd := s.handlers[typeKey]

	finalMeta := make(map[string]any)
	if processorCmd != "" {
		// --- 真实执行逻辑 ---
		// 1. 下载文件到本地临时目录
		ext := ""
		if parts := strings.Split(objectKey, "."); len(parts) > 1 {
			ext = "." + parts[len(parts)-1]
		}

		tempFile, err := os.CreateTemp("", "simhub-resource-*"+ext)
		if err != nil {
			slog.Error("创建临时文件失败", "error", err)
			return // 应该上报 ERROR 状态
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		// 从 MinIO 下载
		obj, err := s.store.Get(ctx, s.bucket, objectKey)
		if err != nil {
			slog.Error("下载资源文件失败", "key", objectKey, "error", err)
			return
		}

		if _, err := io.Copy(tempFile, obj); err != nil {
			obj.Close()
			slog.Error("保存临时文件失败", "error", err)
			return
		}
		obj.Close()

		slog.Info("文件已下载至本地，准备处理", "path", tempFile.Name())

		// 2. 执行外部命令
		// 避免使用 sh -c 防止命令注入
		parts := strings.Fields(processorCmd)
		if len(parts) == 0 {
			slog.Error("处理器命令为空", "type", typeKey)
			return
		}

		head := parts[0]
		args := parts[1:]
		args = append(args, tempFile.Name())

		cmd := exec.CommandContext(ctx, head, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		slog.Debug("执行外部处理器", "cmd", cmd.String())
		startTime := time.Now()
		if err := cmd.Run(); err != nil {
			slog.Error("外部处理器执行失败", "error", err, "stderr", stderr.String())
			// 上报错误状态
			s.notifyResult(ctx, versionID, ProcessResultRequest{
				State:   "ERROR",
				Message: fmt.Sprintf("Processor failed: %v, stderr: %s", err, stderr.String()),
			})
			return
		}

		duration := time.Since(startTime)
		slog.Info("外部处理器执行完成", "duration", duration)

		// 3. 解析结果
		if err := json.Unmarshal(stdout.Bytes(), &finalMeta); err != nil {
			slog.Warn("处理器输出非 JSON 格式，忽略元数据", "output", stdout.String())
			finalMeta["raw_output"] = stdout.String()
		} else {
			// 追加系统级元数据
			finalMeta["processed_by"] = "simhub-worker"
			finalMeta["processed_at"] = time.Now().Format(time.RFC3339)
			finalMeta["processor_duration_ms"] = duration.Milliseconds()
		}
	} else {
		slog.Debug("未配置该类型的处理器，跳过计算", "type", typeKey)
		finalMeta["status"] = "skipped"
	}

	// 2. 上报结果
	err := s.notifyResult(ctx, versionID, ProcessResultRequest{
		MetaData: finalMeta,
		State:    "ACTIVE",
	})

	if err != nil {
		slog.Error("处理结果上报失败", "error", err)
	} else {
		slog.Debug("资源处理结果已成功同步", "key", objectKey)
	}
}

// syncSidecarInternal 仅执行元数据同步到存储 (不涉及外部 Processor)
func (s *Scheduler) syncSidecarInternal(ctx context.Context, objectKey, versionID string) {
	var ver model.ResourceVersion
	var res model.Resource
	if err := s.data.DB.Preload("Resource").First(&ver, "id = ?", versionID).Error; err != nil {
		slog.Error("同步 Sidecar 时找不到版本记录", "id", versionID, "error", err)
		return
	}
	res = ver.Resource

	sidecarKey := objectKey + ".meta.json"
	sidecarData := map[string]any{
		"resource_id":   res.ID,
		"resource_name": res.Name,
		"tags":          res.Tags,
		"version_id":    ver.ID,
		"type_key":      res.TypeKey,
		"metadata":      ver.MetaData,
		"synced_at":     time.Now().Format(time.RFC3339),
	}

	if sidecarBytes, err := json.Marshal(sidecarData); err == nil {
		if err := s.store.Put(ctx, s.bucket, sidecarKey, bytes.NewReader(sidecarBytes), int64(len(sidecarBytes)), "application/json"); err != nil {
			slog.Error("更新 Sidecar 失败", "key", sidecarKey, "error", err)
		} else {
			slog.Debug("Sidecar 刷新成功", "key", sidecarKey)
		}
	}
}

// notifyResult 根据节点角色选择上报方式（直接写库或通过 HTTP API）
func (s *Scheduler) notifyResult(ctx context.Context, versionID string, req ProcessResultRequest) error {
	if s.role == "api" || s.role == "combined" {
		// 本地模式：直接调用内部方法写库
		if s.resultHandler != nil {
			return s.resultHandler(ctx, versionID, req)
		}
		return fmt.Errorf("resultHandler not set for local mode")
	}

	// 远程 Worker 模式：通过 HTTP Callback 上报给 API 节点
	callbackURL := fmt.Sprintf("%s/api/v1/resources/%s/process-result", s.apiBaseURL, versionID)
	body, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(ctx, "PATCH", callbackURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("callback failed with status: %d", resp.StatusCode)
	}

	return nil
}
