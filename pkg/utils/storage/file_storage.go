package storage

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	MaxFileSize = 5 * 1024 * 1024 // 5MB
	BucketName  = "estepage-images"
	Region      = "eu-central-1"
)

var (
	s3Client     *s3.Client
	allowedTypes = map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
	}
)

func InitStorage() error {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(Region),
	)
	if err != nil {
		return fmt.Errorf("unable to load SDK config: %v", err)
	}

	s3Client = s3.NewFromConfig(cfg)
	return nil
}

// UploadImage resmi kontrol eder, optimize eder ve yükler
func UploadImage(file *multipart.FileHeader, userID uint, propertyID uint) (string, error) {
	// Dosya boyutu kontrolü
	if file.Size > MaxFileSize {
		return "", fmt.Errorf("file size too large. Maximum size is %d bytes", MaxFileSize)
	}

	// Dosya tipini kontrol et
	contentType := file.Header.Get("Content-Type")
	if !allowedTypes[contentType] {
		return "", fmt.Errorf("invalid file type. Allowed types are: jpeg, png")
	}

	// Dosyayı aç
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// Resmi decode et
	img, format, err := image.Decode(src)
	if err != nil {
		return "", fmt.Errorf("could not decode image: %v", err)
	}

	// Buffer oluştur
	buf := new(bytes.Buffer)

	// Resmi optimize et ve encode et
	switch format {
	case "jpeg":
		err = jpeg.Encode(buf, img, &jpeg.Options{Quality: 85})
	case "png":
		err = png.Encode(buf, img)
	default:
		return "", fmt.Errorf("unsupported image format: %s", format)
	}

	if err != nil {
		return "", fmt.Errorf("could not encode image: %v", err)
	}

	// Dosya adını oluştur: user_id/property_id/timestamp_original_name
	fileName := fmt.Sprintf("%d/%d/%d_%s",
		userID,
		propertyID,
		time.Now().Unix(),
		filepath.Base(file.Filename),
	)

	// S3'e yükle
	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(BucketName),
		Key:         aws.String(fileName),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return "", fmt.Errorf("could not upload to S3: %v", err)
	}

	// Public URL döndür
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", BucketName, Region, fileName), nil
}

// DeleteImage S3'ten resmi siler
func DeleteImage(imageURL string) error {
	// URL'den key'i çıkar
	parts := strings.Split(imageURL, "/")
	key := strings.Join(parts[3:], "/")

	_, err := s3Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(BucketName),
		Key:    aws.String(key),
	})

	return err
}
