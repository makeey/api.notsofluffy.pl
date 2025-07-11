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

type Material struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MaterialRequest struct {
	Name string `json:"name" binding:"required,min=1,max=256"`
}

type MaterialResponse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type MaterialListResponse struct {
	Materials []MaterialResponse `json:"materials"`
	Total     int                `json:"total"`
	Page      int                `json:"page"`
	Limit     int                `json:"limit"`
}

type Color struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	ImageID    *int      `json:"image_id"`
	Custom     bool      `json:"custom"`
	MaterialID int       `json:"material_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ColorWithRelations struct {
	ID         int              `json:"id"`
	Name       string           `json:"name"`
	ImageID    *int             `json:"image_id"`
	Custom     bool             `json:"custom"`
	MaterialID int              `json:"material_id"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
	Image      *ImageResponse   `json:"image,omitempty"`
	Material   *MaterialResponse `json:"material,omitempty"`
}

type ColorRequest struct {
	Name       string `json:"name" binding:"required,min=1,max=256"`
	ImageID    *int   `json:"image_id"`
	Custom     bool   `json:"custom"`
	MaterialID int    `json:"material_id" binding:"required"`
}

type ColorResponse struct {
	ID         int              `json:"id"`
	Name       string           `json:"name"`
	ImageID    *int             `json:"image_id"`
	Custom     bool             `json:"custom"`
	MaterialID int              `json:"material_id"`
	CreatedAt  string           `json:"created_at"`
	UpdatedAt  string           `json:"updated_at"`
	Image      *ImageResponse   `json:"image,omitempty"`
	Material   *MaterialResponse `json:"material,omitempty"`
}

type ColorListResponse struct {
	Colors []ColorResponse `json:"colors"`
	Total  int             `json:"total"`
	Page   int             `json:"page"`
	Limit  int             `json:"limit"`
}

type AdditionalService struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AdditionalServiceWithImages struct {
	ID          int              `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Price       float64          `json:"price"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Images      []ImageResponse  `json:"images"`
}

type AdditionalServiceRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=256"`
	Description string  `json:"description" binding:"required,min=1,max=256"`
	Price       float64 `json:"price" binding:"required,min=0"`
	ImageIDs    []int   `json:"image_ids"`
}

type AdditionalServiceResponse struct {
	ID          int              `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Price       float64          `json:"price"`
	CreatedAt   string           `json:"created_at"`
	UpdatedAt   string           `json:"updated_at"`
	Images      []ImageResponse  `json:"images"`
}

type AdditionalServiceListResponse struct {
	AdditionalServices []AdditionalServiceResponse `json:"additional_services"`
	Total             int                          `json:"total"`
	Page              int                          `json:"page"`
	Limit             int                          `json:"limit"`
}

type Product struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	ShortDescription string    `json:"short_description"`
	Description      string    `json:"description"`
	MaterialID       *int      `json:"material_id"`
	MainImageID      int       `json:"main_image_id"`
	CategoryID       *int      `json:"category_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ProductWithRelations struct {
	ID                 int                           `json:"id"`
	Name               string                        `json:"name"`
	ShortDescription   string                        `json:"short_description"`
	Description        string                        `json:"description"`
	MaterialID         *int                          `json:"material_id"`
	MainImageID        int                           `json:"main_image_id"`
	CategoryID         *int                          `json:"category_id"`
	CreatedAt          time.Time                     `json:"created_at"`
	UpdatedAt          time.Time                     `json:"updated_at"`
	Material           *MaterialResponse             `json:"material,omitempty"`
	MainImage          ImageResponse                 `json:"main_image"`
	Category           *CategoryResponse             `json:"category,omitempty"`
	Images             []ImageResponse               `json:"images"`
	AdditionalServices []AdditionalServiceResponse   `json:"additional_services"`
}

type ProductRequest struct {
	Name                   string `json:"name" binding:"required,min=1,max=256"`
	ShortDescription       string `json:"short_description" binding:"required,min=1,max=512"`
	Description            string `json:"description" binding:"required,min=1"`
	MaterialID             *int   `json:"material_id"`
	MainImageID            int    `json:"main_image_id" binding:"required"`
	CategoryID             *int   `json:"category_id"`
	ImageIDs               []int  `json:"image_ids" binding:"required,min=1"`
	AdditionalServiceIDs   []int  `json:"additional_service_ids"`
}

type ProductResponse struct {
	ID                 int                           `json:"id"`
	Name               string                        `json:"name"`
	ShortDescription   string                        `json:"short_description"`
	Description        string                        `json:"description"`
	MaterialID         *int                          `json:"material_id"`
	MainImageID        int                           `json:"main_image_id"`
	CategoryID         *int                          `json:"category_id"`
	CreatedAt          string                        `json:"created_at"`
	UpdatedAt          string                        `json:"updated_at"`
	Material           *MaterialResponse             `json:"material,omitempty"`
	MainImage          ImageResponse                 `json:"main_image"`
	Category           *CategoryResponse             `json:"category,omitempty"`
	Images             []ImageResponse               `json:"images"`
	AdditionalServices []AdditionalServiceResponse   `json:"additional_services"`
}

type ProductListResponse struct {
	Products []ProductResponse `json:"products"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	Limit    int               `json:"limit"`
}

type Size struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	ProductID int       `json:"product_id"`
	BasePrice float64   `json:"base_price"`
	A         float64   `json:"a"`
	B         float64   `json:"b"`
	C         float64   `json:"c"`
	D         float64   `json:"d"`
	E         float64   `json:"e"`
	F         float64   `json:"f"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SizeWithProduct struct {
	ID        int             `json:"id"`
	Name      string          `json:"name"`
	ProductID int             `json:"product_id"`
	BasePrice float64         `json:"base_price"`
	A         float64         `json:"a"`
	B         float64         `json:"b"`
	C         float64         `json:"c"`
	D         float64         `json:"d"`
	E         float64         `json:"e"`
	F         float64         `json:"f"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Product   ProductResponse `json:"product"`
}

type SizeRequest struct {
	Name      string  `json:"name" binding:"required,min=1,max=256"`
	ProductID int     `json:"product_id" binding:"required"`
	BasePrice float64 `json:"base_price" binding:"required,min=0"`
	A         float64 `json:"a" binding:"required,min=0"`
	B         float64 `json:"b" binding:"required,min=0"`
	C         float64 `json:"c" binding:"required,min=0"`
	D         float64 `json:"d" binding:"required,min=0"`
	E         float64 `json:"e" binding:"required,min=0"`
	F         float64 `json:"f" binding:"required,min=0"`
}

type SizeResponse struct {
	ID        int             `json:"id"`
	Name      string          `json:"name"`
	ProductID int             `json:"product_id"`
	BasePrice float64         `json:"base_price"`
	A         float64         `json:"a"`
	B         float64         `json:"b"`
	C         float64         `json:"c"`
	D         float64         `json:"d"`
	E         float64         `json:"e"`
	F         float64         `json:"f"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
	Product   ProductResponse `json:"product"`
}

type SizeListResponse struct {
	Sizes []SizeResponse `json:"sizes"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}