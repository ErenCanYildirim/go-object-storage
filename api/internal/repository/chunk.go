package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"object-storage/internal/database"
	"object-storage/internal/models"
)

type ChunkRepository struct {
	db *database.DB
}

func NewChunkRepository(db *database.DB) *ChunkRepository {
	return &ChunkRepository{db: db}
}

func (r *ChunkRepository) Create(ctx context.Context, hash string, size int64, minioKey string, isCompressed bool, originalSize int64) (*models.Chunk, error) {
	chunk := &models.Chunk{}
	query := `
		INSERT INTO chunks (hash, size, minio_key, reference_count, is_compressed, original_size)
		VALUES ($1, $2, $3, 0, $4, $5)
		RETURNING id, hash, size, minio_key, reference_count, created_at, is_compressed, original_size
	`
	err := r.db.QueryRowContext(ctx, query, hash, size, minioKey, isCompressed, originalSize).Scan(
		&chunk.ID, &chunk.Hash, &chunk.Size, &chunk.MinioKey, &chunk.ReferenceCount, &chunk.CreatedAt, &chunk.IsCompressed, &chunk.OriginalSize,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunk: %w", err)
	}
	return chunk, nil
}

func (r *ChunkRepository) GetByHash(ctx context.Context, hash string) (*models.Chunk, error) {
	chunk := &models.Chunk{}
	query := `
		SELECT id, hash, size, minio_key, reference_count, created_at, is_compressed, original_size
		FROM chunks
		WHERE hash = $1
	`
	err := r.db.QueryRowContext(ctx, query, hash).Scan(
		&chunk.ID, &chunk.Hash, &chunk.Size, &chunk.MinioKey, &chunk.ReferenceCount, &chunk.CreatedAt, &chunk.IsCompressed, &chunk.OriginalSize,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}
	return chunk, nil
}

func (r *ChunkRepository) IncrementReference(ctx context.Context, chunkID uuid.UUID) error {
	query := `UPDATE chunks SET reference_count = reference_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, chunkID)
	return err
}

func (r *ChunkRepository) DecrementReference(ctx context.Context, chunkID uuid.UUID) error {
	query := `UPDATE chunks SET reference_count = reference_count - 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, chunkID)
	return err
}

func (r *ChunkRepository) GetUnreferencedChunks(ctx context.Context) ([]models.Chunk, error) {
	query := `
		SELECT id, hash, size, minio_key, reference_count, created_at
		FROM chunks
		WHERE reference_count <= 0
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get unreferenced chunks: %w", err)
	}
	defer rows.Close()

	chunks := []models.Chunk{}
	for rows.Next() {
		var chunk models.Chunk
		if err := rows.Scan(&chunk.ID, &chunk.Hash, &chunk.Size, &chunk.MinioKey, &chunk.ReferenceCount, &chunk.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

func (r *ChunkRepository) Delete(ctx context.Context, chunkID uuid.UUID) error {
	query := `DELETE FROM chunks WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, chunkID)
	return err
}

func (r *ChunkRepository) CreateObjectChunk(ctx context.Context, objectID, chunkID uuid.UUID, index int) error {
	query := `
		INSERT INTO object_chunks (object_id, chunk_id, chunk_index)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.ExecContext(ctx, query, objectID, chunkID, index)
	if err != nil {
		return fmt.Errorf("failed to create object chunk: %w", err)
	}
	return nil
}

func (r *ChunkRepository) GetObjectChunks(ctx context.Context, objectID uuid.UUID) ([]models.Chunk, error) {
	query := `
		SELECT c.id, c.hash, c.size, c.minio_key, c.reference_count, c.created_at, c.is_compressed, c.original_size
		FROM chunks c
		INNER JOIN object_chunks oc ON c.id = oc.chunk_id
		WHERE oc.object_id = $1
		ORDER BY oc.chunk_index
	`
	rows, err := r.db.QueryContext(ctx, query, objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get object chunks: %w", err)
	}
	defer rows.Close()

	chunks := []models.Chunk{}
	for rows.Next() {
		var chunk models.Chunk
		if err := rows.Scan(&chunk.ID, &chunk.Hash, &chunk.Size, &chunk.MinioKey, &chunk.ReferenceCount, &chunk.CreatedAt, &chunk.IsCompressed, &chunk.OriginalSize); err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

func (r *ChunkRepository) DeleteObjectChunks(ctx context.Context, objectID uuid.UUID) error {
	query := `DELETE FROM object_chunks WHERE object_id = $1`
	_, err := r.db.ExecContext(ctx, query, objectID)
	return err
}
