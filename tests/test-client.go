package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const baseURL = "http://localhost:8080/api/v1"

type CreateBucketRequest struct {
	Name string `json:"name"`
}

type Bucket struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Object struct {
	ID          string    `json:"id"`
	BucketID    string    `json:"bucket_id"`
	Key         string    `json:"key"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	ETag        string    `json:"etag"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ListObjectsResponse struct {
	Objects []Object `json:"objects"`
}

func main() {
	fmt.Println("Testing Object Storage API")
	fmt.Println("=====================================\n")

	// Test 1: Health Check
	fmt.Println("Test 1: Health Check")
	if err := testHealth(); err != nil {
		fmt.Printf("Failed: %v\n\n", err)
	} else {
		fmt.Println("Passed\n")
	}

	// Test 2: Create Bucket
	bucketName := fmt.Sprintf("test-bucket-%d", time.Now().Unix())
	fmt.Printf("Test 2: Create Bucket (%s)\n", bucketName)
	if err := testCreateBucket(bucketName); err != nil {
		fmt.Printf("Failed: %v\n\n", err)
		return
	}
	fmt.Println("Passed\n")

	// Test 3: List Buckets
	fmt.Println("Test 3: List Buckets")
	if err := testListBuckets(); err != nil {
		fmt.Printf("Failed: %v\n\n", err)
	} else {
		fmt.Println("Passed\n")
	}

	// Test 4: Upload Small File
	fmt.Println("Test 4: Upload Small File")
	content := []byte("Hello from Go! This is a test file.")
	if err := testUploadFile(bucketName, "test.txt", content); err != nil {
		fmt.Printf("Failed: %v\n\n", err)
		return
	}
	fmt.Println("Passed\n")

	// Test 5: Download File
	fmt.Println("Test 5: Download File")
	downloaded, err := testDownloadFile(bucketName, "test.txt")
	if err != nil {
		fmt.Printf("Failed: %v\n\n", err)
	} else if string(downloaded) != string(content) {
		fmt.Printf("Content mismatch! Expected: %s, Got: %s\n\n", content, downloaded)
	} else {
		fmt.Printf("Passed (Downloaded: %s)\n\n", downloaded)
	}

	// Test 6: Upload Large File (Tests Chunking)
	fmt.Println("Test 6: Upload Large File (8MB - tests chunking)")
	largeContent := make([]byte, 8*1024*1024) // 8MB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	if err := testUploadFile(bucketName, "large.bin", largeContent); err != nil {
		fmt.Printf("Failed: %v\n\n", err)
	} else {
		fmt.Println("Passed\n")
	}

	// Test 7: Deduplication Test
	fmt.Println("Test 7: Test Deduplication")
	dupContent := []byte("Duplicate content for testing")
	if err := testUploadFile(bucketName, "file1.txt", dupContent); err != nil {
		fmt.Printf("Failed: %v\n\n", err)
	} else if err := testUploadFile(bucketName, "file2.txt", dupContent); err != nil {
		fmt.Printf("Failed: %v\n\n", err)
	} else {
		fmt.Println("Passed (Check database for reference_count = 2)\n")
	}

	// Test 8: List Objects
	fmt.Println("Test 8: List Objects")
	if err := testListObjects(bucketName); err != nil {
		fmt.Printf("Failed: %v\n\n", err)
	} else {
		fmt.Println("Passed\n")
	}

	// Test 9: Delete Object
	fmt.Println("Test 9: Delete Object")
	if err := testDeleteObject(bucketName, "test.txt"); err != nil {
		fmt.Printf("Failed: %v\n\n", err)
	} else {
		fmt.Println("Passed\n")
	}

	// Test 10: Delete Bucket
	fmt.Println("Test 10: Delete Bucket")
	if err := testDeleteBucket(bucketName); err != nil {
		fmt.Printf("Failed: %v\n\n", err)
	} else {
		fmt.Println("Passed\n")
	}

	// Test 11: Compression test
	fmt.Println("Test 11: Compression test")
	if err := testCompression(); err != nil {
		fmt.Printf("Failed: %v\n\n", err)
	} else {
		fmt.Println("Passed\n")
	}

	fmt.Println("=====================================")
	fmt.Println("All tests completed!")

}

// Test functions

func testHealth() error {
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("   Response: %s\n", body)
	return nil
}

func testCreateBucket(name string) error {
	reqBody := CreateBucketRequest{Name: name}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := http.Post(baseURL+"/buckets", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	var bucket Bucket
	if err := json.NewDecoder(resp.Body).Decode(&bucket); err != nil {
		return err
	}

	fmt.Printf("   Bucket created: %s (ID: %s)\n", bucket.Name, bucket.ID)
	return nil
}

func testListBuckets() error {
	resp, err := http.Get(baseURL + "/buckets")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Buckets []Bucket `json:"buckets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	fmt.Printf("   Found %d bucket(s)\n", len(result.Buckets))
	for _, bucket := range result.Buckets {
		fmt.Printf("   - %s\n", bucket.Name)
	}
	return nil
}

func testUploadFile(bucket, key string, content []byte) error {
	url := fmt.Sprintf("%s/buckets/%s/objects/%s", baseURL, bucket, key)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(content))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	fmt.Printf("   Uploaded: %s (%v bytes, ETag: %s)\n", key, result["size"], result["etag"])
	return nil
}

func testDownloadFile(bucket, key string) ([]byte, error) {
	url := fmt.Sprintf("%s/buckets/%s/objects/%s", baseURL, bucket, key)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected 200, got %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func testListObjects(bucket string) error {
	url := fmt.Sprintf("%s/buckets/%s/objects", baseURL, bucket)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result ListObjectsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	fmt.Printf("   Found %d object(s)\n", len(result.Objects))
	for _, obj := range result.Objects {
		fmt.Printf("   - %s (%d bytes)\n", obj.Key, obj.Size)
	}
	return nil
}

func testDeleteObject(bucket, key string) error {
	url := fmt.Sprintf("%s/buckets/%s/objects/%s", baseURL, bucket, key)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("expected 204, got %d: %s", resp.StatusCode, body)
	}

	fmt.Printf("   Deleted: %s\n", key)
	return nil
}

func testDeleteBucket(bucket string) error {
	url := fmt.Sprintf("%s/buckets/%s", baseURL, bucket)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("expected 204, got %d: %s", resp.StatusCode, body)
	}

	fmt.Printf("   Deleted bucket: %s\n", bucket)
	return nil
}

func testCompression() error {
	compBucket := "compression-test"
	reqBody := CreateBucketRequest{Name: compBucket}
	jsonData, _ := json.Marshal(reqBody)
	resp, _ := http.Post(baseURL+"/buckets", "application/json", bytes.NewBuffer(jsonData))
	resp.Body.Close()

	//repeated test for high compressibility
	compressibleContent := []byte("This is repeated text that compresses very well! " +
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit. " +
		"The quick brown fox jumps over the lazy dog. ")

	//Repeats the content so we get to 5KB to compress it
	fullContent := bytes.Repeat(compressibleContent, 30)

	fmt.Printf("Original size: %d bytes\n", len(fullContent))

	if err := testUploadFile(compBucket, "compressible.txt", fullContent); err != nil {
		return err
	}

	randomContent := make([]byte, 5000)
	for i := range randomContent {
		randomContent[i] = byte(i % 256)
	}
	if err := testUploadFile(compBucket, "random.bin", randomContent); err != nil {
		return err
	}

	fmt.Println("Uploaded compressible (text) and non-compressible (binary) files")
	fmt.Println("Check compression stats:")
	fmt.Println(`docker exec -it go-object-storage-postgres-1 psql -U objectstore -c "SELECT is_compressed, COUNT(*) as chunks, pg_size_pretty(SUM(size)) as stored, pg_size_pretty(SUM(original_size)) as original FROM chunks GROUP BY is_compressed;"`)

	// Clean up
	testDeleteObject(compBucket, "compressible.txt")
	testDeleteObject(compBucket, "random.bin")
	testDeleteBucket(compBucket)

	return nil
}

func init() {
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		fmt.Println("Cannot connect to API. Make sure it's running:")
		fmt.Println("   docker-compose up -d")
		os.Exit(1)
	}
	resp.Body.Close()
}
