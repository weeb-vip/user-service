package resolvers

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/weeb-vip/user-service/graph/model"
	"github.com/weeb-vip/user-service/http/handlers/requestinfo"
	"github.com/weeb-vip/user-service/internal/services/image"
	"github.com/weeb-vip/user-service/internal/services/users"
)

func UploadProfileImage(ctx context.Context, userService users.User, imageService *image.ImageService, upload graphql.Upload) (*model.User, error) {
	// Get user ID from request context
	req := requestinfo.FromContext(ctx)
	if req.UserID == nil {
		return nil, fmt.Errorf("unauthorized")
	}

	userID := *req.UserID

	// Get current user to check for existing profile image
	currentUser, err := userService.GetUserDetails(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Store old image path for deletion after successful upload
	oldImagePath := ""
	if currentUser.ProfileImageURL != nil && *currentUser.ProfileImageURL != "" {
		oldImagePath = *currentUser.ProfileImageURL
	}

	// Upload new image to MinIO
	imagePath, err := imageService.UploadProfileImage(ctx, userID, upload)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	// Update user profile with new image URL
	updatedUser, err := userService.UpdateProfileImageURL(ctx, userID, imagePath)
	if err != nil {
		// Try to clean up the uploaded image if database update fails
		_ = imageService.DeleteProfileImage(ctx, imagePath)
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}

	// Delete old image from storage after successful update (if it existed)
	if oldImagePath != "" {
		// Use goroutine to delete old image asynchronously to not block the response
		go func() {
			_ = imageService.DeleteProfileImage(context.Background(), oldImagePath)
		}()
	}

	// Convert to GraphQL model
	language := model.Language(updatedUser.Language)
	
	return &model.User{
		ID:              updatedUser.ID,
		Firstname:       updatedUser.FirstName,
		Lastname:        updatedUser.LastName,
		Username:        updatedUser.Username,
		Language:        language,
		Email:           updatedUser.Email,
		ProfileImageURL: updatedUser.ProfileImageURL,
	}, nil
}