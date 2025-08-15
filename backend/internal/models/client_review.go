package models

import (
	"time"
)

// ClientReview represents a client review with photo and optional Instagram handle
type ClientReview struct {
	ID              int       `json:"id"`
	ClientName      string    `json:"client_name"`
	InstagramHandle *string   `json:"instagram_handle,omitempty"`
	ImageID         int       `json:"image_id"`
	DisplayOrder    int       `json:"display_order"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	// Related data
	Image           *Image    `json:"image,omitempty"`
}

// CreateClientReviewRequest represents the request to create a new client review
type CreateClientReviewRequest struct {
	ClientName      string  `json:"client_name" binding:"required,min=1,max=255"`
	InstagramHandle *string `json:"instagram_handle,omitempty" binding:"omitempty,max=100"`
	ImageID         int     `json:"image_id" binding:"required,min=1"`
	DisplayOrder    int     `json:"display_order"`
	IsActive        bool    `json:"is_active"`
}

// UpdateClientReviewRequest represents the request to update an existing client review
type UpdateClientReviewRequest struct {
	ClientName      string  `json:"client_name" binding:"required,min=1,max=255"`
	InstagramHandle *string `json:"instagram_handle,omitempty" binding:"omitempty,max=100"`
	ImageID         int     `json:"image_id" binding:"required,min=1"`
	DisplayOrder    int     `json:"display_order"`
	IsActive        bool    `json:"is_active"`
}

// ClientReviewListResponse represents the response for listing client reviews
type ClientReviewListResponse struct {
	ClientReviews []ClientReview `json:"client_reviews"`
	Total         int            `json:"total"`
	Page          int            `json:"page"`
	Limit         int            `json:"limit"`
}

// ReorderClientReviewsRequest represents the request to reorder client reviews
type ReorderClientReviewsRequest struct {
	ReviewOrders []struct {
		ID           int `json:"id" binding:"required"`
		DisplayOrder int `json:"display_order" binding:"required"`
	} `json:"review_orders" binding:"required,min=1"`
}