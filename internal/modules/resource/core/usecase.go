package core

import (
	"context"
	"io"

	"sim-hub/internal/data"
	"sim-hub/internal/model"
	"sim-hub/pkg/storage"
)

type UseCase struct {
	scheduler *Scheduler
	reader    *ResourceReader
	writer    *ResourceWriter
	uploader  *UploadManager
}

func NewUseCase(d *data.Data, store storage.MultipartBlobStore, stsProvider storage.SecurityTokenProvider, bucket string, natsClient *data.NATSClient, role string, apiBaseURL string, handlers map[string]string) *UseCase {
	// 1. 初始化事件发射器 (用于业务事件推送)
	emitter := NewEventEmitter(natsClient)

	// 2. 初始化核心 Writer (暂时不传 Scheduler，因为它依赖结果上报链)
	writer := NewResourceWriter(d, store, bucket, nil, emitter, handlers)

	// 3. 配置结果上报链 (Worker -> API)
	var resEmitter ResultEmitter
	if natsClient != nil && natsClient.Config.Enabled {
		// 分布式模式：结果通过 NATS 上报
		resEmitter = &NatsResultEmitter{Nats: natsClient}

		// 如果节点具有 API 职责，启动监听器来处理结果并更新 DB
		if role == "api" || role == "combined" {
			watcher := NewResultWatcher(natsClient, writer.ReportProcessResult)
			watcher.Start()
		}
	} else {
		// 单机模式：直接在内存中回调 Writer
		resEmitter = &LocalResultEmitter{ResultHandler: writer.ReportProcessResult}
	}

	// 4. 初始化 Worker (执行器)
	var worker *Worker
	if role == "worker" || role == "combined" {
		worker = NewWorker(store, bucket, handlers, resEmitter, apiBaseURL)
	}

	// 5. 初始化 Scheduler (分发器)
	scheduler := NewScheduler(d, store, natsClient, bucket, role, worker)

	// 6. 完善 Writer：注入分发器实现环路闭合 (接口解耦)
	writer.SetDispatcher(scheduler)

	// 7. 初始化 Reader
	reader := NewResourceReader(d, store, bucket, natsClient)

	// 8. 初始化 Uploader
	uploader := NewUploadManager(d, store, stsProvider, bucket, writer)

	return &UseCase{
		scheduler: scheduler,
		reader:    reader,
		writer:    writer,
		uploader:  uploader,
	}
}

// --- Delegate Methods ---

// Uploader Delegates
func (uc *UseCase) RequestUploadToken(ctx context.Context, req ApplyUploadTokenRequest) (*UploadTicket, error) {
	return uc.uploader.RequestUploadToken(ctx, req)
}

func (uc *UseCase) ConfirmUpload(ctx context.Context, req ConfirmUploadRequest) error {
	return uc.uploader.ConfirmUpload(ctx, req)
}

func (uc *UseCase) InitMultipartUpload(ctx context.Context, req InitMultipartUploadRequest) (*InitMultipartUploadResponse, error) {
	return uc.uploader.InitMultipartUpload(ctx, req)
}

func (uc *UseCase) GetMultipartUploadPartURL(ctx context.Context, req GetPartURLRequest) (*GetPartURLResponse, error) {
	return uc.uploader.GetMultipartUploadPartURL(ctx, req)
}

func (uc *UseCase) CompleteMultipartUpload(ctx context.Context, req CompleteMultipartUploadRequest) error {
	return uc.uploader.CompleteMultipartUpload(ctx, req)
}

// Reader Delegates
func (uc *UseCase) GetResource(ctx context.Context, id string) (*ResourceDTO, error) {
	return uc.reader.GetResource(ctx, id)
}

func (uc *UseCase) ListResources(ctx context.Context, typeKey string, categoryID string, ownerID string, scope string, keyword string, page, size int) ([]*ResourceDTO, int64, error) {
	return uc.reader.ListResources(ctx, typeKey, categoryID, ownerID, scope, keyword, page, size)
}

func (uc *UseCase) GetDashboardStats(ctx context.Context, ownerID string) (*DashboardStatsDTO, error) {
	return uc.reader.GetDashboardStats(ctx, ownerID)
}

func (uc *UseCase) ListResourceTypes(ctx context.Context) ([]model.ResourceType, error) {
	return uc.reader.GetResourceTypes(ctx)
}

func (uc *UseCase) CreateCategory(ctx context.Context, req CreateCategoryRequest) (*CategoryDTO, error) {
	return uc.writer.CreateCategory(ctx, req)
}

func (uc *UseCase) ListCategories(ctx context.Context, typeKey string) ([]*CategoryDTO, error) {
	return uc.reader.ListCategories(ctx, typeKey)
}

func (uc *UseCase) DeleteCategory(ctx context.Context, id string) error {
	return uc.writer.DeleteCategory(ctx, id)
}

func (uc *UseCase) UpdateCategory(ctx context.Context, id string, req UpdateCategoryRequest) error {
	return uc.writer.UpdateCategory(ctx, id, req)
}

func (uc *UseCase) CreateResourceFromData(ctx context.Context, req CreateResourceFromDataRequest) (*ResourceDTO, error) {
	return uc.writer.CreateResourceFromData(ctx, req)
}

func (uc *UseCase) UpdateResourceTags(ctx context.Context, id string, tags []string) error {
	return uc.writer.UpdateResourceTags(ctx, id, tags)
}

func (uc *UseCase) UpdateResourceScope(ctx context.Context, id string, scope string) error {
	return uc.writer.UpdateResourceScope(ctx, id, scope)
}

func (uc *UseCase) UpdateResource(ctx context.Context, id string, req UpdateResourceRequest) error {
	return uc.writer.UpdateResource(ctx, id, req)
}

func (uc *UseCase) UpdateVersionMetadata(ctx context.Context, versionID string, meta map[string]any) error {
	return uc.writer.UpdateVersionMetadata(ctx, versionID, meta)
}

func (uc *UseCase) SyncFromStorage(ctx context.Context) (int, error) {
	return uc.writer.SyncFromStorage(ctx)
}

func (uc *UseCase) DeleteResource(ctx context.Context, id string) error {
	return uc.writer.DeleteResource(ctx, id)
}

func (uc *UseCase) ClearResources(ctx context.Context, typeKey string) error {
	return uc.writer.ClearResources(ctx, typeKey)
}

func (uc *UseCase) GetResourceDependencies(ctx context.Context, vid string) ([]DependencyDTO, error) {
	return uc.reader.GetResourceDependencies(ctx, vid)
}

func (uc *UseCase) GetDependencyTree(ctx context.Context, vid string) ([]map[string]any, error) {
	return uc.reader.GetDependencyTree(ctx, vid)
}

func (uc *UseCase) UpdateResourceDependencies(ctx context.Context, vid string, deps []DependencyDTO) error {
	return uc.writer.UpdateResourceDependencies(ctx, vid, deps)
}

func (uc *UseCase) ListResourceVersions(ctx context.Context, id string) ([]*ResourceVersionDTO, error) {
	return uc.reader.ListResourceVersions(ctx, id)
}

func (uc *UseCase) SetResourceLatestVersion(ctx context.Context, id, versionID string) error {
	return uc.writer.SetResourceLatestVersion(ctx, id, versionID)
}

func (uc *UseCase) GetResourceBundle(ctx context.Context, vid string) ([]map[string]any, error) {
	return uc.reader.GetResourceBundle(ctx, vid)
}

func (uc *UseCase) DownloadBundleZip(ctx context.Context, vid string, w io.Writer) error {
	return uc.reader.DownloadBundleZip(ctx, vid, w)
}

func (uc *UseCase) ReportProcessResult(ctx context.Context, id string, req ProcessResultRequest) error {
	return uc.writer.ReportProcessResult(ctx, id, req)
}
