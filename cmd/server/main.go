package main

import (
	"log"
	"time"

	"marvaron/internal/config"
	"marvaron/internal/database"
	"marvaron/internal/handlers"
	"marvaron/internal/middleware"
	"marvaron/internal/models"
	"marvaron/internal/utils"

	"github.com/gin-gonic/gin"
)

const (
	dbRetryAttempts = 15
	dbRetryDelay    = 2 * time.Second
)

func main() {
	if err := config.Load(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	router := setupRouter()

	// Listen on all interfaces so healthcheck can reach us (e.g. Railway)
	port := config.AppConfig.Server.Port
	addr := ":" + port
	log.Printf("Server starting on %s", addr)

	// Start HTTP server immediately so /health responds within retry window
	go func() {
		if err := router.Run(addr); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Give the server a moment to bind
	time.Sleep(100 * time.Millisecond)

	// Connect to DB with retries (DB may not be ready when container starts)
	var dbErr error
	for i := 0; i < dbRetryAttempts; i++ {
		dbErr = database.Connect()
		if dbErr == nil {
			break
		}
		log.Printf("Database connect attempt %d/%d failed: %v", i+1, dbRetryAttempts, dbErr)
		time.Sleep(dbRetryDelay)
	}
	if dbErr != nil {
		log.Fatalf("Failed to connect to database after %d attempts: %v", dbRetryAttempts, dbErr)
	}
	defer database.Close()

	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	if err := database.ConnectRedis(); err != nil {
		log.Printf("Warning: Redis unavailable: %v", err)
	} else {
		defer database.CloseRedis()
	}

	createSuperAdminIfNotExists()

	// Keep main alive
	select {}
}

func setupRouter() *gin.Engine {
	if config.AppConfig.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Middleware globali
	router.Use(middleware.CORSMiddleware())
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Inizializza handlers
	authHandler := &handlers.AuthHandler{}
	qrHandler := &handlers.QRHandler{}
	productHandler := &handlers.ProductHandler{}
	orderHandler := &handlers.OrderHandler{}
	adminHandler := &handlers.AdminHandler{}
	distributorHandler := &handlers.DistributorHandler{}

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Autenticazione (pubblica)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/otp/send", authHandler.SendOTP)
			auth.POST("/otp/verify", authHandler.VerifyOTP)
		}

		// Autenticazione (protetta)
		authProtected := v1.Group("/auth")
		authProtected.Use(middleware.AuthMiddleware())
		{
			authProtected.GET("/profile", authHandler.GetProfile)
			authProtected.PUT("/profile", authHandler.UpdateProfile)
			authProtected.DELETE("/account", authHandler.CloseAccount)
		}

		// QR Code (scansione pubblica, altre operazioni protette)
		qr := v1.Group("/qr")
		{
			qr.POST("/scan", middleware.OptionalAuthMiddleware(), qrHandler.ScanQR)
			qr.GET("/verify/:token", qrHandler.VerifyQR)
		}

		qrProtected := v1.Group("/qr")
		qrProtected.Use(middleware.AuthMiddleware())
		{
			qrProtected.GET("/history", qrHandler.GetScanHistory)
		}

		// QR Code Admin
		qrAdmin := v1.Group("/qr")
		qrAdmin.Use(middleware.AuthMiddleware())
		qrAdmin.Use(middleware.RoleMiddleware(models.RoleAdmin, models.RoleSuperAdmin))
		{
			qrAdmin.POST("/generate", qrHandler.GenerateQR)
			qrAdmin.PUT("/:id/display-info", qrHandler.UpdateQRDisplayInfo)
		}

		// Prodotti (pubblici)
		products := v1.Group("/products")
		{
			products.GET("", productHandler.GetProducts)
			products.GET("/:id", productHandler.GetProduct)
		}

		// Prodotti Admin
		productsAdmin := v1.Group("/products")
		productsAdmin.Use(middleware.AuthMiddleware())
		productsAdmin.Use(middleware.RoleMiddleware(models.RoleAdmin, models.RoleSuperAdmin))
		{
			productsAdmin.POST("", productHandler.CreateProduct)
			productsAdmin.PUT("/:id", productHandler.UpdateProduct)
			productsAdmin.DELETE("/:id", productHandler.DeleteProduct)
			productsAdmin.POST("/inventory", productHandler.AddInventoryItem)
		}

		// Ordini
		orders := v1.Group("/orders")
		orders.Use(middleware.OptionalAuthMiddleware())
		{
			orders.POST("", orderHandler.CreateOrder)
		}

		ordersProtected := v1.Group("/orders")
		ordersProtected.Use(middleware.AuthMiddleware())
		{
			ordersProtected.GET("", orderHandler.GetOrders)
			ordersProtected.GET("/:id", orderHandler.GetOrder)
			ordersProtected.PUT("/:id", orderHandler.UpdateOrder)
		}

		// Carrello
		cart := v1.Group("/cart")
		cart.Use(middleware.OptionalAuthMiddleware())
		{
			cart.GET("", orderHandler.GetCart)
			cart.POST("", orderHandler.AddToCart)
			cart.DELETE("/:id", orderHandler.RemoveFromCart)
		}

		// Distributore
		distributor := v1.Group("/distributor")
		distributor.Use(middleware.AuthMiddleware())
		distributor.Use(middleware.RoleMiddleware(models.RoleDistributor))
		{
			distributor.GET("/info", distributorHandler.GetDistributorInfo)
			distributor.PUT("/info", distributorHandler.UpdateDistributorInfo)
			distributor.POST("/price-quote", distributorHandler.RequestPriceQuote)
			distributor.GET("/price-quotes", distributorHandler.GetPriceQuotes)
		}

		// Admin
		admin := v1.Group("/admin")
		admin.Use(middleware.AuthMiddleware())
		admin.Use(middleware.RoleMiddleware(models.RoleAdmin, models.RoleSuperAdmin))
		{
			admin.GET("/dashboard", adminHandler.GetDashboard)
			admin.GET("/analytics", adminHandler.GetAnalytics)
			admin.GET("/price-quotes", adminHandler.GetPriceQuotes)
			admin.PUT("/price-quotes/:id", adminHandler.UpdatePriceQuote)
			admin.POST("/distributors/:id/approve", adminHandler.ApproveDistributor)
			admin.POST("/qr/badge", adminHandler.BadgeQRCode)
		}

		// Super Admin
		superAdmin := v1.Group("/admin")
		superAdmin.Use(middleware.AuthMiddleware())
		superAdmin.Use(middleware.RoleMiddleware(models.RoleSuperAdmin))
		{
			superAdmin.POST("/admins", adminHandler.CreateAdmin)
		}
	}

	return router
}

func createSuperAdminIfNotExists() {
	var count int64
	database.DB.Model(&models.User{}).Where("role = ?", models.RoleSuperAdmin).Count(&count)

	if count == 0 {
		log.Println("Creating default super admin...")
		
		// Crea super admin con credenziali di default
		// IMPORTANTE: Cambiare password al primo accesso!
		passwordHash, err := utils.HashPassword("admin123") // Cambiare in produzione!
		if err != nil {
			log.Printf("Warning: Failed to hash super admin password: %v", err)
			return
		}

		superAdmin := models.User{
			Email:        "admin@marvaron.com",
			PasswordHash: passwordHash,
			FirstName:    "Super",
			LastName:     "Admin",
			Role:         models.RoleSuperAdmin,
			IsActive:     true,
			IsEmailVerified: true,
		}

		if err := database.DB.Create(&superAdmin).Error; err != nil {
			log.Printf("Warning: Failed to create super admin: %v", err)
		} else {
			log.Println("Super admin created: admin@marvaron.com / admin123")
		}
	}
}
