package config

import (
	"log"
	"my-ecomm/models"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "test.db"
	}
	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database", err)
	}
	//Auto Migrate the schema
	if err := DB.AutoMigrate(&models.User{}, &models.Product{}, &models.ChatRoom{}, &models.Message{}, &models.RoomMember{}); err != nil {
		log.Fatal("failed to migrate database schema", err)
	}
	log.Println("Database connection establish and migrated successfully")
}

func GetDB() *gorm.DB {
	return DB
}
