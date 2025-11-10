package storage

import (
	"context"
	"fmt"
	"io"

	"object-storage/internal/config"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	SystemBucket = "system-chunks"
)

type MinioStorage struct {
	client *minio.Client
}

func NewMinioStorage(cfg *config.Config) (*MinioStorage, error) {
	client, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	// Ensure system bucket exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, SystemBucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check system bucket: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, SystemBucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create system bucket: %w", err)
		}
	}

	return &MinioStorage{client: client}, nil
}

func (m *MinioStorage) CreateBucket(ctx context.Context, bucketName string) error {
	exists, err := m.client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket: %w", err)
	}

	if exists {
		return fmt.Errorf("bucket already exists")
	}

	return m.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
}

func (m *MinioStorage) DeleteBucket(ctx context.Context, bucketName string) error {
	return m.client.RemoveBucket(ctx, bucketName)
}

func (m *MinioStorage) PutChunk(ctx context.Context, chunkKey string, reader io.Reader, size int64, contentType string) error {
	_, err := m.client.PutObject(ctx, SystemBucket, chunkKey, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (m *MinioStorage) GetChunk(ctx context.Context, chunkKey string) (*minio.Object, error) {
	return m.client.GetObject(ctx, SystemBucket, chunkKey, minio.GetObjectOptions{})
}

func (m *MinioStorage) DeleteChunk(ctx context.Context, chunkKey string) error {
	return m.client.RemoveObject(ctx, SystemBucket, chunkKey, minio.RemoveObjectOptions{})
}

func (m *MinioStorage) GenerateChunkKey() string {
	return fmt.Sprintf("chunks/%s", uuid.New().String())
}
