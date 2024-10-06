package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"time"

)



func NewMediaStorage(bucketName string) (*MediaStorage, error) {
	// Initialize the Google Cloud Storage client
	client, err := storage.NewClient(context.Background(), option.WithCredentialsFile("path/to/your/service-account-file.json"))
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
