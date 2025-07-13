package models

import (
	"time"
)

// Discount type constants
const (
	DiscountTypePercentage  = "percentage"
	DiscountTypeFixedAmount = "fixed_amount"
)

// Usage type constants
const (
	UsageTypeOneTime     = "one_time"
	UsageTypeOncePerUser = "once_per_user"
	UsageTypeUnlimited   = "unlimited"
)

// DiscountCode represents a discount code in the database
type DiscountCode struct {
	ID             int       `json:"id"`
	Code           string    `json:"code"`
	Description    string    `json:"description"`
	DiscountType   string    `json:"discount_type"`
	DiscountValue  float64   `json:"discount_value"`
	MinOrderAmount float64   `json:"min_order_amount"`
	UsageType      string    `json:"usage_type"`
	MaxUses        *int      `json:"max_uses,omitempty"`
	UsedCount      int       `json:"used_count"`
	Active         bool      `json:"active"`
	StartDate      time.Time `json:"start_date"`
	EndDate        *time.Time `json:"end_date,omitempty"`
	CreatedBy      *int      `json:"created_by,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// DiscountCodeRequest represents a request to create or update a discount code
type DiscountCodeRequest struct {
	Code           string     `json:"code" binding:"required,min=2,max=50"`
	Description    string     `json:"description" binding:"required,min=1,max=500"`
	DiscountType   string     `json:"discount_type" binding:"required,oneof=percentage fixed_amount"`
	DiscountValue  float64    `json:"discount_value" binding:"required,gt=0"`
	MinOrderAmount float64    `json:"min_order_amount" binding:"gte=0"`
	UsageType      string     `json:"usage_type" binding:"required,oneof=one_time once_per_user unlimited"`
	MaxUses        *int       `json:"max_uses,omitempty"`
	Active         bool       `json:"active"`
	StartDate      time.Time  `json:"start_date" binding:"required"`
	EndDate        *time.Time `json:"end_date,omitempty"`
}

// DiscountCodeResponse represents a discount code response with additional information
type DiscountCodeResponse struct {
	ID             int       `json:"id"`
	Code           string    `json:"code"`
	Description    string    `json:"description"`
	DiscountType   string    `json:"discount_type"`
	DiscountValue  float64   `json:"discount_value"`
	MinOrderAmount float64   `json:"min_order_amount"`
	UsageType      string    `json:"usage_type"`
	MaxUses        *int      `json:"max_uses,omitempty"`
	UsedCount      int       `json:"used_count"`
	Active         bool      `json:"active"`
	StartDate      time.Time `json:"start_date"`
	EndDate        *time.Time `json:"end_date,omitempty"`
	CreatedBy      *int      `json:"created_by,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	IsExpired      bool      `json:"is_expired"`
	IsUsageExceeded bool     `json:"is_usage_exceeded"`
}

// DiscountCodeListResponse represents a paginated list of discount codes
type DiscountCodeListResponse struct {
	DiscountCodes []DiscountCodeResponse `json:"discount_codes"`
	Total         int                    `json:"total"`
	Page          int                    `json:"page"`
	Limit         int                    `json:"limit"`
}

// ApplyDiscountRequest represents a request to apply a discount code to cart
type ApplyDiscountRequest struct {
	Code string `json:"code" binding:"required"`
}

// ApplyDiscountResponse represents the response when applying a discount code
type ApplyDiscountResponse struct {
	Code            string  `json:"code"`
	Description     string  `json:"description"`
	DiscountType    string  `json:"discount_type"`
	DiscountValue   float64 `json:"discount_value"`
	DiscountAmount  float64 `json:"discount_amount"`
	OriginalTotal   float64 `json:"original_total"`
	DiscountedTotal float64 `json:"discounted_total"`
	Message         string  `json:"message"`
}

// DiscountCodeUsage represents a record of discount code usage
type DiscountCodeUsage struct {
	ID             int       `json:"id"`
	DiscountCodeID int       `json:"discount_code_id"`
	UserID         *int      `json:"user_id,omitempty"`
	SessionID      string    `json:"session_id"`
	OrderID        *int      `json:"order_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// DiscountValidationResult represents the result of discount code validation
type DiscountValidationResult struct {
	IsValid        bool    `json:"is_valid"`
	ErrorMessage   string  `json:"error_message,omitempty"`
	DiscountAmount float64 `json:"discount_amount"`
	DiscountCode   *DiscountCode `json:"discount_code,omitempty"`
}

// CartDiscount represents discount information in cart context
type CartDiscount struct {
	CodeID          int     `json:"code_id"`
	Code            string  `json:"code"`
	Description     string  `json:"description"`
	DiscountType    string  `json:"discount_type"`
	DiscountValue   float64 `json:"discount_value"`
	DiscountAmount  float64 `json:"discount_amount"`
}

// OrderDiscount represents discount information stored with an order
type OrderDiscount struct {
	CodeID      *int    `json:"code_id,omitempty"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}