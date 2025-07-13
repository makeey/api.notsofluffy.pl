package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"notsofluffy-backend/internal/database"
	"notsofluffy-backend/internal/models"
)

type DiscountHandler struct {
	discountQueries *database.DiscountQueries
	cartQueries     *database.CartQueries
}

func NewDiscountHandler(discountQueries *database.DiscountQueries, cartQueries *database.CartQueries) *DiscountHandler {
	return &DiscountHandler{
		discountQueries: discountQueries,
		cartQueries:     cartQueries,
	}
}

// ApplyDiscountToCart applies a discount code to the current cart
func (h *DiscountHandler) ApplyDiscountToCart(c *gin.Context) {
	var req models.ApplyDiscountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Normalize discount code to uppercase
	code := strings.ToUpper(strings.TrimSpace(req.Code))

	// Get session ID
	sessionID, exists := c.Get("session_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No session found"})
		return
	}
	sessionIDStr := sessionID.(string)

	// Get user ID if authenticated
	var userID *int
	if userIDValue, exists := c.Get("user_id"); exists {
		if id, ok := userIDValue.(int); ok {
			userID = &id
		}
	}

	// Get or create cart session
	cartSession, err := h.cartQueries.GetOrCreateCartSession(sessionIDStr, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cart session"})
		return
	}

	// Get current cart
	cart, err := h.cartQueries.GetCartItems(cartSession.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cart items"})
		return
	}

	if len(cart) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cart is empty"})
		return
	}

	// Calculate current cart total
	var cartTotal float64
	for _, item := range cart {
		cartTotal += item.TotalPrice
	}

	// Validate discount code
	validationResult, err := h.discountQueries.ValidateDiscountCode(code, cartTotal, userID, sessionIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate discount code"})
		return
	}

	if !validationResult.IsValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": validationResult.ErrorMessage})
		return
	}

	// Apply discount to cart session
	err = h.discountQueries.ApplyDiscountToCartSession(cartSession.ID, validationResult.DiscountCode.ID, validationResult.DiscountAmount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply discount"})
		return
	}

	// Return success response
	response := models.ApplyDiscountResponse{
		Code:            validationResult.DiscountCode.Code,
		Description:     validationResult.DiscountCode.Description,
		DiscountType:    validationResult.DiscountCode.DiscountType,
		DiscountValue:   validationResult.DiscountCode.DiscountValue,
		DiscountAmount:  validationResult.DiscountAmount,
		OriginalTotal:   cartTotal,
		DiscountedTotal: cartTotal - validationResult.DiscountAmount,
		Message:         "Discount applied successfully",
	}

	c.JSON(http.StatusOK, response)
}

// RemoveDiscountFromCart removes the applied discount from cart
func (h *DiscountHandler) RemoveDiscountFromCart(c *gin.Context) {
	// Get session ID
	sessionID, exists := c.Get("session_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No session found"})
		return
	}
	sessionIDStr := sessionID.(string)

	// Get user ID if authenticated
	var userID *int
	if userIDValue, exists := c.Get("user_id"); exists {
		if id, ok := userIDValue.(int); ok {
			userID = &id
		}
	}

	// Get cart session
	cartSession, err := h.cartQueries.GetOrCreateCartSession(sessionIDStr, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cart session"})
		return
	}

	// Remove discount from cart session
	err = h.discountQueries.RemoveDiscountFromCartSession(cartSession.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove discount"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Discount removed successfully"})
}

// Admin endpoints below

// CreateDiscountCode creates a new discount code (admin only)
func (h *DiscountHandler) CreateDiscountCode(c *gin.Context) {
	var req models.DiscountCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get admin user ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Normalize code to uppercase
	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))

	// Validate percentage discount
	if req.DiscountType == models.DiscountTypePercentage && req.DiscountValue > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Percentage discount cannot exceed 100%"})
		return
	}

	// Validate date range
	if req.EndDate != nil && req.EndDate.Before(req.StartDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End date must be after start date"})
		return
	}

	// Create discount code
	adminUserID := userID.(int)
	discountCode, err := h.discountQueries.CreateDiscountCode(&req, adminUserID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{"error": "Discount code already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create discount code"})
		return
	}

	c.JSON(http.StatusCreated, discountCode)
}

// GetDiscountCodes lists all discount codes (admin only)
func (h *DiscountHandler) GetDiscountCodes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	active := c.Query("active")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var activeFilter *bool
	if active == "true" {
		activeTrue := true
		activeFilter = &activeTrue
	} else if active == "false" {
		activeFalse := false
		activeFilter = &activeFalse
	}

	discountCodes, err := h.discountQueries.GetDiscountCodes(page, limit, activeFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get discount codes"})
		return
	}

	c.JSON(http.StatusOK, discountCodes)
}

// GetDiscountCode gets a specific discount code (admin only)
func (h *DiscountHandler) GetDiscountCode(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid discount code ID"})
		return
	}

	discountCode, err := h.discountQueries.GetDiscountCodeByID(id)
	if err != nil {
		if err.Error() == "discount code not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Discount code not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get discount code"})
		return
	}

	c.JSON(http.StatusOK, discountCode)
}

// UpdateDiscountCode updates a discount code (admin only)
func (h *DiscountHandler) UpdateDiscountCode(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid discount code ID"})
		return
	}

	var req models.DiscountCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Normalize code to uppercase
	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))

	// Validate percentage discount
	if req.DiscountType == models.DiscountTypePercentage && req.DiscountValue > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Percentage discount cannot exceed 100%"})
		return
	}

	// Validate date range
	if req.EndDate != nil && req.EndDate.Before(req.StartDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End date must be after start date"})
		return
	}

	discountCode, err := h.discountQueries.UpdateDiscountCode(id, &req)
	if err != nil {
		if err.Error() == "discount code not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Discount code not found"})
			return
		}
		if strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{"error": "Discount code already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update discount code"})
		return
	}

	c.JSON(http.StatusOK, discountCode)
}

// DeleteDiscountCode deletes a discount code (admin only)
func (h *DiscountHandler) DeleteDiscountCode(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid discount code ID"})
		return
	}

	err = h.discountQueries.DeleteDiscountCode(id)
	if err != nil {
		if err.Error() == "discount code not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Discount code not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete discount code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Discount code deleted successfully"})
}

// GetDiscountCodeUsage gets usage statistics for a discount code (admin only)
func (h *DiscountHandler) GetDiscountCodeUsage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid discount code ID"})
		return
	}

	usage, err := h.discountQueries.GetDiscountCodeUsage(id)
	if err != nil {
		if err.Error() == "discount code not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Discount code not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get usage statistics"})
		return
	}

	c.JSON(http.StatusOK, usage)
}