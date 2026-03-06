package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"marvaron/internal/config"
	"marvaron/internal/database"
)

// GenerateOTP genera un OTP numerico
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

// StoreOTP salva l'OTP in Redis con scadenza
func StoreOTP(identifier string, otp string) error {
	key := fmt.Sprintf("otp:%s", identifier)
	expiration := time.Duration(config.AppConfig.OTP.ExpiryMinutes) * time.Minute
	return database.SetCache(key, otp, expiration)
}

// VerifyOTP verifica l'OTP da Redis
func VerifyOTP(identifier string, otp string) (bool, error) {
	key := fmt.Sprintf("otp:%s", identifier)
	storedOTP, err := database.GetCache(key)
	if err != nil {
		return false, err
	}

	if storedOTP != otp {
		return false, nil
	}

	// Elimina l'OTP dopo la verifica
	_ = database.DeleteCache(key)
	return true, nil
}

// SendOTP invia l'OTP (da implementare con SMS/Email service)
func SendOTP(identifier string, otp string, method string) error {
	// TODO: Implementare invio OTP via SMS o Email
	// Per ora solo log
	fmt.Printf("OTP %s inviato a %s via %s\n", otp, identifier, method)
	return nil
}
