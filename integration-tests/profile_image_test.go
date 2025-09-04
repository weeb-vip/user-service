//go:build integration

package integration_tests

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testServerManager *RealServerManager

// TestMain runs before all tests and handles server lifecycle
func TestMain(m *testing.M) {
	// Set environment to use docker config for integration tests
	os.Setenv("APP_ENV", "docker")

	// Start the GraphQL server once for all tests
	testServerManager = NewRealServerManager(3002)
	
	// Start server (we'll handle errors in the server manager)
	fmt.Println("Starting test server for integration tests...")
	cleanup := testServerManager.StartServerForMain()

	// Run all tests
	code := m.Run()

	// Cleanup server
	fmt.Println("Shutting down test server...")
	cleanup()

	// Exit with the same code as the tests
	os.Exit(code)
}

// TestProfileImageUpload_Integration tests the profile image upload functionality end-to-end
func TestProfileImageUpload_Integration(t *testing.T) {
	// Create GraphQL client using the shared test server
	client := NewGraphQLClient(testServerManager.GetBaseURL())

	// Since we need authentication, we'll need to mock the JWT context
	// For now, we'll create a test that assumes we can bypass auth or use a test token
	// You might need to adjust this based on your actual auth setup

	// Run the integration tests
	t.Run("UploadProfileImage_Success", func(t *testing.T) {
		testUploadProfileImageSuccess(t, client)
	})

	t.Run("UploadProfileImage_InvalidFormat", func(t *testing.T) {
		testUploadProfileImageInvalidFormat(t, client)
	})

	t.Run("UploadProfileImage_LargeFile", func(t *testing.T) {
		testUploadProfileImageLargeFile(t, client)
	})

	t.Run("UploadProfileImage_MultipleFiles", func(t *testing.T) {
		testUploadProfileImageMultipleFiles(t, client)
	})
}

func testUploadProfileImageSuccess(t *testing.T, client *GraphQLClient) {
	// Create a test user first
	testUser := client.CreateTestUser(t, "user_integration_test")
	t.Logf("Created test user: %s", testUser.ID)

	// Test uploading a valid image file
	fileName := "test-profile.jpg"
	fileContent := "fake jpg content for integration test"

	user := client.UploadProfileImage(t, fileName, fileContent)

	// Verify the response
	assert.NotEmpty(t, user.ID)
	assert.NotNil(t, user.ProfileImageURL)
	assert.NotEmpty(t, *user.ProfileImageURL)
	assert.Contains(t, *user.ProfileImageURL, "profiles/")
	assert.Contains(t, *user.ProfileImageURL, ".jpg")

	t.Logf("Successfully uploaded profile image: %s", *user.ProfileImageURL)
}

func testUploadProfileImageInvalidFormat(t *testing.T, client *GraphQLClient) {
	// Test uploading an invalid file format
	fileName := "test-document.pdf"
	fileContent := "fake pdf content"

	// This should fail
	query := `
		mutation UploadProfileImage($image: Upload!) {
			UploadProfileImage(image: $image) {
				id
				profileImageUrl
			}
		}
	`

	variables := map[string]interface{}{
		"image": nil,
	}

	resp := client.UploadFile(t, query, variables, fileName, fileContent)

	// Should have errors
	assert.NotEmpty(t, resp.Errors)
	assert.Contains(t, resp.Errors[0].Message, "invalid file extension")
	
	t.Logf("Successfully rejected invalid file format: %s", resp.Errors[0].Message)
}

func testUploadProfileImageLargeFile(t *testing.T, client *GraphQLClient) {
	// Create a test user first
	testUser := client.CreateTestUser(t, "user_integration_test")
	t.Logf("Created test user for large file test: %s", testUser.ID)

	// Test uploading a large file (1MB)
	fileName := "large-image.png"
	
	// Create 1MB of content
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	user := client.UploadProfileImage(t, fileName, string(largeContent))

	// Verify the response
	assert.NotEmpty(t, user.ID)
	assert.NotNil(t, user.ProfileImageURL)
	assert.NotEmpty(t, *user.ProfileImageURL)
	assert.Contains(t, *user.ProfileImageURL, "profiles/")
	assert.Contains(t, *user.ProfileImageURL, ".png")

	t.Logf("Successfully uploaded large file: %s", *user.ProfileImageURL)
}

func testUploadProfileImageMultipleFiles(t *testing.T, client *GraphQLClient) {
	// Create a test user first
	testUser := client.CreateTestUser(t, "user_integration_test")
	t.Logf("Created test user for multiple files test: %s", testUser.ID)

	// Create a minimal valid GIF (1x1 transparent pixel)
	// This is base64 decoded from: R0lGODlhAQABAIAAAP///wAAACH5BAEAAAAALAAAAAABAAEAAAICRAEAOw==
	gifBytes := []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, // GIF89a
		0x01, 0x00, 0x01, 0x00, // 1x1 dimensions
		0x80, 0x00, 0x00, // Global Color Table info
		0xFF, 0xFF, 0xFF, // White color
		0x00, 0x00, 0x00, // Black color
		0x21, 0xF9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, // Graphics Control Extension
		0x2C, 0x00, 0x00, 0x00, 0x00, // Image Descriptor
		0x01, 0x00, 0x01, 0x00, 0x00, // Width, Height, flags
		0x02, 0x02, 0x44, 0x01, 0x00, // LZW compressed image data
		0x3B, // Trailer
	}

	// Test uploading multiple files to ensure path uniqueness and GIF conversion
	testFiles := []struct {
		filename       string
		content        string
		expectedExt    string // expected extension after processing
	}{
		{"profile1.jpg", "content 1", ".jpg"},
		{"profile2.png", "content 2", ".png"},
		{"profile3.gif", string(gifBytes), ".png"}, // GIF should be converted to PNG
		{"profile4.webp", "content 4", ".webp"},
	}

	var uploadedPaths []string

	for i, file := range testFiles {
		user := client.UploadProfileImage(t, file.filename, file.content)

		assert.NotEmpty(t, user.ID)
		assert.NotNil(t, user.ProfileImageURL)
		assert.NotEmpty(t, *user.ProfileImageURL)

		// Verify the expected file extension after processing
		assert.Contains(t, *user.ProfileImageURL, file.expectedExt)

		uploadedPaths = append(uploadedPaths, *user.ProfileImageURL)

		t.Logf("Upload %d: %s -> %s (expected %s)", i+1, file.filename, *user.ProfileImageURL, file.expectedExt)
	}

	// Verify that paths follow the expected pattern and are unique
	// Note: Since we now replace old images, each upload creates a new file with timestamp
	pathMap := make(map[string]bool)
	for _, path := range uploadedPaths {
		assert.False(t, pathMap[path], "Duplicate path found: %s", path)
		pathMap[path] = true
		// Verify the path contains the profile pattern
		assert.Contains(t, path, "profiles/user_integration_test/profile_")
	}

	t.Logf("Successfully verified %d upload paths with replacement logic", len(uploadedPaths))
}

// TestUserWorkflow_Integration tests a complete user workflow including image upload
func TestUserWorkflow_Integration(t *testing.T) {
	// Create GraphQL client using the shared test server
	client := NewGraphQLClient(testServerManager.GetBaseURL())

	ctx := context.Background()
	_ = ctx // Use context if needed

	t.Run("CompleteUserFlow", func(t *testing.T) {
		// This would test a complete user flow:
		// 1. User registration/creation
		// 2. Profile image upload
		// 3. Profile retrieval with image
		// 4. Profile image update
		
		_ = fmt.Sprintf("integration-test-%d", 123456789)

		// Step 1: Create user
		testUser := client.CreateTestUser(t, "user_integration_test")
		assert.NotEmpty(t, testUser.ID)
		assert.Equal(t, "user_integration_test", testUser.ID)

		// Step 2: Upload profile image
		fileName := "user-profile.jpg"
		fileContent := "user profile image content"
		
		user := client.UploadProfileImage(t, fileName, fileContent)
		assert.NotNil(t, user.ProfileImageURL)
		assert.Contains(t, *user.ProfileImageURL, ".jpg")

		// Step 3: Verify profile retrieval includes image
		userDetails := client.GetUserDetails(t)
		assert.Equal(t, user.ID, userDetails.ID)
		assert.Equal(t, user.ProfileImageURL, userDetails.ProfileImageURL)

		// Step 4: Update profile image
		newFileName := "updated-profile.png"
		newFileContent := "updated profile image content"
		
		updatedUser := client.UploadProfileImage(t, newFileName, newFileContent)
		assert.NotEqual(t, user.ProfileImageURL, updatedUser.ProfileImageURL)
		assert.Contains(t, *updatedUser.ProfileImageURL, ".png")

		t.Logf("Complete user workflow test passed for user %s", updatedUser.ID)
	})
}