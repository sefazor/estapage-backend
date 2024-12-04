package cloudflare

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/gosimple/slug"
)

func getS3Client() (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			os.Getenv("R2_ACCESS_KEY"),
			os.Getenv("R2_SECRET_KEY"),
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", os.Getenv("R2_ACCOUNT_ID")))
		o.UsePathStyle = true
		o.Region = "auto"
	})

	return client, nil
}

type UploadImageConfig struct {
	File         *multipart.FileHeader
	Username     string
	PropertySlug string
}

type UploadResult struct {
	URL          string
	CloudflareID string
}

func UploadImage(config UploadImageConfig) (UploadResult, error) {

	// Klasör isimlerini URL-safe hale getir
	safeUsername := slug.Make(config.Username)
	safePropertySlug := slug.Make(config.PropertySlug)

	// Unique dosya adı oluştur
	ext := filepath.Ext(config.File.Filename)
	uniqueID := fmt.Sprintf("%d-%s", time.Now().UnixNano(), uuid.New().String())
	uniqueFilename := uniqueID + ext

	// Organize edilmiş ve URL-safe path yapısı
	objectKey := filepath.Join("users", safeUsername, "properties", safePropertySlug, "images", uniqueFilename)

	client, err := getS3Client()
	if err != nil {
		return UploadResult{}, err
	}

	src, err := config.File.Open()
	if err != nil {
		return UploadResult{}, fmt.Errorf("could not open file: %v", err)
	}
	defer src.Close()

	input := &s3.PutObjectInput{
		Bucket: aws.String(os.Getenv("R2_BUCKET_NAME")),
		Key:    aws.String(objectKey),
		Body:   src,
	}

	_, err = client.PutObject(context.TODO(), input)
	if err != nil {
		return UploadResult{}, fmt.Errorf("could not upload file to R2: %v", err)
	}

	return UploadResult{
		URL:          fmt.Sprintf("https://cdn.estapage.com/%s", objectKey),
		CloudflareID: uniqueID,
	}, nil
}

func DeleteImage(fullURL string) error {
	// URL'den object key'i çıkar
	objectKey := getObjectKeyFromURL(fullURL)

	client, err := getS3Client()
	if err != nil {
		return err
	}

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(os.Getenv("R2_BUCKET_NAME")),
		Key:    aws.String(objectKey),
	}

	_, err = client.DeleteObject(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("could not delete file from R2: %v", err)
	}

	return nil
}

// GetFileNameFromURL sadece dosya adını döndürür
func GetFileNameFromURL(url string) string {
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}

// getObjectKeyFromURL tam object key'i döndürür (users/username/properties/slug/images/filename)
func getObjectKeyFromURL(url string) string {
	// https://cdn.estapage.com/ kısmını kaldır
	return strings.TrimPrefix(url, "https://cdn.estapage.com/")
}
