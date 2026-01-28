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
	"github.com/liny/sim-hub/pkg/storage"
)

// ResultEmitter 定义了 Worker 如何上报任务执行结果
type ResultEmitter interface {
	EmitResult(ctx context.Context, versionID string, result ProcessResultRequest) error
}

// Worker 负责具体任务的执行 (Processor 调用、下载、命令执行等)
type Worker struct {
	store      storage.MultipartBlobStore
	bucket     string
	handlers   map[string]string // 资源类型与处理器的映射
	emitter    ResultEmitter
	apiBaseURL string // 用于 HTTP Callback 回退
}

func NewWorker(store storage.MultipartBlobStore, bucket string, handlers map[string]string, emitter ResultEmitter, apiBaseURL string) *Worker {
	return &Worker{
		store:      store,
		bucket:     bucket,
		handlers:   handlers,
		emitter:    emitter,
		apiBaseURL: apiBaseURL,
	}
}

func (w *Worker) HandleJob(ctx context.Context, job ProcessJob) {
	switch job.Action {
	case ActionProcess:
		w.processResourceInternal(ctx, job.TypeKey, job.ObjectKey, job.VersionID)
	case ActionRefresh:
		// Refresh 通常由 API 节点本地执行，因为需要访问 DB 生成 Sidecar
		// 如果 Worker 连不上 DB，这个 Action 应该被限制运行或通过 API 转发
		slog.Warn("Worker 接收到 REFRESH 任务，但 Worker 通常不具备 DB 权限，请检查架构设计", "version", job.VersionID)
	}
}

func (w *Worker) processResourceInternal(ctx context.Context, typeKey, objectKey, versionID string) {
	slog.Log(ctx, slog.LevelDebug, "Worker 开始处理资源", "key", objectKey, "type", typeKey)

	processorCmd := w.handlers[typeKey]
	finalMeta := make(map[string]any)

	if processorCmd != "" {
		// 1. 下载文件到本地临时目录
		ext := ""
		if parts := strings.Split(objectKey, "."); len(parts) > 1 {
			ext = "." + parts[len(parts)-1]
		}

		tempFile, err := os.CreateTemp("", "simhub-resource-*"+ext)
		if err != nil {
			w.reportError(ctx, versionID, fmt.Errorf("failed to create temp file: %w", err))
			return
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		obj, err := w.store.Get(ctx, w.bucket, objectKey)
		if err != nil {
			w.reportError(ctx, versionID, fmt.Errorf("failed to download from store: %w", err))
			return
		}

		if _, err := io.Copy(tempFile, obj); err != nil {
			obj.Close()
			w.reportError(ctx, versionID, fmt.Errorf("failed to save temp file: %w", err))
			return
		}
		obj.Close()

		// 2. 执行外部命令
		parts := strings.Fields(processorCmd)
		if len(parts) == 0 {
			w.reportError(ctx, versionID, fmt.Errorf("processor command for %s is empty", typeKey))
			return
		}

		head := parts[0]
		args := parts[1:]
		args = append(args, tempFile.Name())

		cmd := exec.CommandContext(ctx, head, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		startTime := time.Now()
		if err := cmd.Run(); err != nil {
			w.reportError(ctx, versionID, fmt.Errorf("processor failed: %v, stderr: %s", err, stderr.String()))
			return
		}
		duration := time.Since(startTime)

		// 3. 解析结果
		if err := json.Unmarshal(stdout.Bytes(), &finalMeta); err != nil {
			slog.Log(ctx, slog.LevelWarn, "处理器输出非 JSON 格式，尝试记录原始输出", "output", stdout.String())
			finalMeta["raw_output"] = stdout.String()
		}
		finalMeta["processed_by"] = "simhub-worker"
		finalMeta["processed_at"] = time.Now().Format(time.RFC3339)
		finalMeta["processor_duration_ms"] = duration.Milliseconds()
	}

	// 上报成功结果
	w.emitter.EmitResult(ctx, versionID, ProcessResultRequest{
		MetaData: finalMeta,
		State:    "ACTIVE",
	})
}

func (w *Worker) reportError(ctx context.Context, versionID string, err error) {
	slog.Log(ctx, slog.LevelError, "任务执行失败", "version", versionID, "error", err)
	w.emitter.EmitResult(ctx, versionID, ProcessResultRequest{
		State:   "ERROR",
		Message: err.Error(),
	})
}

// HttpResultEmitter 实现了通过 HTTP API 上报结果 (旧模式兼容)
type HttpResultEmitter struct {
	BaseURL string
}

func (e *HttpResultEmitter) EmitResult(ctx context.Context, versionID string, result ProcessResultRequest) error {
	callbackURL := fmt.Sprintf("%s/api/v1/resources/%s/process-result", e.BaseURL, versionID)
	body, _ := json.Marshal(result)

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

// NatsResultEmitter 实现了通过 NATS 上报结果 (推荐模式)
type NatsResultEmitter struct {
	Nats *data.NATSClient
}

func (e *NatsResultEmitter) EmitResult(ctx context.Context, versionID string, result ProcessResultRequest) error {
	if e.Nats == nil || !e.Nats.Config.Enabled {
		return fmt.Errorf("nats not enabled")
	}

	// 包裹一层，带上 ID
	payload := struct {
		VersionID string               `json:"version_id"`
		Result    ProcessResultRequest `json:"result"`
	}{
		VersionID: versionID,
		Result:    result,
	}

	subject := "simhub.results.resource"
	return e.Nats.Encoded.Publish(subject, &payload)
}
