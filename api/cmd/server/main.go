package main

import (
	"fmt"
	"log"

	"object-storage/internal/config"
	"object-storage/internal/database"
	"object-storage/internal/handlers"
	"object-storage/internal/repository"
	"object-storage/internal/service"
	"object-storage/internal/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("Connected to PostgreSQL")

	minioStorage, err := storage.NewMinioStorage(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO: %v", err)
	}

	log.Println("Connected to MinIO")

	bucketRepo := repository.NewBucketRepository(db)
	objectRepo := repository.NewObjectRepository(db)
	chunkRepo := repository.NewChunkRepository(db)

	bucketService := service.NewBucketService(bucketRepo, minioStorage)
	objectService := service.NewObjectService(objectRepo, bucketRepo, chunkRepo, minioStorage, cfg)

	bucketHandler := handlers.NewBucketHandler(bucketService)
	objectHandler := handlers.NewObjectHandler(objectService)

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := router.Group("/api/v1")
	{
		api.POST("/buckets", bucketHandler.CreateBucket)
		api.GET("/buckets", bucketHandler.ListBuckets)
		api.GET("/buckets/:bucket", bucketHandler.GetBucket)
		api.DELETE("/buckets/:bucket", bucketHandler.DeleteBucket)

		api.PUT("/buckets/:bucket/objects/*key", objectHandler.PutObject)
		api.GET("/buckets/:bucket/objects/*key", objectHandler.GetObject)
		api.DELETE("/buckets/:bucket/objects/*key", objectHandler.DeleteObject)
		api.GET("/buckets/:bucket/objects", objectHandler.ListObjects)
	}

	port := "8080"
	log.Printf("Starting server on port %s", port)
	if err := router.Run(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
