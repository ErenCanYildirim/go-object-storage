package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"object-storage/internal/models"
	"object-storage/internal/service"
)

type BucketHandler struct {
	bucketService *service.BucketService
}

func NewBucketHandler(bucketService *service.BucketService) *BucketHandler {
	return &BucketHandler{bucketService: bucketService}
}

func (h *BucketHandler) CreateBucket(c *gin.Context) {
	var req models.CreateBucketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bucket, err := h.bucketService.CreateBucket(c.Request.Context(), req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, bucket)
}

func (h *BucketHandler) GetBucket(c *gin.Context) {
	bucketName := c.Param("bucket")

	bucket, err := h.bucketService.GetBucket(c.Request.Context(), bucketName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, bucket)
}

func (h *BucketHandler) ListBuckets(c *gin.Context) {
	buckets, err := h.bucketService.ListBuckets(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.ListBucketsResponse{Buckets: buckets})
}

func (h *BucketHandler) DeleteBucket(c *gin.Context) {
	bucketName := c.Param("bucket")

	if err := h.bucketService.DeleteBucket(c.Request.Context(), bucketName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
