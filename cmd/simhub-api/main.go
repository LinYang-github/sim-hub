package main

import (
	"log"
	"log/slog"

	"os"

	"github.com/gin-gonic/gin"
	"github.com/liny/sim-hub/internal/conf"
	"github.com/liny/sim-hub/internal/core/module"
	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/internal/modules/resource"
	"github.com/liny/sim-hub/internal/ui"
	"github.com/liny/sim-hub/pkg/logger"
	"github.com/liny/sim-hub/pkg/storage"
	"github.com/liny/sim-hub/pkg/storage/minio"
	"github.com/spf13/viper"
)

func main() {
	// 1. 加载配置信息 (Master/API 专用)
	viper.SetConfigName("config-api")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		slog.Error("读取 API 配置文件出错 (config-api.yaml)", "error", err)
		os.Exit(1)
	}

	var cfg conf.Data
	if err := viper.Unmarshal(&cfg); err != nil {
		slog.Error("配置解码失败", "error", err)
		os.Exit(1)
	}

	// 1.5 初始化日志系统
	logger.InitLogger(&cfg.Log)

	// 2. 初始化核心数据组件 (数据库与 MinIO)
	dbConn, cleanup, err := data.NewData(&cfg)
	if err != nil {
		slog.Error("数据库初始化失败", "error", err)
		os.Exit(1)
	}
	defer cleanup()

	// 3. 初始化 MinIO 客户端
	minioClientWrapper, err := data.NewMinIO(&cfg.MinIO)
	if err != nil {
		slog.Warn("MinIO 初始化失败", "error", err)
	} else {
		slog.Info("MinIO 连接成功")
	}

	// 3.5 初始化 NATS 客户端 (如果开启)
	natsClient, err := data.NewNATS(&cfg.NATS)
	if err != nil {
		slog.Warn("NATS 初始化失败", "error", err)
	} else if natsClient != nil {
		defer natsClient.Close()
	}

	// 4. 初始化存储层 (MinIO Adapter)
	// MinIOStore 同时实现了 MultipartBlobStore 和 SecurityTokenProvider
	var blobStore storage.MultipartBlobStore
	var stsProvider storage.SecurityTokenProvider

	if minioClientWrapper != nil {
		store := minio.NewMinIOStore(minioClientWrapper.Client, cfg.MinIO.AccessKey, cfg.MinIO.SecretKey)
		blobStore = store
		stsProvider = store
	}

	// 5. 业务模块注册
	handlers := make(map[string]string)
	for _, rt := range cfg.ResourceTypes {
		if rt.ProcessConf != nil {
			// 如果配置了 pipeline，则视为有处理器（目前简单以 typeKey 为索引）
			handlers[rt.TypeKey] = "placeholder"
		}
	}

	registry := module.NewRegistry()
	registry.Register(resource.NewModule(dbConn, blobStore, stsProvider, cfg.MinIO.Bucket, natsClient, "api", "", handlers))

	// 6. 配置 HTTP 路由
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	v1 := r.Group("/api/v1")
	registry.MapRoutes(v1)

	// 7. 注册嵌入的前端页面 (UI Handlers)
	// 必须放在 API 注册之后，因为 UI 包含 NoRoute 的兜底逻辑
	ui.RegisterUIHandlers(r)

	slog.Info("服务器正在启动", "port", 30030)
	if err := r.Run(":30030"); err != nil {
		log.Fatal(err)
	}
}
