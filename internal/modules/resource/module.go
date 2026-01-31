package resource

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"sim-hub/internal/conf"
	"sim-hub/internal/core/module"
	"sim-hub/internal/data"
	"sim-hub/internal/model"
	"sim-hub/internal/modules/resource/core"
	"sim-hub/pkg/storage"

	"github.com/gin-gonic/gin"

	"gorm.io/gorm/clause"
)

// Module 实现了 module.Module 接口
type Module struct {
	uc          *core.UseCase
	auth        *core.AuthManager
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
		if err := d.DB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "type_key"}}, UpdateAll: true}).Create(&types).Error; err != nil {
			slog.Error("Sync resource types failed", "error", err)
		} else {
			slog.Info("Synced resource types to DB", "count", len(types))
		}
	}

	orderedKeys := make([]string, 0, len(resourceTypes))
	for _, rt := range resourceTypes {
		orderedKeys = append(orderedKeys, rt.TypeKey)
	}

	// 确保 Admin 用户存在
	authMgr := core.NewAuthManager(d)
	if err := authMgr.EnsureAdminUser(context.Background(), "123456"); err != nil {
		slog.Error("Failed to ensure admin user", "error", err)
	}

	return &Module{
		uc:          core.NewUseCase(d, store, stsProvider, bucket, natsClient, role, apiBaseURL, handlers),
		auth:        authMgr,
		orderedKeys: orderedKeys,
	}
}

func (m *Module) RegisterRoutes(g *gin.RouterGroup) {
	authGrp := g.Group("/auth")
	{
		authGrp.POST("/login", m.Login)
		authGrp.GET("/me", AuthMiddleware(m.auth), m.Me)
		authGrp.GET("/tokens", AuthMiddleware(m.auth), m.ListTokens)
		authGrp.POST("/tokens", AuthMiddleware(m.auth), m.CreateToken)
		authGrp.DELETE("/tokens/:id", AuthMiddleware(m.auth), m.RevokeToken)
	}

	// /api/v1/integration/upload/... 路径组 (增加鉴权与 RBAC)
	integration := g.Group("/integration", AuthMiddleware(m.auth))
	{
		// 只有具备 resource:create 权限的角色才能上传
		upload := integration.Group("", RBACMiddleware(m.auth, "resource:create"))
		{
			upload.POST("/upload/token", m.ApplyUploadToken)
			upload.POST("/upload/confirm", m.ConfirmUpload)
			upload.POST("/upload/multipart/init", m.InitMultipartUpload)
			upload.POST("/upload/multipart/part-url", m.GetMultipartUploadPartURL)
			upload.POST("/upload/multipart/complete", m.CompleteMultipartUpload)
		}
	}

	// /api/v1/resource-types
	rTypes := g.Group("/resource-types")
	{
		rTypes.GET("", m.ListResourceTypes)
	}

	// /api/v1/resources 路径组
	resources := g.Group("/resources")
	{
		// Public Read
		resources.GET("", m.ListResources)
		resources.GET("/:id", m.GetResource)
		resources.GET("/:id/versions", m.ListVersions)
		resources.GET("/versions/:vid/dependencies", m.GetDependencies)
		resources.GET("/versions/:vid/dependency-tree", m.GetDependencyTree)
		resources.GET("/versions/:vid/bundle", m.GetBundle)

		// Protected Write
		protected := resources.Group("", AuthMiddleware(m.auth))
		{
			protected.POST("/sync", RBACMiddleware(m.auth, "resource:sync"), m.SyncFromStorage)
			protected.POST("/clear", RBACMiddleware(m.auth, "resource:delete"), m.ClearResources)
			protected.POST("/create", RBACMiddleware(m.auth, "resource:create"), m.CreateResourceFromData)
			protected.PATCH("/:id", RBACMiddleware(m.auth, "resource:update"), m.UpdateResource)
			protected.DELETE("/:id", RBACMiddleware(m.auth, "resource:delete"), m.DeleteResource)
			protected.PATCH("/:id/tags", RBACMiddleware(m.auth, "resource:update"), m.UpdateResourceTags)
			protected.PATCH("/:id/scope", RBACMiddleware(m.auth, "resource:update"), m.UpdateResourceScope)
			protected.PATCH("/:id/process-result", m.ReportProcessResult)

			protected.POST("/:id/latest", RBACMiddleware(m.auth, "resource:update"), m.SetLatestVersion)
			protected.PATCH("/versions/:vid/dependencies", RBACMiddleware(m.auth, "resource:update"), m.UpdateResourceDependencies)
			protected.PATCH("/versions/:vid/meta", RBACMiddleware(m.auth, "resource:update"), m.UpdateVersionMetadata)
			// 下载需要鉴权么？暂时公开吧，或者是 protected？
			// SDK 下载是带 Token 的，但是浏览器直接下载可能不方便带 Header。
			// 鉴于目前是 PAT，浏览器下载如果是通过 URL，可能需要 Query Param Token 或者是 Cookie。
			// 简单起见，下载先公开，或者保留在 protected 里但是前端暂时无法下载。
			// 考虑到前端没有下载功能，主要是 SDK 下载，SDK 有 Token。
			// 但是 walkthrough 里提到 "浏览器直接下载..."
			// 既然暂时前端没登录，先公开下载吧，方便调试。
		}
		// 暂时公开下载
		resources.GET("/versions/:vid/download-pack", m.DownloadBundle)
	}

	// /api/v1/dashboard 路径组
	dashboard := g.Group("/dashboard")
	{
		dashboard.GET("/stats", m.GetDashboardStats)
	}

	// /api/v1/categories 路径组
	categories := g.Group("/categories")
	{
		categories.GET("", m.ListCategories)

		protected := categories.Group("", AuthMiddleware(m.auth))
		{
			protected.POST("", RBACMiddleware(m.auth, "resource:update"), m.CreateCategory)
			protected.DELETE("/:id", RBACMiddleware(m.auth, "resource:delete"), m.DeleteCategory)
			protected.PATCH("/:id", RBACMiddleware(m.auth, "resource:update"), m.UpdateCategory)
		}
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

// GetDashboardStats 获取概览统计
func (m *Module) GetDashboardStats(c *gin.Context) {
	// 默认使用 Admin 用户 ID，实际应从 Token 获取
	ownerID := "admin"

	stats, err := m.uc.GetDashboardStats(c.Request.Context(), ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// --- Token Management ---

// Me 获取当前用户信息
func (m *Module) Me(c *gin.Context) {
	// 优先从 Context 获取已识别的 userID (JWT 或 Proxy 注入)
	userID, exists := c.Get("user_id")
	var user *model.User
	var err error

	if exists {
		uid := userID.(string)
		user, err = m.auth.GetUserWithRole(c.Request.Context(), uid)
	} else {
		// 备选方案：通过 username 查询 (仅用于兼容或特殊调试)
		username := c.Query("username")
		if username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "identity not found"})
			return
		}
		user = &model.User{}
		err = m.auth.DB().Preload("Role").Where("username = ?", username).First(user).Error
	}

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// Login 用户登录
func (m *Module) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := m.auth.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// CreateToken 创建个人访问令牌
func (m *Module) CreateToken(c *gin.Context) {
	var req struct {
		UserID       string `json:"user_id"`
		Name         string `json:"name"`
		ExpireInDays int    `json:"expire_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 临时方案：如果没传 userID，使用默认的 admin
	if req.UserID == "" {
		req.UserID = "admin"
	}

	resp, err := m.auth.CreateAccessToken(c.Request.Context(), req.UserID, req.Name, req.ExpireInDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

// ListTokens 列出令牌
func (m *Module) ListTokens(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = "admin"
	}

	tokens, err := m.auth.ListAccessTokens(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tokens)
}

// RevokeToken 撤销令牌
func (m *Module) RevokeToken(c *gin.Context) {
	id := c.Param("id")
	userID := c.Query("user_id")
	if userID == "" {
		userID = "admin"
	}

	if err := m.auth.RevokeAccessToken(c.Request.Context(), userID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Token revoked"})
}
