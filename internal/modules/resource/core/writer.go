package core

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

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
	emitter    *EventEmitter
	handlers   map[string]string // 为了判断是否需要调度任务
}

func NewResourceWriter(d *data.Data, store storage.MultipartBlobStore, bucket string, dispatcher JobDispatcher, emitter *EventEmitter, handlers map[string]string) *ResourceWriter {
	return &ResourceWriter{
		data:       d,
		store:      store,
		bucket:     bucket,
		dispatcher: dispatcher,
		emitter:    emitter,
		handlers:   handlers,
	}
}

func (w *ResourceWriter) SetDispatcher(dispatcher JobDispatcher) {
	w.dispatcher = dispatcher
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

// UpdateCategory 更新分类
func (w *ResourceWriter) UpdateCategory(ctx context.Context, id string, req UpdateCategoryRequest) error {
	updates := make(map[string]any)
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.ParentID != nil {
		updates["parent_id"] = *req.ParentID
	}

	if len(updates) == 0 {
		return nil
	}

	return w.data.DB.Model(&model.Category{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteCategory 删除分类
func (w *ResourceWriter) DeleteCategory(ctx context.Context, id string) error {
	return w.data.DB.Delete(&model.Category{}, "id = ?", id).Error
}

func (w *ResourceWriter) DeleteResource(ctx context.Context, id string) error {
	// 软删除
	if err := w.data.DB.Model(&model.Resource{}).Where("id = ?", id).Update("is_deleted", true).Error; err != nil {
		return err
	}

	// 发送删除事件
	w.emitter.Emit(LifecycleEvent{
		Type:       EventResourceDeleted,
		ResourceID: id,
		Timestamp:  time.Now(),
	})
	return nil
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
			w.dispatcher.Dispatch(ctx, ProcessJob{
				Action:    ActionRefresh,
				ObjectKey: v.FilePath,
				VersionID: v.ID,
			})
		}
		return nil
	})
}

// ClearResources 清空指定类型的资源库
func (w *ResourceWriter) ClearResources(ctx context.Context, typeKey string) error {
	if typeKey == "" {
		return fmt.Errorf("type_key is required")
	}
	// 批量软删除
	return w.data.DB.Model(&model.Resource{}).Where("type_key = ?", typeKey).Update("is_deleted", true).Error
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
			w.dispatcher.Dispatch(ctx, ProcessJob{
				Action:    ActionRefresh,
				ObjectKey: v.FilePath,
				VersionID: v.ID,
			})
		}
		return nil
	})
}

// UpdateResourceDependencies 更新资源版本的依赖关联
func (w *ResourceWriter) UpdateResourceDependencies(ctx context.Context, versionID string, deps []DependencyDTO) error {
	return w.data.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 删除旧依赖
		if err := tx.Delete(&model.ResourceDependency{}, "source_version_id = ?", versionID).Error; err != nil {
			return err
		}

		// 2. 写入新依赖
		for _, d := range deps {
			dependency := model.ResourceDependency{
				SourceVersionID:  versionID,
				TargetResourceID: d.TargetResourceID,
				Constraint:       d.Constraint,
			}
			if err := tx.Create(&dependency).Error; err != nil {
				return err
			}
		}

		// 3. 触发异步刷新 Sidecar (以便下游感知依赖变化)
		var v model.ResourceVersion
		if err := tx.First(&v, "id = ?", versionID).Error; err == nil {
			w.dispatcher.Dispatch(ctx, ProcessJob{
				Action:    ActionRefresh,
				ObjectKey: v.FilePath,
				VersionID: v.ID,
			})
		}

		return nil
	})
}

// UpdateResource 更新资源基本信息 (更名、移动分类)
func (w *ResourceWriter) UpdateResource(ctx context.Context, id string, req UpdateResourceRequest) error {
	updates := make(map[string]any)
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.CategoryID != "" {
		updates["category_id"] = req.CategoryID
	}

	if len(updates) == 0 {
		return nil
	}

	return w.data.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Resource{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}

		// 触发异步刷新 Sidecar (获取最新版本)
		var v model.ResourceVersion
		if err := tx.Order("version_num desc").First(&v, "resource_id = ?", id).Error; err == nil {
			w.dispatcher.Dispatch(ctx, ProcessJob{
				Action:    ActionRefresh,
				ObjectKey: v.FilePath,
				VersionID: v.ID,
			})
		}

		// 发送资源更新事件
		if w.emitter != nil {
			w.emitter.Emit(LifecycleEvent{
				Type:       EventResourceUpdated,
				ResourceID: id,
				Timestamp:  time.Now(),
				Data:       updates,
			})
		}

		return nil
	})
}

// UpdateVersionMetadata 更新指定版本的元数据
func (w *ResourceWriter) UpdateVersionMetadata(ctx context.Context, versionID string, meta map[string]any) error {
	return w.data.DB.Transaction(func(tx *gorm.DB) error {
		var ver model.ResourceVersion
		if err := tx.First(&ver, "id = ?", versionID).Error; err != nil {
			return err
		}

		// 合并元数据
		if ver.MetaData == nil {
			ver.MetaData = make(map[string]any)
		}
		for k, v := range meta {
			ver.MetaData[k] = v
		}

		if err := tx.Model(&ver).Update("meta_data", ver.MetaData).Error; err != nil {
			return err
		}

		// 触发更新 Sidecar
		w.dispatcher.Dispatch(ctx, ProcessJob{
			Action:    ActionRefresh,
			ObjectKey: ver.FilePath,
			VersionID: ver.ID,
		})

		return nil
	})
}

// SetResourceLatestVersion 回滚/设置最新版本
// SetResourceLatestVersion 回滚/设置最新版本
func (w *ResourceWriter) SetResourceLatestVersion(ctx context.Context, resourceID string, versionID string) error {
	return w.data.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Verify existence and ownership
		var ver model.ResourceVersion
		if err := tx.First(&ver, "id = ? AND resource_id = ?", versionID, resourceID).Error; err != nil {
			return err
		}

		// 2. Ensure version is ACTIVE (optional constraint, but safer for "latest")
		// if ver.State != "ACTIVE" {
		// 	 return fmt.Errorf("cannot set non-active version as latest")
		// }

		// 3. Update Resource
		if err := tx.Model(&model.Resource{}).Where("id = ?", resourceID).Update("latest_version_id", versionID).Error; err != nil {
			return err
		}

		// 4. Trigger Sidecar Refresh
		// Only trigger if state is active, otherwise sidecar gen might fail or be empty?
		// Assuming we want to refresh anyway.
		w.dispatcher.Dispatch(ctx, ProcessJob{
			Action:    ActionRefresh,
			ObjectKey: ver.FilePath,
			VersionID: ver.ID,
		})

		return nil
	})
}

// CreateResourceAndVersion 核心注册逻辑
func (w *ResourceWriter) CreateResourceAndVersion(tx *gorm.DB, typeKey, categoryID, name, ownerID, scope, objectKey string, size int64, tags []string, semver string, deps []DependencyDTO, meta map[string]any) error {
	if tx == nil {
		tx = w.data.DB
	}

	if scope == "" {
		scope = "PRIVATE"
	}

	// 1. 查找或创建资源主体 (引入 category_id 实现目录命名空间隔离)
	var res model.Resource
	err := tx.Where("type_key = ? AND category_id = ? AND name = ? AND owner_id = ? AND is_deleted = ?", typeKey, categoryID, name, ownerID, false).First(&res).Error

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
		if err := tx.Model(&res).Updates(map[string]any{
			"category_id": categoryID,
			"tags":        tags,
			"scope":       scope,
		}).Error; err != nil {
			return err
		}
	}

	// 2. 检查版本是否存在
	var ver model.ResourceVersion
	err = tx.Where(&model.ResourceVersion{ResourceID: res.ID, SemVer: semver}).First(&ver).Error

	// 3. 决定初始状态
	initialState := "PENDING"
	hasHandler := w.handlers[typeKey] != ""
	if !hasHandler {
		initialState = "ACTIVE"
	}

	if err == nil {
		// 版本已存在
		if ver.State == "ACTIVE" {
			return fmt.Errorf("version %s already exists and is ACTIVE", semver)
		}
		// 允许覆盖 PENDING/ERROR 状态的版本
		ver.FilePath = objectKey
		ver.FileSize = size
		ver.MetaData = meta
		ver.State = initialState

		if err := tx.Save(&ver).Error; err != nil {
			return err
		}

		// 清理旧依赖
		if err := tx.Delete(&model.ResourceDependency{}, "source_version_id = ?", ver.ID).Error; err != nil {
			return err
		}
	} else if err == gorm.ErrRecordNotFound {
		// 创建新版本
		var count int64
		// 注意：这里的 VersionNum 仍然可能存在并发问题，但在同一资源下冲突概率较低
		tx.Model(&model.ResourceVersion{}).Where("resource_id = ?", res.ID).Count(&count)
		currentVer := int(count) + 1

		ver = model.ResourceVersion{
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

		// 如果资源还没有最新版本 ID，或者这是手动创建的第一个版本，设为最新
		if res.LatestVersionID == "" {
			tx.Model(&res).Update("latest_version_id", ver.ID)
		}
	} else {
		return err
	}

	// 5. 处理依赖关系
	for _, d := range deps {
		dependency := model.ResourceDependency{
			SourceVersionID:  ver.ID,
			TargetResourceID: d.TargetResourceID,
			Constraint:       d.Constraint,
		}
		if err := tx.Create(&dependency).Error; err != nil {
			return err
		}
	}

	// 6. 只有在有处理器的情况下才触发异步处理
	if hasHandler {
		w.dispatcher.Dispatch(tx.Statement.Context, ProcessJob{
			Action:    ActionProcess,
			TypeKey:   typeKey,
			ObjectKey: objectKey,
			VersionID: ver.ID,
		})
	} else {
		slog.InfoContext(tx.Statement.Context, "资源类型无需后端处理，跳过 NATS 任务分发", "type", typeKey, "name", name)

		// 发送版本就绪事件
		if w.emitter != nil {
			w.emitter.Emit(LifecycleEvent{
				Type:       EventVersionActivated,
				ResourceID: res.ID,
				VersionID:  ver.ID,
				Timestamp:  time.Now(),
				Data: map[string]any{
					"semver":    ver.SemVer,
					"file_path": ver.FilePath,
				},
			})
		}
	}

	// 如果是新资源，发送创建事件
	if res.CreatedAt.After(time.Now().Add(-5*time.Second)) && w.emitter != nil {
		w.emitter.Emit(LifecycleEvent{
			Type:       EventResourceCreated,
			ResourceID: res.ID,
			TypeKey:    res.TypeKey,
			Timestamp:  time.Now(),
			Data: map[string]any{
				"name": res.Name,
			},
		})
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
		// 修复：忽略目录（通常以 / 结尾且大小为 0）
		if strings.HasSuffix(object.Key, "/") && object.Size == 0 {
			continue
		}

		// Expected format: resources/<typeKey>/<resourceUUID>/<filename>
		// Example: resources/model_glb/3004b46b-8960-4e4a-bf9e-a4e7481f2b6f/sample_triangle.glb
		slashParts := strings.Split(object.Key, "/")
		if len(slashParts) < 4 {
			continue
		}

		typeKey := slashParts[1]
		resourceIDOrName := slashParts[2]         // This might be UUID or Name depending on how it was uploaded
		fileName := slashParts[len(slashParts)-1] // last part

		// Simple heuristic: if 3. part is UUID, use it as ID, else use it as Name
		// But current uploader uses UUID folder.
		// Let's assume folder name IS the Resource ID for now, or we treat it as "Imported-" + folder

		// For robustness: Auto-register
		err := w.data.DB.Transaction(func(tx *gorm.DB) error {
			var res model.Resource
			// Try to find by ID (if folder is UUID)
			if err := tx.First(&res, "id = ?", resourceIDOrName).Error; err != nil {
				// If not found by ID, create a new Resource
				// Use folder name as ID if it looks like UUID, otherwise generate new
				res = model.Resource{
					ID:        resourceIDOrName, // Assuming folder is stable ID
					TypeKey:   typeKey,
					Name:      fileName, // Use filename as resource name initially
					OwnerID:   "admin",  // Default owner
					Scope:     "PRIVATE",
					IsDeleted: false,
				}
				if len(resourceIDOrName) != 36 { // Not a UUID?
					// Fallback: create random ID, put folder name in Name?
					// Actually uploader enforces UUID. But if manual upload:
					res.ID = "" // Let GORM/BeforeCreate gen UUID
					res.Name = resourceIDOrName + "-" + fileName
				}

				if err := tx.Create(&res).Error; err != nil {
					// Fallback if ID conflict (unlikely if UUID)
					return err
				}
			}

			// Create Version
			var ver model.ResourceVersion
			// Check if this file path is already registered
			if err := tx.First(&ver, "file_path = ?", object.Key).Error; err == nil {
				return nil // Already synced
			}

			// Not found, insert version
			// Determine version num
			var count int64
			tx.Model(&model.ResourceVersion{}).Where("resource_id = ?", res.ID).Count(&count)

			newVer := model.ResourceVersion{
				ResourceID: res.ID,
				VersionNum: int(count) + 1,
				SemVer:     fmt.Sprintf("v1.0.%d", int(count)), // Dummy Semver
				FilePath:   object.Key,
				FileSize:   object.Size,
				State:      "ACTIVE", // Auto synced is active
				MetaData:   map[string]any{"imported": true},
			}
			if err := tx.Create(&newVer).Error; err != nil {
				return err
			}

			// 修复：更新资源的最新版本 ID，否则 UI 会显示为没有版本的资源
			if res.LatestVersionID == "" {
				return tx.Model(&res).Update("latest_version_id", newVer.ID).Error
			}
			return nil
		})

		if err == nil {
			syncedCount++
		}
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

		updates := model.ResourceVersion{
			State:    req.State,
			MetaData: ver.MetaData,
		}
		if err := tx.Model(&ver).Select("state", "meta_data").Updates(updates).Error; err != nil {
			return err
		}

		// Sidecar update is triggered via UseCase/Scheduler usually?
		// Or we trigger it here.
		if req.State == "ACTIVE" {
			w.dispatcher.Dispatch(ctx, ProcessJob{
				Action:    ActionRefresh,
				ObjectKey: ver.FilePath,
				VersionID: ver.ID,
			})

			// 发送版本就绪事件
			if w.emitter != nil {
				w.emitter.Emit(LifecycleEvent{
					Type:       EventVersionActivated,
					ResourceID: ver.ResourceID,
					VersionID:  ver.ID,
					Timestamp:  time.Now(),
					Data: map[string]any{
						"semver":    ver.SemVer,
						"file_path": ver.FilePath,
					},
				})
			}
		}

		return nil
	})
}
