package service

import (
	"context"
	"fmt"

	"object-storage/internal/models"
	"object-storage/internal/repository"
	"object-storage/internal/storage"
)

type BucketService struct {
	bucketRepo *repository.BucketRepository
	storage    *storage.MinioStorage
}

func NewBucketService(bucketRepo *repository.BucketRepository, storage *storage.MinioStorage) *BucketService {
	return &BucketService{
		bucketRepo: bucketRepo,
		storage:    storage,
	}
}

func (s *BucketService) CreateBucket(ctx context.Context, name string) (*models.Bucket, error) {

	if err := s.storage.CreateBucket(ctx, name); err != nil {
		return nil, fmt.Errorf("failed to create bucket in storage: %w", err)
	}

	bucket, err := s.bucketRepo.Create(ctx, name)
	if err != nil {
		//rollback if DB creation fails
		_ = s.storage.DeleteBucket(ctx, name)
		return nil, fmt.Errorf("failed to create bucket in database: %w", err)
	}

	return bucket, nil
}

func (s *BucketService) GetBucket(ctx context.Context, name string) (*models.Bucket, error) {
	return s.bucketRepo.GetByName(ctx, name)
}

func (s *BucketService) ListBuckets(ctx context.Context) ([]models.Bucket, error) {
	return s.bucketRepo.List(ctx)
}

func (s *BucketService) DeleteBucket(ctx context.Context, name string) error {
	//delete from DB first

	if err := s.bucketRepo.Delete(ctx, name); err != nil {
		return fmt.Errorf("failed to delete bucket from database: %w", err)
	}

	//afterwards removal from MinIO
	if err := s.storage.DeleteBucket(ctx, name); err != nil {
		return fmt.Errorf("failed to delete bucket from storage: %w", err)
	}

	return nil
}
