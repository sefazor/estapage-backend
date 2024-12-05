package cloudflare

import (
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/utils/jwt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/gosimple/slug"
)

// Utility functions
func getObjectKeyFromURL(url string) string {
	return strings.TrimPrefix(url, "https://cdn.estapage.com/")
}

func GetFileNameFromURL(url string) string {
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}

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

// Types
type UploadImageConfig struct {
	File         *multipart.FileHeader
	Username     string
	PropertySlug string
}

type UploadResult struct {
	URL          string
	CloudflareID string
}

type UploadAvatarConfig struct {
	File     *multipart.FileHeader
	Username string
}

// Core functions
func DeleteImage(fullURL string) error {
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

func UploadImage(config UploadImageConfig) (UploadResult, error) {
	safeUsername := slug.Make(config.Username)
	safePropertySlug := slug.Make(config.PropertySlug)

	ext := filepath.Ext(config.File.Filename)
	uniqueID := fmt.Sprintf("%d-%s", time.Now().UnixNano(), uuid.New().String())
	uniqueFilename := uniqueID + ext

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

func UploadAvatar(config UploadAvatarConfig) (string, error) {
	ext := filepath.Ext(config.File.Filename)
	uniqueID := fmt.Sprintf("%d-%s", time.Now().UnixNano(), uuid.New().String())
	uniqueFilename := uniqueID + ext

	objectKey := filepath.Join("users", config.Username, "profile", uniqueFilename)

	client, err := getS3Client()
	if err != nil {
		return "", err
	}

	src, err := config.File.Open()
	if err != nil {
		return "", fmt.Errorf("could not open file: %v", err)
	}
	defer src.Close()

	input := &s3.PutObjectInput{
		Bucket: aws.String(os.Getenv("R2_BUCKET_NAME")),
		Key:    aws.String(objectKey),
		Body:   src,
	}

	_, err = client.PutObject(context.TODO(), input)
	if err != nil {
		return "", fmt.Errorf("could not upload avatar to R2: %v", err)
	}

	return fmt.Sprintf("https://cdn.estapage.com/%s", objectKey), nil
}

func UploadAvatarHandler(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	var user model.User
	if err := database.GetDB().First(&user, claims.UserID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No avatar image provided",
		})
	}

	contentType := file.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File must be an image",
		})
	}

	if file.Size > 5*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File size must be less than 5MB",
		})
	}

	if user.Avatar != "" {
		if err := DeleteImage(user.Avatar); err != nil {
			log.Printf("Error deleting old avatar: %v", err)
		}
	}

	config := UploadAvatarConfig{
		File:     file,
		Username: user.Username,
	}

	avatarURL, err := UploadAvatar(config)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Could not upload avatar: %v", err),
		})
	}

	if err := database.GetDB().Model(&user).Update("avatar", avatarURL).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update avatar in database",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Avatar uploaded successfully",
		"avatar":  avatarURL,
	})
}
