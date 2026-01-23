package resource

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liny/sim-hub/internal/core/module"
	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/internal/modules/resource/core"
	"github.com/liny/sim-hub/pkg/storage"
)

// Module 实现了 module.Module 接口
type Module struct {
	uc *core.UseCase
}

func NewModule(d *data.Data, store storage.MultipartBlobStore, stsProvider storage.SecurityTokenProvider, bucket string, natsClient *data.NATSClient, role string, apiBaseURL string, handlers map[string]string) module.Module {
	return &Module{
		uc: core.NewUseCase(d, store, stsProvider, bucket, natsClient, role, apiBaseURL, handlers),
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

	// /api/v1/resources 路径组
	resources := g.Group("/resources")
	{
		resources.GET("", m.ListResources)
		resources.POST("/sync", m.SyncFromStorage) // 新增：同步存储
		resources.GET("/:id", m.GetResource)
		resources.DELETE("/:id", m.DeleteResource)
		resources.PATCH("/:id/tags", m.UpdateResourceTags)
		resources.PATCH("/:id/scope", m.UpdateResourceScope) // 新增：更新作用域
		resources.PATCH("/:id/process-result", m.ReportProcessResult)

		// 新增：依赖管理
		resources.GET("/versions/:vid/dependencies", m.GetDependencies)
		resources.GET("/versions/:vid/dependency-tree", m.GetDependencyTree)
	}

	// /api/v1/categories 路径组
	categories := g.Group("/categories")
	{
		categories.GET("", m.ListCategories)
		categories.POST("", m.CreateCategory)
		categories.DELETE("/:id", m.DeleteCategory)
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
	page := 1
	size := 20
	typeKey := c.Query("type")
	categoryID := c.Query("category_id")
	ownerID := c.Query("owner_id")
	scope := c.Query("scope")

	list, total, err := m.uc.ListResources(c.Request.Context(), typeKey, categoryID, ownerID, scope, page, size)
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
