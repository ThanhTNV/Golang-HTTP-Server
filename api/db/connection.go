package db

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DB is non-nil only after a successful connect and migrate. Check before use if the app may run without a database.
var DB *gorm.DB

func InitDB() {
	db_host := os.Getenv("POSTGRES_HOST")
	db_port := os.Getenv("POSTGRES_PORT")
	db_user := os.Getenv("POSTGRES_USER")
	db_password := os.Getenv("POSTGRES_PASSWORD")
	db_name := os.Getenv("POSTGRES_DB")
	if db_host == "" || db_port == "" || db_user == "" || db_password == "" || db_name == "" {
		log.Error().Msg("database: environment variables are not set")
		return
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", db_host, db_user, db_password, db_name, db_port)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	err = DB.AutoMigrate(&User{}, &Pet{})
	if err != nil {
		log.Error().Err(err).Msg("database: auto-migrate failed; continuing without DB")
		sqlDB, cerr := DB.DB()
		if cerr == nil {
			_ = sqlDB.Close()
		}
		DB = nil
		return
	}

	log.Info().Msg("database: connected and migrated successfully")
}
