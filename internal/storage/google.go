package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"net/http"

	"path/filepath"
	"strings"

	buck "cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

type BucketStorage interface {
	UploadFileToBucket(file multipart.File, originalFileName, filePath string, c context.Context) (string, error)
	GenerateSignedURL(filePath string) (string, error)
}

type GoogleUpload struct {
	
}

func NewStorageClient() (*buck.Client, error) {
	fmt.Printf("new storage client\n")
	client, err := buck.NewClient(context.Background(), option.WithCredentialsFile("/home/kontentski/Documents/programing/github/chat/KEY_S3.json"))

	if err != nil {
		fmt.Printf("Error creating storage client: %v\n", err) // Log the error
		return nil, fmt.Errorf("failed to create storage client: %v", err)
	}
	fmt.Printf("client: %v\n", client)
	return client, nil
}

var (
	bucketName = "chat-app-bucket-1"
)

func (GoogleUpload) UploadFileToBucket(file multipart.File, originalFileName, filePath string, c context.Context) (string, error) {
	client, err := NewStorageClient()
	if err != nil {
		return "", fmt.Errorf("failed to create storage client: %v", err)
	}
	defer client.Close()

	fmt.Printf("Starting upload of file: %s to path: %s\n", originalFileName, filePath)

	// Determine the content type
	contentType := getContentType(originalFileName)

	// Upload the file with write permissions
	writer := client.Bucket(bucketName).Object(filePath).NewWriter(c)
	writer.ContentType = contentType                                                     // Set the correct content type
	writer.ContentDisposition = fmt.Sprintf("inline; filename=\"%s\"", originalFileName) // Change to inline
	writer.Metadata = map[string]string{
		"originalFilename": originalFileName,
	}

	// Read file into a buffer
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, file)
	if err != nil {
		return "", fmt.Errorf("failed to read file content: %v", err)
	}
	fmt.Println("File content read into buffer successfully")

	// Write the file to the bucket
	_, err = writer.Write(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to write to bucket: %v", err)
	}
	fmt.Println("File written to bucket successfully")

	// Close the writer to finalize the upload
	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close writer: %v", err)
	}
	fmt.Println("Writer closed successfully")

	// Generate a signed URL (valid for 15 minutes)

	return filePath, nil
}

func (GoogleUpload) GenerateSignedURL(filePath string) (string, error) {
	fmt.Printf("Generating signed URL for filePath: %s\n", filePath)

	ctx := context.Background()
	client, err := buck.NewClient(ctx, option.WithCredentialsFile("/home/kontentski/Documents/programing/github/chat/KEY_S3.json"))
	if err != nil {
		return "", fmt.Errorf("failed to create storage client: %v", err)
	}
	defer client.Close()

	expiration := time.Now().Add(15 * time.Minute)
	url, err := client.Bucket(bucketName).SignedURL(filePath, &buck.SignedURLOptions{
		Method:  http.MethodGet,
		Expires: expiration,
		QueryParameters: map[string][]string{
			"response-content-disposition": {"inline"},
			"response-content-type":        {getContentType(filePath)},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %v", err)
	}

	fmt.Printf("Signed URL generated: %s\n", url)
	return url, nil
}

// Add this helper function
func getContentType(filename string) string {
	ext := filepath.Ext(filename)
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".mp4":
		return "video/mp4"
	// Add more types as needed
	default:
		return "application/octet-stream"
	}
}

/* func NewMediaStorage(bucketName string) (*MediaStorage, error) {
	// Initialize the Google Cloud Storage client
	client, err := storage.NewClient(context.Background(), option.WithCredentialsFile("/home/kontentski/Documents/programing/github/chat/KEY_S3.json"))
	if err != nil {
		return nil, fmt.Errorf("Failed to create storage client: %v", err)
	}

	return &MediaStorage{
		BucketName: bucketName,
		Client:     client,
	}, nil
}

// UploadFile uploads the file to Google Cloud Storage
func (m *MediaStorage) UploadFile(file multipart.File, fileHeader *multipart.FileHeader, filePath string) (string, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	// Define the object (file) that will be uploaded
	wc := m.Client.Bucket(m.BucketName).Object(filePath).NewWriter(ctx)

	// Set the content type (e.g., image/jpeg, video/mp4)
	wc.ContentType = fileHeader.Header.Get("Content-Type")

	// Copy the file's content to the GCS writer
	if _, err := io.Copy(wc, file); err != nil {
		return "", fmt.Errorf("Failed to copy file: %v", err)
	}

	// Close the writer
	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("Failed to close writer: %v", err)
	}

	// Return the public URL of the uploaded file
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", m.BucketName, filePath), nil
}
*/
