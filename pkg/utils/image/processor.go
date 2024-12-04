package image

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"mime/multipart"

	"github.com/chai2010/webp"
)

const (
	MaxImageSize = 10 * 1024 * 1024 // 10MB
)

var (
	AllowedImageTypes = map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
	}
)

func ProcessImage(file *multipart.FileHeader) (*bytes.Buffer, string, error) {
	// Dosyayı aç
	src, err := file.Open()
	if err != nil {
		return nil, "", fmt.Errorf("could not open file: %v", err)
	}
	defer src.Close()

	// Resmi decode et
	img, format, err := image.Decode(src)
	if err != nil {
		return nil, "", fmt.Errorf("could not decode image: %v", err)
	}

	// Buffer oluştur
	buf := new(bytes.Buffer)

	// Resmi optimize et ve encode et
	switch format {
	case "jpeg":
		err = jpeg.Encode(buf, img, &jpeg.Options{Quality: 85})
	case "png":
		err = png.Encode(buf, img)
	case "webp":
		err = webp.Encode(buf, img, &webp.Options{Lossless: false, Quality: 85})
	default:
		return nil, "", fmt.Errorf("unsupported image format: %s", format)
	}

	if err != nil {
		return nil, "", fmt.Errorf("could not encode image: %v", err)
	}

	// Content type belirle
	contentType := fmt.Sprintf("image/%s", format)

	return buf, contentType, nil
}
