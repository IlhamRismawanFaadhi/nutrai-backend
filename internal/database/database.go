package database

import (
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"nutrai/internal/models"
)

// Connect mencoba beberapa kali karena di Railway DB bisa start lebih lambat dari app.
func Connect(dsn string) (*gorm.DB, error) {
	var err error
	for i := 1; i <= 10; i++ {
		var db *gorm.DB
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
		if err == nil {
			sqlDB, e := db.DB()
			if e == nil {
				if err = sqlDB.Ping(); err == nil {
					sqlDB.SetMaxOpenConns(20)
					sqlDB.SetMaxIdleConns(10)
					sqlDB.SetConnMaxLifetime(30 * time.Minute)
					return db, nil
				}
			} else {
				err = e
			}
		}
		log.Printf("db connect attempt %d/10 failed: %v", i, err)
		time.Sleep(3 * time.Second)
	}
	return nil, err
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&models.User{}, &models.UserProfile{}, &models.MealLog{}, &models.WaterLog{}, &models.FavoriteMeal{}, &models.FoodItem{}, &models.Ingredient{}, &models.FoodComponent{})
}
