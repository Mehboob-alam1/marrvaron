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

type AuthHandler struct{}

type RegisterRequest struct {
	Email         string          `json:"email" binding:"required,email"`
	Password      string          `json:"password" binding:"required,min=8"`
	Phone         string          `json:"phone"`
	FirstName     string          `json:"first_name"`
	LastName      string          `json:"last_name"`
	Role          models.UserRole `json:"role" binding:"required"`
	MarketingOptIn bool          `json:"marketing_opt_in"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type OTPRequest struct {
	Identifier string `json:"identifier" binding:"required"` // email o phone
	Method     string `json:"method" binding:"required"`     // email o sms
}

type OTPVerifyRequest struct {
	Identifier string `json:"identifier" binding:"required"`
	OTP        string `json:"otp" binding:"required"`
}

// Register gestisce la registrazione di un nuovo utente
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verifica che l'email non esista già
	var existingUser models.User
	result := database.DB.Where("email = ?", req.Email).First(&existingUser)
	if result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	// Hash della password
	passwordHash, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Crea nuovo utente
	user := models.User{
		Email:          req.Email,
		Phone:          req.Phone,
		PasswordHash:   passwordHash,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Role:           req.Role,
		MarketingOptIn: req.MarketingOptIn,
		IsActive:       true,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Se è un distributore, crea il record Distributor
	if req.Role == models.RoleDistributor {
		distributor := models.Distributor{
			UserID:       user.ID,
			IsApproved:   false,
		}
		database.DB.Create(&distributor)
	}

	// Genera token JWT
	token, err := utils.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"token":   token,
		"user":    user,
	})
}

// Login gestisce il login dell'utente
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Trova l'utente
	var user models.User
	result := database.DB.Where("email = ?", req.Email).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Verifica password
	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Verifica che l'utente sia attivo
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "User account is inactive"})
		return
	}

	// Aggiorna last login
	now := time.Now()
	user.LastLoginAt = &now
	database.DB.Save(&user)

	// Genera token JWT
	token, err := utils.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
		"user":    user,
	})
}

// SendOTP invia un OTP all'utente
func (h *AuthHandler) SendOTP(c *gin.Context) {
	var req OTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Genera OTP
	otp, err := utils.GenerateOTP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
		return
	}

	// Salva OTP in Redis
	if err := utils.StoreOTP(req.Identifier, otp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store OTP"})
		return
	}

	// Invia OTP
	if err := utils.SendOTP(req.Identifier, otp, req.Method); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send OTP"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OTP sent successfully",
	})
}

// VerifyOTP verifica l'OTP
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req OTPVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verifica OTP
	valid, err := utils.VerifyOTP(req.Identifier, req.OTP)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify OTP"})
		return
	}

	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid OTP"})
		return
	}

	// Trova l'utente per email o phone
	var user models.User
	result := database.DB.Where("email = ? OR phone = ?", req.Identifier, req.Identifier).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Aggiorna verifica
	if user.Email == req.Identifier {
		user.IsEmailVerified = true
	} else {
		user.IsPhoneVerified = true
	}
	database.DB.Save(&user)

	// Genera token JWT
	token, err := utils.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OTP verified successfully",
		"token":   token,
		"user":    user,
	})
}

// GetProfile restituisce il profilo dell'utente autenticato
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var user models.User
	if err := database.DB.Preload("DistributorInfo").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// UpdateProfile aggiorna il profilo dell'utente
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var updateData struct {
		FirstName      *string `json:"first_name"`
		LastName       *string `json:"last_name"`
		Phone          *string `json:"phone"`
		MarketingOptIn *bool   `json:"marketing_opt_in"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if updateData.FirstName != nil {
		user.FirstName = *updateData.FirstName
	}
	if updateData.LastName != nil {
		user.LastName = *updateData.LastName
	}
	if updateData.Phone != nil {
		user.Phone = *updateData.Phone
	}
	if updateData.MarketingOptIn != nil {
		user.MarketingOptIn = *updateData.MarketingOptIn
	}

	database.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"user":    user,
	})
}

// CloseAccount chiude l'account dell'utente
func (h *AuthHandler) CloseAccount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.IsActive = false
	database.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{"message": "Account closed successfully"})
}
