package database

import (
	"fmt"
	"log"

	"marvaron/internal/config"
	"marvaron/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() error {
	dsn := config.AppConfig.GetDSN()
	
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connected successfully")
	return nil
}

func AutoMigrate() error {
	err := DB.AutoMigrate(
		&models.User{},
		&models.Distributor{},
		&models.AdminPermission{},
		&models.Product{},
		&models.InventoryItem{},
		&models.QRCode{},
		&models.QRScanHistory{},
		&models.PriceQuote{},
		&models.Order{},
		&models.OrderItem{},
		&models.Payment{},
		&models.Cart{},
	)
	
	if err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	log.Println("Database migration completed successfully")
	return nil
}

func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
