package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"notsofluffy-backend/internal/database"
	"notsofluffy-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// PublicHandler handles public API requests
type PublicHandler struct {
	db              *sql.DB
	categoryQueries *database.CategoryQueries
	productQueries  *database.ProductQueries
}

// NewPublicHandler creates a new public handler
func NewPublicHandler(db *sql.DB) *PublicHandler {
	return &PublicHandler{
		db:              db,
		categoryQueries: database.NewCategoryQueries(db),
		productQueries:  database.NewProductQueries(db),
	}
}

// GetActiveCategories returns all active categories with images
func (h *PublicHandler) GetActiveCategories(c *gin.Context) {
	categories, err := h.categoryQueries.GetActiveCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}

	// Convert to response format
	categoryResponses := make([]models.CategoryResponse, len(categories))
	for i, cat := range categories {
		categoryResponses[i] = models.CategoryResponse{
			ID:        cat.ID,
			Name:      cat.Name,
			Slug:      cat.Slug,
			ImageID:   cat.ImageID,
			Active:    cat.Active,
			ChartOnly: cat.ChartOnly,
			CreatedAt: cat.CreatedAt.Format(time.RFC3339),
			UpdatedAt: cat.UpdatedAt.Format(time.RFC3339),
			Image:     cat.Image,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"categories": categoryResponses,
		"total":      len(categoryResponses),
	})
}

// GetPublicProducts returns products with filtering and pagination for public access
func (h *PublicHandler) GetPublicProducts(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))
	search := c.Query("search")
	
	// Parse category filter (can be multiple)
	categoryNames := c.QueryArray("category")
	var categoryIDs []int
	if len(categoryNames) > 0 {
		// Convert category names to IDs
		for _, name := range categoryNames {
			name = strings.TrimSpace(name)
			if name != "" {
				// Get category by name/slug
				categories, err := h.categoryQueries.GetActiveCategories()
				if err == nil {
					for _, cat := range categories {
						if cat.Name == name || cat.Slug == name {
							categoryIDs = append(categoryIDs, cat.ID)
							break
						}
					}
				}
			}
		}
	}

	// Call the database query method
	products, err := h.productQueries.GetPublicProducts(page, limit, search, categoryIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products", "details": err.Error()})
		return
	}

	// Get total count for pagination
	total, err := h.productQueries.GetPublicProductsCount(search, categoryIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product count", "details": err.Error()})
		return
	}

	// Convert to response format
	productResponses := make([]models.ProductResponse, len(products))
	for i, product := range products {
		productResponses[i] = models.ProductResponse{
			ID:               product.ID,
			Name:             product.Name,
			ShortDescription: product.ShortDescription,
			Description:      product.Description,
			MaterialID:       product.MaterialID,
			MainImageID:      product.MainImageID,
			CategoryID:       product.CategoryID,
			CreatedAt:        product.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
			Material:         product.Material,
			MainImage:        product.MainImage,
			Category:         product.Category,
			Images:           product.Images,
			AdditionalServices: product.AdditionalServices,
			MinPrice:         product.MinPrice,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"products": productResponses,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

// GetPublicProduct returns a single product with all details for public access
func (h *PublicHandler) GetPublicProduct(c *gin.Context) {
	// Parse product ID from URL
	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// Get product with all relations
	product, err := h.productQueries.GetProduct(productID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product", "details": err.Error()})
		return
	}

	// Convert to response format
	productResponse := models.ProductResponse{
		ID:               product.ID,
		Name:             product.Name,
		ShortDescription: product.ShortDescription,
		Description:      product.Description,
		MaterialID:       product.MaterialID,
		MainImageID:      product.MainImageID,
		CategoryID:       product.CategoryID,
		CreatedAt:        product.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
		Material:         product.Material,
		MainImage:        product.MainImage,
		Category:         product.Category,
		Images:           product.Images,
		AdditionalServices: product.AdditionalServices,
		MinPrice:         product.MinPrice,
	}

	// Get product variants
	variants, err := h.productQueries.GetProductVariants(productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product variants", "details": err.Error()})
		return
	}

	// Get product sizes
	sizes, err := h.productQueries.GetProductSizes(productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product sizes", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"product":  productResponse,
		"variants": variants,
		"sizes":    sizes,
	})
}

// SearchProducts handles dedicated search functionality with enhanced features
func (h *PublicHandler) SearchProducts(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))
	query := strings.TrimSpace(c.Query("q"))
	sortBy := c.DefaultQuery("sort", "relevance") // relevance, price_asc, price_desc, newest
	
	// Parse category filter
	categoryNames := c.QueryArray("category")
	var categoryIDs []int
	if len(categoryNames) > 0 {
		// Convert category names to IDs
		for _, name := range categoryNames {
			name = strings.TrimSpace(name)
			if name != "" {
				categories, err := h.categoryQueries.GetActiveCategories()
				if err == nil {
					for _, cat := range categories {
						if cat.Name == name || cat.Slug == name {
							categoryIDs = append(categoryIDs, cat.ID)
							break
						}
					}
				}
			}
		}
	}

	// Validate and set limits
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 12
	}

	// If no search query, return popular/recent products
	if query == "" {
		products, err := h.productQueries.GetPublicProducts(page, limit, "", categoryIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products", "details": err.Error()})
			return
		}

		total, err := h.productQueries.GetPublicProductsCount("", categoryIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product count", "details": err.Error()})
			return
		}

		// Convert to response format
		productResponses := make([]models.ProductResponse, len(products))
		for i, product := range products {
			productResponses[i] = models.ProductResponse{
				ID:               product.ID,
				Name:             product.Name,
				ShortDescription: product.ShortDescription,
				Description:      product.Description,
				MaterialID:       product.MaterialID,
				MainImageID:      product.MainImageID,
				CategoryID:       product.CategoryID,
				CreatedAt:        product.CreatedAt.Format(time.RFC3339),
				UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
				Material:         product.Material,
				MainImage:        product.MainImage,
				Category:         product.Category,
				Images:           product.Images,
				AdditionalServices: product.AdditionalServices,
				MinPrice:         product.MinPrice,
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"products": productResponses,
			"total":    total,
			"page":     page,
			"limit":    limit,
			"query":    query,
			"sort":     sortBy,
			"suggestion": "Try searching for 'sweater', 'coat', or browse our categories",
		})
		return
	}

	// Perform search with the query
	products, err := h.productQueries.SearchProductsEnhanced(page, limit, query, categoryIDs, sortBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed", "details": err.Error()})
		return
	}

	// Get total count for pagination
	total, err := h.productQueries.GetSearchProductsCount(query, categoryIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get search count", "details": err.Error()})
		return
	}

	// Convert to response format
	productResponses := make([]models.ProductResponse, len(products))
	for i, product := range products {
		productResponses[i] = models.ProductResponse{
			ID:               product.ID,
			Name:             product.Name,
			ShortDescription: product.ShortDescription,
			Description:      product.Description,
			MaterialID:       product.MaterialID,
			MainImageID:      product.MainImageID,
			CategoryID:       product.CategoryID,
			CreatedAt:        product.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
			Material:         product.Material,
			MainImage:        product.MainImage,
			Category:         product.Category,
			Images:           product.Images,
			AdditionalServices: product.AdditionalServices,
			MinPrice:         product.MinPrice,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"products": productResponses,
		"total":    total,
		"page":     page,
		"limit":    limit,
		"query":    query,
		"sort":     sortBy,
	})
}

// GetSearchSuggestions provides autocomplete suggestions for search
func (h *PublicHandler) GetSearchSuggestions(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))
	
	if limit < 1 || limit > 10 {
		limit = 5
	}

	if len(query) < 2 {
		c.JSON(http.StatusOK, gin.H{
			"suggestions": []string{},
			"query":       query,
		})
		return
	}

	suggestions, err := h.productQueries.GetSearchSuggestions(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get suggestions", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"suggestions": suggestions,
		"query":       query,
	})
}