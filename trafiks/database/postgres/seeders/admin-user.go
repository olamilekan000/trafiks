package seeders

import (
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/trafiks/trafiks/cfg"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
)

func SeedAdminUser(db *gorm.DB) error {
	config := cfg.GetConf()
	email := config.Bootstrap.User.Email
	password := config.Bootstrap.User.Password
	firstName := config.Bootstrap.User.FirstName
	lastName := config.Bootstrap.User.LastName

	var count int64
	if err := db.Model(&models.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		log.Println("Admin user already exists, skipping seeding.")
		return nil
	}

	hashedPassword, err := pkg.HashPassword(password)
	if err != nil {
		return err
	}

	now := time.Now().Local()
	adminUser := models.User{
		FirstName:  firstName,
		LastName:   lastName,
		Email:      email,
		Role:       "super-admin",
		Password:   hashedPassword,
		VerifiedAt: &now,
		AuthMethod: "email",
	}

	if err := db.Create(&adminUser).Error; err != nil {
		return err
	}

	log.Println("Admin user seeded successfully.")

	return nil
}
