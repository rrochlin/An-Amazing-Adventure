package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// S3Client wraps the AWS S3 client with our configuration
type S3Client struct {
	client     *s3.Client
	bucketName string
	region     string
}

// NewS3Client creates a new S3 client configured with IAM user credentials
func NewS3Client(ctx context.Context) (*S3Client, error) {
	bucketName := os.Getenv("S3_BUCKET_NAME")
	if bucketName == "" {
		return nil, fmt.Errorf("S3_BUCKET_NAME environment variable not set")
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-west-2" // Default to same region as DynamoDB
	}

	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	if accessKeyID == "" || secretAccessKey == "" {
		return nil, fmt.Errorf("AWS credentials not set (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)")
	}

	// Create AWS config with explicit credentials
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretAccessKey,
			"", // session token not needed for IAM user
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &S3Client{
		client:     s3.NewFromConfig(cfg),
		bucketName: bucketName,
		region:     region,
	}, nil
}

// UploadMapImage uploads a map image to S3 and returns the public URL
func (s *S3Client) UploadMapImage(ctx context.Context, sessionID uuid.UUID, imageType string, imageData []byte) (string, error) {
	// Determine file extension based on image type
	ext := ".png"
	contentType := "image/png"
	if imageType == "jpeg" || imageType == "jpg" {
		ext = ".jpg"
		contentType = "image/jpeg"
	}

	// Generate S3 key (path) for the image
	// Format: {sessionUUID}/world-map.png or {sessionUUID}/zones/{zoneName}.png
	var key string
	switch imageType {
	case "world-map":
		key = fmt.Sprintf("%s/world-map%s", sessionID.String(), ext)
	default:
		// For zone maps, imageType will be "zone-{zoneName}"
		key = fmt.Sprintf("%s/%s%s", sessionID.String(), imageType, ext)
	}

	// Upload to S3
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(imageData),
		ContentType: aws.String(contentType),
		// ACL will be set via bucket policy, not here
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload image to S3: %w", err)
	}

	// Generate public URL
	// Format: https://{bucket}.s3.{region}.amazonaws.com/{key}
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucketName, s.region, key)

	return url, nil
}

// DeleteMapImages deletes all map images for a game session
func (s *S3Client) DeleteMapImages(ctx context.Context, sessionID uuid.UUID) error {
	// List all objects with the session prefix
	prefix := fmt.Sprintf("%s/", sessionID.String())

	listOutput, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return fmt.Errorf("failed to list objects for deletion: %w", err)
	}

	// Delete each object
	for _, obj := range listOutput.Contents {
		_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucketName),
			Key:    obj.Key,
		})
		if err != nil {
			return fmt.Errorf("failed to delete object %s: %w", *obj.Key, err)
		}
	}

	return nil
}

// GetMapImageURL returns the public URL for a map image without uploading
func (s *S3Client) GetMapImageURL(sessionID uuid.UUID, imageType string) string {
	ext := ".png"
	var key string

	switch imageType {
	case "world-map":
		key = fmt.Sprintf("%s/world-map%s", sessionID.String(), ext)
	default:
		key = fmt.Sprintf("%s/%s%s", sessionID.String(), imageType, ext)
	}

	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucketName, s.region, key)
}

// SaveImageLocally saves image data to a local file (for debugging)
func SaveImageLocally(sessionID uuid.UUID, imageType string, imageData []byte) error {
	dir := filepath.Join(".", "debug-images", sessionID.String())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create debug directory: %w", err)
	}

	filename := filepath.Join(dir, imageType+".png")
	if err := os.WriteFile(filename, imageData, 0644); err != nil {
		return fmt.Errorf("failed to write debug image: %w", err)
	}

	fmt.Printf("Debug: Saved image to %s\n", filename)
	return nil
}
