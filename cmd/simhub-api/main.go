package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/liny/sim-hub/internal/conf"
	"github.com/liny/sim-hub/internal/core/module"
	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/internal/modules/resource"
	"github.com/liny/sim-hub/pkg/sts"
	"github.com/spf13/viper"
)

func main() {
	// 1. 加载配置信息
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("读取配置文件出错: %s", err)
	}

	var cfg conf.Data
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("将配置解码为结构体时出错: %v", err)
	}

	// 2. 初始化核心数据组件 (数据库与 MinIO)
	dbConn, cleanup, err := data.NewData(&cfg)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer cleanup()

	// 3. 初始化 MinIO 客户端
	minioClientWrapper, err := data.NewMinIO(&cfg.MinIO)
	if err != nil {
		log.Printf("警告: MinIO 初始化失败: %v", err)
	} else {
		log.Println("MinIO 连接成功")
	}

	// 4. 初始化令牌分发器 (TokenVendor)
	// NewTokenVendor 需要 *minio.Client，该客户端位于包装类中
	var tokenVendor *sts.TokenVendor
	if minioClientWrapper != nil {
		tokenVendor = sts.NewTokenVendor(minioClientWrapper.Client, cfg.MinIO.AccessKey, cfg.MinIO.SecretKey)
	}

	// 5. 业务模块注册
	registry := module.NewRegistry()
	// NewModule 需要 *data.Data、*sts.TokenVendor 以及 bucket 名称
	registry.Register(resource.NewModule(dbConn, tokenVendor, cfg.MinIO.Bucket))

	// 6. 配置 HTTP 路由
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	v1 := r.Group("/api/v1")
	registry.MapRoutes(v1)

	log.Println("服务器正在启动，端口为 :30030")
	if err := r.Run(":30030"); err != nil {
		log.Fatal(err)
	}
}
