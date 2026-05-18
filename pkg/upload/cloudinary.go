package upload

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type UploadService interface {
	UploadImage(file multipart.File, folder string) (string, error)
	UploadDocument(file multipart.File, filename string, folder string) (string, error)
}

type cloudinaryService struct {
	cld       *cloudinary.Cloudinary
	cloudName string
	apiKey    string
	apiSecret string
}

func NewUploadService() (UploadService, error) {
	cldURL := os.Getenv("CLOUDINARY_URL")
	if cldURL == "" {
		return nil, fmt.Errorf("CLOUDINARY_URL is not set")
	}

	cld, err := cloudinary.NewFromURL(cldURL)
	if err != nil {
		return nil, err
	}

	parsedURL, err := url.Parse(cldURL)
	if err != nil {
		return nil, err
	}
	apiKey := parsedURL.User.Username()
	apiSecret, _ := parsedURL.User.Password()
	cloudName := strings.TrimPrefix(parsedURL.Host, "/")
	if apiKey == "" || apiSecret == "" || cloudName == "" {
		return nil, fmt.Errorf("CLOUDINARY_URL must include api key, api secret, and cloud name")
	}

	return &cloudinaryService{
		cld:       cld,
		cloudName: cloudName,
		apiKey:    apiKey,
		apiSecret: apiSecret,
	}, nil
}

func (s *cloudinaryService) UploadImage(file multipart.File, folder string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := s.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder:         folder,
		Transformation: "q_auto,f_auto", // Auto-optimize quality and format
	})

	if err != nil {
		return "", err
	}

	return resp.SecureURL, nil
}

func (s *cloudinaryService) UploadDocument(file multipart.File, filename string, folder string) (string, error) {
	bodyReader, bodyWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(bodyWriter)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	go func() {
		defer bodyWriter.Close()
		defer multipartWriter.Close()

		_ = multipartWriter.WriteField("api_key", s.apiKey)
		_ = multipartWriter.WriteField("folder", folder)
		_ = multipartWriter.WriteField("timestamp", timestamp)
		_ = multipartWriter.WriteField("use_filename", "true")
		_ = multipartWriter.WriteField("unique_filename", "true")
		_ = multipartWriter.WriteField("signature", s.signUploadParams(map[string]string{
			"folder":          folder,
			"timestamp":       timestamp,
			"use_filename":    "true",
			"unique_filename": "true",
		}))

		fileWriter, err := multipartWriter.CreateFormFile("file", filename)
		if err != nil {
			_ = bodyWriter.CloseWithError(err)
			return
		}
		if _, err := io.Copy(fileWriter, file); err != nil {
			_ = bodyWriter.CloseWithError(err)
			return
		}
	}()

	requestURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/raw/upload", url.PathEscape(s.cloudName))
	req, err := http.NewRequest(http.MethodPost, requestURL, bodyReader)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var uploadResp struct {
		SecureURL    string `json:"secure_url"`
		ErrorMessage struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		if uploadResp.ErrorMessage.Message != "" {
			return "", fmt.Errorf("cloudinary raw upload failed: %s", uploadResp.ErrorMessage.Message)
		}
		return "", fmt.Errorf("cloudinary raw upload failed with status %d", resp.StatusCode)
	}
	if uploadResp.SecureURL == "" {
		return "", fmt.Errorf("cloudinary raw upload did not return a secure URL")
	}
	return uploadResp.SecureURL, nil
}

func (s *cloudinaryService) signUploadParams(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, key+"="+params[key])
	}
	rawSignature := strings.Join(pairs, "&") + s.apiSecret
	sum := sha1.Sum([]byte(rawSignature))
	return hex.EncodeToString(sum[:])
}
