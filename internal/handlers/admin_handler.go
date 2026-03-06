package handlers

import (
	"net/http"
	"time"

	"marvaron/internal/database"
	"marvaron/internal/models"
	"marvaron/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminHandler struct{}

// CreateAdmin crea un nuovo account admin (Super Admin only)
func (h *AdminHandler) CreateAdmin(c *gin.Context) {
	var req struct {
		Email         string `json:"email" binding:"required,email"`
		Password      string `json:"password" binding:"required,min=8"`
		FirstName     string `json:"first_name"`
		LastName      string `json:"last_name"`
		Phone         string `json:"phone"`
		Permissions   struct {
			CanUpdateInventory bool `json:"can_update_inventory"`
			CanGenerateQR      bool `json:"can_generate_qr"`
			CanManageOrders    bool `json:"can_manage_orders"`
			CanManageUsers     bool `json:"can_manage_users"`
			CanSendPromotions  bool `json:"can_send_promotions"`
			CanViewRawStore    bool `json:"can_view_raw_store"`
			CanEditRawStore    bool `json:"can_edit_raw_store"`
		} `json:"permissions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verifica email esistente
	var existing models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	}

	// Hash password
	passwordHash, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Crea utente admin
	user := models.User{
		Email:        req.Email,
		PasswordHash: passwordHash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Phone:        req.Phone,
		Role:         models.RoleAdmin,
		IsActive:     true,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create admin"})
		return
	}

	// Crea permessi
	permissions := models.AdminPermission{
		AdminID:            user.ID,
		CanUpdateInventory: req.Permissions.CanUpdateInventory,
		CanGenerateQR:      req.Permissions.CanGenerateQR,
		CanManageOrders:    req.Permissions.CanManageOrders,
		CanManageUsers:     req.Permissions.CanManageUsers,
		CanSendPromotions:  req.Permissions.CanSendPromotions,
		CanViewRawStore:    req.Permissions.CanViewRawStore,
		CanEditRawStore:    req.Permissions.CanEditRawStore,
	}

	database.DB.Create(&permissions)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Admin created successfully",
		"admin":   user,
	})
}

// GetDashboard restituisce le statistiche della dashboard admin
func (h *AdminHandler) GetDashboard(c *gin.Context) {
	var stats struct {
		TotalProducts      int64   `json:"total_products"`
		TotalOrders        int64   `json:"total_orders"`
		TotalRevenue       float64 `json:"total_revenue"`
		TotalUsers         int64   `json:"total_users"`
		TotalDistributors  int64   `json:"total_distributors"`
		TotalQRScans       int64   `json:"total_qr_scans"`
		FakeProductAlerts  int64   `json:"fake_product_alerts"`
		PendingOrders      int64   `json:"pending_orders"`
		LowStockItems       int64   `json:"low_stock_items"`
	}

	// Conta prodotti
	database.DB.Model(&models.Product{}).Where("status = ?", models.ProductStatusActive).Count(&stats.TotalProducts)

	// Conta ordini
	database.DB.Model(&models.Order{}).Count(&stats.TotalOrders)

	// Revenue totale
	database.DB.Model(&models.Order{}).
		Where("payment_status = ?", models.PaymentStatusPaid).
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&stats.TotalRevenue)

	// Conta utenti
	database.DB.Model(&models.User{}).Where("is_active = ?", true).Count(&stats.TotalUsers)

	// Conta distributori
	database.DB.Model(&models.Distributor{}).Where("is_approved = ?", true).Count(&stats.TotalDistributors)

	// Conta scansioni QR
	database.DB.Model(&models.QRScanHistory{}).Count(&stats.TotalQRScans)

	// Alert prodotti falsi
	database.DB.Model(&models.QRScanHistory{}).
		Where("is_fake_alert = ?", true).
		Where("created_at > ?", time.Now().AddDate(0, 0, -30)).
		Count(&stats.FakeProductAlerts)

	// Ordini pending
	database.DB.Model(&models.Order{}).
		Where("status = ?", models.OrderStatusPending).
		Count(&stats.PendingOrders)

	// Items a basso stock (esempio: < 10)
	database.DB.Model(&models.InventoryItem{}).
		Where("quantity < ?", 10).
		Where("status = ?", "in_stock").
		Count(&stats.LowStockItems)

	c.JSON(http.StatusOK, gin.H{"dashboard": stats})
}

// GetAnalytics restituisce analytics dettagliate
func (h *AdminHandler) GetAnalytics(c *gin.Context) {
	// Analytics per periodo (default: ultimi 30 giorni)
	days := 30
	if d := c.Query("days"); d != "" {
		// TODO: Parse days
	}

	startDate := time.Now().AddDate(0, 0, -days)

	var analytics struct {
		SalesByDay        []map[string]interface{} `json:"sales_by_day"`
		TopProducts       []map[string]interface{}   `json:"top_products"`
		SalesByRegion     []map[string]interface{}   `json:"sales_by_region"`
		QRScansByDay      []map[string]interface{}   `json:"qr_scans_by_day"`
		AuthenticationRate float64                   `json:"authentication_rate"`
	}

	// Sales by day
	var salesByDay []struct {
		Date  time.Time `gorm:"column:date"`
		Total float64   `gorm:"column:total"`
	}
	database.DB.Model(&models.Order{}).
		Where("created_at >= ? AND payment_status = ?", startDate, models.PaymentStatusPaid).
		Select("DATE(created_at) as date, SUM(total_amount) as total").
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&salesByDay)

	for _, s := range salesByDay {
		analytics.SalesByDay = append(analytics.SalesByDay, map[string]interface{}{
			"date":  s.Date.Format("2006-01-02"),
			"total": s.Total,
		})
	}

	// Top products
	var topProducts []struct {
		ProductID   uuid.UUID `gorm:"column:product_id"`
		ProductName string    `gorm:"column:product_name"`
		TotalSold   int64     `gorm:"column:total_sold"`
		Revenue     float64   `gorm:"column:revenue"`
	}
	database.DB.Table("order_items").
		Select("product_id, products.name as product_name, SUM(quantity) as total_sold, SUM(total_price) as revenue").
		Joins("JOIN products ON order_items.product_id = products.id").
		Joins("JOIN orders ON order_items.order_id = orders.id").
		Where("orders.created_at >= ? AND orders.payment_status = ?", startDate, models.PaymentStatusPaid).
		Group("product_id, products.name").
		Order("total_sold DESC").
		Limit(10).
		Scan(&topProducts)

	for _, tp := range topProducts {
		analytics.TopProducts = append(analytics.TopProducts, map[string]interface{}{
			"product_id":   tp.ProductID,
			"product_name": tp.ProductName,
			"total_sold":   tp.TotalSold,
			"revenue":      tp.Revenue,
		})
	}

	// QR Scans by day
	var scansByDay []struct {
		Date  time.Time `gorm:"column:date"`
		Count int64     `gorm:"column:count"`
	}
	database.DB.Model(&models.QRScanHistory{}).
		Where("created_at >= ?", startDate).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&scansByDay)

	for _, s := range scansByDay {
		analytics.QRScansByDay = append(analytics.QRScansByDay, map[string]interface{}{
			"date":  s.Date.Format("2006-01-02"),
			"count": s.Count,
		})
	}

	// Authentication rate
	var totalScans int64
	var authenticScans int64
	database.DB.Model(&models.QRScanHistory{}).
		Where("created_at >= ?", startDate).
		Count(&totalScans)
	database.DB.Model(&models.QRScanHistory{}).
		Where("created_at >= ? AND is_authentic = ?", startDate, true).
		Count(&authenticScans)

	if totalScans > 0 {
		analytics.AuthenticationRate = float64(authenticScans) / float64(totalScans) * 100
	}

	c.JSON(http.StatusOK, gin.H{"analytics": analytics})
}

// ApproveDistributor approva un distributore
func (h *AdminHandler) ApproveDistributor(c *gin.Context) {
	id := c.Param("id")

	var distributor models.Distributor
	if err := database.DB.Preload("User").First(&distributor, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Distributor not found"})
		return
	}

	adminID, _ := c.Get("user_id")
	adminUUID, _ := adminID.(uuid.UUID)

	now := time.Now()
	distributor.IsApproved = true
	distributor.ApprovedAt = &now
	distributor.ApprovedBy = &adminUUID

	database.DB.Save(&distributor)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Distributor approved successfully",
		"distributor": distributor,
	})
}

// BadgeQRCode assegna un QR code a un distributore e regione (quando il distributore paga)
func (h *AdminHandler) BadgeQRCode(c *gin.Context) {
	var req struct {
		QRCodeID     string `json:"qr_code_id" binding:"required"`
		DistributorID string `json:"distributor_id" binding:"required"`
		RegionID     string `json:"region_id" binding:"required"`
		RegionName   string `json:"region_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	qrUUID, err := uuid.Parse(req.QRCodeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid QR code ID"})
		return
	}

	distributorUUID, err := uuid.Parse(req.DistributorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid distributor ID"})
		return
	}

	regionUUID, err := uuid.Parse(req.RegionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid region ID"})
		return
	}

	var qrCode models.QRCode
	if err := database.DB.First(&qrCode, qrUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "QR code not found"})
		return
	}

	// Aggiorna QR code
	qrCode.DistributorID = &distributorUUID
	qrCode.RegionID = &regionUUID
	database.DB.Save(&qrCode)

	// Aggiorna inventory item se esiste
	if qrCode.InventoryItemID != nil {
		database.DB.Model(&models.InventoryItem{}).
			Where("id = ?", qrCode.InventoryItemID).
			Updates(map[string]interface{}{
				"distributor_id": distributorUUID,
				"region_id":      regionUUID,
			})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "QR code badged successfully",
		"qr_code": qrCode,
	})
}

// GetPriceQuotes restituisce le richieste di preventivo
func (h *AdminHandler) GetPriceQuotes(c *gin.Context) {
	status := c.Query("status") // pending, approved, rejected

	var quotes []models.PriceQuote
	query := database.DB.Preload("Distributor").Preload("Product")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Order("created_at DESC").Find(&quotes)

	c.JSON(http.StatusOK, gin.H{"price_quotes": quotes})
}

// UpdatePriceQuote aggiorna un preventivo
func (h *AdminHandler) UpdatePriceQuote(c *gin.Context) {
	id := c.Param("id")

	var quote models.PriceQuote
	if err := database.DB.First(&quote, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Price quote not found"})
		return
	}

	var req struct {
		QuotedPrice *float64 `json:"quoted_price"`
		Status      *string  `json:"status"`
		Notes       *string  `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID, _ := c.Get("user_id")
	adminUUID, _ := adminID.(uuid.UUID)

	if req.QuotedPrice != nil {
		quote.QuotedPrice = req.QuotedPrice
	}
	if req.Status != nil {
		quote.Status = *req.Status
	}
	if req.Notes != nil {
		quote.Notes = *req.Notes
	}

	if quote.QuotedPrice != nil && quote.Status == "approved" {
		now := time.Now()
		quote.QuotedAt = &now
		quote.QuotedBy = &adminUUID
	}

	database.DB.Save(&quote)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Price quote updated successfully",
		"price_quote": quote,
	})
}
