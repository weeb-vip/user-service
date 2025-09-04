package repositories

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/weeb-vip/user-service/internal/db"
	"github.com/weeb-vip/user-service/internal/services/users/models"
)

type UsersRepository interface {
	AddUser(
		ctx context.Context,
		username string,
		userID string,
		firstName string,
		lastName string,
		language string,
	) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserById(ctx context.Context, id string) (*models.User, error)
	UpdateUser(ctx context.Context, id string, username *string, firstName *string, lastName *string, language *string, email *string) (*models.User, error)
	UpdateProfileImageURL(ctx context.Context, id string, profileImageURL string) (*models.User, error)
	DeleteUser(ctx context.Context, username string) error
}

type userRepository struct {
	DBService db.DB
}

var userRepositorySingleton UsersRepository // nolint

func NewUsersRepository() UsersRepository {
	dbService := db.GetDBService()

	return &userRepository{
		DBService: dbService,
	}
}

func (repository *userRepository) AddUser(
	ctx context.Context,
	username string,
	userID string,
	firstName string,
	lastName string,
	language string,
) (*models.User, error) {
	database := repository.DBService.GetDB()

	credentials := models.User{
		BaseModel: db.BaseModel{ID: userID},
		Username:  username,
		FirstName: firstName,
		LastName:  lastName,
		Language:  language,
	}
	err := database.WithContext(ctx).FirstOrCreate(&credentials, models.User{Username: username}).Error

	if err != nil {
		return nil, err
	}

	return &credentials, nil
}

func (repository *userRepository) GetUserById(ctx context.Context, id string) (*models.User, error) {
	database := repository.DBService.GetDB()

	var credentials models.User

	err := database.WithContext(ctx).Where("id = ?", id).First(&credentials).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return &credentials, nil
}

func (repository *userRepository) DeleteUser(ctx context.Context, username string) error {
	database := repository.DBService.GetDB()

	return database.WithContext(ctx).Where("username = ?", username).Delete(&models.User{}).Error
}

func (repository *userRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	database := repository.DBService.GetDB()

	var credentials models.User

	err := database.WithContext(ctx).Where("username = ?", username).First(&credentials).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &credentials, nil
}

func (repository *userRepository) UpdateUser(
	ctx context.Context,
	id string,
	username *string,
	firstName *string,
	lastName *string,
	language *string,
	email *string,
) (*models.User, error) {
	database := repository.DBService.GetDB()

	user, err := repository.GetUserById(ctx, id)
	if err != nil {
		return nil, err
	}

	if username != nil {
		user.Username = *username
	}

	if firstName != nil {
		user.FirstName = *firstName
	}

	if lastName != nil {
		user.LastName = *lastName
	}

	if language != nil {
		user.Language = *language
	}

	if email != nil {
		user.Email = email
	}

	err = database.WithContext(ctx).Save(&user).Error
	if err != nil {
		return nil, err
	}

	// get updated user
	return repository.GetUserById(ctx, id)
}

func (repository *userRepository) UpdateProfileImageURL(
	ctx context.Context,
	id string,
	profileImageURL string,
) (*models.User, error) {
	database := repository.DBService.GetDB()

	user, err := repository.GetUserById(ctx, id)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New("user not found")
	}

	user.ProfileImageURL = &profileImageURL

	err = database.WithContext(ctx).Save(&user).Error
	if err != nil {
		return nil, err
	}

	// get updated user
	return repository.GetUserById(ctx, id)
}

func GetUsersRepository() UsersRepository {
	if userRepositorySingleton == nil {
		userRepositorySingleton = NewUsersRepository()
	}

	return userRepositorySingleton
}
