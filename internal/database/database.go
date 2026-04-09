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

	// Ensure pgcrypto exists for gen_random_uuid() (PostgreSQL < 13)
	_ = DB.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto").Error

	log.Println("Database connected successfully")
	return nil
}

func AutoMigrate() error {
	err := DB.AutoMigrate(
		&models.User{},
		&models.Distributor{},
		&models.AdminPermission{},
		&models.RoleRequest{},
		&models.OTPRecord{},
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

// BackfillUserRolesArray sets roles[] from legacy role column where roles was never populated
func BackfillUserRolesArray() error {
	if DB == nil {
		return nil
	}
	return DB.Exec(`UPDATE users SET roles = ARRAY[role::text]::text[] WHERE roles IS NULL`).Error
}

// BackfillLegacyEmailVerified marks existing completed accounts as email-verified so login keeps working after signup OTP is enforced for new users.
func BackfillLegacyEmailVerified() error {
	if DB == nil {
		return nil
	}
	return DB.Exec(`UPDATE users SET is_email_verified = true WHERE registration_complete = true AND is_email_verified = false`).Error
}

func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
