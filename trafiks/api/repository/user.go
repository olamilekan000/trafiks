package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/trafiks/trafiks/database/postgres"
	"github.com/trafiks/trafiks/models"
)

type UserRepoClient interface {
	Create(ctx context.Context, user *models.User) error
	CreateWithCB(ctx context.Context, user *models.User, cb func(tx *gorm.DB) error) error
	Find(ctx context.Context, filter *models.User) (*models.User, error)
	Updates(ctx context.Context, filter *models.User, updates map[string]interface{}) error
}

type UserRepo struct {
	db *postgres.PostgresDB
}

func NewUserRepo(db postgres.PostgresDB) UserRepoClient {
	return &UserRepo{
		db: &db,
	}
}

func (u *UserRepo) Create(ctx context.Context, user *models.User) error {
	return u.db.WithContext(ctx).Create(user).Error
}

func (u *UserRepo) Find(ctx context.Context, filter *models.User) (*models.User, error) {
	var user models.User

	if err := u.db.WithContext(ctx).Where(filter).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (u *UserRepo) Updates(ctx context.Context, filter *models.User, updates map[string]interface{}) error {
	if err := u.db.WithContext(ctx).Model(&models.User{}).Where(filter).Updates(updates).Error; err != nil {
		return err
	}

	return nil
}

func (u *UserRepo) CreateWithCB(ctx context.Context, user *models.User, cb func(tx *gorm.DB) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		return cb(tx)
	})
}
