package postgres

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"github.com/trafiks/trafiks/cfg"
	"github.com/trafiks/trafiks/database/postgres/seeders"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
)

type PostgresDB struct {
	*gorm.DB
}

func NewDatabase(logger pkg.LoggerClient, configEnv *cfg.Config) PostgresDB {
	dbLogger := logger.LogWithFields(map[string]interface{}{
		"function": "NewDatabase",
	})

	db := configEnv.Database

	dbUrl := fmt.Sprintf("host=%v port=%v user=%v password=%v dbname=%v sslmode=%v",
		db.Host, db.Port, db.User, db.Password, db.Name, db.SSLMode)

	var pgConf *gorm.Config

	logLevel := gormLogger.Info
	if configEnv.Database.LogEnabled != nil && !*configEnv.Database.LogEnabled {
		logLevel = gormLogger.Silent
	}

	pgConf = &gorm.Config{
		Logger: gormLogger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			gormLogger.Config{
				SlowThreshold: time.Second,
				LogLevel:      logLevel,
				Colorful:      true,
			},
		),
	}

	dbClient, err := gorm.Open(postgres.Open(dbUrl), pgConf)
	if err != nil {
		logger.Panicf("error connecting to the database %q:", err)
	}

	dbLogger.Info("Database connection established")

	if err := dbClient.AutoMigrate(
		models.User{},
		models.APIKey{},
		models.Project{},
		models.Service{},
		models.ProxyRequestLog{},
		models.Webhook{},
		models.WebhookDelivery{},
	); err != nil {
		dbLogger.Errorf("error migrating database: %v", err)
		panic(err)
	}

	// Seed bootstrap admin user
	if err := seeders.SeedAdminUser(dbClient); err != nil {
		dbLogger.Errorf("error seeding admin user: %v", err)
		panic(err)
	}

	return PostgresDB{
		DB: dbClient,
	}
}
