package resource

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liny/sim-hub/internal/core/module"
	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/internal/modules/resource/core"
	"github.com/liny/sim-hub/pkg/sts"
)

// Module 实现了 module.Module 接口
type Module struct {
	uc *core.UseCase
}

func NewModule(d *data.Data, tv *sts.TokenVendor, bucket string) module.Module {
	return &Module{
		uc: core.NewUseCase(d, tv, bucket),
	}
}

func (m *Module) RegisterRoutes(g *gin.RouterGroup) {
	// /api/v1/integration/upload/... 路径组
	integration := g.Group("/integration")
	{
		integration.POST("/upload/token", m.ApplyUploadToken)
		integration.POST("/upload/confirm", m.ConfirmUpload)
	}

	// /api/v1/resources 路径组
	resources := g.Group("/resources")
	{
		resources.GET("", m.ListResources)
		resources.POST("/sync", m.SyncFromStorage) // 新增：同步存储
		resources.GET("/:id", m.GetResource)
		resources.PATCH("/:id/tags", m.UpdateResourceTags) // 新增：更新标签
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

// ListResources 列出资源
func (m *Module) ListResources(c *gin.Context) {
	page := 1
	size := 20
	typeKey := c.Query("type")
	categoryID := c.Query("category_id")

	list, total, err := m.uc.ListResources(c.Request.Context(), typeKey, categoryID, page, size)
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

// SyncFromStorage 同步存储中的文件到数据库
func (m *Module) SyncFromStorage(c *gin.Context) {
	count, err := m.uc.SyncFromStorage(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Sync completed", "count": count})
}
