package handlers

import (
	"io"
	"net/http"

	"object-storage/internal/models"
	"object-storage/internal/service"

	"github.com/gin-gonic/gin"
)

type ObjectHandler struct {
	objectService *service.ObjectService
}

func NewObjectHandler(objectService *service.ObjectService) *ObjectHandler {
	return &ObjectHandler{objectService: objectService}
}

func (h *ObjectHandler) PutObject(c *gin.Context) {
	bucketName := c.Param("bucket")
	key := c.Param("key")

	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	contentLength := c.Request.ContentLength
	if contentLength <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content length required"})
		return
	}

	object, err := h.objectService.PutObject(
		c.Request.Context(),
		bucketName,
		key,
		c.Request.Body,
		contentLength,
		contentType,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("ETag", object.ETag)
	c.JSON(http.StatusCreated, gin.H{
		"bucket": bucketName,
		"key":    key,
		"etag":   object.ETag,
		"size":   object.Size,
	})
}

func (h *ObjectHandler) GetObject(c *gin.Context) {
	bucketName := c.Param("bucket")
	key := c.Param("key")

	reader, object, err := h.objectService.GetObject(c.Request.Context(), bucketName, key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", object.ContentType)
	c.Header("Content-Length", string(object.Size))
	c.Header("ETag", object.ETag)
	c.Header("Last-Modified", object.UpdatedAt.Format(http.TimeFormat))

	// Stream the object back
	if _, err := io.Copy(c.Writer, reader); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

func (h *ObjectHandler) DeleteObject(c *gin.Context) {
	bucketName := c.Param("bucket")
	key := c.Param("key")

	if err := h.objectService.DeleteObject(c.Request.Context(), bucketName, key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *ObjectHandler) ListObjects(c *gin.Context) {
	bucketName := c.Param("bucket")
	prefix := c.DefaultQuery("prefix", "")

	objects, err := h.objectService.ListObjects(c.Request.Context(), bucketName, prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.ListObjectsResponse{Objects: objects})
}
