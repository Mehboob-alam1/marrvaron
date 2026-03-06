package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"time"

	"marvaron/internal/config"
)

// QRPayload contiene i dati da crittografare nel QR code
type QRPayload struct {
	ProductID     string `json:"product_id"`
	BatchNumber   string `json:"batch_number"`
	SerialNumber  string `json:"serial_number"`
	InventoryID   string `json:"inventory_id"`
	Timestamp     int64  `json:"timestamp"`
}

// EncryptQRCode crittografa i dati del QR code usando AES-256
func EncryptQRCode(payload QRPayload) (string, string, error) {
	// Serializza il payload
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	// Prepara la chiave AES (deve essere 32 byte per AES-256)
	key := []byte(config.AppConfig.QR.EncryptionKey)
	if len(key) != 32 {
		// Padding o hashing se necessario
		hash := sha256.Sum256(key)
		key = hash[:]
	}

	// Crea il cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", err
	}

	// Crea GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}

	// Genera nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", err
	}

	// Crittografa
	ciphertext := aesGCM.Seal(nonce, nonce, payloadJSON, nil)
	encryptedToken := base64.URLEncoding.EncodeToString(ciphertext)

	// Crea firma digitale HMAC
	signature := createSignature(encryptedToken)

	return encryptedToken, signature, nil
}

// DecryptQRCode decrittografa il token QR
func DecryptQRCode(encryptedToken string) (*QRPayload, error) {
	// Decodifica base64
	ciphertext, err := base64.URLEncoding.DecodeString(encryptedToken)
	if err != nil {
		return nil, err
	}

	// Prepara la chiave
	key := []byte(config.AppConfig.QR.EncryptionKey)
	if len(key) != 32 {
		hash := sha256.Sum256(key)
		key = hash[:]
	}

	// Crea cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Crea GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Estrai nonce
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrittografa
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	// Deserializza
	var payload QRPayload
	err = json.Unmarshal(plaintext, &payload)
	if err != nil {
		return nil, err
	}

	return &payload, nil
}

// VerifyQRSignature verifica la firma digitale del QR code
func VerifyQRSignature(encryptedToken string, signature string) bool {
	expectedSignature := createSignature(encryptedToken)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// createSignature crea una firma HMAC-SHA256
func createSignature(data string) string {
	h := hmac.New(sha256.New, []byte(config.AppConfig.QR.SignatureSecret))
	h.Write([]byte(data))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

// GenerateQRCodeData genera i dati per un nuovo QR code
func GenerateQRCodeData(productID, batchNumber, serialNumber, inventoryID string) (string, string, error) {
	payload := QRPayload{
		ProductID:    productID,
		BatchNumber:  batchNumber,
		SerialNumber: serialNumber,
		InventoryID:  inventoryID,
		Timestamp:    getCurrentTimestamp(),
	}

	return EncryptQRCode(payload)
}

func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
