package users

import (
	"context"

	"gorm.io/gorm"

	"auth-session/internal/domains"
	"auth-session/internal/infrastructure/postgres"
	"auth-session/pkg"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) domains.UserRepository {
	return &userRepository{db: db}
}

// ==== User Repository ====

func (r *userRepository) CreateUser(ctx context.Context, user domains.CreateUserEntity) error {
	storage := &postgres.UserStorage{
		ID:       user.ID,
		Name:     user.Name,
		Email:    user.Email,
		Password: user.Password,
	}

	if err := r.db.WithContext(ctx).Create(storage).Error; err != nil {
		return pkg.HandleDBError(err)
	}

	return nil
}

func (r *userRepository) UpdateUser(ctx context.Context, update *domains.UpdateUserEntity) error {
	fields := make(map[string]any)

	if update.Name != nil && *update.Name != "" {
		fields["name"] = *update.Name
	}
	if update.Email != nil && *update.Email != "" {
		fields["email"] = *update.Email
	}
	if update.RememberMe != nil {
		fields["remember_me"] = *update.RememberMe
	}

	result := r.db.WithContext(ctx).
		Model(&postgres.UserStorage{}).
		Where("id = ?", update.ID).
		Updates(fields)

	if result.Error != nil {
		return pkg.HandleDBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return pkg.HandleDBError(gorm.ErrRecordNotFound)
	}

	return nil
}

func (r *userRepository) GetUsers(ctx context.Context) ([]domains.UserEntity, error) {
	var storages []postgres.UserStorage

	if err := r.db.WithContext(ctx).
		Order("created_at ASC").
		Find(&storages).Error; err != nil {
		return nil, pkg.HandleDBError(err)
	}

	return ToUserListEntity(storages), nil
}

func (r *userRepository) GetUserByID(ctx context.Context, id string) (*domains.UserEntity, error) {
	var storage postgres.UserStorage

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&storage).Error

	if err != nil {
		return nil, pkg.HandleDBError(err)
	}

	return ToUserEntity(&storage), nil
}

func (r *userRepository) GetPasswordByEmail(ctx context.Context, email string) (string, string, error) {
	var storage postgres.UserStorage

	err := r.db.WithContext(ctx).
		Select("id", "password").
		Where("email = ?", email).
		First(&storage).Error

	if err != nil {
		return "", "", pkg.HandleDBError(err)
	}

	return storage.Password, storage.ID, nil
}

func (r *userRepository) DeleteUser(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&postgres.UserStorage{})
	
	if result.Error != nil {
		return pkg.HandleDBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return pkg.HandleDBError(gorm.ErrRecordNotFound)
	}

	return nil
}