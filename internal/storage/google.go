package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"time"

	"net/http"

	buck "cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

func NewStorageClient() (*buck.Client, error) {
	fmt.Printf("new storage client\n")
	client, err := buck.NewClient(context.Background(), option.WithCredentialsFile("D:/programming/chat/KEY_S3.json"))

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

func (r *UserQuery) UploadFileToBucket(file multipart.File, originalFileName, filePath string, c context.Context) (string, error) {
	client, err := NewStorageClient()
	if err != nil {
		return "", fmt.Errorf("failed to create storage client: %v", err)
	}
	defer client.Close()

	fmt.Printf("Starting upload of file: %s to path: %s\n", originalFileName, filePath)

	// Upload the file to the specified file path in the bucket
	bucket := client.Bucket(bucketName)
	obj := bucket.Object(filePath)

	// Upload the file with write permissions
	writer := obj.NewWriter(c)
	writer.ContentType = "application/octet-stream"
	writer.ContentDisposition = fmt.Sprintf("attachment; filename=\"%s\"", originalFileName)
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
	expiration := time.Now().Add(15 * time.Minute)
	url, err := generateSignedURL(filePath, originalFileName, expiration)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %v", err)
	}
	fmt.Printf("Generated signed URL: %s\n", url)

	return url, nil
}

func generateSignedURL(filePath string, originalFileName string, expiration time.Time) (string, error) {
	fmt.Printf("Generating signed URL for filePath: %s with expiration: %s\n", filePath, expiration)

	// Load the service account credentials JSON file
	serviceAccountKeyFile := "D:/programming/chat/KEY_S3.json"
	creds, err := os.ReadFile(serviceAccountKeyFile)
	if err != nil {
		return "", fmt.Errorf("failed to read service account key file: %v", err)
	}

	// Create a new context
	ctx := context.Background()

	// Create a new storage client using the credentials
	client, err := buck.NewClient(ctx, option.WithCredentialsJSON(creds))
	if err != nil {
		return "", fmt.Errorf("failed to create storage client: %v", err)
	}
	defer client.Close()

	// Generate the signed URL
	url, err := client.Bucket(bucketName).SignedURL(filePath, &buck.SignedURLOptions{
		Method:  http.MethodGet,
		Expires: expiration,
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %v", err)
	}

	// Append the Content-Disposition to the URL query parameters to set the original file name
	signedURL := fmt.Sprintf("%s&response-content-disposition=attachment; filename=\"%s\"", url, originalFileName)
	fmt.Printf("Signed URL generated: %s\n", signedURL)

	return signedURL, nil
}

/* func NewMediaStorage(bucketName string) (*MediaStorage, error) {
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
*/
