package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"object-storage/internal/database"
	"object-storage/internal/models"
)

type ObjectRepository struct {
	db *database.DB
}

func NewObjectRepository(db *database.DB) *ObjectRepository {
	return &ObjectRepository{db: db}
}

func (r *ObjectRepository) Create(ctx context.Context, bucketID uuid.UUID, key string, size int64, contentType, etag string) (*models.Object, error) {
	object := &models.Object{}
	query := `
		INSERT INTO objects (bucket_id, key, size, content_type, etag)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, bucket_id, key, size, content_type, etag, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query, bucketID, key, size, contentType, etag).Scan(
		&object.ID, &object.BucketID, &object.Key, &object.Size,
		&object.ContentType, &object.ETag, &object.CreatedAt, &object.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create object: %w", err)
	}
	return object, nil
}

func (r *ObjectRepository) GetByKey(ctx context.Context, bucketID uuid.UUID, key string) (*models.Object, error) {
	object := &models.Object{}
	query := `
		SELECT id, bucket_id, key, size, content_type, etag, created_at, updated_at
		FROM objects
		WHERE bucket_id = $1 AND key = $2
	`
	err := r.db.QueryRowContext(ctx, query, bucketID, key).Scan(
		&object.ID, &object.BucketID, &object.Key, &object.Size,
		&object.ContentType, &object.ETag, &object.CreatedAt, &object.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("object not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	return object, nil
}

func (r *ObjectRepository) List(ctx context.Context, bucketID uuid.UUID, prefix string) ([]models.Object, error) {
	query := `
		SELECT id, bucket_id, key, size, content_type, etag, created_at, updated_at
		FROM objects
		WHERE bucket_id = $1 AND key LIKE $2
		ORDER BY key
	`

	rows, err := r.db.QueryContext(ctx, query, bucketID, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}
	defer rows.Close()

	objects := []models.Object{}

	for rows.Next() {
		var object models.Object
		if err := rows.Scan(
			&object.ID, &object.BucketID, &object.Key, &object.Size,
			&object.ContentType, &object.ETag, &object.CreatedAt, &object.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan object: %w", err)
		}
		objects = append(objects, object)
	}
	return objects, nil
}

func (r *ObjectRepository) Delete(ctx context.Context, bucketID uuid.UUID, key string) error {
	query := `DELETE FROM objects WHERE bucket_id = $1 AND key = $2`
	result, err := r.db.ExecContext(ctx, query, bucketID, key)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("object not found")
	}

	return nil
}
