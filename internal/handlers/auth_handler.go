package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"marvaron/internal/database"
	"marvaron/internal/models"
	"marvaron/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type AuthHandler struct{}

// --- Legacy single-step register (still supported for tests / simple clients) ---

type RegisterRequest struct {
	Email            string          `json:"email" binding:"required,email"`
	Password         string          `json:"password" binding:"required,min=8"`
	Phone            string          `json:"phone"`
	FullName         string          `json:"full_name"`
	FirstName        string          `json:"first_name"`
	LastName         string          `json:"last_name"`
	StreetAddress    string          `json:"street_address"`
	City             string          `json:"city"`
	State            string          `json:"state"`
	PostCode         string          `json:"post_code"`
	Country          string          `json:"country"`
	Role             models.UserRole `json:"role" binding:"required"`
	ReferralCodeUsed string          `json:"referral_code"` // optional referrer's code
	MarketingOptIn   bool            `json:"marketing_opt_in"`
}

// --- Multi-step registration ---

type RegisterProfileRequest struct {
	FullName      string `json:"full_name" binding:"required"`
	Phone         string `json:"phone" binding:"required"`
	Email         string `json:"email" binding:"required,email"`
	StreetAddress string `json:"street_address" binding:"required"`
	City          string `json:"city" binding:"required"`
	State         string `json:"state" binding:"required"`
	PostCode      string `json:"post_code" binding:"required"`
	Country       string `json:"country" binding:"required"`
	MarketingOptIn bool `json:"marketing_opt_in"`
}

type RegisterPasswordRequest struct {
	RegistrationToken string `json:"registration_token"` // if not using Authorization header
	Password          string `json:"password" binding:"required,min=8"`
	ReferralCode      string `json:"referral_code"` // optional: another user's code
}

type RegisterVerifyPhoneRequest struct {
	RegistrationToken string `json:"registration_token"`
	OTP               string `json:"otp" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type SwitchRoleRequest struct {
	ActiveRole models.UserRole `json:"active_role" binding:"required"`
}

type EnableRoleRequest struct {
	Role models.UserRole `json:"role" binding:"required"`
}

type OTPRequest struct {
	Identifier string `json:"identifier" binding:"required"`
	Method     string `json:"method" binding:"required"`
}

type OTPVerifyRequest struct {
	Identifier string `json:"identifier" binding:"required"`
	OTP        string `json:"otp" binding:"required"`
}

var allowedRegisterRoles = map[models.UserRole]bool{
	models.RoleCustomer:    true,
	models.RoleDistributor: true,
	models.RoleCourier:     true,
	models.RoleVendor:      true,
}

var selfServiceRoles = map[models.UserRole]bool{
	models.RoleDistributor: true,
	models.RoleCourier:     true,
	models.RoleVendor:      true,
}

func splitFullName(full string) (first, last string) {
	full = strings.TrimSpace(full)
	if full == "" {
		return "", ""
	}
	parts := strings.SplitN(full, " ", 2)
	first = parts[0]
	if len(parts) > 1 {
		last = strings.TrimSpace(parts[1])
	}
	return first, last
}

func generateUniqueReferralCode() (string, error) {
	for i := 0; i < 20; i++ {
		b := make([]byte, 4)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		code := strings.ToUpper(hex.EncodeToString(b))
		var n int64
		database.DB.Model(&models.User{}).Where("referral_code = ?", code).Count(&n)
		if n == 0 {
			return code, nil
		}
	}
	return "", errors.New("could not generate unique referral code")
}

func registrationTokenFromRequest(c *gin.Context, bodyToken string) (uuid.UUID, error) {
	if authHeader := c.GetHeader("Authorization"); authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			if uid, err := utils.ParseRegistrationToken(parts[1]); err == nil {
				return uid, nil
			}
		}
	}
	if strings.TrimSpace(bodyToken) != "" {
		return utils.ParseRegistrationToken(strings.TrimSpace(bodyToken))
	}
	return uuid.Nil, errors.New("missing registration token")
}

// RegisterProfile — step 1: contact & address (no password yet)
func (h *AuthHandler) RegisterProfile(c *gin.Context) {
	var req RegisterProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing models.User
	if database.DB.Where("email = ?", strings.ToLower(strings.TrimSpace(req.Email))).First(&existing).Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	first, last := splitFullName(req.FullName)
	user := models.User{
		Email:                  strings.ToLower(strings.TrimSpace(req.Email)),
		Phone:                  strings.TrimSpace(req.Phone),
		FullName:               strings.TrimSpace(req.FullName),
		FirstName:              first,
		LastName:               last,
		StreetAddress:          strings.TrimSpace(req.StreetAddress),
		City:                   strings.TrimSpace(req.City),
		StateRegion:            strings.TrimSpace(req.State),
		PostCode:               strings.TrimSpace(req.PostCode),
		Country:                strings.TrimSpace(req.Country),
		Role:                   models.RoleCustomer,
		Roles:                  pq.StringArray{string(models.RoleCustomer)},
		RegistrationComplete:   false,
		PasswordHash:           "",
		MarketingOptIn:         req.MarketingOptIn,
		IsActive:               true,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user", "details": err.Error()})
		return
	}

	regTok, err := utils.GenerateRegistrationToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to issue registration token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":             "Profile saved. Set password and optional referral code next, then verify phone OTP.",
		"registration_token":  regTok,
		"user_id":             user.ID,
		"next_step":           "password",
		"authorization_hint": "Send this token as: Authorization: Bearer <registration_token> on the next requests, or pass registration_token in JSON body.",
	})
}

// RegisterPassword — step 2: password + optional referral; sends OTP to phone
func (h *AuthHandler) RegisterPassword(c *gin.Context) {
	var req RegisterPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid, err := registrationTokenFromRequest(c, req.RegistrationToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Valid registration_token required (Bearer or body)"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Registration not found"})
		return
	}
	if user.RegistrationComplete {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Registration already completed; use login"})
		return
	}

	if user.PasswordHash == "" {
		hash, err := utils.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		user.PasswordHash = hash

		if code := strings.TrimSpace(req.ReferralCode); code != "" {
			var ref models.User
			if err := database.DB.Where("referral_code = ? AND id <> ?", code, user.ID).First(&ref).Error; err == nil {
				user.ReferredByUserID = &ref.ID
			}
		}

		if err := database.DB.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save password"})
			return
		}
	}

	otp, err := utils.GenerateOTP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
		return
	}
	if err := utils.StoreOTP(user.Phone, otp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store OTP"})
		return
	}
	_ = utils.SendOTP(user.Phone, otp, "sms")

	regTok, _ := utils.GenerateRegistrationToken(user.ID)
	c.JSON(http.StatusOK, gin.H{
		"message":            "OTP sent to your phone number",
		"registration_token": regTok,
		"next_step":          "verify_phone",
	})
}

// RegisterVerifyPhone — step 3: OTP on phone completes signup
func (h *AuthHandler) RegisterVerifyPhone(c *gin.Context) {
	var req RegisterVerifyPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid, err := registrationTokenFromRequest(c, req.RegistrationToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Valid registration_token required"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Registration not found"})
		return
	}
	if user.RegistrationComplete {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Already verified; use login"})
		return
	}
	if user.PasswordHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Set password first (register/password)"})
		return
	}

	ok, err := utils.VerifyOTP(user.Phone, req.OTP)
	if err != nil || !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired OTP"})
		return
	}

	code, err := generateUniqueReferralCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate referral code"})
		return
	}
	user.ReferralCode = &code
	user.IsPhoneVerified = true
	user.RegistrationComplete = true
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize registration"})
		return
	}

	token, err := utils.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Registration complete",
		"token":   token,
		"user":    user,
	})
}

// Register legacy single-step (password + role at once)
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !allowedRegisterRoles[req.Role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role. Allowed: customer, distributor, courier, vendor"})
		return
	}

	var existingUser models.User
	if database.DB.Where("email = ?", req.Email).First(&existingUser).Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	passwordHash, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	first, last := splitFullName(req.FullName)
	if req.FirstName != "" {
		first = req.FirstName
	}
	if req.LastName != "" {
		last = req.LastName
	}
	displayName := strings.TrimSpace(req.FullName)
	if displayName == "" {
		displayName = strings.TrimSpace(strings.TrimSpace(first + " " + last))
	}

	roles := pq.StringArray{string(req.Role)}
	user := models.User{
		Email:                req.Email,
		Phone:                req.Phone,
		PasswordHash:         passwordHash,
		FullName:             displayName,
		FirstName:            first,
		LastName:             last,
		StreetAddress:        req.StreetAddress,
		City:                 req.City,
		StateRegion:          req.State,
		PostCode:             req.PostCode,
		Country:              req.Country,
		Role:                 req.Role,
		Roles:                roles,
		MarketingOptIn:       req.MarketingOptIn,
		IsActive:             true,
		RegistrationComplete: true,
		IsPhoneVerified:      false,
	}

	if code := strings.TrimSpace(req.ReferralCodeUsed); code != "" {
		var ref models.User
		if err := database.DB.Where("referral_code = ?", code).First(&ref).Error; err == nil {
			user.ReferredByUserID = &ref.ID
		}
	}

	refCode, err := generateUniqueReferralCode()
	if err == nil {
		user.ReferralCode = &refCode
	}

	if err := database.DB.Create(&user).Error; err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "localhost") || strings.Contains(errMsg, "connection refused") {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Database not configured. On Railway: add PostgreSQL, then in your service Variables add DATABASE_URL (Add Reference → Postgres → DATABASE_URL).",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user", "details": errMsg})
		return
	}

	if req.Role == models.RoleDistributor {
		distributor := models.Distributor{UserID: user.ID, IsApproved: false}
		database.DB.Create(&distributor)
	}

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

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !user.RegistrationComplete || user.PasswordHash == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Complete registration first (profile → password → phone OTP)"})
		return
	}

	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "User account is inactive"})
		return
	}

	now := time.Now()
	user.LastLoginAt = &now
	database.DB.Save(&user)

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

func (h *AuthHandler) SendOTP(c *gin.Context) {
	var req OTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	otp, err := utils.GenerateOTP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
		return
	}

	if err := utils.StoreOTP(req.Identifier, otp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store OTP"})
		return
	}

	_ = utils.SendOTP(req.Identifier, otp, req.Method)

	c.JSON(http.StatusOK, gin.H{
		"message": "OTP sent successfully",
	})
}

func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req OTPVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	valid, err := utils.VerifyOTP(req.Identifier, req.OTP)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify OTP"})
		return
	}

	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid OTP"})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ? OR phone = ?", req.Identifier, req.Identifier).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if !user.RegistrationComplete || user.PasswordHash == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Complete registration (password + phone OTP) before using this OTP flow"})
		return
	}

	if user.Email == req.Identifier {
		user.IsEmailVerified = true
	} else {
		user.IsPhoneVerified = true
	}
	database.DB.Save(&user)

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

// SwitchRole sets active role context (must already be in roles list)
func (h *AuthHandler) SwitchRole(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return
	}

	var req SwitchRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	target := req.ActiveRole
	if target == models.RoleSuperAdmin || target == models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot switch to admin via this endpoint"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if !user.HasRole(target) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have this role. Enable it first.", "your_roles": user.RolesAsStrings()})
		return
	}

	user.Role = target
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update role"})
		return
	}

	token, err := utils.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Active role updated",
		"token":       token,
		"active_role": user.Role,
		"roles":       user.RolesAsStrings(),
	})
}

// EnableRole adds distributor / courier / vendor to the user (MVP: no approval workflow)
func (h *AuthHandler) EnableRole(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return
	}

	var req EnableRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !selfServiceRoles[req.Role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only distributor, courier, or vendor can be self-enabled"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.HasRole(req.Role) {
		c.JSON(http.StatusOK, gin.H{"message": "Role already enabled", "roles": user.RolesAsStrings()})
		return
	}

	next := make(pq.StringArray, 0, len(user.Roles)+1)
	next = append(next, user.Roles...)
	next = append(next, string(req.Role))
	user.Roles = next
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save roles"})
		return
	}

	if req.Role == models.RoleDistributor {
		var d models.Distributor
		if err := database.DB.Where("user_id = ?", user.ID).First(&d).Error; err != nil {
			database.DB.Create(&models.Distributor{UserID: user.ID, IsApproved: false})
		}
	}

	token, err := utils.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Role enabled",
		"roles":   user.RolesAsStrings(),
		"token":   token,
		"user":    user,
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
		return
	}

	var user models.User
	if err := database.DB.Preload("DistributorInfo").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
		"roles": user.RolesAsStrings(),
		"active_role": user.Role,
	})
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var updateData struct {
		FullName      *string `json:"full_name"`
		FirstName     *string `json:"first_name"`
		LastName      *string `json:"last_name"`
		Phone         *string `json:"phone"`
		StreetAddress *string `json:"street_address"`
		City          *string `json:"city"`
		State         *string `json:"state"`
		PostCode      *string `json:"post_code"`
		Country       *string `json:"country"`
		MarketingOptIn *bool  `json:"marketing_opt_in"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if updateData.FullName != nil {
		user.FullName = *updateData.FullName
		f, l := splitFullName(*updateData.FullName)
		user.FirstName, user.LastName = f, l
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
	if updateData.StreetAddress != nil {
		user.StreetAddress = *updateData.StreetAddress
	}
	if updateData.City != nil {
		user.City = *updateData.City
	}
	if updateData.State != nil {
		user.StateRegion = *updateData.State
	}
	if updateData.PostCode != nil {
		user.PostCode = *updateData.PostCode
	}
	if updateData.Country != nil {
		user.Country = *updateData.Country
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

func (h *AuthHandler) CloseAccount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
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
