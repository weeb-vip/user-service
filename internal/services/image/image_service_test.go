package image

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weeb-vip/user-service/mocks"
	"go.uber.org/mock/gomock"
)

func TestNewImageService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	service := NewImageService(mockStorage)

	assert.NotNil(t, service)
	assert.Equal(t, mockStorage, service.storage)
}

func TestImageService_UploadProfileImage(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		filename      string
		fileContent   string
		setupMock     func(*mocks.MockStorage)
		expectedError string
		validatePath  func(t *testing.T, path string)
	}{
		{
			name:        "successful upload with jpg",
			userID:      "user123",
			filename:    "profile.jpg",
			fileContent: "fake image content",
			setupMock: func(ms *mocks.MockStorage) {
				ms.EXPECT().
					Put(gomock.Any(), []byte("fake image content"), gomock.Any()).
					DoAndReturn(func(ctx context.Context, data []byte, path string) error {
						// Validate the path format
						assert.Contains(t, path, "profiles/user123/")
						assert.True(t, strings.HasSuffix(path, ".jpg"))
						return nil
					})
			},
			validatePath: func(t *testing.T, path string) {
				assert.Contains(t, path, "profiles/user123/")
				assert.True(t, strings.HasSuffix(path, ".jpg"))
			},
		},
		{
			name:        "successful upload with png",
			userID:      "user456",
			filename:    "avatar.png",
			fileContent: "fake png content",
			setupMock: func(ms *mocks.MockStorage) {
				ms.EXPECT().
					Put(gomock.Any(), []byte("fake png content"), gomock.Any()).
					Return(nil)
			},
			validatePath: func(t *testing.T, path string) {
				assert.Contains(t, path, "profiles/user456/")
				assert.True(t, strings.HasSuffix(path, ".png"))
			},
		},
		{
			name:        "successful upload with uppercase extension",
			userID:      "user789",
			filename:    "photo.PNG",
			fileContent: "fake png content",
			setupMock: func(ms *mocks.MockStorage) {
				ms.EXPECT().
					Put(gomock.Any(), []byte("fake png content"), gomock.Any()).
					Return(nil)
			},
			validatePath: func(t *testing.T, path string) {
				assert.Contains(t, path, "profiles/user789/")
				assert.True(t, strings.HasSuffix(path, ".PNG"))
			},
		},
		{
			name:        "successful upload with gif",
			userID:      "user111",
			filename:    "animated.gif",
			fileContent: "fake gif content",
			setupMock: func(ms *mocks.MockStorage) {
				ms.EXPECT().
					Put(gomock.Any(), []byte("fake gif content"), gomock.Any()).
					Return(nil)
			},
			validatePath: func(t *testing.T, path string) {
				assert.Contains(t, path, "profiles/user111/")
				assert.True(t, strings.HasSuffix(path, ".gif"))
			},
		},
		{
			name:        "successful upload with webp",
			userID:      "user222",
			filename:    "modern.webp",
			fileContent: "fake webp content",
			setupMock: func(ms *mocks.MockStorage) {
				ms.EXPECT().
					Put(gomock.Any(), []byte("fake webp content"), gomock.Any()).
					Return(nil)
			},
			validatePath: func(t *testing.T, path string) {
				assert.Contains(t, path, "profiles/user222/")
				assert.True(t, strings.HasSuffix(path, ".webp"))
			},
		},
		{
			name:        "file without extension defaults to jpg",
			userID:      "user333",
			filename:    "noextension",
			fileContent: "fake content",
			setupMock: func(ms *mocks.MockStorage) {
				ms.EXPECT().
					Put(gomock.Any(), []byte("fake content"), gomock.Any()).
					Return(nil)
			},
			validatePath: func(t *testing.T, path string) {
				assert.Contains(t, path, "profiles/user333/")
				assert.True(t, strings.HasSuffix(path, ".jpg"))
			},
		},
		{
			name:          "invalid file extension",
			userID:        "user444",
			filename:      "document.pdf",
			fileContent:   "fake pdf content",
			setupMock:     func(ms *mocks.MockStorage) {},
			expectedError: "invalid file extension: .pdf",
		},
		{
			name:          "invalid executable extension",
			userID:        "user555",
			filename:      "malware.exe",
			fileContent:   "fake exe content",
			setupMock:     func(ms *mocks.MockStorage) {},
			expectedError: "invalid file extension: .exe",
		},
		{
			name:        "storage upload fails",
			userID:      "user666",
			filename:    "profile.jpg",
			fileContent: "fake content",
			setupMock: func(ms *mocks.MockStorage) {
				ms.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("storage error"))
			},
			expectedError: "failed to upload to storage: storage error",
		},
		{
			name:        "empty file content",
			userID:      "user777",
			filename:    "empty.jpg",
			fileContent: "",
			setupMock: func(ms *mocks.MockStorage) {
				ms.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, data []byte, path string) error {
						// Validate empty content
						assert.Empty(t, data)
						return nil
					})
			},
			validatePath: func(t *testing.T, path string) {
				assert.Contains(t, path, "profiles/user777/")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStorage := mocks.NewMockStorage(ctrl)
			tt.setupMock(mockStorage)

			service := NewImageService(mockStorage)

			// Create a mock upload
			upload := graphql.Upload{
				File:     strings.NewReader(tt.fileContent),
				Filename: tt.filename,
			}

			ctx := context.Background()
			path, err := service.UploadProfileImage(ctx, tt.userID, upload)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Empty(t, path)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, path)
				if tt.validatePath != nil {
					tt.validatePath(t, path)
				}
			}
		})
	}
}

func TestImageService_DeleteProfileImage(t *testing.T) {
	tests := []struct {
		name          string
		imagePath     string
		setupMock     func(*mocks.MockStorage)
		expectedError string
	}{
		{
			name:      "successful deletion",
			imagePath: "profiles/user123/20240101120000_abc123.jpg",
			setupMock: func(ms *mocks.MockStorage) {
				ms.EXPECT().
					Delete(gomock.Any(), "profiles/user123/20240101120000_abc123.jpg").
					Return(nil)
			},
		},
		{
			name:      "empty path - no deletion",
			imagePath: "",
			setupMock: func(ms *mocks.MockStorage) {
				// No expectations - Delete should not be called
			},
		},
		{
			name:      "storage deletion fails",
			imagePath: "profiles/user456/image.png",
			setupMock: func(ms *mocks.MockStorage) {
				ms.EXPECT().
					Delete(gomock.Any(), "profiles/user456/image.png").
					Return(errors.New("storage error"))
			},
			expectedError: "failed to delete image from storage: storage error",
		},
		{
			name:      "deletion with not found error",
			imagePath: "profiles/nonexistent/image.jpg",
			setupMock: func(ms *mocks.MockStorage) {
				ms.EXPECT().
					Delete(gomock.Any(), "profiles/nonexistent/image.jpg").
					Return(errors.New("not found"))
			},
			expectedError: "failed to delete image from storage: not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStorage := mocks.NewMockStorage(ctrl)
			tt.setupMock(mockStorage)

			service := NewImageService(mockStorage)

			ctx := context.Background()
			err := service.DeleteProfileImage(ctx, tt.imagePath)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestImageService_UploadProfileImage_LargeFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a large file content (1MB)
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	mockStorage := mocks.NewMockStorage(ctrl)
	mockStorage.EXPECT().
		Put(gomock.Any(), largeContent, gomock.Any()).
		Return(nil)

	service := NewImageService(mockStorage)

	upload := graphql.Upload{
		File:     bytes.NewReader(largeContent),
		Filename: "large.jpg",
	}

	ctx := context.Background()
	path, err := service.UploadProfileImage(ctx, "user999", upload)

	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "profiles/user999/")
}

func TestImageService_UploadProfileImage_FileReadError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	service := NewImageService(mockStorage)

	// Create a reader that will fail
	failingReader := &failingReadCloser{
		err: errors.New("read failed"),
	}

	upload := graphql.Upload{
		File:     failingReader,
		Filename: "error.jpg",
	}

	ctx := context.Background()
	path, err := service.UploadProfileImage(ctx, "user000", upload)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
	assert.Empty(t, path)
}

func TestImageService_UploadProfileImage_PathUniqueness(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	
	paths := make(map[string]bool)
	
	// Expect 3 different uploads with unique paths
	mockStorage.EXPECT().
		Put(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(3).
		DoAndReturn(func(ctx context.Context, data []byte, path string) error {
			// Check that each path is unique
			if paths[path] {
				t.Errorf("Duplicate path generated: %s", path)
			}
			paths[path] = true
			return nil
		})

	service := NewImageService(mockStorage)
	ctx := context.Background()

	// Upload the same file 3 times
	for i := 0; i < 3; i++ {
		upload := graphql.Upload{
			File:     strings.NewReader("content"),
			Filename: "test.jpg",
		}
		
		path, err := service.UploadProfileImage(ctx, "user123", upload)
		require.NoError(t, err)
		assert.NotEmpty(t, path)
	}

	// Verify we got 3 unique paths
	assert.Len(t, paths, 3)
}

func TestImageService_ValidateExtensions(t *testing.T) {
	validExtensions := []string{
		".jpg", ".jpeg", ".png", ".gif", ".webp",
		".JPG", ".JPEG", ".PNG", ".GIF", ".WEBP",
		".Jpg", ".Jpeg", ".Png", ".Gif", ".WebP",
	}

	invalidExtensions := []string{
		".pdf", ".doc", ".txt", ".exe", ".sh", ".bat",
		".zip", ".rar", ".mp4", ".avi", ".mov",
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	
	// For valid extensions, expect Put to be called
	for range validExtensions {
		mockStorage.EXPECT().
			Put(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil).
			AnyTimes()
	}

	service := NewImageService(mockStorage)
	ctx := context.Background()

	// Test valid extensions
	for _, ext := range validExtensions {
		t.Run(fmt.Sprintf("valid_extension_%s", ext), func(t *testing.T) {
			upload := graphql.Upload{
				File:     strings.NewReader("content"),
				Filename: "file" + ext,
			}
			
			path, err := service.UploadProfileImage(ctx, "user", upload)
			require.NoError(t, err)
			assert.NotEmpty(t, path)
		})
	}

	// Test invalid extensions
	for _, ext := range invalidExtensions {
		t.Run(fmt.Sprintf("invalid_extension_%s", ext), func(t *testing.T) {
			upload := graphql.Upload{
				File:     strings.NewReader("content"),
				Filename: "file" + ext,
			}
			
			path, err := service.UploadProfileImage(ctx, "user", upload)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid file extension")
			assert.Empty(t, path)
		})
	}
}

// Helper type for testing read errors
type failingReadCloser struct {
	err error
}

func (f *failingReadCloser) Read(p []byte) (n int, err error) {
	return 0, f.err
}

func (f *failingReadCloser) Close() error {
	return nil
}

func (f *failingReadCloser) Seek(offset int64, whence int) (int64, error) {
	return 0, f.err
}

// Helper to create a ReadSeeker from string
func newReadSeeker(content string) io.ReadSeeker {
	return strings.NewReader(content)
}

// Helper to create a ReadSeeker from bytes
func newByteReadSeeker(content []byte) io.ReadSeeker {
	return bytes.NewReader(content)
}