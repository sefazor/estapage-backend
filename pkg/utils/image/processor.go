// pkg/utils/image/processor.go
package image

import (
	"errors"
	"path/filepath"
	"strings"
)

const (
	MaxImageSize = 10 * 1024 * 1024 // 10MB
)

var (
	ErrFileSize = errors.New("file size exceeds limit")
	ErrFileType = errors.New("invalid file type")

	AllowedImageTypes = map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}
)

// ValidateImage dosya kontrollerini yapar
func ValidateImage(filename string, size int64) error {
	if size > MaxImageSize {
		return ErrFileSize
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if !AllowedImageTypes[ext] {
		return ErrFileType
	}

	return nil
}
