package db

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	var err error

	// Get database connection string from environment variable
	// You can set this with: export DATABASE_URL="host=localhost user=postgres password=password dbname=helloworld port=5432 sslmode=disable"
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Default local PostgreSQL connection
		dsn = "host=localhost user=postgres password=postgres dbname=mydb port=5432 sslmode=disable"
	}

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate the schema
	err = DB.AutoMigrate(&User{}, &Pet{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	fmt.Println("Database connected and migrated successfully")
}
