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

	// 3. Init MinIO
	minioClientWrapper, err := data.NewMinIO(&cfg.MinIO)
	if err != nil {
		log.Printf("Warning: MinIO init failed: %v", err)
	} else {
		log.Println("MinIO connected successfully")
	}

	// 4. Init TokenVendor
	// NewTokenVendor expects *minio.Client, which is inside our wrapper
	var tokenVendor *sts.TokenVendor
	if minioClientWrapper != nil {
		tokenVendor = sts.NewTokenVendor(minioClientWrapper.Client, cfg.MinIO.AccessKey, cfg.MinIO.SecretKey)
	}

	// 5. Module Registration
	registry := module.NewRegistry()
	// NewModule expects *data.Data, *sts.TokenVendor, bucket string
	registry.Register(resource.NewModule(dbConn, tokenVendor, cfg.MinIO.Bucket))

	// 6. Setup Router
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	v1 := r.Group("/api/v1")
	registry.MapRoutes(v1)

	log.Println("Server starting on :30030")
	if err := r.Run(":30030"); err != nil {
		log.Fatal(err)
	}
}
