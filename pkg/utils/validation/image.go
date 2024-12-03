// pkg/utils/validation/image.go
package validation

import (
	"errors"
	"mime/multipart"
	"path/filepath"
	"strings"
)

var (
	ErrFileSize     = errors.New("file size exceeds limit of 10MB")
	ErrFileType     = errors.New("invalid file type. Allowed types: JPG, PNG, WEBP")
	ErrFileRequired = errors.New("no file provided")
)

const MaxImageSize = 10 * 1024 * 1024 // 10MB

var AllowedImageTypes = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
}

func ValidateImage(file *multipart.FileHeader) error {
	if file == nil {
		return ErrFileRequired
	}

	// Boyut kontrolü
	if file.Size > MaxImageSize {
		return ErrFileSize
	}

	// Tip kontrolü
	ext := filepath.Ext(strings.ToLower(file.Filename))
	if !AllowedImageTypes[ext] {
		return ErrFileType
	}

	return nil
}
