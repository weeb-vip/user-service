package image

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/weeb-vip/user-service/internal/storage"
	"golang.org/x/image/draw"
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
	
	// Get base filename without extension for creating multiple versions
	baseFilename := fmt.Sprintf("profiles/%s/profile_%s", userID, timestamp)
	originalFilename := baseFilename + processedExt

	// Upload original image
	err = s.storage.Put(ctx, processedData, originalFilename)
	if err != nil {
		return "", fmt.Errorf("failed to upload original image to storage: %w", err)
	}

	// Generate and upload thumbnails
	err = s.generateAndUploadThumbnails(ctx, processedData, baseFilename, processedExt)
	if err != nil {
		// If thumbnail generation fails, delete the original and return error
		_ = s.storage.Delete(ctx, originalFilename)
		return "", fmt.Errorf("failed to generate thumbnails: %w", err)
	}

	// Return the original image path
	return originalFilename, nil
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

// generateAndUploadThumbnails creates 32x32 and 64x64 thumbnails and uploads them
func (s *ImageService) generateAndUploadThumbnails(ctx context.Context, imageData []byte, baseFilename, ext string) error {
	// Decode the original image
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return fmt.Errorf("failed to decode image for thumbnails: %w", err)
	}

	// Generate 32x32 thumbnail
	thumb32, err := s.resizeImage(img, 32, 32)
	if err != nil {
		return fmt.Errorf("failed to create 32x32 thumbnail: %w", err)
	}

	// Generate 64x64 thumbnail
	thumb64, err := s.resizeImage(img, 64, 64)
	if err != nil {
		return fmt.Errorf("failed to create 64x64 thumbnail: %w", err)
	}

	// Encode and upload 32x32 thumbnail
	thumb32Data, err := s.encodeImage(thumb32, format)
	if err != nil {
		return fmt.Errorf("failed to encode 32x32 thumbnail: %w", err)
	}
	
	thumb32Filename := baseFilename + "_32" + ext
	err = s.storage.Put(ctx, thumb32Data, thumb32Filename)
	if err != nil {
		return fmt.Errorf("failed to upload 32x32 thumbnail: %w", err)
	}

	// Encode and upload 64x64 thumbnail
	thumb64Data, err := s.encodeImage(thumb64, format)
	if err != nil {
		return fmt.Errorf("failed to encode 64x64 thumbnail: %w", err)
	}
	
	thumb64Filename := baseFilename + "_64" + ext
	err = s.storage.Put(ctx, thumb64Data, thumb64Filename)
	if err != nil {
		// If 64x64 upload fails, try to clean up the 32x32 thumbnail
		_ = s.storage.Delete(ctx, thumb32Filename)
		return fmt.Errorf("failed to upload 64x64 thumbnail: %w", err)
	}

	return nil
}

// resizeImage resizes an image to the specified dimensions
func (s *ImageService) resizeImage(src image.Image, width, height int) (image.Image, error) {
	// Create a new image with the target size
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Use BiLinear scaling for good quality thumbnails
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	
	return dst, nil
}

// encodeImage encodes an image based on the original format
func (s *ImageService) encodeImage(img image.Image, format string) ([]byte, error) {
	var buf bytes.Buffer
	
	switch format {
	case "jpeg":
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
		if err != nil {
			return nil, fmt.Errorf("failed to encode JPEG: %w", err)
		}
	case "png":
		err := png.Encode(&buf, img)
		if err != nil {
			return nil, fmt.Errorf("failed to encode PNG: %w", err)
		}
	default:
		// Default to PNG for other formats
		err := png.Encode(&buf, img)
		if err != nil {
			return nil, fmt.Errorf("failed to encode as PNG: %w", err)
		}
	}
	
	return buf.Bytes(), nil
}

func (s *ImageService) DeleteProfileImage(ctx context.Context, imagePath string) error {
	if imagePath == "" {
		return nil
	}

	// Delete original image
	err := s.storage.Delete(ctx, imagePath)
	if err != nil {
		return fmt.Errorf("failed to delete original image from storage: %w", err)
	}

	// Generate thumbnail paths and delete them
	// Extract base filename and extension
	ext := filepath.Ext(imagePath)
	baseWithoutExt := strings.TrimSuffix(imagePath, ext)
	
	// Delete 32x32 thumbnail
	thumb32Path := baseWithoutExt + "_32" + ext
	_ = s.storage.Delete(ctx, thumb32Path) // Don't fail if thumbnail doesn't exist
	
	// Delete 64x64 thumbnail
	thumb64Path := baseWithoutExt + "_64" + ext
	_ = s.storage.Delete(ctx, thumb64Path) // Don't fail if thumbnail doesn't exist

	return nil
}