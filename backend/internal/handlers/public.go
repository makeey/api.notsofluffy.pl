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