package image

import (
	"bytes"
	"context"
	"fmt"
	"image/gif"
	"image/png"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/weeb-vip/user-service/internal/storage"
)

type ImageService struct {
	storage storage.Storage
}

func NewImageService(storage storage.Storage) *ImageService {
	return &ImageService{
		storage: storage,
	}
}

func (s *ImageService) UploadProfileImage(ctx context.Context, userID string, file graphql.Upload) (string, error) {
	// Read file content
	buf := bytes.NewBuffer(nil)
	_, err := io.Copy(buf, file.File)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Get file extension
	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = ".jpg" // default extension
	}

	// Validate file extension - only allow image types
	allowedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	isValidExt := false
	for _, allowedExt := range allowedExts {
		if strings.EqualFold(ext, allowedExt) {
			isValidExt = true
			break
		}
	}
	if !isValidExt {
		return "", fmt.Errorf("invalid file extension: %s", ext)
	}

	// Process the image data
	processedData, processedExt, err := s.processImage(buf.Bytes(), ext)
	if err != nil {
		return "", fmt.Errorf("failed to process image: %w", err)
	}

	// Use a consistent filename pattern for each user
	// Include timestamp with milliseconds to avoid collisions and for cache busting
	timestamp := time.Now().Format("20060102150405.000")
	// Replace dots in timestamp to avoid issues with file extensions
	timestamp = strings.ReplaceAll(timestamp, ".", "")
	filename := fmt.Sprintf("profiles/%s/profile_%s%s", userID, timestamp, processedExt)

	// Upload to MinIO
	err = s.storage.Put(ctx, processedData, filename)
	if err != nil {
		return "", fmt.Errorf("failed to upload to storage: %w", err)
	}

	// Return the path (URL will be constructed based on your MinIO configuration)
	return filename, nil
}

// processImage handles image processing, converting GIFs to still images
func (s *ImageService) processImage(data []byte, ext string) ([]byte, string, error) {
	// If it's a GIF, convert to PNG (still image)
	if strings.EqualFold(ext, ".gif") {
		return s.convertGifToStill(data)
	}

	// For other image types, return as-is
	return data, ext, nil
}

// convertGifToStill converts a GIF to a still PNG image (first frame)
func (s *ImageService) convertGifToStill(gifData []byte) ([]byte, string, error) {
	// Decode the GIF
	reader := bytes.NewReader(gifData)
	gifImage, err := gif.DecodeAll(reader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode GIF: %w", err)
	}

	// Check if GIF has frames
	if len(gifImage.Image) == 0 {
		return nil, "", fmt.Errorf("GIF has no frames")
	}

	// Get the first frame
	firstFrame := gifImage.Image[0]

	// Convert to PNG
	var buf bytes.Buffer
	err = png.Encode(&buf, firstFrame)
	if err != nil {
		return nil, "", fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf.Bytes(), ".png", nil
}

func (s *ImageService) DeleteProfileImage(ctx context.Context, imagePath string) error {
	if imagePath == "" {
		return nil
	}

	err := s.storage.Delete(ctx, imagePath)
	if err != nil {
		return fmt.Errorf("failed to delete image from storage: %w", err)
	}

	return nil
}