package handlers

import (
	"net/http"

	"marvaron/internal/database"
	"marvaron/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DistributorHandler struct{}

// RequestPriceQuote richiede un preventivo per un prodotto
func (h *DistributorHandler) RequestPriceQuote(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	uid, _ := userID.(uuid.UUID)

	// Verifica che l'utente sia un distributore
	var distributor models.Distributor
	if err := database.DB.Where("user_id = ?", uid).First(&distributor).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "User is not a distributor"})
		return
	}

	var req struct {
		ProductID     string  `json:"product_id" binding:"required"`
		Quantity      int     `json:"quantity" binding:"required,min=1"`
		RequestedPrice float64 `json:"requested_price"`
		Notes         string  `json:"notes"`
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

	// Verifica che il prodotto esista
	var product models.Product
	if err := database.DB.First(&product, productUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Crea richiesta preventivo
	priceQuote := models.PriceQuote{
		DistributorID:  distributor.ID,
		ProductID:      productUUID,
		Quantity:       req.Quantity,
		RequestedPrice: req.RequestedPrice,
		Status:         "pending",
		Notes:          req.Notes,
	}

	if err := database.DB.Create(&priceQuote).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create price quote request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Price quote requested successfully",
		"price_quote": priceQuote,
	})
}

// GetPriceQuotes restituisce i preventivi del distributore
func (h *DistributorHandler) GetPriceQuotes(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	uid, _ := userID.(uuid.UUID)

	var distributor models.Distributor
	if err := database.DB.Where("user_id = ?", uid).First(&distributor).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "User is not a distributor"})
		return
	}

	var quotes []models.PriceQuote
	database.DB.Where("distributor_id = ?", distributor.ID).
		Preload("Product").
		Order("created_at DESC").
		Find(&quotes)

	c.JSON(http.StatusOK, gin.H{"price_quotes": quotes})
}

// GetDistributorInfo restituisce le informazioni del distributore
func (h *DistributorHandler) GetDistributorInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	uid, _ := userID.(uuid.UUID)

	var distributor models.Distributor
	if err := database.DB.Where("user_id = ?", uid).
		Preload("User").
		First(&distributor).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Distributor not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"distributor": distributor})
}

// UpdateDistributorInfo aggiorna le informazioni del distributore
func (h *DistributorHandler) UpdateDistributorInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	uid, _ := userID.(uuid.UUID)

	var distributor models.Distributor
	if err := database.DB.Where("user_id = ?", uid).First(&distributor).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Distributor not found"})
		return
	}

	var req struct {
		BusinessName *string `json:"business_name"`
		TaxID        *string `json:"tax_id"`
		Address      *string `json:"address"`
		City         *string `json:"city"`
		Country      *string `json:"country"`
		PostalCode   *string `json:"postal_code"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.BusinessName != nil {
		distributor.BusinessName = *req.BusinessName
	}
	if req.TaxID != nil {
		distributor.TaxID = *req.TaxID
	}
	if req.Address != nil {
		distributor.Address = *req.Address
	}
	if req.City != nil {
		distributor.City = *req.City
	}
	if req.Country != nil {
		distributor.Country = *req.Country
	}
	if req.PostalCode != nil {
		distributor.PostalCode = *req.PostalCode
	}

	database.DB.Save(&distributor)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Distributor info updated successfully",
		"distributor": distributor,
	})
}
