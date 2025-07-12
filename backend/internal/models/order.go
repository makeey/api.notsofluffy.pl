package models

import (
	"time"
)

// Order status constants
const (
	OrderStatusPending    = "pending"
	OrderStatusProcessing = "processing"
	OrderStatusShipped    = "shipped"
	OrderStatusDelivered  = "delivered"
	OrderStatusCancelled  = "cancelled"
)

// Payment status constants
const (
	PaymentStatusPending   = "pending"
	PaymentStatusCompleted = "completed"
	PaymentStatusFailed    = "failed"
	PaymentStatusRefunded  = "refunded"
)

// Order represents an order in the database
type Order struct {
	ID            int       `json:"id"`
	UserID        *int      `json:"user_id,omitempty"`
	SessionID     *string   `json:"session_id,omitempty"`
	PublicHash    *string   `json:"public_hash,omitempty"`
	Email         string    `json:"email"`
	Phone         string    `json:"phone"`
	Status        string    `json:"status"`
	TotalAmount   float64   `json:"total_amount"`
	Subtotal      float64   `json:"subtotal"`
	ShippingCost  float64   `json:"shipping_cost"`
	TaxAmount     float64   `json:"tax_amount"`
	PaymentMethod *string   `json:"payment_method,omitempty"`
	PaymentStatus string    `json:"payment_status"`
	Notes         *string   `json:"notes,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ShippingAddress represents a shipping address
type ShippingAddress struct {
	ID           int       `json:"id"`
	OrderID      int       `json:"order_id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Company      *string   `json:"company,omitempty"`
	AddressLine1 string    `json:"address_line1"`
	AddressLine2 *string   `json:"address_line2,omitempty"`
	City         string    `json:"city"`
	StateProvince string   `json:"state_province"`
	PostalCode   string    `json:"postal_code"`
	Country      string    `json:"country"`
	Phone        string    `json:"phone"`
	CreatedAt    time.Time `json:"created_at"`
}

// BillingAddress represents a billing address
type BillingAddress struct {
	ID              int       `json:"id"`
	OrderID         int       `json:"order_id"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	Company         *string   `json:"company,omitempty"`
	AddressLine1    string    `json:"address_line1"`
	AddressLine2    *string   `json:"address_line2,omitempty"`
	City            string    `json:"city"`
	StateProvince   string    `json:"state_province"`
	PostalCode      string    `json:"postal_code"`
	Country         string    `json:"country"`
	Phone           string    `json:"phone"`
	SameAsShipping  bool      `json:"same_as_shipping"`
	CreatedAt       time.Time `json:"created_at"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	ID                   int                     `json:"id"`
	OrderID              int                     `json:"order_id"`
	ProductID            int                     `json:"product_id"`
	ProductName          string                  `json:"product_name"`
	ProductDescription   *string                 `json:"product_description,omitempty"`
	VariantID            int                     `json:"variant_id"`
	VariantName          string                  `json:"variant_name"`
	VariantColorName     *string                 `json:"variant_color_name,omitempty"`
	VariantColorCustom   bool                    `json:"variant_color_custom"`
	SizeID               int                     `json:"size_id"`
	SizeName             string                  `json:"size_name"`
	SizeDimensions       map[string]interface{}  `json:"size_dimensions,omitempty"`
	Quantity             int                     `json:"quantity"`
	UnitPrice            float64                 `json:"unit_price"`
	TotalPrice           float64                 `json:"total_price"`
	MainImage            *ImageResponse          `json:"main_image,omitempty"`
	Services             []OrderItemService      `json:"services,omitempty"`
	CreatedAt            time.Time               `json:"created_at"`
}

// OrderItemService represents a service for an order item
type OrderItemService struct {
	ID                 int       `json:"id"`
	OrderItemID        int       `json:"order_item_id"`
	ServiceID          int       `json:"service_id"`
	ServiceName        string    `json:"service_name"`
	ServiceDescription *string   `json:"service_description,omitempty"`
	ServicePrice       float64   `json:"service_price"`
	CreatedAt          time.Time `json:"created_at"`
}

// Request/Response types

// AddressRequest represents address input from frontend
type AddressRequest struct {
	FirstName     string  `json:"first_name" binding:"required"`
	LastName      string  `json:"last_name" binding:"required"`
	Company       *string `json:"company,omitempty"`
	AddressLine1  string  `json:"address_line1" binding:"required"`
	AddressLine2  *string `json:"address_line2,omitempty"`
	City          string  `json:"city" binding:"required"`
	StateProvince string  `json:"state_province" binding:"required"`
	PostalCode    string  `json:"postal_code" binding:"required"`
	Country       string  `json:"country" binding:"required"`
	Phone         string  `json:"phone" binding:"required"`
}

// OrderRequest represents order creation request
type OrderRequest struct {
	Email           string         `json:"email" binding:"required,email"`
	Phone           string         `json:"phone" binding:"required"`
	ShippingAddress AddressRequest `json:"shipping_address" binding:"required"`
	BillingAddress  AddressRequest `json:"billing_address" binding:"required"`
	SameAsShipping  bool           `json:"same_as_shipping"`
	PaymentMethod   *string        `json:"payment_method,omitempty"`
	Notes           *string        `json:"notes,omitempty"`
}

// OrderResponse represents order response to frontend
type OrderResponse struct {
	ID              int                     `json:"id"`
	UserID          *int                    `json:"user_id,omitempty"`
	SessionID       *string                 `json:"session_id,omitempty"`
	PublicHash      *string                 `json:"public_hash,omitempty"`
	Email           string                  `json:"email"`
	Phone           string                  `json:"phone"`
	Status          string                  `json:"status"`
	TotalAmount     float64                 `json:"total_amount"`
	Subtotal        float64                 `json:"subtotal"`
	ShippingCost    float64                 `json:"shipping_cost"`
	TaxAmount       float64                 `json:"tax_amount"`
	PaymentMethod   *string                 `json:"payment_method,omitempty"`
	PaymentStatus   string                  `json:"payment_status"`
	Notes           *string                 `json:"notes,omitempty"`
	ShippingAddress *ShippingAddress        `json:"shipping_address,omitempty"`
	BillingAddress  *BillingAddress         `json:"billing_address,omitempty"`
	Items           []OrderItem             `json:"items,omitempty"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

// OrderListResponse represents paginated order list response
type OrderListResponse struct {
	Orders []OrderResponse `json:"orders"`
	Total  int             `json:"total"`
	Page   int             `json:"page"`
	Limit  int             `json:"limit"`
}

// OrderStatusUpdateRequest represents order status update request
type OrderStatusUpdateRequest struct {
	Status string `json:"status" binding:"required"`
}