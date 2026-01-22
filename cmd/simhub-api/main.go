package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/liny/sim-hub/internal/biz"
	"github.com/liny/sim-hub/internal/conf"
	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/internal/service"
	"github.com/liny/sim-hub/pkg/sts"
	"github.com/spf13/viper"
)

func main() {
	// 1. Load Config
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	var cfg conf.Data
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Unable to decode into struct: %v", err)
	}

	// 2. Init Data (DB & MinIO)
	dbConn, cleanup, err := data.NewData(&cfg)
	if err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}
	defer cleanup()

	minioClient, err := data.NewMinIO(&cfg.MinIO)
	if err != nil {
		log.Printf("Warning: Failed to init MinIO: %v", err)
	} else {
		log.Println("MinIO connected successfully")
	}

	// 3. Init Biz & Service
	// If MinIO failed, minioClient is nil. In production we might fatal, but for dev we might allow partial start.
	// However, TokenVendor needs a client.
	var tv *sts.TokenVendor
	if minioClient != nil {
		tv = sts.NewTokenVendor(minioClient.Client)
	}

	bucketName := cfg.MinIO.Bucket
	if bucketName == "" {
		bucketName = "simhub-raw"
	}

	uc := biz.NewResourceUseCase(dbConn, tv, bucketName)
	svc := service.NewResourceService(uc)

	// 4. Setup Router
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
			"db":      dbConn.DB != nil,
			"minio":   minioClient != nil,
		})
	})

	v1 := r.Group("/api/v1")
	{
		integration := v1.Group("/integration")
		{
			integration.POST("/upload/token", svc.ApplyUploadToken)
			integration.POST("/upload/confirm", svc.ConfirmUpload)
		}
		resources := v1.Group("/resources")
		{
			resources.GET("", svc.ListResources)
			resources.GET("/:id", svc.GetResource)
		}
	}

	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
