package resource

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/liny/sim-hub/internal/conf"
	"github.com/liny/sim-hub/internal/core/module"
	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/internal/model"
	"github.com/liny/sim-hub/internal/modules/resource/core"
	"github.com/liny/sim-hub/pkg/storage"
)

// Module 实现了 module.Module 接口
type Module struct {
	uc          *core.UseCase
	orderedKeys []string
}

func NewModule(d *data.Data, store storage.MultipartBlobStore, stsProvider storage.SecurityTokenProvider, bucket string, natsClient *data.NATSClient, role string, apiBaseURL string, handlers map[string]string, resourceTypes []conf.ResourceType) module.Module {
	// 启动时同步资源类型定义到数据库
	if len(resourceTypes) > 0 {
		var types []model.ResourceType
		for _, rt := range resourceTypes {
			types = append(types, model.ResourceType{
				TypeKey:         rt.TypeKey,
				TypeName:        rt.TypeName,
				SchemaDef:       rt.SchemaDef,
				CategoryMode:    rt.CategoryMode,
				IntegrationMode: rt.IntegrationMode,
				UploadMode:      rt.UploadMode,
				ProcessConf:     rt.ProcessConf,
				MetaData:        rt.MetaData,
			})
		}
		// 1. Get all existing keys in DB
		var existingKeys []string
		d.DB.Model(&model.ResourceType{}).Pluck("type_key", &existingKeys)

		// 2. Identify keys to delete (present in DB but not in config)
		configKeys := make(map[string]bool)
		for _, t := range types {
			configKeys[t.TypeKey] = true
		}

		var keysToDelete []string
		for _, k := range existingKeys {
			if !configKeys[k] {
				keysToDelete = append(keysToDelete, k)
			}
		}

		// 3. Delete orphans
		if len(keysToDelete) > 0 {
			d.DB.Where("type_key IN ?", keysToDelete).Delete(&model.ResourceType{})
			slog.Info("Deleted orphaned resource types", "count", len(keysToDelete), "keys", keysToDelete)
		}

		// 4. Upsert config types
		if err := d.DB.Save(&types).Error; err != nil {
			slog.Error("Sync resource types failed", "error", err)
		} else {
			slog.Info("Synced resource types to DB", "count", len(types))
		}
	}

	orderedKeys := make([]string, 0, len(resourceTypes))
	for _, rt := range resourceTypes {
		orderedKeys = append(orderedKeys, rt.TypeKey)
	}

	return &Module{
		uc:          core.NewUseCase(d, store, stsProvider, bucket, natsClient, role, apiBaseURL, handlers),
		orderedKeys: orderedKeys,
	}
}

func (m *Module) RegisterRoutes(g *gin.RouterGroup) {
	// /api/v1/integration/upload/... 路径组
	integration := g.Group("/integration")
	{
		integration.POST("/upload/token", m.ApplyUploadToken)
		integration.POST("/upload/confirm", m.ConfirmUpload)

		// 分片上传子路由
		integration.POST("/upload/multipart/init", m.InitMultipartUpload)
		integration.POST("/upload/multipart/part-url", m.GetMultipartUploadPartURL)
		integration.POST("/upload/multipart/complete", m.CompleteMultipartUpload)
	}

	// /api/v1/resource-types
	rTypes := g.Group("/resource-types")
	{
		rTypes.GET("", m.ListResourceTypes)
	}

	// /api/v1/resources 路径组
	resources := g.Group("/resources")
	{
		resources.GET("", m.ListResources)
		resources.POST("/sync", m.SyncFromStorage)          // 新增：同步存储
		resources.POST("/clear", m.ClearResources)          // 新增：清空资源库
		resources.POST("/create", m.CreateResourceFromData) // 新增：在线创建接口
		resources.GET("/:id", m.GetResource)
		resources.PATCH("/:id", m.UpdateResource) // 新增：更新基本信息 (更名/移动)
		resources.DELETE("/:id", m.DeleteResource)
		resources.PATCH("/:id/tags", m.UpdateResourceTags)
		resources.PATCH("/:id/scope", m.UpdateResourceScope) // 新增：更新作用域
		resources.PATCH("/:id/process-result", m.ReportProcessResult)

		// 新增：版本历史与依赖管理
		resources.GET("/:id/versions", m.ListVersions)
		resources.POST("/:id/latest", m.SetLatestVersion)
		resources.GET("/versions/:vid/dependencies", m.GetDependencies)
		resources.PATCH("/versions/:vid/dependencies", m.UpdateResourceDependencies)
		resources.GET("/versions/:vid/dependency-tree", m.GetDependencyTree)
		resources.PATCH("/versions/:vid/meta", m.UpdateVersionMetadata) // 新增：更新版本元数据 (PATCH)
		resources.GET("/versions/:vid/bundle", m.GetBundle)

		// 新增：实时同步打包下载
		resources.GET("/versions/:vid/download-pack", m.DownloadBundle)
	}

	// /api/v1/categories 路径组
	categories := g.Group("/categories")
	{
		categories.GET("", m.ListCategories)
		categories.POST("", m.CreateCategory)
		categories.DELETE("/:id", m.DeleteCategory)
		categories.PATCH("/:id", m.UpdateCategory)
	}
}

// ApplyUploadToken 申请上传令牌
func (m *Module) ApplyUploadToken(c *gin.Context) {
	var req core.ApplyUploadTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ticket, err := m.uc.RequestUploadToken(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ticket)
}

// ConfirmUpload 确认上传已完成
func (m *Module) ConfirmUpload(c *gin.Context) {
	var req core.ConfirmUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := m.uc.ConfirmUpload(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Processing started"})
}

// InitMultipartUpload 初始化分片上传
func (m *Module) InitMultipartUpload(c *gin.Context) {
	var req core.InitMultipartUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := m.uc.InitMultipartUpload(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetMultipartUploadPartURL 获取分片 URL
func (m *Module) GetMultipartUploadPartURL(c *gin.Context) {
	var req core.GetPartURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := m.uc.GetMultipartUploadPartURL(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// CompleteMultipartUpload 完成分片上传
func (m *Module) CompleteMultipartUpload(c *gin.Context) {
	var req core.CompleteMultipartUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := m.uc.CompleteMultipartUpload(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Multipart upload completed and processing started"})
}

// GetResource 获取资源详情
func (m *Module) GetResource(c *gin.Context) {
	id := c.Param("id")
	res, err := m.uc.GetResource(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		return
	}
	c.JSON(http.StatusOK, res)
}

// ReportProcessResult 处理由外部 Worker 回填的结果
func (m *Module) ReportProcessResult(c *gin.Context) {
	id := c.Param("id")
	var req core.ProcessResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := m.uc.ReportProcessResult(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Result reported"})
}

// ListResources 列出资源
func (m *Module) ListResources(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	typeKey := c.Query("type")
	categoryID := c.Query("category_id")
	ownerID := c.Query("owner_id")
	scope := c.Query("scope")
	keyword := c.Query("query")

	list, total, err := m.uc.ListResources(c.Request.Context(), typeKey, categoryID, ownerID, scope, keyword, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items": list,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// CreateCategory 创建分类
func (m *Module) CreateCategory(c *gin.Context) {
	var req core.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := m.uc.CreateCategory(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

// ListCategories 列出分类
func (m *Module) ListCategories(c *gin.Context) {
	typeKey := c.Query("type")
	list, err := m.uc.ListCategories(c.Request.Context(), typeKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// DeleteCategory 删除分类
func (m *Module) DeleteCategory(c *gin.Context) {
	id := c.Param("id")
	if err := m.uc.DeleteCategory(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "Category deleted"})
}

// UpdateCategory 更新分类 (重命名/移动)
func (m *Module) UpdateCategory(c *gin.Context) {
	id := c.Param("id")
	var req core.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := m.uc.UpdateCategory(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Category updated"})
}

// UpdateResourceTags 更新资源标签
func (m *Module) UpdateResourceTags(c *gin.Context) {
	id := c.Param("id")
	var req core.UpdateResourceTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := m.uc.UpdateResourceTags(c.Request.Context(), id, req.Tags); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Tags updated"})
}

// UpdateResourceScope 更新资源作用域
func (m *Module) UpdateResourceScope(c *gin.Context) {
	id := c.Param("id")
	var req core.UpdateResourceScopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := m.uc.UpdateResourceScope(c.Request.Context(), id, req.Scope); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Scope updated"})
}

// SyncFromStorage 同步存储中的文件到数据库
func (m *Module) SyncFromStorage(c *gin.Context) {
	count, err := m.uc.SyncFromStorage(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Sync completed", "count": count})
}

// DeleteResource 删除资源
func (m *Module) DeleteResource(c *gin.Context) {
	id := c.Param("id")
	if err := m.uc.DeleteResource(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Resource deleted"})
}

// ClearResources 清空资源库
func (m *Module) ClearResources(c *gin.Context) {
	typeKey := c.Query("type")
	if err := m.uc.ClearResources(c.Request.Context(), typeKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Repository cleared"})
}

// GetDependencies 获取版本依赖
func (m *Module) GetDependencies(c *gin.Context) {
	vid := c.Param("vid")
	deps, err := m.uc.GetResourceDependencies(c.Request.Context(), vid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, deps)
}

// GetDependencyTree 获取依赖树
func (m *Module) GetDependencyTree(c *gin.Context) {
	vid := c.Param("vid")
	tree, err := m.uc.GetDependencyTree(c.Request.Context(), vid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tree)
}

// ListVersions 获取版本历史
func (m *Module) ListVersions(c *gin.Context) {
	id := c.Param("id")
	versions, err := m.uc.ListResourceVersions(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, versions)
}

// SetLatestVersion 设置主版本（回溯）
func (m *Module) SetLatestVersion(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		VersionID string `json:"version_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := m.uc.SetResourceLatestVersion(c.Request.Context(), id, req.VersionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Latest version updated"})
}

// GetBundle 获取一键打包清单
func (m *Module) GetBundle(c *gin.Context) {
	vid := c.Param("vid")
	bundle, err := m.uc.GetResourceBundle(c.Request.Context(), vid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bundle)
}

// DownloadBundle 实时流式下载 Zip 包
func (m *Module) DownloadBundle(c *gin.Context) {
	vid := c.Param("vid")

	// 1. 设置响应头，告诉浏览器这是一个文件下载
	// 这里可以先查一下资源名称来给文件命名
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"bundle-%s.simpack\"", vid))
	c.Header("Content-Type", "application/octet-stream")

	// 2. 调用 UseCase 直接将 Zip 数据流向 ResponseWriter
	err := m.uc.DownloadBundleZip(c.Request.Context(), vid, c.Writer)
	if err != nil {
		slog.Error("打包下载失败", "error", err)
		// 注意：如果已经开始写入数据，这里再写 JSON 错误可能会破坏响应
		return
	}
}

// UpdateResourceDependencies 更新依赖关联
func (m *Module) UpdateResourceDependencies(c *gin.Context) {
	vid := c.Param("vid")
	var req []core.DependencyDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := m.uc.UpdateResourceDependencies(c.Request.Context(), vid, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Dependencies updated"})
}

// UpdateResource 更新资源基本信息
func (m *Module) UpdateResource(c *gin.Context) {
	id := c.Param("id")
	var req core.UpdateResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := m.uc.UpdateResource(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Resource updated"})
}

// UpdateVersionMetadata 更新版本元数据
func (m *Module) UpdateVersionMetadata(c *gin.Context) {
	vid := c.Param("vid")
	var req core.UpdateVersionMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := m.uc.UpdateVersionMetadata(c.Request.Context(), vid, req.MetaData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Metadata updated"})
}

// ListResourceTypes 获取所有资源类型定义 (包含 Schema)
func (m *Module) ListResourceTypes(c *gin.Context) {
	types, err := m.uc.ListResourceTypes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Sort based on orderedKeys
	if len(m.orderedKeys) > 0 {
		orderMap := make(map[string]int)
		for i, k := range m.orderedKeys {
			orderMap[k] = i
		}

		// Simple bubble sort or slice stable sort (using insertion logic here for clarity or just sort.Slice)
		// Let's use specific sort logic
		sorted := make([]model.ResourceType, 0, len(types))

		// 1. Add types present in config in order
		typeMap := make(map[string]model.ResourceType)
		for _, t := range types {
			typeMap[t.TypeKey] = t
		}

		for _, k := range m.orderedKeys {
			if t, ok := typeMap[k]; ok {
				sorted = append(sorted, t)
				delete(typeMap, k)
			}
		}

		// 2. Add remaining types (not in config or new)
		for _, t := range types {
			if _, ok := typeMap[t.TypeKey]; ok { // if still in map
				sorted = append(sorted, t)
			}
		}
		types = sorted
	}

	c.JSON(http.StatusOK, types)
}

// CreateResourceFromData 在线创建资源
func (m *Module) CreateResourceFromData(c *gin.Context) {
	var req core.CreateResourceFromDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 默认 owner
	if req.OwnerID == "" {
		req.OwnerID = "admin" // 暂定
	}

	resource, err := m.uc.CreateResourceFromData(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resource)
}
