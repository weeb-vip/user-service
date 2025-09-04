package users

import (
	"context"

	"github.com/weeb-vip/user-service/internal/services/users/models"
	"github.com/weeb-vip/user-service/internal/services/users/repositories"
)

type usersService struct {
	usersRepository repositories.UsersRepository
}

func NewUserService() User {
	usersRepository := repositories.GetUsersRepository()

	return &usersService{
		usersRepository: usersRepository,
	}
}

func (service *usersService) AddUser(
	ctx context.Context,
	id string,
	username string,
	firstName string,
	lastName string,
	language string,
) (*models.User, error) {
	// check if user already exists
	user, err := service.usersRepository.GetUserByUsername(ctx, username)

	if err != nil {
		return nil, &Error{
			Code:    UserErrorInternalError,
			Message: "database error",
		}
	}

	if user != nil {
		return nil, &Error{
			Code:    UserErrorUserExists,
			Message: "user already exists",
		}
	}

	return service.usersRepository.AddUser(
		ctx,
		username,
		id,
		firstName,
		lastName,
		language,
	)
}

func (service *usersService) GetUserDetails( //nolint
	ctx context.Context,
	id string,
) (*models.User, error) {
	user, err := service.usersRepository.GetUserById(ctx, id) // nolint
	if err != nil {
		return nil, &Error{
			Code:    UserErrorInternalError,
			Message: "database error",
		}
	}

	if user == nil {
		return nil, &Error{
			Code:    UserErrorInvalidUsers,
			Message: "invalid user",
		}
	}

	return user, nil
}

func (service *usersService) UpdateUser(
	ctx context.Context,
	id string,
	username *string,
	firstName *string,
	lastName *string,
	language *string,
	email *string,
) (*models.User, error) {
	return service.usersRepository.UpdateUser(ctx, id, username, firstName, lastName, language, email)
}

func (service *usersService) UpdateProfileImageURL(
	ctx context.Context,
	id string,
	profileImageURL string,
) (*models.User, error) {
	user, err := service.usersRepository.GetUserById(ctx, id)
	if err != nil {
		return nil, &Error{
			Code:    UserErrorInternalError,
			Message: "database error",
		}
	}

	if user == nil {
		return nil, &Error{
			Code:    UserErrorInvalidUsers,
			Message: "user not found",
		}
	}

	return service.usersRepository.UpdateProfileImageURL(ctx, id, profileImageURL)
}
