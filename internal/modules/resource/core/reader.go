package core

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/internal/model"
	"github.com/liny/sim-hub/pkg/storage"
)

type ResourceReader struct {
	data   *data.Data
	store  storage.MultipartBlobStore
	bucket string
}

func NewResourceReader(d *data.Data, store storage.MultipartBlobStore, bucket string) *ResourceReader {
	return &ResourceReader{
		data:   d,
		store:  store,
		bucket: bucket,
	}
}

// GetResource 获取资源详情
func (r *ResourceReader) GetResource(ctx context.Context, id string) (*ResourceDTO, error) {
	var res model.Resource
	if err := r.data.DB.First(&res, "id = ?", id).Error; err != nil {
		return nil, err
	}

	// 获取当前指定的最新版本
	var v model.ResourceVersion
	if res.LatestVersionID != "" {
		if err := r.data.DB.First(&v, "id = ?", res.LatestVersionID).Error; err != nil {
			// 如果指针失效，退回到最高版本号
			r.data.DB.Order("version_num desc").First(&v, "resource_id = ?", id)
		}
	} else {
		// 默认取最高版本号
		if err := r.data.DB.Order("version_num desc").First(&v, "resource_id = ?", id).Error; err != nil {
			return nil, err
		}
	}

	var url string
	var err error
	if r.store != nil {
		url, err = r.store.PresignGet(ctx, r.bucket, v.FilePath, time.Hour)
	}
	if err != nil {
		return nil, err
	}

	return &ResourceDTO{
		ID:         res.ID,
		TypeKey:    res.TypeKey,
		CategoryID: res.CategoryID,
		Name:       res.Name,
		OwnerID:    res.OwnerID,
		Scope:      res.Scope,
		Tags:       res.Tags,
		CreatedAt:  res.CreatedAt,
		LatestVer: &ResourceVersionDTO{
			ID:          v.ID,
			VersionNum:  v.VersionNum,
			SemVer:      v.SemVer,
			FileSize:    v.FileSize,
			MetaData:    v.MetaData,
			State:       v.State,
			DownloadURL: url,
		},
	}, nil
}

// ListResources 列出资源
func (r *ResourceReader) ListResources(ctx context.Context, typeKey string, categoryID string, ownerID string, scope string, keyword string, page, size int) ([]*ResourceDTO, int64, error) {
	var resources []model.Resource
	var total int64
	offset := (page - 1) * size

	query := r.data.DB.Model(&model.Resource{}).Where("is_deleted = ?", false)
	if typeKey != "" {
		query = query.Where("type_key = ?", typeKey)
	}
	if categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}
	if keyword != "" {
		// 增强搜索：匹配名称或标签 (JSON 字段通过 LIKE 简单搜索，适用于标签数组序列化后的字符串)
		query = query.Where("name LIKE ? OR tags LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 作用域逻辑
	if scope == "PUBLIC" {
		query = query.Where("scope = ?", "PUBLIC")
	} else if scope == "PRIVATE" {
		query = query.Where("scope = ? AND owner_id = ?", "PRIVATE", ownerID)
	} else if ownerID != "" {
		query = query.Where("scope = ? OR (scope = ? AND owner_id = ?)", "PUBLIC", "PRIVATE", ownerID)
	}

	if err := query.Count(&total).Limit(size).Offset(offset).Order("created_at desc").Find(&resources).Error; err != nil {
		return nil, 0, err
	}

	cw := make([]*ResourceDTO, 0, len(resources))
	for _, res := range resources {
		// 获取当前指向的最新版本
		var v model.ResourceVersion
		if res.LatestVersionID != "" {
			if err := r.data.DB.First(&v, "id = ?", res.LatestVersionID).Error; err != nil {
				r.data.DB.Order("version_num desc").First(&v, "resource_id = ?", res.ID)
			}
		} else {
			r.data.DB.Order("version_num desc").First(&v, "resource_id = ?", res.ID)
		}

		dv := &ResourceVersionDTO{
			ID:         v.ID,
			VersionNum: v.VersionNum,
			SemVer:     v.SemVer,
			FileSize:   v.FileSize,
			State:      v.State,
			MetaData:   v.MetaData,
		}
		if v.State == "ACTIVE" && r.store != nil {
			url, _ := r.store.PresignGet(ctx, r.bucket, v.FilePath, time.Hour)
			dv.DownloadURL = url
		}

		cw = append(cw, &ResourceDTO{
			ID:         res.ID,
			TypeKey:    res.TypeKey,
			CategoryID: res.CategoryID,
			Name:       res.Name,
			OwnerID:    res.OwnerID,
			Scope:      res.Scope,
			Tags:       res.Tags,
			CreatedAt:  res.CreatedAt,
			LatestVer:  dv,
		})
	}
	return cw, total, nil
}

// ListCategories 列出分类
func (r *ResourceReader) ListCategories(ctx context.Context, typeKey string) ([]*CategoryDTO, error) {
	var cats []model.Category
	if err := r.data.DB.Where("type_key = ?", typeKey).Find(&cats).Error; err != nil {
		return nil, err
	}

	res := make([]*CategoryDTO, 0, len(cats))
	for _, c := range cats {
		res = append(res, &CategoryDTO{ID: c.ID, Name: c.Name, ParentID: c.ParentID})
	}
	return res, nil
}

// ListResourceVersions 列出版本
func (r *ResourceReader) ListResourceVersions(ctx context.Context, resourceID string) ([]*ResourceVersionDTO, error) {
	var versions []model.ResourceVersion
	if err := r.data.DB.Where("resource_id = ?", resourceID).Order("version_num desc").Find(&versions).Error; err != nil {
		return nil, err
	}

	res := make([]*ResourceVersionDTO, 0, len(versions))
	for _, v := range versions {
		var url string
		if r.store != nil {
			url, _ = r.store.PresignGet(ctx, r.bucket, v.FilePath, time.Hour)
		}
		res = append(res, &ResourceVersionDTO{
			ID:          v.ID,
			VersionNum:  v.VersionNum,
			SemVer:      v.SemVer,
			FileSize:    v.FileSize,
			MetaData:    v.MetaData,
			State:       v.State,
			DownloadURL: url,
		})
	}
	return res, nil
}

// GetResourceDependencies 获取资源依赖
func (r *ResourceReader) GetResourceDependencies(ctx context.Context, versionID string) ([]DependencyDTO, error) {
	var deps []model.ResourceDependency
	if err := r.data.DB.Where("source_version_id = ?", versionID).Find(&deps).Error; err != nil {
		return nil, err
	}

	res := make([]DependencyDTO, 0, len(deps))
	for _, d := range deps {
		res = append(res, DependencyDTO{
			TargetResourceID: d.TargetResourceID,
			Constraint:       d.Constraint,
		})
	}
	return res, nil
}

// GetDependencyTree 获取依赖树（递归）
func (r *ResourceReader) GetDependencyTree(ctx context.Context, versionID string) ([]map[string]any, error) {
	// 递归查询，这里使用简化的递归逻辑
	return r.resolveDependencyTree(ctx, versionID, make(map[string]bool))
}

func (r *ResourceReader) resolveDependencyTree(ctx context.Context, versionID string, visited map[string]bool) ([]map[string]any, error) {
	if visited[versionID] {
		return nil, nil // 避免循环依赖
	}
	visited[versionID] = true

	var deps []model.ResourceDependency
	if err := r.data.DB.Where("source_version_id = ?", versionID).Find(&deps).Error; err != nil {
		return nil, err
	}

	tree := make([]map[string]any, 0, len(deps))
	for _, d := range deps {
		// 查找目标资源的最新版本（根据约束，这里简化为 Latest）
		var targetVer model.ResourceVersion
		// 这里应该解析 Constraint，目前假设总是 Latest
		if err := r.data.DB.Where("resource_id = ?", d.TargetResourceID).Order("version_num desc").First(&targetVer).Error; err != nil {
			continue
		}

		var targetRes model.Resource
		r.data.DB.First(&targetRes, "id = ?", d.TargetResourceID)

		subDeps, _ := r.resolveDependencyTree(ctx, targetVer.ID, visited)

		node := map[string]any{
			"resource_id":   d.TargetResourceID,
			"resource_name": targetRes.Name,
			"type_key":      targetRes.TypeKey,
			"version_id":    targetVer.ID,
			"semver":        targetVer.SemVer,
			"dependencies":  subDeps,
		}
		tree = append(tree, node)
	}

	visited[versionID] = false // 回溯
	return tree, nil
}

// GetResourceBundle 获取打包列表
func (r *ResourceReader) GetResourceBundle(ctx context.Context, versionID string) ([]map[string]any, error) {
	flatList := make(map[string]map[string]any)
	visited := make(map[string]bool)

	if err := r.collectBundleFiles(ctx, versionID, flatList, visited); err != nil {
		return nil, err
	}

	// Convert map to slice
	result := make([]map[string]any, 0, len(flatList))
	for _, item := range flatList {
		result = append(result, item)
	}
	return result, nil
}

func (r *ResourceReader) collectBundleFiles(ctx context.Context, versionID string, flatList map[string]map[string]any, visited map[string]bool) error {
	if visited[versionID] {
		return nil
	}
	visited[versionID] = true

	var ver model.ResourceVersion
	if err := r.data.DB.Preload("Resource").First(&ver, "id = ?", versionID).Error; err != nil {
		return err
	}

	var url string
	if r.store != nil {
		url, _ = r.store.PresignGet(ctx, r.bucket, ver.FilePath, time.Hour*24)
	}

	flatList[versionID] = map[string]any{
		"resource_id":   ver.ResourceID,
		"resource_name": ver.Resource.Name,
		"type_key":      ver.Resource.TypeKey,
		"version_id":    ver.ID,
		"semver":        ver.SemVer,
		"file_path":     ver.FilePath, // Internal MinIO path
		"download_url":  url,
		"size":          ver.FileSize,
	}

	// Recursive dependencies
	var deps []model.ResourceDependency
	if err := r.data.DB.Where("source_version_id = ?", versionID).Find(&deps).Error; err != nil {
		return nil
	}

	for _, d := range deps {
		var targetVer model.ResourceVersion
		// Find best matching version (Simplified: Latest)
		if err := r.data.DB.Order("version_num desc").First(&targetVer, "resource_id = ?", d.TargetResourceID).Error; err == nil {
			if err := r.collectBundleFiles(ctx, targetVer.ID, flatList, visited); err != nil {
				return err
			}
		}
	}
	return nil
}

// DownloadBundleZip 流式下载打包
func (r *ResourceReader) DownloadBundleZip(ctx context.Context, versionID string, w io.Writer) error {
	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	// 1. 获取所有文件清单 (Bundle)
	bundle, err := r.GetResourceBundle(ctx, versionID)
	if err != nil {
		return err
	}

	// 2. 生成并写入 manifest.json
	manifestBytes, _ := json.MarshalIndent(map[string]any{
		"root_version_id": versionID,
		"generated_at":    time.Now(),
		"items":           bundle,
	}, "", "  ")

	fManifest, err := zipWriter.Create("manifest.json")
	if err != nil {
		return err
	}
	if _, err := fManifest.Write(manifestBytes); err != nil {
		return err
	}

	// 3. 遍历并流式写入文件
	for _, item := range bundle {
		// 构造 ZIP 内的友好路径: resources/<type>/<name>-<semver>/<filename>
		// 例如: resources/model/Tank-v1.0.0/tank.glb
		// 注意: FilePath 可能是 "resources/model/uuid/filename.ext"
		rawPath := item["file_path"].(string)
		fileName := rawPath // fallback
		if parts := strings.Split(rawPath, "/"); len(parts) > 0 {
			fileName = parts[len(parts)-1]
		}

		zipPath := fmt.Sprintf("resources/%s/%s-%s/%s",
			item["type_key"],
			item["resource_name"],
			item["semver"],
			fileName,
		)

		// 获取 MinIO 对象流
		if r.store == nil {
			return fmt.Errorf("storage service unavailable")
		}
		obj, err := r.store.Get(ctx, r.bucket, rawPath)
		if err != nil {
			slog.WarnContext(ctx, "跳过文件下载失败", "key", rawPath, "error", err)
			continue
		}

		f, err := zipWriter.Create(zipPath)
		if err != nil {
			obj.Close()
			return err
		}

		if _, err := io.Copy(f, obj); err != nil {
			obj.Close()
			return err
		}
		obj.Close()
	}

	return nil
}

// GetDashboardStats 获取概览统计数据
func (r *ResourceReader) GetDashboardStats(ctx context.Context, ownerID string) (*DashboardStatsDTO, error) {
	stats := &DashboardStatsDTO{
		TotalCounts: make(map[string]int64),
	}

	// 1. 获取各个类型的总量
	var typeCounts []struct {
		TypeKey string
		Count   int64
	}
	r.data.DB.Model(&model.Resource{}).
		Where("is_deleted = ?", false).
		Select("type_key, count(*) as count").
		Group("type_key").
		Scan(&typeCounts)

	for _, tc := range typeCounts {
		stats.TotalCounts[tc.TypeKey] = tc.Count
	}

	// 2. 获取最近上传的 10 个资源 (忽略作用域仅为演示逻辑，实际应过滤权限)
	recent, _, err := r.ListResources(ctx, "", "", ownerID, "ALL", "", 1, 10)
	if err == nil {
		stats.RecentItems = recent
	}

	return stats, nil
}
