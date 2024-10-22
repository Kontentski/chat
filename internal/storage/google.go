package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"net/http"

	"path/filepath"
	"strings"

	buck "cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

type BucketStorage interface {
	UploadFileToBucket(file io.Reader, originalFileName, filePath string, c context.Context) (string, error)
	GenerateSignedURL(filePath string) (string, error)
}

type GoogleUpload struct {
}

func NewStorageClient() (*buck.Client, error) {
	fmt.Printf("new storage client\n")
	client, err := buck.NewClient(context.Background(), option.WithCredentialsFile("/home/kontentski/programming/chat/KEY_S3.json"))

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

func (GoogleUpload) UploadFileToBucket(file io.Reader, originalFileName, filePath string, c context.Context) (string, error) {
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
	writer.ContentDisposition = fmt.Sprintf("inline; filename=\"%s\"", originalFileName)
	writer.Metadata = map[string]string{
		"originalFilename": originalFileName,
	}

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

	return filePath, nil
}

func (GoogleUpload) GenerateSignedURL(filePath string) (string, error) {
	fmt.Printf("Generating signed URL for filePath: %s\n", filePath)

	ctx := context.Background()
	client, err := buck.NewClient(ctx, option.WithCredentialsFile("/home/kontentski/programming/chat/KEY_S3.json"))
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
	default:
		return "application/octet-stream"
	}
}
