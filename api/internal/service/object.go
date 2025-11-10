package service

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"object-storage/internal/config"
	"object-storage/internal/models"
	"object-storage/internal/repository"
	"object-storage/internal/storage"
)

type ObjectService struct {
	objectRepo *repository.ObjectRepository
	bucketRepo *repository.BucketRepository
	chunkRepo  *repository.ChunkRepository
	storage    *storage.MinioStorage
	cfg        *config.Config
}

func NewObjectService(
	objectRepo *repository.ObjectRepository,
	bucketRepo *repository.BucketRepository,
	chunkRepo *repository.ChunkRepository,
	storage *storage.MinioStorage,
	cfg *config.Config,
) *ObjectService {
	return &ObjectService{
		objectRepo: objectRepo,
		bucketRepo: bucketRepo,
		chunkRepo:  chunkRepo,
		storage:    storage,
		cfg:        cfg,
	}
}

func (s *ObjectService) PutObject(ctx context.Context, bucketName, key string, reader io.Reader, size int64, contentType string) (*models.Object, error) {
	// Get bucket
	bucket, err := s.bucketRepo.GetByName(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	// Read entire file into memory for chunking
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	// Calculate overall ETag
	etag := calculateMD5(data)

	// Create object in database
	object, err := s.objectRepo.Create(ctx, bucket.ID, key, int64(len(data)), contentType, etag)
	if err != nil {
		return nil, fmt.Errorf("failed to create object: %w", err)
	}

	// Chunk the data
	chunks := chunkData(data, s.cfg.ChunkSize)

	// Store each chunk with deduplication
	for index, chunkData := range chunks {
		chunkHash := calculateSHA256(chunkData)

		// Check if chunk already exists
		existingChunk, err := s.chunkRepo.GetByHash(ctx, chunkHash)
		if err != nil {
			// Rollback object creation
			_ = s.objectRepo.Delete(ctx, bucket.ID, key)
			return nil, fmt.Errorf("failed to check chunk: %w", err)
		}

		var chunk *models.Chunk
		if existingChunk != nil {
			// Chunk exists, reuse it
			chunk = existingChunk
		} else {
			// New chunk, store it in MinIO
			minioKey := s.storage.GenerateChunkKey()
			err = s.storage.PutChunk(ctx, minioKey, bytes.NewReader(chunkData), int64(len(chunkData)), "application/octet-stream")
			if err != nil {
				// Rollback
				_ = s.objectRepo.Delete(ctx, bucket.ID, key)
				return nil, fmt.Errorf("failed to store chunk: %w", err)
			}

			// Create chunk record
			chunk, err = s.chunkRepo.Create(ctx, chunkHash, int64(len(chunkData)), minioKey)
			if err != nil {
				// Rollback
				_ = s.storage.DeleteChunk(ctx, minioKey)
				_ = s.objectRepo.Delete(ctx, bucket.ID, key)
				return nil, fmt.Errorf("failed to create chunk record: %w", err)
			}
		}

		// Link chunk to object
		if err := s.chunkRepo.CreateObjectChunk(ctx, object.ID, chunk.ID, index); err != nil {
			// Rollback
			_ = s.objectRepo.Delete(ctx, bucket.ID, key)
			return nil, fmt.Errorf("failed to link chunk: %w", err)
		}

		// Increment reference count
		if err := s.chunkRepo.IncrementReference(ctx, chunk.ID); err != nil {
			return nil, fmt.Errorf("failed to increment reference: %w", err)
		}
	}

	return object, nil
}

func (s *ObjectService) GetObject(ctx context.Context, bucketName, key string) (io.Reader, *models.Object, error) {

	bucket, err := s.bucketRepo.GetByName(ctx, bucketName)
	if err != nil {
		return nil, nil, fmt.Errorf("bucket not found: %w", err)
	}

	object, err := s.objectRepo.GetByKey(ctx, bucket.ID, key)
	if err != nil {
		return nil, nil, fmt.Errorf("object not found: %w", err)
	}

	//retrieving all chunks for that object
	chunks, err := s.chunkRepo.GetObjectChunks(ctx, object.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get chunks: %w", err)
	}

	var buffer bytes.Buffer
	//reassembling the chunks

	for _, chunk := range chunks {
		chunkReader, err := s.storage.GetChunk(ctx, chunk.MinioKey)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get chunk: %w", err)
		}
		defer chunkReader.Close()

		if _, err := io.Copy(&buffer, chunkReader); err != nil {
			return nil, nil, fmt.Errorf("failed to read chunk: %w", err)
		}
	}
	return &buffer, object, nil
}

func (s *ObjectService) DeleteObject(ctx context.Context, bucketName, key string) error {

	bucket, err := s.bucketRepo.GetByName(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("bucket not found: %w", err)
	}

	object, err := s.objectRepo.GetByKey(ctx, bucket.ID, key)
	if err != nil {
		return fmt.Errorf("object not found: %w", err)
	}

	chunks, err := s.chunkRepo.GetObjectChunks(ctx, object.ID)
	if err != nil {
		return fmt.Errorf("failed to get chunks: %w", err)
	}

	//decrement reference counts and delete unreferenced chunks

	for _, chunk := range chunks {
		if err := s.chunkRepo.DecrementReference(ctx, chunk.ID); err != nil {
			return fmt.Errorf("failed to decrement reference: %w", err)
		}

		//check if the chunk is unreferenced
		updatedChunk, err := s.chunkRepo.GetByHash(ctx, chunk.Hash)
		if err != nil {
			return fmt.Errorf("failed to get updated chunk: %w", err)
		}

		if updatedChunk.ReferenceCount <= 0 {
			//Delete it from MinIO

			if err := s.storage.DeleteChunk(ctx, chunk.MinioKey); err != nil {
				return fmt.Errorf("failed to delete chunk from storage: %w", err)
			}

			//delete it from the database
			if err := s.chunkRepo.Delete(ctx, chunk.ID); err != nil {
				return fmt.Errorf("failed to delete chunk record: %w", err)
			}
		}
	}

	// Delete object-chunk mappings
	if err := s.chunkRepo.DeleteObjectChunks(ctx, object.ID); err != nil {
		return fmt.Errorf("failed to delete object chunks: %w", err)
	}

	// Delete object
	if err := s.objectRepo.Delete(ctx, bucket.ID, key); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

func (s *ObjectService) ListObjects(ctx context.Context, bucketName, prefix string) ([]models.Object, error) {
	bucket, err := s.bucketRepo.GetByName(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	return s.objectRepo.List(ctx, bucket.ID, prefix)
}

func chunkData(data []byte, chunkSize int64) [][]byte {
	var chunks [][]byte
	dataLen := int64(len(data))

	for i := int64(0); i < dataLen; i += chunkSize {
		end := i + chunkSize
		if end > dataLen {
			end = dataLen
		}
		chunks = append(chunks, data[i:end])
	}
	return chunks
}

func calculateSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func calculateMD5(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}
