package models

import (
	"time"

	"github.com/google/uuid"
)

type Bucket struct {
	ID        uuid.UUID `json:"string"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Object struct {
	ID          uuid.UUID `json:"id"`
	BucketID    uuid.UUID `json:"bucket_id"`
	Key         string    `json:"key"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	ETag        string    `json:"etag"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Chunk struct {
	ID             uuid.UUID `json:"id"`
	Hash           string    `json:"hash"`
	Size           int64     `json:"size"`
	MinioKey       string    `json:"minio_key"`
	ReferenceCount int       `json:"reference_count"`
	CreatedAt      time.Time `json:"created_at"`
	IsCompressed   bool      `json:"is_compressed"`
	OriginalSize   int64     `json:"original_size"`
}

type ObjectChunk struct {
	ID         uuid.UUID `json:"id"`
	ObjectID   uuid.UUID `json:"object_id"`
	ChunkID    uuid.UUID `json:"chunk_id"`
	ChunkIndex int       `json:"chunk_index"`
	CreatedAt  time.Time `json:"created_at"`
}

//DTOs

type CreateBucketRequest struct {
	Name string `json:"name" binding:"required"`
}

type ListBucketsResponse struct {
	Buckets []Bucket `json:"buckets"`
}

type ListObjectsResponse struct {
	Objects []Object `json:"objects"`
}

type ObjectMetadata struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"content_type"`
	ETag         string    `json:"etag"`
	LastModified time.Time `json:"last_modified"`
}
