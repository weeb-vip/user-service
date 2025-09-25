package resolvers

import (
	"context"
	"github.com/weeb-vip/user-service/http/handlers/requestinfo"
	"github.com/weeb-vip/user-service/internal/logger"
	"github.com/weeb-vip/user-service/internal/services/users"

	"github.com/weeb-vip/user-service/graph/model"
)

func CreateUser( // nolint
	ctx context.Context,
	userService users.User,
	input *model.CreateUserInput,
) (*model.User, error) {
	log := logger.FromCtx(ctx)
	req := requestinfo.FromContext(ctx)

	userID := req.UserID
	if userID == nil {
		log.Error().Msg("User ID is missing")
		return nil, nil
	}

	if input.ID != *userID {
		log.Error().Msg("User ID does not match, unauthenticated")
		return nil, nil
	}
	language := input.Language.String()
	createdUser, err := userService.AddUser(ctx, input.ID, input.Username, input.Firstname, input.Lastname, language)

	if err != nil {
		return nil, err
	}

	return &model.User{
		ID: createdUser.ID,
	}, nil
}

func CreateUserFromEvent( // nolint
	ctx context.Context,
	userService users.User,
	input *model.CreateUserInput,
) (*model.User, error) {

	language := input.Language.String()
	createdUser, err := userService.AddUser(ctx, input.ID, input.Username, input.Firstname, input.Lastname, language)

	if err != nil {
		return nil, err
	}

	return &model.User{
		ID: createdUser.ID,
	}, nil
}

func GetUser( // nolint
	ctx context.Context,
	userService users.User,
) (*model.User, error) {
	log := logger.FromCtx(ctx)
	req := requestinfo.FromContext(ctx)

	userID := req.UserID
	if userID == nil {
		log.Error().Msg("User ID is missing")
		return nil, nil
	}

	log.Info().Any("userid", *userID).Msg("Fetching user with ID")
	user, err := userService.GetUserDetails(ctx, *userID)

	if err != nil {
		return nil, err
	}

	// convert user language to model.Language
	language := model.Language(user.Language)

	return &model.User{
		ID:              user.ID,
		Firstname:       user.FirstName,
		Lastname:        user.LastName,
		Username:        user.Username,
		Language:        language,
		Email:           user.Email,
		ProfileImageURL: user.ProfileImageURL,
	}, nil
}

func UpdateUser( // nolint
	ctx context.Context,
	userService users.User,
	input *model.UpdateUserInput,
) (*model.User, error) {
	log := logger.FromCtx(ctx)
	req := requestinfo.FromContext(ctx)

	userID := req.UserID

	var language *string
	if input.Language != nil {
		language = new(string)
		*language = input.Language.String()
	}
	log.Info().Any("userid", userID).Msg("User ID from context")
	log.Info().Any("language", language).Msg("language")
	updatedUser, err := userService.UpdateUser(ctx, *userID, input.Username, input.Firstname, input.Lastname, language, input.Email)

	if err != nil {
		return nil, err
	}

	var userLanguage model.Language
	if updatedUser.Language != "" {
		userLanguage = model.Language(updatedUser.Language)
	}
	return &model.User{
		ID:              updatedUser.ID,
		Firstname:       updatedUser.FirstName,
		Lastname:        updatedUser.LastName,
		Username:        updatedUser.Username,
		Language:        userLanguage,
		Email:           updatedUser.Email,
		ProfileImageURL: updatedUser.ProfileImageURL,
	}, nil
}
