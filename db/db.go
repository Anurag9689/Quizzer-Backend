package db

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"OnlineQuizSystem/models"
)


var DB *gorm.DB


func Init() (*gorm.DB) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env first")
	}

	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	host := os.Getenv("DB_HOST")
	dbport := os.Getenv("DB_PORT")
	dbname := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, pass, host, dbport, dbname)

	DB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	log.Println("✅ Connected to DB!")

	migrationErr := DB.AutoMigrate(
		&models.User{},
		&models.UserDetails{},
		&models.QuizEvent{},
		&models.EventResult{},
	)

	if migrationErr != nil {
		log.Fatalf("❌ AutoMigration failed: %v", migrationErr)
	}

	log.Println("✅ AutoMigration complete!")

	return DB
}
