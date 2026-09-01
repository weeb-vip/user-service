package resolvers

import (
	"context"
	"fmt"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/weeb-vip/user-service/graph/model"
	"github.com/weeb-vip/user-service/http/handlers/requestinfo"
	"github.com/weeb-vip/user-service/internal/services/image"
	"github.com/weeb-vip/user-service/internal/services/users"
	"github.com/weeb-vip/user-service/metrics"
	"github.com/weeb-vip/user-service/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// UploadBannerImage is the banner counterpart of UploadProfileImage: upload the
// object, then point the user's row at it, and clean up the previous banner
// once the row has moved so a failed write never leaves the page imageless.
func UploadBannerImage(ctx context.Context, userService users.User, imageService *image.ImageService, upload graphql.Upload) (*model.User, error) {
	tracer := tracing.GetTracer(ctx)
	ctx, span := tracer.Start(ctx, "UploadBannerImage",
		trace.WithAttributes(
			attribute.String("resolver.name", "UploadBannerImage"),
			attribute.String("upload.filename", upload.Filename),
			attribute.Int64("upload.size", upload.Size),
		),
		tracing.GetEnvironmentAttribute(),
	)
	defer span.End()

	startTime := time.Now()

	req := requestinfo.FromContext(ctx)
	if req.UserID == nil {
		metrics.GetAppMetrics().ResolverMetric(float64(time.Since(startTime).Milliseconds()), "UploadBannerImage", metrics.Error)
		return nil, fmt.Errorf("unauthorized")
	}
	userID := *req.UserID
	span.SetAttributes(attribute.String("user.id", userID))

	currentUser, err := userService.GetUserDetails(ctx, userID)
	if err != nil {
		metrics.GetAppMetrics().ResolverMetric(float64(time.Since(startTime).Milliseconds()), "UploadBannerImage", metrics.Error)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	oldImagePath := ""
	if currentUser.BannerImageURL != nil && *currentUser.BannerImageURL != "" {
		oldImagePath = *currentUser.BannerImageURL
	}

	imagePath, err := imageService.UploadBannerImage(ctx, userID, upload)
	if err != nil {
		metrics.GetAppMetrics().ResolverMetric(float64(time.Since(startTime).Milliseconds()), "UploadBannerImage", metrics.Error)
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}
	span.SetAttributes(attribute.String("image.path", imagePath))

	updatedUser, err := userService.UpdateBannerImageURL(ctx, userID, imagePath)
	if err != nil {
		_ = imageService.DeleteProfileImage(ctx, imagePath)
		metrics.GetAppMetrics().ResolverMetric(float64(time.Since(startTime).Milliseconds()), "UploadBannerImage", metrics.Error)
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}

	if oldImagePath != "" {
		go func() {
			_ = imageService.DeleteProfileImage(context.Background(), oldImagePath)
		}()
	}

	metrics.GetAppMetrics().ResolverMetric(float64(time.Since(startTime).Milliseconds()), "UploadBannerImage", metrics.Success)
	return toGraphUser(updatedUser), nil
}
