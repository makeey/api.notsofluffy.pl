package models

import (
	"mime/multipart"
	"time"
)

type User struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	User         User   `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

const (
	RoleClient = "client"
	RoleAdmin  = "admin"
)

type Image struct {
	ID           int       `json:"id"`
	Filename     string    `json:"filename"`
	OriginalName string    `json:"original_name"`
	Path         string    `json:"path"`
	SizeBytes    int64     `json:"size_bytes"`
	MimeType     string    `json:"mime_type"`
	UploadedBy   int       `json:"uploaded_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ImageUploadRequest struct {
	File multipart.File `json:"-"`
	FileHeader *multipart.FileHeader `json:"-"`
}

type ImageResponse struct {
	ID           int    `json:"id"`
	Filename     string `json:"filename"`
	OriginalName string `json:"original_name"`
	Path         string `json:"path"`
	SizeBytes    int64  `json:"size_bytes"`
	MimeType     string `json:"mime_type"`
	UploadedBy   int    `json:"uploaded_by"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type ImageListResponse struct {
	Images []ImageResponse `json:"images"`
	Total  int             `json:"total"`
	Page   int             `json:"page"`
	Limit  int             `json:"limit"`
}

type UserListResponse struct {
	Users []User `json:"users"`
	Total int    `json:"total"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}

type AdminUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password,omitempty" binding:"min=6"`
	Role     string `json:"role" binding:"required,oneof=client admin"`
}

type Category struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	ImageID   *int      `json:"image_id"`
	Active    bool      `json:"active"`
	ChartOnly bool      `json:"chart_only"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CategoryWithImage struct {
	ID        int            `json:"id"`
	Name      string         `json:"name"`
	Slug      string         `json:"slug"`
	ImageID   *int           `json:"image_id"`
	Active    bool           `json:"active"`
	ChartOnly bool           `json:"chart_only"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Image     *ImageResponse `json:"image,omitempty"`
}

type CategoryRequest struct {
	Name      string `json:"name" binding:"required,min=1,max=256"`
	Slug      string `json:"slug" binding:"required,min=1,max=256"`
	ImageID   *int   `json:"image_id"`
	Active    bool   `json:"active"`
	ChartOnly bool   `json:"chart_only"`
}

type CategoryResponse struct {
	ID        int            `json:"id"`
	Name      string         `json:"name"`
	Slug      string         `json:"slug"`
	ImageID   *int           `json:"image_id"`
	Active    bool           `json:"active"`
	ChartOnly bool           `json:"chart_only"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
	Image     *ImageResponse `json:"image,omitempty"`
}

type CategoryListResponse struct {
	Categories []CategoryResponse `json:"categories"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
}