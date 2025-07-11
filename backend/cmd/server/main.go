package main

import (
	"log"
	"os"

	"notsofluffy-backend/internal/config"
	"notsofluffy-backend/internal/database"
	"notsofluffy-backend/internal/handlers"
	"notsofluffy-backend/internal/middleware"

	"github.com/gin-contrib/cors"
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

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Session middleware
	r.Use(middleware.SessionMiddleware())

	// Static file serving for uploads
	r.Static("/uploads", "./uploads")

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, cfg.JWTSecret)
	adminHandler := handlers.NewAdminHandler(db)
	publicHandler := handlers.NewPublicHandler(db)
	cartHandler := handlers.NewCartHandler(db)

	// Public routes
	public := r.Group("/api")
	{
		public.GET("/categories", publicHandler.GetActiveCategories)
		public.GET("/products", publicHandler.GetPublicProducts)
		public.GET("/products/:id", publicHandler.GetPublicProduct)
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
	}

	// Auth routes
	auth := r.Group("/api/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.GET("/profile", middleware.AuthMiddleware(cfg.JWTSecret), authHandler.Profile)
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
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
