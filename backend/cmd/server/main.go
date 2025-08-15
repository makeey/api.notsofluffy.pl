package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"notsofluffy-backend/internal/config"
	"notsofluffy-backend/internal/database"
	"notsofluffy-backend/internal/handlers"
	"notsofluffy-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Ensure uploads directory exists
	if err := os.MkdirAll("uploads/images", 0755); err != nil {
		log.Fatal("Failed to create uploads directory:", err)
	}

	r := gin.Default()

	// Initialize session store
	middleware.InitSessionStore(cfg.JWTSecret)

	// Security and proxy middleware (must be first)
	r.Use(middleware.TrustedProxyHeaders())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.RequestLogger())

	// Health check endpoint (before other middleware)
	r.Use(middleware.HealthCheck("/health"))

	// CORS middleware with proxy support
	r.Use(middleware.CORSWithProxy(cfg.AllowedOrigins))

	// Session middleware
	r.Use(middleware.SessionMiddleware())

	// Maintenance mode middleware
	r.Use(middleware.MaintenanceMiddleware(db, cfg.JWTSecret))

	// Static file serving for uploads
	r.Static("/uploads", "./uploads")

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, cfg.JWTSecret)
	adminHandler := handlers.NewAdminHandler(db)
	publicHandler := handlers.NewPublicHandler(db)
	cartHandler := handlers.NewCartHandler(db)
	profileHandler := handlers.NewProfileHandler(db)
	
	// Initialize order handler
	orderQueries := database.NewOrderQueries(db)
	cartQueries := database.NewCartQueries(db)
	stockQueries := database.NewStockQueries(db)
	discountQueries := database.NewDiscountQueries(db)
	orderHandler := handlers.NewOrderHandler(orderQueries, cartQueries, stockQueries, discountQueries)
	
	// Initialize discount handler
	discountHandler := handlers.NewDiscountHandler(discountQueries, cartQueries)

	// Public routes
	public := r.Group("/api")
	{
		public.GET("/categories", publicHandler.GetActiveCategories)
		public.GET("/products", publicHandler.GetPublicProducts)
		public.GET("/products/:id", publicHandler.GetPublicProduct)
		public.GET("/search", publicHandler.SearchProducts)
		public.GET("/search/suggestions", publicHandler.GetSearchSuggestions)
		public.GET("/maintenance-status", publicHandler.GetMaintenanceStatus)
		public.GET("/client-reviews", publicHandler.GetActiveClientReviews)
	}

	// Cart routes (public but require session)
	cart := r.Group("/api/cart")
	{
		cart.GET("", cartHandler.GetCart)
		cart.POST("/add", cartHandler.AddToCart)
		cart.PUT("/update/:id", cartHandler.UpdateCartItem)
		cart.DELETE("/remove/:id", cartHandler.RemoveFromCart)
		cart.POST("/clear", cartHandler.ClearCart)
		cart.GET("/count", cartHandler.GetCartCount)
		
		// Discount routes for cart
		cart.POST("/discount/apply", discountHandler.ApplyDiscountToCart)
		cart.DELETE("/discount/remove", discountHandler.RemoveDiscountFromCart)
	}

	// Auth routes
	auth := r.Group("/api/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.GET("/profile", middleware.AuthMiddleware(cfg.JWTSecret), authHandler.Profile)
	}

	// Order routes (with optional auth for user association)
	orders := r.Group("/api/orders")
	{
		orders.POST("", middleware.OptionalAuthMiddleware(cfg.JWTSecret), orderHandler.CreateOrder)
		orders.GET("/:id", middleware.OptionalAuthMiddleware(cfg.JWTSecret), orderHandler.GetOrder)
		orders.GET("/hash/:hash", orderHandler.GetOrderByHash)
	}

	// User routes (authenticated)
	user := r.Group("/api/user")
	user.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		user.GET("/orders", orderHandler.GetUserOrders)
		
		// Profile management
		user.GET("/profile", profileHandler.GetProfile)
		user.PUT("/profile", profileHandler.UpdateProfile)
		
		// Address management
		user.GET("/addresses", profileHandler.GetAddresses)
		user.POST("/addresses", profileHandler.CreateAddress)
		user.PUT("/addresses/:id", profileHandler.UpdateAddress)
		user.DELETE("/addresses/:id", profileHandler.DeleteAddress)
		user.PATCH("/addresses/:id/default", profileHandler.SetDefaultAddress)
	}

	// Admin routes
	admin := r.Group("/api/admin")
	admin.Use(middleware.AdminMiddleware(cfg.JWTSecret))
	{
		// User management
		admin.GET("/users", adminHandler.ListUsers)
		admin.POST("/users", adminHandler.CreateUser)
		admin.PUT("/users/:id", adminHandler.UpdateUser)
		admin.DELETE("/users/:id", adminHandler.DeleteUser)

		// Image management
		admin.POST("/images/upload", adminHandler.UploadImage)
		admin.GET("/images", adminHandler.ListImages)
		admin.DELETE("/images/:id", adminHandler.DeleteImage)

		// Category management
		admin.GET("/categories", adminHandler.ListCategories)
		admin.POST("/categories", adminHandler.CreateCategory)
		admin.GET("/categories/:id", adminHandler.GetCategory)
		admin.PUT("/categories/:id", adminHandler.UpdateCategory)
		admin.DELETE("/categories/:id", adminHandler.DeleteCategory)
		admin.PATCH("/categories/:id/toggle", adminHandler.ToggleCategoryActive)

		// Material management
		admin.GET("/materials", adminHandler.ListMaterials)
		admin.POST("/materials", adminHandler.CreateMaterial)
		admin.GET("/materials/:id", adminHandler.GetMaterial)
		admin.PUT("/materials/:id", adminHandler.UpdateMaterial)
		admin.DELETE("/materials/:id", adminHandler.DeleteMaterial)

		// Color management
		admin.GET("/colors", adminHandler.ListColors)
		admin.POST("/colors", adminHandler.CreateColor)
		admin.GET("/colors/:id", adminHandler.GetColor)
		admin.PUT("/colors/:id", adminHandler.UpdateColor)
		admin.DELETE("/colors/:id", adminHandler.DeleteColor)

		// Additional Service management
		admin.GET("/additional-services", adminHandler.ListAdditionalServices)
		admin.POST("/additional-services", adminHandler.CreateAdditionalService)
		admin.GET("/additional-services/:id", adminHandler.GetAdditionalService)
		admin.PUT("/additional-services/:id", adminHandler.UpdateAdditionalService)
		admin.DELETE("/additional-services/:id", adminHandler.DeleteAdditionalService)

		// Product management
		admin.GET("/products", adminHandler.ListProducts)
		admin.POST("/products", adminHandler.CreateProduct)
		admin.GET("/products/:id", adminHandler.GetProduct)
		admin.PUT("/products/:id", adminHandler.UpdateProduct)
		admin.DELETE("/products/:id", adminHandler.DeleteProduct)

		// Size management
		admin.GET("/sizes", adminHandler.ListSizes)
		admin.POST("/sizes", adminHandler.CreateSize)
		admin.GET("/sizes/:id", adminHandler.GetSize)
		admin.PUT("/sizes/:id", adminHandler.UpdateSize)
		admin.DELETE("/sizes/:id", adminHandler.DeleteSize)

		// Product Variant management
		admin.GET("/product-variants", adminHandler.ListProductVariants)
		admin.POST("/product-variants", adminHandler.CreateProductVariant)
		admin.GET("/product-variants/:id", adminHandler.GetProductVariant)
		admin.PUT("/product-variants/:id", adminHandler.UpdateProductVariant)
		admin.DELETE("/product-variants/:id", adminHandler.DeleteProductVariant)

		// Order management
		admin.GET("/orders", adminHandler.ListOrders)
		admin.GET("/orders/:id", adminHandler.GetOrderDetails)
		admin.PUT("/orders/:id/status", adminHandler.UpdateOrderStatus)
		admin.DELETE("/orders/:id", adminHandler.DeleteOrder)
		
		// Discount code management
		admin.GET("/discount-codes", discountHandler.GetDiscountCodes)
		admin.POST("/discount-codes", discountHandler.CreateDiscountCode)
		admin.GET("/discount-codes/:id", discountHandler.GetDiscountCode)
		admin.PUT("/discount-codes/:id", discountHandler.UpdateDiscountCode)
		admin.DELETE("/discount-codes/:id", discountHandler.DeleteDiscountCode)
		admin.GET("/discount-codes/:id/usage", discountHandler.GetDiscountCodeUsage)
		
		// Settings management
		admin.GET("/settings", adminHandler.GetSettings)
		admin.PUT("/settings/:key", adminHandler.UpdateSetting)
		
		// Client reviews management
		admin.GET("/client-reviews", adminHandler.ListClientReviews)
		admin.POST("/client-reviews", adminHandler.CreateClientReview)
		admin.GET("/client-reviews/:id", adminHandler.GetClientReview)
		admin.PUT("/client-reviews/:id", adminHandler.UpdateClientReview)
		admin.DELETE("/client-reviews/:id", adminHandler.DeleteClientReview)
		admin.POST("/client-reviews/reorder", adminHandler.ReorderClientReviews)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create HTTP server with proper timeouts
	server := &http.Server{
		Addr:           ":" + port,
		Handler:        r,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Configure trusted proxies for Gin
	if err := r.SetTrustedProxies([]string{"127.0.0.1", "::1"}); err != nil {
		log.Printf("Warning: Failed to set trusted proxies: %v", err)
	}

	// Log startup information
	log.Printf("=== NotSoFluffy API Server ===")
	log.Printf("Environment: %s", getEnv("ENVIRONMENT", "development"))
	log.Printf("Port: %s", port)
	log.Printf("Database SSL: %s", cfg.DBSSLMode)
	log.Printf("Allowed Origins: %v", cfg.AllowedOrigins)
	
	// Log SSL database info if enabled
	if cfg.DBSSLMode != "disable" {
		sslInfo := database.GetSSLInfo(cfg.DatabaseURL)
		log.Printf("Database SSL Info: %+v", sslInfo)
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server:", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	} else {
		log.Println("Server exited gracefully")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
