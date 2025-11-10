package service

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
)

// determines if the content should be compressed
func shouldCompress(contentType string) bool {
	compressibleTypes := []string{
		"text/",
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-javascript",
		"application/xhtml+xml",
		"application/rss+xml",
		"application/atom+xml",
	}

	for _, ct := range compressibleTypes {
		if strings.HasPrefix(contentType, ct) {
			return true
		}
	}
	return false
}

// compression using the gzip algorithm
func compress(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}

	if _, err := writer.Write(data); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}
