package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"sim-hub/internal/conf"
	"sim-hub/internal/data"
	"sim-hub/internal/modules/resource/core"
	"sim-hub/internal/search"
	"sim-hub/pkg/logger"
	"sim-hub/pkg/storage/minio"

	"github.com/spf13/viper"
)

func main() {
	// 1. 加载配置 (Worker 专用)
	viper.SetConfigName("config-worker")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		slog.Error("读取 Worker 配置文件出错 (config-worker.yaml)", "error", err)
		os.Exit(1)
	}

	var cfg conf.Data
	if err := viper.Unmarshal(&cfg); err != nil {
		slog.Error("配置解码失败", "error", err)
		os.Exit(1)
	}

	if !cfg.NATS.Enabled {
		slog.Error("Worker 节点必须开启 NATS 才能运行")
		os.Exit(1)
	}

	// 2. 初始化日志
	logger.InitLogger(&cfg.Log)

	// 3. Worker 现在不需要直接操作数据库
	// 它通过 API Callback 上报结果

	// 4. 初始化存储
	minioClientWrapper, err := data.NewMinIO(&cfg.MinIO)
	if err != nil {
		slog.Error("MinIO 初始化失败", "error", err)
		os.Exit(1)
	}

	store := minio.NewMinIOStore(minioClientWrapper.Client, cfg.MinIO.AccessKey, cfg.MinIO.SecretKey)

	// 5. 初始化 NATS
	natsClient, err := data.NewNATS(&cfg.NATS)
	if err != nil {
		slog.Error("NATS 初始化失败", "error", err)
		os.Exit(1)
	}
	defer natsClient.Close()

	// 6. 启动 UseCase (Worker 模式)
	_ = core.NewUseCase(nil, store, store, cfg.MinIO.Bucket, natsClient, "worker", cfg.Worker.ApiBaseURL, cfg.Worker.Handlers)

	slog.Info("SimHub 计算 Worker 已启动", "subject", cfg.NATS.Subject)

	// 7. 启动 ES Search Worker (可选)
	if len(cfg.Elasticsearch.Addresses) > 0 {
		esClient, err := data.NewElasticsearch(&cfg.Elasticsearch)
		if err != nil {
			slog.Error("Elasticsearch 初始化失败，将跳过搜索增强功能", "error", err)
		} else {
			var tikaClient *search.TikaClient
			if cfg.Tika.URL != "" {
				tikaClient = search.NewTikaClient(cfg.Tika.URL)
				slog.Info("Apache Tika 文本提取增强已启用", "url", cfg.Tika.URL)
			}
			esWorker := search.NewESWorker(esClient, natsClient, cfg.Worker.ApiBaseURL, cfg.Elasticsearch.Index, tikaClient, store, cfg.MinIO.Bucket)
			go esWorker.Start()
		}
	}

	// 优雅停机
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Worker 正在关闭...")
}
