package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"object-storage/internal/database"
	"object-storage/internal/models"
)

type BucketRepository struct {
	db *database.DB
}

func NewBucketRepository(db *database.DB) *BucketRepository {
	return &BucketRepository{db: db}
}

func (r *BucketRepository) Create(ctx context.Context, name string) (*models.Bucket, error) {
	bucket := &models.Bucket{}
	query := `
		INSERT INTO buckets (name)
		VALUES ($1)
		RETURNING id, name, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&bucket.ID, &bucket.Name, &bucket.CreatedAt, &bucket.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}
	return bucket, nil
}

func (r *BucketRepository) GetByName(ctx context.Context, name string) (*models.Bucket, error) {
	bucket := &models.Bucket{}
	query := `
		SELECT id, name, created_at, updated_at
		FROM buckets
		WHERE name = $1
	`
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&bucket.ID, &bucket.Name, &bucket.CreatedAt, &bucket.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("bucket not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}
	return bucket, nil
}

func (r *BucketRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Bucket, error) {
	bucket := &models.Bucket{}

	query := `
		SELECT id, name, created_at, updated_at
		FROM buckets
		WHERE id = $1
	`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&bucket.ID, &bucket.Name, &bucket.CreatedAt, &bucket.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("bucket not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}
	return bucket, nil
}

func (r *BucketRepository) List(ctx context.Context) ([]models.Bucket, error) {
	query := `
		SELECT id, name, created_at, updated_at
		FROM buckets
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}
	defer rows.Close()

	buckets := []models.Bucket{}
	for rows.Next() {
		var bucket models.Bucket
		if err := rows.Scan(&bucket.ID, &bucket.Name, &bucket.CreatedAt, &bucket.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan bucket: %w", err)
		}
		buckets = append(buckets, bucket)
	}
	return buckets, nil
}

func (r *BucketRepository) Delete(ctx context.Context, name string) error {
	query := `DELETE FROM buckets WHERE name = $1`
	result, err := r.db.ExecContext(ctx, query, name)
	if err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("bucket not found")
	}

	return nil
}
