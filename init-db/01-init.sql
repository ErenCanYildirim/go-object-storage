CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS buckets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS objects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bucket_id UUID NOT NULL REFERENCES buckets(id) ON DELETE CASCADE,
    key VARCHAR(1024) NOT NULL,
    size BIGINT NOT NULL,
    content_type VARCHAR(255),
    etag VARCHAR(64) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(bucket_id, key)
);

CREATE TABLE IF NOT EXISTS chunks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hash VARCHAR(64) UNIQUE NOT NULL,
    size BIGINT NOT NULL,
    minio_key VARCHAR(255) UNIQUE NOT NULL,
    reference_count INTEGER DEFAULT 0,
    is_compressed BOOLEAN DEFAULT false,
    original_size BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS object_chunks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    object_id UUID NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
    chunk_id UUID NOT NULL REFERENCES chunks(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(object_id, chunk_index)
);

CREATE INDEX idx_objects_bucket_id ON objects(bucket_id);
CREATE INDEX idx_objects_key ON objects(key);
CREATE INDEX idx_chunks_hash ON chunks(hash);
CREATE INDEX idx_object_chunks_object_id ON object_chunks(object_id);
CREATE INDEX idx_object_chunks_chunk_id ON object_chunks(chunk_id);
CREATE INDEX idx_chunks_compressed ON chunks(is_compressed);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN 
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_buckets_updated_at BEFORE UPDATE ON buckets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_objects_updated_at BEFORE UPDATE ON objects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();


