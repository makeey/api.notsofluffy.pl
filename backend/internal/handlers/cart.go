package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"notsofluffy-backend/internal/database"
	"notsofluffy-backend/internal/middleware"
	"notsofluffy-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// CartHandler handles cart-related requests
type CartHandler struct {
	db              *sql.DB
	cartQueries     *database.CartQueries
	productQueries  *database.ProductQueries
	variantQueries  *database.ProductVariantQueries
	sizeQueries     *database.SizeQueries
	serviceQueries  *database.AdditionalServiceQueries
	stockQueries    *database.StockQueries
	discountQueries *database.DiscountQueries
}

// NewCartHandler creates a new cart handler
func NewCartHandler(db *sql.DB) *CartHandler {
	return &CartHandler{
		db:              db,
		cartQueries:     database.NewCartQueries(db),
		productQueries:  database.NewProductQueries(db),
		variantQueries:  database.NewProductVariantQueries(db),
		sizeQueries:     database.NewSizeQueries(db),
		serviceQueries:  database.NewAdditionalServiceQueries(db),
		stockQueries:    database.NewStockQueries(db),
		discountQueries: database.NewDiscountQueries(db),
	}
}

// GetCart returns the current cart contents
func (h *CartHandler) GetCart(c *gin.Context) {
	sessionID := middleware.GetSessionID(c)
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No session found"})
		return
	}

	// Get user ID if authenticated
	var userID *int
	if userIDInterface, exists := c.Get("user_id"); exists {
		uid := userIDInterface.(int)
		userID = &uid
	}

	// Get or create cart session
	cartSession, err := h.cartQueries.GetOrCreateCartSession(sessionID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cart session", "details": err.Error()})
		return
	}

	// Get cart items
	items, err := h.cartQueries.GetCartItems(cartSession.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cart items", "details": err.Error()})
		return
	}

	// Calculate totals
	var totalItems int
	var subtotal float64
	for _, item := range items {
		totalItems += item.Quantity
		subtotal += item.TotalPrice
	}

	// Get discount information if applied
	var appliedDiscount *models.CartDiscount
	discountAmount := cartSession.DiscountAmount
	if cartSession.AppliedDiscountCodeID != nil {
		// Get discount code details
		discountCode, err := h.discountQueries.GetDiscountCodeByID(*cartSession.AppliedDiscountCodeID)
		if err == nil {
			appliedDiscount = &models.CartDiscount{
				CodeID:          discountCode.ID,
				Code:            discountCode.Code,
				Description:     discountCode.Description,
				DiscountType:    discountCode.DiscountType,
				DiscountValue:   discountCode.DiscountValue,
				DiscountAmount:  discountAmount,
			}
		}
	}

	totalPrice := subtotal - discountAmount
	if totalPrice < 0 {
		totalPrice = 0
	}

	response := models.CartResponse{
		Items:           items,
		TotalItems:      totalItems,
		Subtotal:        subtotal,
		DiscountAmount:  discountAmount,
		TotalPrice:      totalPrice,
		AppliedDiscount: appliedDiscount,
	}

	c.JSON(http.StatusOK, response)
}

// AddToCart adds an item to the cart
func (h *CartHandler) AddToCart(c *gin.Context) {
	var req models.CartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessionID := middleware.GetSessionID(c)
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No session found"})
		return
	}

	// Get user ID if authenticated
	var userID *int
	if userIDInterface, exists := c.Get("user_id"); exists {
		uid := userIDInterface.(int)
		userID = &uid
	}

	// Get or create cart session
	cartSession, err := h.cartQueries.GetOrCreateCartSession(sessionID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cart session", "details": err.Error()})
		return
	}

	// Validate product exists
	_, err = h.productQueries.GetProduct(req.ProductID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Validate variant exists and belongs to product
	variant, err := h.variantQueries.GetProductVariantByID(req.VariantID)
	if err != nil || variant.ProductID != req.ProductID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid variant for this product"})
		return
	}

	// Validate size exists and belongs to product
	size, err := h.sizeQueries.GetSizeByID(req.SizeID)
	if err != nil || size.Product.ID != req.ProductID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid size for this product"})
		return
	}

	// Check stock availability
	available, availableStock, err := h.stockQueries.CheckStockAvailability(req.SizeID, req.Quantity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check stock availability", "details": err.Error()})
		return
	}
	
	if !available {
		if availableStock == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "This size is out of stock"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Insufficient stock available", 
				"available_stock": availableStock,
				"requested_quantity": req.Quantity,
			})
		}
		return
	}

	// Validate additional services exist
	var totalServicePrice float64
	for _, serviceID := range req.AdditionalServiceIDs {
		service, err := h.serviceQueries.GetAdditionalServiceByID(serviceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid additional service ID"})
			return
		}
		totalServicePrice += service.Price
	}

	// Calculate price per item
	pricePerItem := size.BasePrice
	
	// Apply 10% markup for custom colors
	if variant.Color.Custom {
		pricePerItem *= 1.1
	}
	
	// Add additional services price
	pricePerItem += totalServicePrice

	// Add item to cart
	_, err = h.cartQueries.AddCartItem(cartSession.ID, &req, pricePerItem)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add item to cart", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Item added to cart successfully"})
}

// UpdateCartItem updates the quantity of a cart item
func (h *CartHandler) UpdateCartItem(c *gin.Context) {
	cartItemID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cart item ID"})
		return
	}

	var req models.CartItemUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessionID := middleware.GetSessionID(c)
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No session found"})
		return
	}

	// Get user ID if authenticated
	var userID *int
	if userIDInterface, exists := c.Get("user_id"); exists {
		uid := userIDInterface.(int)
		userID = &uid
	}

	// Get cart session
	cartSession, err := h.cartQueries.GetOrCreateCartSession(sessionID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cart session", "details": err.Error()})
		return
	}

	// Verify the cart item belongs to this session (security check)
	items, err := h.cartQueries.GetCartItems(cartSession.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cart items", "details": err.Error()})
		return
	}

	var currentItem *models.CartItemResponse
	for _, item := range items {
		if item.ID == cartItemID {
			currentItem = &item
			break
		}
	}

	if currentItem == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cart item not found"})
		return
	}

	// Check stock availability for the new quantity
	available, availableStock, err := h.stockQueries.CheckStockAvailability(currentItem.SizeID, req.Quantity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check stock availability", "details": err.Error()})
		return
	}
	
	if !available {
		if availableStock == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "This size is out of stock"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Insufficient stock available", 
				"available_stock": availableStock,
				"requested_quantity": req.Quantity,
			})
		}
		return
	}

	// Update quantity
	_, err = h.cartQueries.UpdateCartItemQuantity(cartItemID, req.Quantity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update cart item", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cart item updated successfully"})
}

// RemoveFromCart removes an item from the cart
func (h *CartHandler) RemoveFromCart(c *gin.Context) {
	cartItemID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cart item ID"})
		return
	}

	sessionID := middleware.GetSessionID(c)
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No session found"})
		return
	}

	// Get user ID if authenticated
	var userID *int
	if userIDInterface, exists := c.Get("user_id"); exists {
		uid := userIDInterface.(int)
		userID = &uid
	}

	// Get cart session
	cartSession, err := h.cartQueries.GetOrCreateCartSession(sessionID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cart session", "details": err.Error()})
		return
	}

	// Verify the cart item belongs to this session (security check)
	items, err := h.cartQueries.GetCartItems(cartSession.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cart items", "details": err.Error()})
		return
	}

	found := false
	for _, item := range items {
		if item.ID == cartItemID {
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cart item not found"})
		return
	}

	// Remove item
	err = h.cartQueries.RemoveCartItem(cartItemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove cart item", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item removed from cart successfully"})
}

// ClearCart removes all items from the cart
func (h *CartHandler) ClearCart(c *gin.Context) {
	sessionID := middleware.GetSessionID(c)
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No session found"})
		return
	}

	// Get user ID if authenticated
	var userID *int
	if userIDInterface, exists := c.Get("user_id"); exists {
		uid := userIDInterface.(int)
		userID = &uid
	}

	// Get cart session
	cartSession, err := h.cartQueries.GetOrCreateCartSession(sessionID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cart session", "details": err.Error()})
		return
	}

	// Clear cart
	err = h.cartQueries.ClearCart(cartSession.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear cart", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cart cleared successfully"})
}

// GetCartCount returns the number of items in the cart
func (h *CartHandler) GetCartCount(c *gin.Context) {
	sessionID := middleware.GetSessionID(c)
	if sessionID == "" {
		c.JSON(http.StatusOK, models.CartCountResponse{Count: 0})
		return
	}

	// Get user ID if authenticated
	var userID *int
	if userIDInterface, exists := c.Get("user_id"); exists {
		uid := userIDInterface.(int)
		userID = &uid
	}

	// Get cart session
	cartSession, err := h.cartQueries.GetOrCreateCartSession(sessionID, userID)
	if err != nil {
		c.JSON(http.StatusOK, models.CartCountResponse{Count: 0})
		return
	}

	// Get cart count
	count, err := h.cartQueries.GetCartItemCount(cartSession.ID)
	if err != nil {
		c.JSON(http.StatusOK, models.CartCountResponse{Count: 0})
		return
	}

	c.JSON(http.StatusOK, models.CartCountResponse{Count: count})
}