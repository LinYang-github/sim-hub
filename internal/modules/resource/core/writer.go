package core

import (
	"context"
	"log/slog"
	"strings"

	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/internal/model"
	"github.com/liny/sim-hub/pkg/storage"
	"gorm.io/gorm"
)

type ResourceWriter struct {
	data       *data.Data
	store      storage.MultipartBlobStore
	bucket     string
	dispatcher JobDispatcher
	handlers   map[string]string // 为了判断是否需要调度任务
}

func NewResourceWriter(d *data.Data, store storage.MultipartBlobStore, bucket string, dispatcher JobDispatcher, handlers map[string]string) *ResourceWriter {
	return &ResourceWriter{
		data:       d,
		store:      store,
		bucket:     bucket,
		dispatcher: dispatcher,
		handlers:   handlers,
	}
}

// CreateCategory 创建分类
func (w *ResourceWriter) CreateCategory(ctx context.Context, req CreateCategoryRequest) (*CategoryDTO, error) {
	cat := model.Category{
		TypeKey:  req.TypeKey,
		Name:     req.Name,
		ParentID: req.ParentID,
	}
	if err := w.data.DB.Create(&cat).Error; err != nil {
		return nil, err
	}
	return &CategoryDTO{ID: cat.ID, Name: cat.Name, ParentID: cat.ParentID}, nil
}

// DeleteCategory 删除分类
func (w *ResourceWriter) DeleteCategory(ctx context.Context, id string) error {
	return w.data.DB.Delete(&model.Category{}, "id = ?", id).Error
}

// DeleteResource 删除资源
func (w *ResourceWriter) DeleteResource(ctx context.Context, id string) error {
	// 软删除
	return w.data.DB.Model(&model.Resource{}).Where("id = ?", id).Update("is_deleted", true).Error
}

// UpdateResourceTags 更新资源标签 并同步刷新 Sidecar
func (w *ResourceWriter) UpdateResourceTags(ctx context.Context, id string, tags []string) error {
	return w.data.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Resource{}).Where("id = ?", id).Select("Tags").Updates(model.Resource{Tags: tags}).Error; err != nil {
			return err
		}

		// 触发异步刷新 Sidecar (获取最新版本)
		var v model.ResourceVersion
		if err := tx.Order("version_num desc").First(&v, "resource_id = ?", id).Error; err == nil {
			w.dispatcher.Dispatch(ProcessJob{
				Action:    ActionRefresh,
				ObjectKey: v.FilePath,
				VersionID: v.ID,
			})
		}
		return nil
	})
}

// UpdateResourceScope 更新资源作用域 (公开/私有)
func (w *ResourceWriter) UpdateResourceScope(ctx context.Context, id string, scope string) error {
	return w.data.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Resource{}).Where("id = ?", id).Update("scope", scope).Error; err != nil {
			return err
		}

		// 触发异步刷新 Sidecar
		var v model.ResourceVersion
		if err := tx.Order("version_num desc").First(&v, "resource_id = ?", id).Error; err == nil {
			w.dispatcher.Dispatch(ProcessJob{
				Action:    ActionRefresh,
				ObjectKey: v.FilePath,
				VersionID: v.ID,
			})
		}
		return nil
	})
}

// SetResourceLatestVersion 回滚/设置最新版本
func (w *ResourceWriter) SetResourceLatestVersion(ctx context.Context, resourceID string, versionID string) error {
	// 这个逻辑待定，可能需要更新 Resource 表指向 LatestVersionID，或者只是插入一个新版本作为 Copy？
	// 暂时空实现或简单的 Log
	slog.Info("Set latest version not fully implemented", "resource", resourceID, "ver", versionID)
	return nil
}

// CreateResourceAndVersion 核心注册逻辑
func (w *ResourceWriter) CreateResourceAndVersion(tx *gorm.DB, typeKey, categoryID, name, ownerID, scope, objectKey string, size int64, tags []string, semver string, deps []DependencyDTO, meta map[string]any) error {
	if tx == nil {
		tx = w.data.DB
	}

	if scope == "" {
		scope = "PRIVATE"
	}

	// 1. 查找或创建资源主体
	var res model.Resource
	err := tx.Where("type_key = ? AND name = ? AND owner_id = ? AND is_deleted = ?", typeKey, name, ownerID, false).First(&res).Error

	if err == gorm.ErrRecordNotFound {
		res = model.Resource{
			TypeKey:    typeKey,
			CategoryID: categoryID,
			Name:       name,
			OwnerID:    ownerID,
			Scope:      scope,
			Tags:       tags,
		}
		if err := tx.Create(&res).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		// 如果资源已存在，更新其标签和分类（可选）
		tx.Model(&res).Updates(map[string]any{
			"category_id": categoryID,
			"tags":        tags,
			"scope":       scope,
		})
	}

	// 2. 确定版本号
	var lastVer int
	// 注意：这里如果在 Transaction 中，应该能读到？需注意隔离级别
	// 简单起见，假设 version_num 自增
	var count int64
	tx.Model(&model.ResourceVersion{}).Where("resource_id = ?", res.ID).Count(&count)
	lastVer = int(count)
	currentVer := lastVer + 1

	// 3. 决定初始状态
	initialState := "PENDING"
	hasHandler := w.handlers[typeKey] != ""
	if !hasHandler {
		initialState = "ACTIVE"
	}

	// 4. 创建版本
	ver := model.ResourceVersion{
		ResourceID: res.ID,
		VersionNum: currentVer,
		SemVer:     semver,
		FilePath:   objectKey,
		FileSize:   size,
		MetaData:   meta,
		State:      initialState,
	}
	if err := tx.Create(&ver).Error; err != nil {
		return err
	}

	// 5. 处理依赖关系
	for _, d := range deps {
		dependency := model.ResourceDependency{
			SourceVersionID:  ver.ID,
			TargetResourceID: d.TargetResourceID,
			Constraint:       d.Constraint,
		}
		tx.Create(&dependency)
	}

	// 6. 只有在有处理器的情况下才触发异步处理
	if hasHandler {
		w.dispatcher.Dispatch(ProcessJob{
			Action:    ActionProcess,
			TypeKey:   typeKey,
			ObjectKey: objectKey,
			VersionID: ver.ID,
		})
	} else {
		slog.Info("资源类型无需后端处理，跳过 NATS 任务分发", "type", typeKey, "name", name)
	}
	return nil
}

// SyncFromStorage 从存储扫描并同步资源到数据库
func (w *ResourceWriter) SyncFromStorage(ctx context.Context) (int, error) {
	// 注意：这里需要调用 w.store.ListObjects，但这通常是 Reader 的工作？
	// 或者 Sync 作为一个单独的功能。
	// 为简化，这里保留逻辑。

	objectCh := w.store.ListObjects(ctx, w.bucket, "resources/", true)
	syncedCount := 0

	// 由于 ListObjects 是同步阻塞或 Channel，我们在外部循环
	for object := range objectCh {
		if strings.HasSuffix(object.Key, ".meta.json") {
			continue
		}

		slashParts := strings.Split(object.Key, "/")
		if len(slashParts) < 4 {
			continue
		}
		// typeKey := slashParts[1]
		// resourceID := slashParts[2] // 这里其实是 UUID，不是 DB ID，逻辑需调整
		// fileName := slashParts[3]

		// 暂不完整实现 Sync 逻辑迁移，保持原有功能的骨架
		syncedCount++
	}

	return syncedCount, nil
}

// ReportProcessResult 更新处理结果
func (w *ResourceWriter) ReportProcessResult(ctx context.Context, versionID string, req ProcessResultRequest) error {
	return w.data.DB.Transaction(func(tx *gorm.DB) error {
		var ver model.ResourceVersion
		if err := tx.First(&ver, "id = ?", versionID).Error; err != nil {
			return err
		}

		// Merge metadata
		if ver.MetaData == nil {
			ver.MetaData = make(map[string]any)
		}
		for k, v := range req.MetaData {
			ver.MetaData[k] = v
		}

		updates := map[string]any{
			"state":     req.State,
			"meta_data": ver.MetaData,
		}
		if err := tx.Model(&ver).Updates(updates).Error; err != nil {
			return err
		}

		// Sidecar update is triggered via UseCase/Scheduler usually?
		// Or we trigger it here.
		if req.State == "ACTIVE" {
			w.dispatcher.Dispatch(ProcessJob{
				Action:    ActionRefresh,
				ObjectKey: ver.FilePath,
				VersionID: ver.ID,
			})
		}

		return nil
	})
}
