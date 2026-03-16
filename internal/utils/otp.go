package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"marvaron/internal/config"
	"marvaron/internal/database"
	"marvaron/internal/models"

	"github.com/redis/go-redis/v9"
)

// GenerateOTP generates a numeric OTP
func GenerateOTP() (string, error) {
	length := config.AppConfig.OTP.Length
	max := big.NewInt(int64(1))
	for i := 0; i < length; i++ {
		max.Mul(max, big.NewInt(10))
	}
	max.Sub(max, big.NewInt(1))

	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}

	otp := fmt.Sprintf("%0*d", length, n.Int64())
	return otp, nil
}

// StoreOTP stores OTP in Redis or DB (DB used when Redis is unavailable)
func StoreOTP(identifier string, otp string) error {
	expiration := time.Duration(config.AppConfig.OTP.ExpiryMinutes) * time.Minute
	key := fmt.Sprintf("otp:%s", identifier)

	if database.RedisClient != nil {
		if err := database.SetCache(key, otp, expiration); err == nil {
			return nil
		}
	}

	// Fallback: store in database
	if database.DB != nil {
		database.DB.Where("identifier = ?", identifier).Delete(&models.OTPRecord{})
		record := models.OTPRecord{
			Identifier: identifier,
			OTP:        otp,
			ExpiresAt:  time.Now().Add(expiration),
		}
		return database.DB.Create(&record).Error
	}

	return fmt.Errorf("no OTP storage available (Redis and DB)")
}

// VerifyOTP verifies OTP from Redis or DB
func VerifyOTP(identifier string, otp string) (bool, error) {
	key := fmt.Sprintf("otp:%s", identifier)

	if database.RedisClient != nil {
		storedOTP, err := database.GetCache(key)
		if err == nil {
			if storedOTP == otp {
				_ = database.DeleteCache(key)
				return true, nil
			}
			return false, nil
		}
		if err != redis.Nil {
			return false, err
		}
	}

	// Fallback: verify from database
	if database.DB != nil {
		var record models.OTPRecord
		err := database.DB.Where("identifier = ? AND otp = ? AND expires_at > ?", identifier, otp, time.Now()).
			First(&record).Error
		if err != nil {
			return false, nil // invalid or expired OTP
		}
		_ = database.DB.Delete(&record)
		return true, nil
	}

	return false, fmt.Errorf("no OTP storage available")
}

// SendOTP invia l'OTP (da implementare con SMS/Email service)
func SendOTP(identifier string, otp string, method string) error {
	// TODO: Implementare invio OTP via SMS o Email
	// Per ora solo log
	fmt.Printf("OTP %s inviato a %s via %s\n", otp, identifier, method)
	return nil
}
