package handlers

import (
	"net/http"

	"marvaron/internal/database"
	"marvaron/internal/models"
	"marvaron/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type QRHandler struct{}

type QRScanRequest struct {
	EncryptedToken string  `json:"encrypted_token" binding:"required"`
	Signature      string  `json:"signature" binding:"required"`
	DeviceID       string  `json:"device_id"`
	DeviceType     string  `json:"device_type"` // ios, android, web
	LocationLat    float64 `json:"location_lat"`
	LocationLng    float64 `json:"location_lng"`
	LocationAddress string `json:"location_address"`
	ScanMethod     string  `json:"scan_method"` // camera, file_upload
}

// ScanQR gestisce la scansione di un QR code
func (h *QRHandler) ScanQR(c *gin.Context) {
	var req QRScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verifica la firma digitale
	if !utils.VerifyQRSignature(req.EncryptedToken, req.Signature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid QR code signature"})
		return
	}

	// Decrittografa il token
	payload, err := utils.DecryptQRCode(req.EncryptedToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to decrypt QR code"})
		return
	}

	// Trova il QR code nel database
	var qrCode models.QRCode
	result := database.DB.Where("encrypted_token = ?", req.EncryptedToken).First(&qrCode)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "QR code not found in database"})
		return
	}

	// Verifica che il QR code sia attivo
	if !qrCode.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "QR code is inactive"})
		return
	}

	// Verifica autenticità confrontando i dati
	isAuthentic := true
	if qrCode.BatchNumber != payload.BatchNumber || 
	   qrCode.SerialNumber != payload.SerialNumber {
		isAuthentic = false
	}

	// Ottieni user_id se autenticato (opzionale)
	var userID *uuid.UUID
	if uid, exists := c.Get("user_id"); exists {
		if u, ok := uid.(uuid.UUID); ok {
			userID = &u
		}
	}

	// Registra la scansione
	scanHistory := models.QRScanHistory{
		QRCodeID:        qrCode.ID,
		UserID:          userID,
		DeviceID:        req.DeviceID,
		DeviceType:      req.DeviceType,
		LocationLat:     req.LocationLat,
		LocationLng:     req.LocationLng,
		LocationAddress: req.LocationAddress,
		IsAuthentic:     isAuthentic,
		IsFakeAlert:     !isAuthentic,
		ScanMethod:      req.ScanMethod,
	}

	if err := database.DB.Create(&scanHistory).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record scan"})
		return
	}

	// Se è un prodotto falso, invia alert all'admin (via Kafka)
	if !isAuthentic {
		// TODO: Pubblicare evento su Kafka per alert admin
	}

	// Carica il prodotto associato
	var product models.Product
	database.DB.First(&product, qrCode.ProductID)

	// Prepara la risposta
	response := gin.H{
		"is_authentic": isAuthentic,
		"qr_code":      qrCode,
		"product":      product,
		"scan_id":      scanHistory.ID,
	}

	// Se l'utente è autenticato, permette di aggiungere al carrello
	if userID != nil {
		response["can_add_to_cart"] = true
	} else {
		response["can_add_to_cart"] = false
		response["message"] = "Login required to add to cart"
	}

	c.JSON(http.StatusOK, response)
}

// VerifyQR verifica un token QR senza registrare la scansione
func (h *QRHandler) VerifyQR(c *gin.Context) {
	token := c.Param("token")

	var qrCode models.QRCode
	result := database.DB.Where("encrypted_token = ?", token).First(&qrCode)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "QR code not found"})
		return
	}

	// Verifica firma
	if !utils.VerifyQRSignature(qrCode.EncryptedToken, qrCode.DigitalSignature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	var product models.Product
	database.DB.First(&product, qrCode.ProductID)

	c.JSON(http.StatusOK, gin.H{
		"is_valid":  true,
		"is_active": qrCode.IsActive,
		"qr_code":   qrCode,
		"product":   product,
	})
}

// GetScanHistory restituisce lo storico delle scansioni dell'utente
func (h *QRHandler) GetScanHistory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var scans []models.QRScanHistory
	database.DB.Where("user_id = ?", userID).
		Preload("QRCode").
		Preload("QRCode.Product").
		Order("created_at DESC").
		Limit(100).
		Find(&scans)

	c.JSON(http.StatusOK, gin.H{"scans": scans})
}

// GenerateQR genera un nuovo QR code per un prodotto (Admin only)
func (h *QRHandler) GenerateQR(c *gin.Context) {
	var req struct {
		ProductID     string `json:"product_id" binding:"required"`
		BatchNumber   string `json:"batch_number" binding:"required"`
		SerialNumber  string `json:"serial_number" binding:"required"`
		InventoryID   string `json:"inventory_id" binding:"required"`
		DisplayInfo   string `json:"display_info"` // JSON string con info personalizzabili
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	productUUID, err := uuid.Parse(req.ProductID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	inventoryUUID, err := uuid.Parse(req.InventoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid inventory ID"})
		return
	}

	// Genera token crittografato
	encryptedToken, signature, err := utils.GenerateQRCodeData(
		req.ProductID,
		req.BatchNumber,
		req.SerialNumber,
		req.InventoryID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate QR code"})
		return
	}

	// Crea record QR code
	qrCode := models.QRCode{
		ProductID:       productUUID,
		InventoryItemID: &inventoryUUID,
		EncryptedToken:  encryptedToken,
		DigitalSignature: signature,
		BatchNumber:     req.BatchNumber,
		SerialNumber:    req.SerialNumber,
		DisplayInfo:     req.DisplayInfo,
		IsActive:        true,
	}

	if err := database.DB.Create(&qrCode).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save QR code"})
		return
	}

	// Aggiorna inventory item con QR code ID
	database.DB.Model(&models.InventoryItem{}).
		Where("id = ?", inventoryUUID).
		Update("qr_code_id", qrCode.ID)

	c.JSON(http.StatusCreated, gin.H{
		"message": "QR code generated successfully",
		"qr_code": qrCode,
	})
}

// UpdateQRDisplayInfo aggiorna le informazioni visualizzate del QR code (Admin)
func (h *QRHandler) UpdateQRDisplayInfo(c *gin.Context) {
	qrID := c.Param("id")

	var qrCode models.QRCode
	if err := database.DB.First(&qrCode, qrID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "QR code not found"})
		return
	}

	var req struct {
		DisplayInfo string `json:"display_info" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	qrCode.DisplayInfo = req.DisplayInfo
	database.DB.Save(&qrCode)

	c.JSON(http.StatusOK, gin.H{
		"message": "QR code display info updated",
		"qr_code": qrCode,
	})
}
