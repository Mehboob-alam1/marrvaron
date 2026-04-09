package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"marvaron/internal/config"
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
	Role             models.UserRole `json:"role"` // ignored; new accounts are always role "user"
	ReferralCodeUsed string          `json:"referral_code"` // optional referrer's code
	MarketingOptIn   bool            `json:"marketing_opt_in"`
}

// SignupRequest is the primary one-step registration payload (email OTP completes signup).
type SignupRequest struct {
	FullName       string `json:"full_name" binding:"required"`
	Phone          string `json:"phone" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	StreetAddress  string `json:"street_address" binding:"required"`
	City           string `json:"city" binding:"required"`
	State          string `json:"state" binding:"required"`
	PostCode       string `json:"post_code" binding:"required"`
	Country        string `json:"country" binding:"required"`
	Password       string `json:"password" binding:"required,min=8"`
	ReferralCode   string `json:"referral_code"`
	MarketingOptIn bool   `json:"marketing_opt_in"`
}

type SignupVerifyOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
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

type RequestRoleBody struct {
	RequestedRole models.UserRole `json:"requested_role" binding:"required"`
}

type OTPRequest struct {
	Identifier string `json:"identifier" binding:"required"`
	Method     string `json:"method" binding:"required"`
}

type OTPVerifyRequest struct {
	Identifier string `json:"identifier" binding:"required"`
	OTP        string `json:"otp" binding:"required"`
}

// Elevated roles a user may ask for; admin must approve via /admin/role-requests.
var requestableElevatedRoles = map[models.UserRole]bool{
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

func emailOTPKey(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Signup creates a user with role "user", hashes password, and emails an OTP. Login is allowed only after SignupVerifyOTP.
func (h *AuthHandler) Signup(c *gin.Context) {
	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	email := emailOTPKey(req.Email)
	var existing models.User
	if database.DB.Where("email = ?", email).First(&existing).Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	first, last := splitFullName(req.FullName)
	user := models.User{
		Email:                 email,
		Phone:                 strings.TrimSpace(req.Phone),
		FullName:              strings.TrimSpace(req.FullName),
		FirstName:             first,
		LastName:              last,
		StreetAddress:         strings.TrimSpace(req.StreetAddress),
		City:                  strings.TrimSpace(req.City),
		StateRegion:           strings.TrimSpace(req.State),
		PostCode:              strings.TrimSpace(req.PostCode),
		Country:               strings.TrimSpace(req.Country),
		PasswordHash:          hash,
		Role:                  models.RoleUser,
		Roles:                 pq.StringArray{string(models.RoleUser)},
		RegistrationComplete:  false,
		IsEmailVerified:       false,
		IsPhoneVerified:       false,
		MarketingOptIn:        req.MarketingOptIn,
		IsActive:              true,
	}

	if code := strings.TrimSpace(req.ReferralCode); code != "" {
		var ref models.User
		if err := database.DB.Where("referral_code = ?", code).First(&ref).Error; err == nil {
			user.ReferredByUserID = &ref.ID
		}
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user", "details": err.Error()})
		return
	}

	otp, err := utils.GenerateOTP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
		return
	}
	if err := utils.StoreOTP(email, otp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store OTP"})
		return
	}
	_ = utils.SendOTP(email, otp, "email")

	c.JSON(http.StatusCreated, gin.H{
		"message":    "We sent a verification code to your email. Submit it to /auth/signup/verify-otp to finish signup.",
		"user_id":    user.ID,
		"created_at": user.CreatedAt,
	})
}

// SignupVerifyOTP completes registration after email OTP; returns JWT.
func (h *AuthHandler) SignupVerifyOTP(c *gin.Context) {
	var req SignupVerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key := emailOTPKey(req.Email)
	var user models.User
	if err := database.DB.Where("email = ?", key).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No signup found for this email"})
		return
	}
	if user.RegistrationComplete && user.IsEmailVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email already verified; use login"})
		return
	}

	ok, err := utils.VerifyOTP(key, req.OTP)
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
	user.IsEmailVerified = true
	user.RegistrationComplete = true
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize signup"})
		return
	}

	token, err := utils.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Signup complete",
		"token":   token,
		"user":    user,
	})
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
		Role:                   models.RoleUser,
		Roles:                  pq.StringArray{string(models.RoleUser)},
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
		"message":             "Profile saved. Set password and optional referral code next, then verify email OTP.",
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
	ident := emailOTPKey(user.Email)
	if err := utils.StoreOTP(ident, otp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store OTP"})
		return
	}
	_ = utils.SendOTP(ident, otp, "email")

	regTok, _ := utils.GenerateRegistrationToken(user.ID)
	c.JSON(http.StatusOK, gin.H{
		"message":            "OTP sent to your email address",
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

	ok, err := utils.VerifyOTP(emailOTPKey(user.Email), req.OTP)
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
	user.IsEmailVerified = true
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

// Register legacy single-step: always creates role "user" (elevated roles require admin approval after /auth/roles/request).
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	email := emailOTPKey(req.Email)
	var existingUser models.User
	if database.DB.Where("email = ?", email).First(&existingUser).Error == nil {
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

	roles := pq.StringArray{string(models.RoleUser)}
	user := models.User{
		Email:                 email,
		Phone:                 req.Phone,
		PasswordHash:          passwordHash,
		FullName:              displayName,
		FirstName:             first,
		LastName:              last,
		StreetAddress:         req.StreetAddress,
		City:                  req.City,
		StateRegion:           req.State,
		PostCode:              req.PostCode,
		Country:               req.Country,
		Role:                  models.RoleUser,
		Roles:                 roles,
		MarketingOptIn:        req.MarketingOptIn,
		IsActive:              true,
		RegistrationComplete:  true,
		IsEmailVerified:       true,
		IsPhoneVerified:       false,
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
	if err := database.DB.Where("email = ?", emailOTPKey(req.Email)).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !user.RegistrationComplete || user.PasswordHash == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Complete registration first (e.g. verify email OTP after signup)"})
		return
	}

	if !user.IsEmailVerified {
		c.JSON(http.StatusForbidden, gin.H{"error": "Verify your email with the code we sent you before logging in"})
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

// ForgotPassword emails a reset link (always responds OK to avoid email enumeration).
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	email := emailOTPKey(req.Email)
	var user models.User
	if err := database.DB.Where("email = ? AND is_active = ?", email, true).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "If an account exists for that email, we sent a password reset link."})
		return
	}
	if user.PasswordHash == "" || !user.RegistrationComplete {
		c.JSON(http.StatusOK, gin.H{"message": "If an account exists for that email, we sent a password reset link."})
		return
	}

	tok, err := utils.GeneratePasswordResetToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not issue reset link"})
		return
	}

	base := config.AppConfig.AppPublicURL
	resetURL := base + "/reset-password?token=" + url.QueryEscape(tok)
	if base == "" {
		resetURL = "(set APP_PUBLIC_URL) /reset-password?token=" + tok
	}

	_ = utils.SendPasswordResetEmail(email, resetURL)

	c.JSON(http.StatusOK, gin.H{"message": "If an account exists for that email, we sent a password reset link."})
}

// ResetPassword sets a new password using the token from the forgot-password email.
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid, err := utils.ParsePasswordResetToken(strings.TrimSpace(req.Token))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired reset link"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is inactive"})
		return
	}

	hash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	user.PasswordHash = hash
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password updated. You can log in with your new password."})
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
		c.JSON(http.StatusForbidden, gin.H{"error": "Complete registration before using this OTP flow"})
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

// RequestRole submits a pending request for distributor, courier, or vendor; an admin approves via /admin/role-requests.
func (h *AuthHandler) RequestRole(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		return
	}

	var req RequestRoleBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !requestableElevatedRoles[req.RequestedRole] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You can only request: distributor, courier, or vendor"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.HasRole(req.RequestedRole) {
		c.JSON(http.StatusOK, gin.H{"message": "You already have this role", "roles": user.RolesAsStrings()})
		return
	}

	var pending int64
	database.DB.Model(&models.RoleRequest{}).
		Where("user_id = ? AND requested_role = ? AND status = ?", user.ID, req.RequestedRole, models.RoleRequestPending).
		Count(&pending)
	if pending > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "A pending request for this role already exists"})
		return
	}

	rr := models.RoleRequest{
		UserID:          user.ID,
		RequestedRole:   req.RequestedRole,
		Status:          models.RoleRequestPending,
	}
	if err := database.DB.Create(&rr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":        "Role request submitted. An administrator will review it.",
		"role_request_id": rr.ID,
		"status":         rr.Status,
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
