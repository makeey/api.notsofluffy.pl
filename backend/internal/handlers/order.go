package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"notsofluffy-backend/internal/database"
	"notsofluffy-backend/internal/models"
)

type OrderHandler struct {
	orderQueries *database.OrderQueries
	cartQueries  *database.CartQueries
	stockQueries *database.StockQueries
}

func NewOrderHandler(orderQueries *database.OrderQueries, cartQueries *database.CartQueries, stockQueries *database.StockQueries) *OrderHandler {
	return &OrderHandler{
		orderQueries: orderQueries,
		cartQueries:  cartQueries,
		stockQueries: stockQueries,
	}
}

// CreateOrder creates a new order from cart
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req models.OrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

	// Get cart items
	items, err := h.cartQueries.GetCartItems(cartSession.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cart items"})
		return
	}

	if len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cart is empty"})
		return
	}

	// Calculate totals
	var totalItems int
	var totalPrice float64
	for _, item := range items {
		totalItems += item.Quantity
		totalPrice += item.TotalPrice
	}

	cart := models.CartResponse{
		Items:      items,
		TotalItems: totalItems,
		TotalPrice: totalPrice,
	}

	// Calculate totals
	subtotal := cart.TotalPrice
	shippingCost := 0.0 // TODO: implement shipping calculation
	taxAmount := 0.0    // TODO: implement tax calculation
	totalAmount := subtotal + shippingCost + taxAmount

	// Create order
	order := &models.Order{
		UserID:        userID,
		SessionID:     &sessionIDStr,
		Email:         req.Email,
		Phone:         req.Phone,
		Status:        models.OrderStatusPending,
		TotalAmount:   totalAmount,
		Subtotal:      subtotal,
		ShippingCost:  shippingCost,
		TaxAmount:     taxAmount,
		PaymentMethod: req.PaymentMethod,
		PaymentStatus: models.PaymentStatusPending,
		Notes:         req.Notes,
	}

	// Create shipping address
	shippingAddr := &models.ShippingAddress{
		FirstName:     req.ShippingAddress.FirstName,
		LastName:      req.ShippingAddress.LastName,
		Company:       req.ShippingAddress.Company,
		AddressLine1:  req.ShippingAddress.AddressLine1,
		AddressLine2:  req.ShippingAddress.AddressLine2,
		City:          req.ShippingAddress.City,
		StateProvince: req.ShippingAddress.StateProvince,
		PostalCode:    req.ShippingAddress.PostalCode,
		Country:       req.ShippingAddress.Country,
		Phone:         req.ShippingAddress.Phone,
	}

	// Create billing address
	billingAddr := &models.BillingAddress{
		FirstName:      req.BillingAddress.FirstName,
		LastName:       req.BillingAddress.LastName,
		Company:        req.BillingAddress.Company,
		AddressLine1:   req.BillingAddress.AddressLine1,
		AddressLine2:   req.BillingAddress.AddressLine2,
		City:           req.BillingAddress.City,
		StateProvince:  req.BillingAddress.StateProvince,
		PostalCode:     req.BillingAddress.PostalCode,
		Country:        req.BillingAddress.Country,
		Phone:          req.BillingAddress.Phone,
		SameAsShipping: req.SameAsShipping,
	}

	// Reserve stock for all items first
	var stockReservations []struct {
		SizeID   int
		Quantity int
	}
	
	for _, cartItem := range cart.Items {
		// Check and reserve stock
		available, availableStock, err := h.stockQueries.CheckStockAvailability(cartItem.SizeID, cartItem.Quantity)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check stock availability", "details": err.Error()})
			return
		}
		
		if !available {
			if availableStock == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "One or more items are out of stock", "size_id": cartItem.SizeID})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Insufficient stock for one or more items",
					"size_id": cartItem.SizeID,
					"available_stock": availableStock,
					"requested_quantity": cartItem.Quantity,
				})
			}
			return
		}
		
		// Reserve stock
		err = h.stockQueries.ReserveStock(cartItem.SizeID, cartItem.Quantity)
		if err != nil {
			// Release any previously reserved stock
			for _, reservation := range stockReservations {
				h.stockQueries.ReleaseStock(reservation.SizeID, reservation.Quantity)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reserve stock", "details": err.Error()})
			return
		}
		
		stockReservations = append(stockReservations, struct {
			SizeID   int
			Quantity int
		}{cartItem.SizeID, cartItem.Quantity})
	}

	// Convert cart items to order items
	var orderItems []models.OrderItem
	for _, cartItem := range cart.Items {
		// Create size dimensions map
		sizeDimensions := map[string]interface{}{
			"a": cartItem.Size.A,
			"b": cartItem.Size.B,
			"c": cartItem.Size.C,
			"d": cartItem.Size.D,
			"e": cartItem.Size.E,
			"f": cartItem.Size.F,
		}

		orderItem := models.OrderItem{
			ProductID:          cartItem.ProductID,
			ProductName:        cartItem.Product.Name,
			ProductDescription: &cartItem.Product.Description,
			VariantID:          cartItem.VariantID,
			VariantName:        cartItem.Variant.Name,
			VariantColorName:   &cartItem.Variant.Color.Name,
			VariantColorCustom: cartItem.Variant.Color.Custom,
			SizeID:             cartItem.SizeID,
			SizeName:           cartItem.Size.Name,
			SizeDimensions:     sizeDimensions,
			Quantity:           cartItem.Quantity,
			UnitPrice:          cartItem.PricePerItem,
			TotalPrice:         cartItem.TotalPrice,
		}

		// Convert additional services
		for _, service := range cartItem.AdditionalServices {
			orderItem.Services = append(orderItem.Services, models.OrderItemService{
				ServiceID:          service.ID,
				ServiceName:        service.Name,
				ServiceDescription: &service.Description,
				ServicePrice:       service.Price,
			})
		}

		orderItems = append(orderItems, orderItem)
	}

	// Create order in database
	orderResponse, err := h.orderQueries.CreateOrder(order, shippingAddr, billingAddr, orderItems)
	if err != nil {
		// Release reserved stock if order creation fails
		for _, reservation := range stockReservations {
			h.stockQueries.ReleaseStock(reservation.SizeID, reservation.Quantity)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	// Decrement stock for all items after successful order creation
	for _, reservation := range stockReservations {
		err = h.stockQueries.DecrementStock(reservation.SizeID, reservation.Quantity)
		if err != nil {
			// Log error but don't fail the request since order was created
			// TODO: implement proper logging
			// In a production system, you might want to track this for inventory correction
		}
	}

	// Clear cart after successful order
	err = h.cartQueries.ClearCart(cartSession.ID)
	if err != nil {
		// Log error but don't fail the request since order was created
		// TODO: implement proper logging
	}

	c.JSON(http.StatusCreated, orderResponse)
}

// GetOrder retrieves an order by ID
func (h *OrderHandler) GetOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	order, err := h.orderQueries.GetOrderByID(id)
	if err != nil {
		if err.Error() == "order not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get order"})
		return
	}

	// Check if user has permission to view this order
	if userIDValue, exists := c.Get("user_id"); exists {
		if userID, ok := userIDValue.(int); ok {
			// User can view their own orders
			if order.UserID != nil && *order.UserID == userID {
				c.JSON(http.StatusOK, order)
				return
			}
			// Admin can view all orders (check role)
			if userRole, roleExists := c.Get("user_role"); roleExists {
				if role, roleOk := userRole.(string); roleOk && role == "admin" {
					c.JSON(http.StatusOK, order)
					return
				}
			}
		}
	}

	// For guest orders, check session
	sessionID, exists := c.Get("session_id")
	if exists && order.SessionID != nil && *order.SessionID == sessionID.(string) {
		c.JSON(http.StatusOK, order)
		return
	}

	c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
}

// ListOrders lists orders for admin
func (h *OrderHandler) ListOrders(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	email := c.Query("email")
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	orders, err := h.orderQueries.ListOrders(page, limit, nil, email, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get orders"})
		return
	}

	c.JSON(http.StatusOK, orders)
}

// UpdateOrderStatus updates order status (admin only)
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var req models.OrderStatusUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate status
	validStatuses := []string{
		models.OrderStatusPending,
		models.OrderStatusProcessing,
		models.OrderStatusShipped,
		models.OrderStatusDelivered,
		models.OrderStatusCancelled,
	}
	
	isValid := false
	for _, status := range validStatuses {
		if req.Status == status {
			isValid = true
			break
		}
	}
	
	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	err = h.orderQueries.UpdateOrderStatus(id, req.Status)
	if err != nil {
		if err.Error() == "order not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order status updated successfully"})
}

// GetUserOrders retrieves orders for the authenticated user
func (h *OrderHandler) GetUserOrders(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, ok := userID.(int)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	orders, err := h.orderQueries.GetOrdersByUserIDWithItems(id, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get orders"})
		return
	}

	c.JSON(http.StatusOK, orders)
}

// GetOrderByHash retrieves an order by public hash for guest access
func (h *OrderHandler) GetOrderByHash(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hash is required"})
		return
	}

	order, err := h.orderQueries.GetOrderByHash(hash)
	if err != nil {
		if err.Error() == "order not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get order"})
		return
	}

	c.JSON(http.StatusOK, order)
}