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

type OrderHandler struct{}

type CreateOrderRequest struct {
	Items           []OrderItemRequest `json:"items" binding:"required"`
	ShippingAddress string             `json:"shipping_address"`
	BillingAddress  string             `json:"billing_address"`
	PaymentMethod   models.PaymentMethod `json:"payment_method"`
	Notes           string             `json:"notes"`
	SaveForLater    bool               `json:"save_for_later"` // Per distributori
}

type OrderItemRequest struct {
	ProductID string  `json:"product_id" binding:"required"`
	QRCodeID  *string `json:"qr_code_id"` // Opzionale, se aggiunto tramite scan
	Quantity  int     `json:"quantity" binding:"required,min=1"`
}

// CreateOrder crea un nuovo ordine
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ottieni user_id se autenticato
	var userID *uuid.UUID
	var distributorID *uuid.UUID
	var isGuest bool

	if uid, exists := c.Get("user_id"); exists {
		if u, ok := uid.(uuid.UUID); ok {
			userID = &u
			
			// Verifica se è un distributore
			var user models.User
			database.DB.Preload("DistributorInfo").First(&user, u)
			if user.Role == models.RoleDistributor && user.DistributorInfo != nil {
				distributorID = &user.DistributorInfo.ID
			}
		}
	} else {
		isGuest = true
	}

	// Calcola totali
	var subTotal float64
	var orderItems []models.OrderItem

	for _, itemReq := range req.Items {
		var product models.Product
		if err := database.DB.First(&product, itemReq.ProductID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found: " + itemReq.ProductID})
			return
		}

		// Determina il prezzo (potrebbe essere un prezzo speciale per distributore)
		unitPrice := product.BasePrice
		if distributorID != nil {
			// TODO: Verifica se c'è un price quote per questo distributore
		}

		totalPrice := unitPrice * float64(itemReq.Quantity)

		orderItem := models.OrderItem{
			ProductID:  product.ID,
			Quantity:   itemReq.Quantity,
			UnitPrice:  unitPrice,
			TotalPrice: totalPrice,
		}

		// Se c'è un QR code, associalo
		if itemReq.QRCodeID != nil {
			qrUUID, err := uuid.Parse(*itemReq.QRCodeID)
			if err == nil {
				orderItem.QRCodeID = &qrUUID
			}
		}

		orderItems = append(orderItems, orderItem)
		subTotal += totalPrice
	}

	// Crea ordine
	order := models.Order{
		UserID:          userID,
		DistributorID:   distributorID,
		Status:          models.OrderStatusPending,
		PaymentStatus:   models.PaymentStatusPending,
		PaymentMethod:   req.PaymentMethod,
		SubTotal:        subTotal,
		TaxAmount:       0, // TODO: Calcola tasse
		ShippingCost:    0, // TODO: Calcola shipping
		TotalAmount:     subTotal,
		ShippingAddress: req.ShippingAddress,
		BillingAddress:  req.BillingAddress,
		Notes:           req.Notes,
		IsGuestOrder:    isGuest,
	}

	if req.SaveForLater {
		order.Status = models.OrderStatusSaved
	}

	if err := database.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	// Crea order items
	for i := range orderItems {
		orderItems[i].OrderID = order.ID
	}
	database.DB.Create(&orderItems)

	// Se il pagamento è immediato, processa il pagamento
	if !req.SaveForLater && req.PaymentMethod != "" {
		// TODO: Integrare gateway di pagamento
		// Per ora, se il metodo è "cash" o "bank_transfer", lo lasciamo pending
		if req.PaymentMethod == models.PaymentMethodCard {
			// Processa pagamento con gateway
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Order created successfully",
		"order":   order,
	})
}

// GetOrders restituisce la lista degli ordini dell'utente
func (h *OrderHandler) GetOrders(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var orders []models.Order
	database.DB.Where("user_id = ?", userID).
		Preload("Items").
		Preload("Items.Product").
		Order("created_at DESC").
		Find(&orders)

	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

// GetOrder restituisce i dettagli di un ordine
func (h *OrderHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")

	var order models.Order
	if err := database.DB.Preload("Items").
		Preload("Items.Product").
		Preload("User").
		Preload("Distributor").
		First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Verifica che l'utente abbia accesso a questo ordine
	userID, exists := c.Get("user_id")
	if exists {
		uid, _ := userID.(uuid.UUID)
		if order.UserID != nil && *order.UserID != uid {
			// Verifica se è admin
			role, _ := c.Get("role")
			if role != models.RoleAdmin && role != models.RoleSuperAdmin {
				c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"order": order})
}

// UpdateOrder aggiorna un ordine (Admin o Distributore per i propri ordini)
func (h *OrderHandler) UpdateOrder(c *gin.Context) {
	id := c.Param("id")

	var order models.Order
	if err := database.DB.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var req struct {
		Status        *models.OrderStatus   `json:"status"`
		PaymentStatus *models.PaymentStatus `json:"payment_status"`
		Notes         *string               `json:"notes"`
		ShippingAddress *string             `json:"shipping_address"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Status != nil {
		order.Status = *req.Status
		if *req.Status == models.OrderStatusShipped {
			now := time.Now()
			order.ShippedAt = &now
		}
		if *req.Status == models.OrderStatusDelivered {
			now := time.Now()
			order.DeliveredAt = &now
		}
	}
	if req.PaymentStatus != nil {
		order.PaymentStatus = *req.PaymentStatus
		if *req.PaymentStatus == models.PaymentStatusPaid {
			now := time.Now()
			order.PaidAt = &now
		}
	}
	if req.Notes != nil {
		order.Notes = *req.Notes
	}
	if req.ShippingAddress != nil {
		order.ShippingAddress = *req.ShippingAddress
	}

	database.DB.Save(&order)

	c.JSON(http.StatusOK, gin.H{
		"message": "Order updated successfully",
		"order":   order,
	})
}

// AddToCart aggiunge un prodotto al carrello
func (h *OrderHandler) AddToCart(c *gin.Context) {
	var req struct {
		ProductID string  `json:"product_id" binding:"required"`
		QRCodeID  *string `json:"qr_code_id"`
		Quantity  int     `json:"quantity" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var product models.Product
	if err := database.DB.First(&product, req.ProductID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	var userID *uuid.UUID
	var sessionID string

	if uid, exists := c.Get("user_id"); exists {
		if u, ok := uid.(uuid.UUID); ok {
			userID = &u
		}
	} else {
		// Per utenti guest, usa session ID
		sessionID = c.GetHeader("X-Session-ID")
		if sessionID == "" {
			sessionID = uuid.New().String()
		}
	}

	// Verifica se l'item esiste già nel carrello
	var existingCart models.Cart
	query := database.DB
	if userID != nil {
		query = query.Where("user_id = ? AND product_id = ?", userID, req.ProductID)
	} else {
		query = query.Where("session_id = ? AND product_id = ?", sessionID, req.ProductID)
	}

	if err := query.First(&existingCart).Error; err == nil {
		// Aggiorna quantità
		existingCart.Quantity += req.Quantity
		database.DB.Save(&existingCart)
		c.JSON(http.StatusOK, gin.H{
			"message": "Cart item updated",
			"cart":    existingCart,
		})
		return
	}

	// Crea nuovo item nel carrello
	cart := models.Cart{
		UserID:    userID,
		SessionID: sessionID,
		ProductID: product.ID,
		Quantity:  req.Quantity,
		UnitPrice: product.BasePrice,
	}

	if req.QRCodeID != nil {
		qrUUID, err := uuid.Parse(*req.QRCodeID)
		if err == nil {
			cart.QRCodeID = &qrUUID
		}
	}

	database.DB.Create(&cart)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Item added to cart",
		"cart":    cart,
	})
}

// GetCart restituisce il carrello dell'utente
func (h *OrderHandler) GetCart(c *gin.Context) {
	var userID *uuid.UUID
	var sessionID string

	if uid, exists := c.Get("user_id"); exists {
		if u, ok := uid.(uuid.UUID); ok {
			userID = &u
		}
	} else {
		sessionID = c.GetHeader("X-Session-ID")
	}

	var cartItems []models.Cart
	query := database.DB.Preload("Product").Preload("QRCode")
	if userID != nil {
		query = query.Where("user_id = ?", userID)
	} else if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	} else {
		c.JSON(http.StatusOK, gin.H{"cart": []models.Cart{}})
		return
	}

	query.Find(&cartItems)

	var total float64
	for _, item := range cartItems {
		total += item.UnitPrice * float64(item.Quantity)
	}

	c.JSON(http.StatusOK, gin.H{
		"cart":  cartItems,
		"total": total,
	})
}

// RemoveFromCart rimuove un item dal carrello
func (h *OrderHandler) RemoveFromCart(c *gin.Context) {
	id := c.Param("id")

	var cart models.Cart
	query := database.DB

	// Verifica ownership
	if uid, exists := c.Get("user_id"); exists {
		u, _ := uid.(uuid.UUID)
		query = query.Where("id = ? AND user_id = ?", id, u)
	} else {
		sessionID := c.GetHeader("X-Session-ID")
		query = query.Where("id = ? AND session_id = ?", id, sessionID)
	}

	if err := query.First(&cart).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cart item not found"})
		return
	}

	database.DB.Delete(&cart)

	c.JSON(http.StatusOK, gin.H{"message": "Item removed from cart"})
}
