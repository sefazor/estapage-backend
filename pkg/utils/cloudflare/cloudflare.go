// pkg/utils/cloudflare/cloudflare.go
package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type CloudflareUploadResponse struct {
	Success bool `json:"success"`
	Result  struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"result"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func UploadImage(file *multipart.FileHeader) (string, error) {
	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")

	if accountID == "" || apiToken == "" {
		return "", fmt.Errorf("missing Cloudflare credentials")
	}

	// API endpoint
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/images/v1", accountID)

	// Dosyayı aç
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("could not open file: %v", err)
	}
	defer src.Close()

	// Multipart form oluştur
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Dosyayı forma ekle
	part, err := writer.CreateFormFile("file", filepath.Base(file.Filename))
	if err != nil {
		return "", fmt.Errorf("could not create form file: %v", err)
	}

	if _, err = io.Copy(part, src); err != nil {
		return "", fmt.Errorf("could not copy file: %v", err)
	}
	writer.Close()

	// HTTP isteği oluştur
	request, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", fmt.Errorf("could not create request: %v", err)
	}

	request.Header.Set("Authorization", "Bearer "+apiToken)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	// İsteği gönder
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("could not send request: %v", err)
	}
	defer response.Body.Close()

	// Yanıtı parse et
	var result CloudflareUploadResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("could not decode response: %v", err)
	}

	if !result.Success {
		if len(result.Errors) > 0 {
			return "", fmt.Errorf("cloudflare error: %s", result.Errors[0].Message)
		}
		return "", fmt.Errorf("unknown cloudflare error")
	}

	return result.Result.URL, nil
}

func DeleteImage(cloudflareID string) error {
	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")

	if accountID == "" || apiToken == "" {
		return fmt.Errorf("missing Cloudflare credentials")
	}

	// API endpoint
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/images/v1/%s",
		accountID, cloudflareID)

	// HTTP isteği oluştur
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("could not create request: %v", err)
	}

	request.Header.Set("Authorization", "Bearer "+apiToken)

	// İsteği gönder
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("could not send request: %v", err)
	}
	defer response.Body.Close()

	// Yanıtı parse et
	var result struct {
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return fmt.Errorf("could not decode response: %v", err)
	}

	if !result.Success {
		if len(result.Errors) > 0 {
			return fmt.Errorf("cloudflare error: %s", result.Errors[0].Message)
		}
		return fmt.Errorf("unknown cloudflare error")
	}

	return nil
}
