package models

import (
	"time"
)

// CartSession represents a shopping cart session
type CartSession struct {
	ID        int       `json:"id"`
	SessionID string    `json:"session_id"`
	UserID    *int      `json:"user_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CartItem represents an item in the cart
type CartItem struct {
	ID            int       `json:"id"`
	CartSessionID int       `json:"cart_session_id"`
	ProductID     int       `json:"product_id"`
	VariantID     int       `json:"variant_id"`
	SizeID        int       `json:"size_id"`
	Quantity      int       `json:"quantity"`
	PricePerItem  float64   `json:"price_per_item"`
	ServicesHash  string    `json:"services_hash"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CartItemService represents the many-to-many relationship between cart items and additional services
type CartItemService struct {
	CartItemID          int `json:"cart_item_id"`
	AdditionalServiceID int `json:"additional_service_id"`
}

// CartItemRequest represents the request to add an item to cart
type CartItemRequest struct {
	ProductID            int   `json:"product_id" binding:"required"`
	VariantID            int   `json:"variant_id" binding:"required"`
	SizeID               int   `json:"size_id" binding:"required"`
	Quantity             int   `json:"quantity" binding:"required,min=1"`
	AdditionalServiceIDs []int `json:"additional_service_ids"`
}

// CartItemUpdateRequest represents the request to update cart item quantity
type CartItemUpdateRequest struct {
	Quantity int `json:"quantity" binding:"required,min=1"`
}

// CartItemResponse represents a cart item with full product details
type CartItemResponse struct {
	ID                 int                          `json:"id"`
	ProductID          int                          `json:"product_id"`
	Product            ProductResponse              `json:"product"`
	VariantID          int                          `json:"variant_id"`
	Variant            ProductVariantResponse       `json:"variant"`
	SizeID             int                          `json:"size_id"`
	Size               SizeResponse                 `json:"size"`
	Quantity           int                          `json:"quantity"`
	PricePerItem       float64                      `json:"price_per_item"`
	TotalPrice         float64                      `json:"total_price"`
	AdditionalServices []AdditionalServiceResponse  `json:"additional_services"`
	CreatedAt          string                       `json:"created_at"`
	UpdatedAt          string                       `json:"updated_at"`
}

// CartResponse represents the full cart with items
type CartResponse struct {
	Items      []CartItemResponse `json:"items"`
	TotalItems int                `json:"total_items"`
	TotalPrice float64            `json:"total_price"`
}

// CartCountResponse represents the cart item count
type CartCountResponse struct {
	Count int `json:"count"`
}