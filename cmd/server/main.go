package main

import (
	"fmt"
	"log"

	"marvaron/internal/config"
	"marvaron/internal/database"
	"marvaron/internal/handlers"
	"marvaron/internal/middleware"
	"marvaron/internal/models"
	"marvaron/internal/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// Carica configurazione
	if err := config.Load(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connetti al database
	if err := database.Connect(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Esegui migrations
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Connetti a Redis
	if err := database.ConnectRedis(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
	} else {
		defer database.CloseRedis()
	}

	// Inizializza Kafka (opzionale)
	// kafka.Init()
	// defer kafka.Close()

	// Crea super admin se non esiste
	createSuperAdminIfNotExists()

	// Setup router
	router := setupRouter()

	// Avvia server
	addr := fmt.Sprintf("%s:%s", config.AppConfig.Server.Host, config.AppConfig.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
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
