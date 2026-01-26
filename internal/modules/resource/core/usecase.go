package core

import (
	"context"
	"io"

	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/pkg/storage"
)

type UseCase struct {
	scheduler *Scheduler
	reader    *ResourceReader
	writer    *ResourceWriter
	uploader  *UploadManager
}

func NewUseCase(d *data.Data, store storage.MultipartBlobStore, stsProvider storage.SecurityTokenProvider, bucket string, natsClient *data.NATSClient, role string, apiBaseURL string, handlers map[string]string) *UseCase {
	// 1. 初始化 Scheduler
	scheduler := NewScheduler(d, store, natsClient, bucket, role, apiBaseURL, handlers)

	// 2. 初始化 Writer (依赖 Scheduler)
	writer := NewResourceWriter(d, store, bucket, scheduler, handlers)

	// 3. 解决 Reciprocating Dependency: Scheduler 需要回调 Writer
	// (通过闭包或接口设置)
	scheduler.SetResultHandler(writer.ReportProcessResult)

	// 4. 初始化 Reader
	reader := NewResourceReader(d, store, bucket)

	// 5. 初始化 Uploader (依赖 Writer)
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

func (uc *UseCase) ListResources(ctx context.Context, typeKey string, categoryID string, ownerID string, scope string, page, size int) ([]*ResourceDTO, int64, error) {
	return uc.reader.ListResources(ctx, typeKey, categoryID, ownerID, scope, page, size)
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

func (uc *UseCase) UpdateResourceTags(ctx context.Context, id string, tags []string) error {
	return uc.writer.UpdateResourceTags(ctx, id, tags)
}

func (uc *UseCase) UpdateResourceScope(ctx context.Context, id string, scope string) error {
	return uc.writer.UpdateResourceScope(ctx, id, scope)
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
	// 注意，这里可能需要转给 Writer
	return uc.writer.ReportProcessResult(ctx, id, req)
}
