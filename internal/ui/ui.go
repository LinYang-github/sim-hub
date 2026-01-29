package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// 这些文件夹将在构建时由 Makefile 填充
// 我们使用 all: 前缀确保所有文件（包括隐藏文件）都被包含

//go:embed all:dist_web
var webFS embed.FS

//go:embed all:dist_terrain
var terrainFS embed.FS

//go:embed all:dist_ext_apps
var extAppsFS embed.FS

// RegisterUIHandlers registers all embedded frontend routes
func RegisterUIHandlers(r *gin.Engine) {
	// 1. Terrain App (Prefix: /terrain)
	terrainSub, _ := fs.Sub(terrainFS, "dist_terrain")
	r.StaticFS("/terrain", http.FS(terrainSub))

	// 2. Consolidated External Apps (MPA)
	extSub, _ := fs.Sub(extAppsFS, "dist_ext_apps")

	r.StaticFS("/demo-repo", http.FS(extSub))
	r.StaticFS("/demo-preview", http.FS(extSub))
	r.StaticFS("/ext", http.FS(extSub)) // New unified prefix

	// 3. Main Web (Prefix: /) - SPA Support
	webSub, _ := fs.Sub(webFS, "dist_web")
	webStatic := http.FileServer(http.FS(webSub))

	r.NoRoute(func(c *gin.Context) {
		// API 请求不处理
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			return
		}

		// 检查文件是否存在于主应用中
		filePath := strings.TrimPrefix(c.Request.URL.Path, "/")
		if filePath == "" {
			filePath = "index.html"
		}

		f, err := webSub.Open(filePath)
		if err == nil {
			f.Close()
			webStatic.ServeHTTP(c.Writer, c.Request)
			return
		}

		// SPA 回退到 index.html
		c.Request.URL.Path = "/"
		webStatic.ServeHTTP(c.Writer, c.Request)
	})
}
