package db

import (
	"os"

	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DB is non-nil only after a successful connect and migrate. Check before use if the app may run without a database.
var DB *gorm.DB

func InitDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=mydb port=5432 sslmode=disable"
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Error().Err(err).Msg("database: connection failed; continuing without DB")
		DB = nil
		return
	}

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
