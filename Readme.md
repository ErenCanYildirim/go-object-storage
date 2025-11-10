# S3-Like Object Storage System

An object storage system built with **Go**, **MinIO** and **PostgreSQL**.
Uses chunking and deduplication for file storage and an API oriented from S3.

## Features

- **Object storage** using buckets
- **Chunking** - automatic file splitting into 5MB chunks
- **Deduplication** using SHA-256 hashing
- **ACID consistency** for metadata
- **REST-API** inspired by S3

## Components

1. API Layer (Golang + Gin)

2. Storage layer (MinIO)

3. Metadata layer (PostgreSQL)

## Functionality

The system has the core functionalities of a file storage, open to extension.
Currently files are chunked into 5 MB chunks, which are each hashed (SHA-256) for deduplication and then stored in a MinIO system-chunks bucket.

The database checks for existing chunk hashes, which can be reused. The DB also stores reference counts. When an object is deleted, only chunks with ref_count = 0 are deleted.

E.g.

```text
    Object A: [Chunk1, Chunk2, Chunk3]
    Object B: [Chunk2, Chunk3, Chunk4]

    Chunk2 reference_count = 2
    Chunk3 reference_count = 2
```

upon deleting object A only Chunk1 is deleted.

## Quick start

This is open to adapting it for your own needs.

Start via ```bash docker-compose up --build```. 
Check the health:
```bash
    curl http://localhost:8080/health
```

## Test the API

You can run the added test-client.go for a complete check, or use the following endpoint checks:

```bash

#Create bucket
curl -X POST http://localhost:8080/api/v1/buckets \
  -H "Content-Type: application/json" \
  -d '{"name": "my-bucket"}'

# Upload file
curl -X PUT http://localhost:8080/api/v1/buckets/my-bucket/objects/test.txt \
  -H "Content-Type: text/plain" \
  --data-binary "Hello, World!"


# Download file
curl http://localhost:8080/api/v1/buckets/my-bucket/objects/test.txt

# List objects
curl http://localhost:8080/api/v1/buckets/my-bucket/objects

# Delete object
curl -X DELETE http://localhost:8080/api/v1/buckets/my-bucket/objects/test.txt
```

## API reference

Buckets

Create bucket 

```http
    POST /api/v1/buckets
    Content-Type: application/json
    {
        "name": "bucket"
    }
```

List buckets

```http
    GET /api/v1/buckets
```

Get bucket

```http
    GET /api/v1/buckets/{bucket}
```

Delete bucket

```http
    DELETE /api/v1/buckets/{bucket}
```

Objects

Put Object

```http
    PUT /api/v1/buckets/{bucket}/objects/{key}
    Content-Type: application/octet-stream
    Content-Length: {size}

    {binary-data}
```

Get Object

```http
    GET /api/v1/buckets/{bucket}/objects/{key}
```

Delete Object

```http
    DELETE /api/v1/buckets/{bucket}/objects/{key}
```

List Objects

```http
    GET /api/v1/buckets/{bucket}/objects?prefix={prefix}
```

## Database schema

The database has the following schema:

```sql
buckets
    - id (uuid, pk)
    - name (varchar, unique)
    - created_at, updated_at

objects
    - id (uuid, pk)
    - bucket_id (uuid, fk -> buckets)
    - key (varchar)
    - size (bigint)
    - content_type (varchar)
    - etag (varchar)
    - created_at, updated_at 

chunks
    - id (uuid, pk)
    - hash (varchar, unique)
    - size (bigint)
    - minio_key (varchar, unique)
    - reference_count (int)
    - created_at

object_chunks
    - id (uuid, pk)
    - objet_id (uuid, fk -> objects)
    - chunk_id (uuid, fk -> chunks)
    - chunk_index (int)
```

## Configuration

You can adapt this to your liking, the current docker-compose is also insecure with its variables.

```yaml
# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=objectstore
DB_PASSWORD=secret
DB_NAME=objectstore

# MinIO
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_USE_SSL=false

# Chunking
CHUNK_SIZE=5242880  # 5 MB
```

## Sys. design fundamentals

The repo was also intended as an exercise in system design. Fault tolerance is ensured via DB translations using atomicity, rolling back on MinIO failure and reference counting in the DB to prevent orphaned chunks.

The system can be scaled by adding MinIO nodes on the horizontal, increasing the chunk size on the vertical and adding DB connection poolng for throughput.

## Enhacements

Potential enhancements for the future are:
    - multipart uploads
    - streaming
    - versioning
    - compression