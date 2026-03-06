package handlers

import (
	"net/http"
	"strconv"

	"marvaron/internal/database"
	"marvaron/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProductHandler struct{}

// GetProducts restituisce la lista dei prodotti
func (h *ProductHandler) GetProducts(c *gin.Context) {
	var products []models.Product

	query := database.DB.Where("status = ?", models.ProductStatusActive)

	// Filtri opzionali
	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}
	if brand := c.Query("brand"); brand != "" {
		query = query.Where("brand = ?", brand)
	}
	if search := c.Query("search"); search != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Paginazione
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	var total int64
	query.Model(&models.Product{}).Count(&total)

	query.Offset(offset).Limit(limit).Find(&products)

	c.JSON(http.StatusOK, gin.H{
		"products": products,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

// GetProduct restituisce i dettagli di un prodotto
func (h *ProductHandler) GetProduct(c *gin.Context) {
	id := c.Param("id")

	var product models.Product
	if err := database.DB.Preload("InventoryItems").First(&product, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"product": product})
}

// CreateProduct crea un nuovo prodotto (Admin)
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req struct {
		Name              string  `json:"name" binding:"required"`
		Description       string  `json:"description"`
		SKU               string  `json:"sku" binding:"required"`
		Barcode           string  `json:"barcode"`
		Category          string  `json:"category"`
		Brand             string  `json:"brand"`
		BasePrice         float64 `json:"base_price" binding:"required"`
		Currency          string  `json:"currency"`
		ImageURL          string  `json:"image_url"`
		Weight            float64 `json:"weight"`
		Dimensions        string  `json:"dimensions"`
		IsAuthenticatable bool    `json:"is_authenticatable"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verifica SKU univoco
	var existing models.Product
	if err := database.DB.Where("sku = ?", req.SKU).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "SKU already exists"})
		return
	}

	product := models.Product{
		Name:              req.Name,
		Description:       req.Description,
		SKU:               req.SKU,
		Barcode:           req.Barcode,
		Category:          req.Category,
		Brand:             req.Brand,
		BasePrice:         req.BasePrice,
		Currency:          getOrDefault(req.Currency, "USD"),
		ImageURL:          req.ImageURL,
		Weight:            req.Weight,
		Dimensions:        req.Dimensions,
		IsAuthenticatable: req.IsAuthenticatable,
		Status:            models.ProductStatusActive,
	}

	if err := database.DB.Create(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Product created successfully",
		"product": product,
	})
}

// UpdateProduct aggiorna un prodotto (Admin)
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id := c.Param("id")

	var product models.Product
	if err := database.DB.First(&product, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	var req struct {
		Name              *string          `json:"name"`
		Description       *string          `json:"description"`
		Category          *string          `json:"category"`
		Brand             *string          `json:"brand"`
		BasePrice         *float64         `json:"base_price"`
		ImageURL          *string          `json:"image_url"`
		Weight            *float64         `json:"weight"`
		Dimensions        *string          `json:"dimensions"`
		Status            *models.ProductStatus `json:"status"`
		IsAuthenticatable *bool            `json:"is_authenticatable"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Aggiorna solo i campi forniti
	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Description != nil {
		product.Description = *req.Description
	}
	if req.Category != nil {
		product.Category = *req.Category
	}
	if req.Brand != nil {
		product.Brand = *req.Brand
	}
	if req.BasePrice != nil {
		product.BasePrice = *req.BasePrice
	}
	if req.ImageURL != nil {
		product.ImageURL = *req.ImageURL
	}
	if req.Weight != nil {
		product.Weight = *req.Weight
	}
	if req.Dimensions != nil {
		product.Dimensions = *req.Dimensions
	}
	if req.Status != nil {
		product.Status = *req.Status
	}
	if req.IsAuthenticatable != nil {
		product.IsAuthenticatable = *req.IsAuthenticatable
	}

	database.DB.Save(&product)

	c.JSON(http.StatusOK, gin.H{
		"message": "Product updated successfully",
		"product": product,
	})
}

// DeleteProduct elimina un prodotto (Admin)
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id := c.Param("id")

	var product models.Product
	if err := database.DB.First(&product, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Soft delete
	database.DB.Delete(&product)

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}

// AddInventoryItem aggiunge un item all'inventario (Admin)
func (h *ProductHandler) AddInventoryItem(c *gin.Context) {
	var req struct {
		ProductID    string  `json:"product_id" binding:"required"`
		BatchNumber  string  `json:"batch_number" binding:"required"`
		SerialNumber string  `json:"serial_number" binding:"required"`
		Quantity     int     `json:"quantity" binding:"required"`
		CostPrice    float64 `json:"cost_price"`
		Location     string  `json:"location"`
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

	// Verifica seriale univoco
	var existing models.InventoryItem
	if err := database.DB.Where("serial_number = ?", req.SerialNumber).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Serial number already exists"})
		return
	}

	inventoryItem := models.InventoryItem{
		ProductID:    productUUID,
		BatchNumber:  req.BatchNumber,
		SerialNumber: req.SerialNumber,
		Quantity:     req.Quantity,
		CostPrice:    req.CostPrice,
		Location:     req.Location,
		Status:       "in_stock",
	}

	if err := database.DB.Create(&inventoryItem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create inventory item"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":        "Inventory item added successfully",
		"inventory_item": inventoryItem,
	})
}

func getOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
