package handlers

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"notsofluffy-backend/internal/database"
	"notsofluffy-backend/internal/models"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	db                       *sql.DB
	userQueries              *database.UserQueries
	imageQueries             *database.ImageQueries
	categoryQueries          *database.CategoryQueries
	materialQueries          *database.MaterialQueries
	colorQueries             *database.ColorQueries
	additionalServiceQueries *database.AdditionalServiceQueries
	productQueries           *database.ProductQueries
	sizeQueries              *database.SizeQueries
	productVariantQueries    *database.ProductVariantQueries
	orderQueries             *database.OrderQueries
	settingsQueries          *database.SettingsQueries
	clientReviewQueries      *database.ClientReviewQueries
}

func NewAdminHandler(db *sql.DB) *AdminHandler {
	return &AdminHandler{
		db:                       db,
		userQueries:              database.NewUserQueries(db),
		imageQueries:             database.NewImageQueries(db),
		categoryQueries:          database.NewCategoryQueries(db),
		materialQueries:          database.NewMaterialQueries(db),
		colorQueries:             database.NewColorQueries(db),
		additionalServiceQueries: database.NewAdditionalServiceQueries(db),
		productQueries:           database.NewProductQueries(db),
		sizeQueries:              database.NewSizeQueries(db),
		productVariantQueries:    database.NewProductVariantQueries(db),
		orderQueries:             database.NewOrderQueries(db),
		settingsQueries:          database.NewSettingsQueries(db),
		clientReviewQueries:      database.NewClientReviewQueries(db),
	}
}

// User Management

func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	users, total, err := h.userQueries.ListUsers(page, limit, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}

	response := models.UserListResponse{
		Users: users,
		Total: total,
		Page:  page,
		Limit: limit,
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req models.AdminUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if email already exists
	exists, err := h.userQueries.EmailExists(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check email"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	}

	user, err := h.userQueries.CreateAdminUser(req.Email, req.Password, req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req models.AdminUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userQueries.UpdateUser(id, req.Email, req.Password, req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	err = h.userQueries.DeleteUser(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// Image Management

func (h *AdminHandler) UploadImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Validate file type
	if !isValidImageType(header.Header.Get("Content-Type")) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type. Only JPEG, PNG, and GIF are allowed"})
		return
	}

	// Validate file size (10MB limit)
	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File size too large. Maximum 10MB allowed"})
		return
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := generateUUID() + ext
	
	// Create upload directory if it doesn't exist
	uploadDir := "uploads/images"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	// Save file
	filePath := filepath.Join(uploadDir, filename)
	out, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create file"})
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Get user ID from context
	userID, _ := c.Get("user_id")
	userIDInt, _ := userID.(int)

	// Save image metadata to database
	image := &models.Image{
		Filename:     filename,
		OriginalName: header.Filename,
		Path:         filePath,
		SizeBytes:    header.Size,
		MimeType:     header.Header.Get("Content-Type"),
		UploadedBy:   userIDInt,
	}

	err = h.imageQueries.CreateImage(image)
	if err != nil {
		// Clean up file if database save fails
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image metadata"})
		return
	}

	response := models.ImageResponse{
		ID:           image.ID,
		Filename:     image.Filename,
		OriginalName: image.OriginalName,
		Path:         image.Path,
		SizeBytes:    image.SizeBytes,
		MimeType:     image.MimeType,
		UploadedBy:   image.UploadedBy,
		CreatedAt:    image.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    image.UpdatedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusCreated, response)
}

func (h *AdminHandler) ListImages(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	images, total, err := h.imageQueries.ListImages(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve images"})
		return
	}

	// Convert to response format
	imageResponses := make([]models.ImageResponse, len(images))
	for i, img := range images {
		imageResponses[i] = models.ImageResponse{
			ID:           img.ID,
			Filename:     img.Filename,
			OriginalName: img.OriginalName,
			Path:         img.Path,
			SizeBytes:    img.SizeBytes,
			MimeType:     img.MimeType,
			UploadedBy:   img.UploadedBy,
			CreatedAt:    img.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    img.UpdatedAt.Format(time.RFC3339),
		}
	}

	response := models.ImageListResponse{
		Images: imageResponses,
		Total:  total,
		Page:   page,
		Limit:  limit,
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) DeleteImage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}

	// Get image details before deletion
	image, err := h.imageQueries.GetImageByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	// Delete from database
	err = h.imageQueries.DeleteImage(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete image"})
		return
	}

	// Delete file from filesystem
	os.Remove(image.Path)

	c.JSON(http.StatusOK, gin.H{"message": "Image deleted successfully"})
}

// Category Management

func (h *AdminHandler) ListCategories(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Parse filter parameters
	var activeOnly *bool
	var chartOnly *bool

	if activeParam := c.Query("active"); activeParam != "" {
		active := activeParam == "true"
		activeOnly = &active
	}

	if chartParam := c.Query("chart_only"); chartParam != "" {
		chart := chartParam == "true"
		chartOnly = &chart
	}

	categories, total, err := h.categoryQueries.ListCategories(page, limit, search, activeOnly, chartOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve categories"})
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

	response := models.CategoryListResponse{
		Categories: categoryResponses,
		Total:      total,
		Page:       page,
		Limit:      limit,
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) CreateCategory(c *gin.Context) {
	var req models.CategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if slug already exists
	exists, err := h.categoryQueries.SlugExists(req.Slug, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check slug"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Slug already exists"})
		return
	}

	// Validate image ID if provided
	if req.ImageID != nil {
		_, err := h.imageQueries.GetImageByID(*req.ImageID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
			return
		}
	}

	category := &models.Category{
		Name:      req.Name,
		Slug:      req.Slug,
		ImageID:   req.ImageID,
		Active:    req.Active,
		ChartOnly: req.ChartOnly,
	}

	err = h.categoryQueries.CreateCategory(category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
		return
	}

	response := models.CategoryResponse{
		ID:        category.ID,
		Name:      category.Name,
		Slug:      category.Slug,
		ImageID:   category.ImageID,
		Active:    category.Active,
		ChartOnly: category.ChartOnly,
		CreatedAt: category.CreatedAt.Format(time.RFC3339),
		UpdatedAt: category.UpdatedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusCreated, response)
}

func (h *AdminHandler) GetCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	category, err := h.categoryQueries.GetCategoryByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	response := models.CategoryResponse{
		ID:        category.ID,
		Name:      category.Name,
		Slug:      category.Slug,
		ImageID:   category.ImageID,
		Active:    category.Active,
		ChartOnly: category.ChartOnly,
		CreatedAt: category.CreatedAt.Format(time.RFC3339),
		UpdatedAt: category.UpdatedAt.Format(time.RFC3339),
		Image:     category.Image,
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) UpdateCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	var req models.CategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if slug already exists (excluding current category)
	exists, err := h.categoryQueries.SlugExists(req.Slug, &id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check slug"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Slug already exists"})
		return
	}

	// Validate image ID if provided
	if req.ImageID != nil {
		_, err := h.imageQueries.GetImageByID(*req.ImageID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
			return
		}
	}

	category, err := h.categoryQueries.UpdateCategory(id, req.Name, req.Slug, req.ImageID, req.Active, req.ChartOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update category"})
		return
	}

	response := models.CategoryResponse{
		ID:        category.ID,
		Name:      category.Name,
		Slug:      category.Slug,
		ImageID:   category.ImageID,
		Active:    category.Active,
		ChartOnly: category.ChartOnly,
		CreatedAt: category.CreatedAt.Format(time.RFC3339),
		UpdatedAt: category.UpdatedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) DeleteCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	err = h.categoryQueries.DeleteCategory(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category deleted successfully"})
}

func (h *AdminHandler) ToggleCategoryActive(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	err = h.categoryQueries.ToggleActive(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle category status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category status toggled successfully"})
}

// Material Management

func (h *AdminHandler) ListMaterials(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	materials, total, err := h.materialQueries.ListMaterials(page, limit, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve materials"})
		return
	}

	// Convert to response format
	materialResponses := make([]models.MaterialResponse, len(materials))
	for i, mat := range materials {
		materialResponses[i] = models.MaterialResponse{
			ID:        mat.ID,
			Name:      mat.Name,
			CreatedAt: mat.CreatedAt.Format(time.RFC3339),
			UpdatedAt: mat.UpdatedAt.Format(time.RFC3339),
		}
	}

	response := models.MaterialListResponse{
		Materials: materialResponses,
		Total:     total,
		Page:      page,
		Limit:     limit,
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) CreateMaterial(c *gin.Context) {
	var req models.MaterialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if name already exists
	exists, err := h.materialQueries.NameExists(req.Name, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check material name"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Material name already exists"})
		return
	}

	material := &models.Material{
		Name: req.Name,
	}

	err = h.materialQueries.CreateMaterial(material)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create material"})
		return
	}

	response := models.MaterialResponse{
		ID:        material.ID,
		Name:      material.Name,
		CreatedAt: material.CreatedAt.Format(time.RFC3339),
		UpdatedAt: material.UpdatedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusCreated, response)
}

func (h *AdminHandler) GetMaterial(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid material ID"})
		return
	}

	material, err := h.materialQueries.GetMaterialByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Material not found"})
		return
	}

	response := models.MaterialResponse{
		ID:        material.ID,
		Name:      material.Name,
		CreatedAt: material.CreatedAt.Format(time.RFC3339),
		UpdatedAt: material.UpdatedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) UpdateMaterial(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid material ID"})
		return
	}

	var req models.MaterialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if name already exists (excluding current material)
	exists, err := h.materialQueries.NameExists(req.Name, &id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check material name"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Material name already exists"})
		return
	}

	material, err := h.materialQueries.UpdateMaterial(id, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update material"})
		return
	}

	response := models.MaterialResponse{
		ID:        material.ID,
		Name:      material.Name,
		CreatedAt: material.CreatedAt.Format(time.RFC3339),
		UpdatedAt: material.UpdatedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) DeleteMaterial(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid material ID"})
		return
	}

	err = h.materialQueries.DeleteMaterial(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete material"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Material deleted successfully"})
}

// Color Management

func (h *AdminHandler) ListColors(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Parse filter parameters
	var materialID *int
	var customOnly *bool

	if materialParam := c.Query("material_id"); materialParam != "" {
		if matID, err := strconv.Atoi(materialParam); err == nil {
			materialID = &matID
		}
	}

	if customParam := c.Query("custom"); customParam != "" {
		custom := customParam == "true"
		customOnly = &custom
	}

	colors, total, err := h.colorQueries.ListColors(page, limit, search, materialID, customOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve colors"})
		return
	}

	// Convert to response format
	colorResponses := make([]models.ColorResponse, len(colors))
	for i, color := range colors {
		colorResponses[i] = models.ColorResponse{
			ID:         color.ID,
			Name:       color.Name,
			ImageID:    color.ImageID,
			Custom:     color.Custom,
			MaterialID: color.MaterialID,
			CreatedAt:  color.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  color.UpdatedAt.Format(time.RFC3339),
			Image:      color.Image,
			Material:   color.Material,
		}
	}

	response := models.ColorListResponse{
		Colors: colorResponses,
		Total:  total,
		Page:   page,
		Limit:  limit,
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) CreateColor(c *gin.Context) {
	var req models.ColorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate material exists
	_, err := h.materialQueries.GetMaterialByID(req.MaterialID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid material ID"})
		return
	}

	// Validate image ID if provided
	if req.ImageID != nil {
		_, err := h.imageQueries.GetImageByID(*req.ImageID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
			return
		}
	}

	// Check if name already exists for this material
	exists, err := h.colorQueries.NameExistsForMaterial(req.Name, req.MaterialID, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check color name"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Color name already exists for this material"})
		return
	}

	color := &models.Color{
		Name:       req.Name,
		ImageID:    req.ImageID,
		Custom:     req.Custom,
		MaterialID: req.MaterialID,
	}

	err = h.colorQueries.CreateColor(color)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create color"})
		return
	}

	response := models.ColorResponse{
		ID:         color.ID,
		Name:       color.Name,
		ImageID:    color.ImageID,
		Custom:     color.Custom,
		MaterialID: color.MaterialID,
		CreatedAt:  color.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  color.UpdatedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusCreated, response)
}

func (h *AdminHandler) GetColor(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid color ID"})
		return
	}

	color, err := h.colorQueries.GetColorByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Color not found"})
		return
	}

	response := models.ColorResponse{
		ID:         color.ID,
		Name:       color.Name,
		ImageID:    color.ImageID,
		Custom:     color.Custom,
		MaterialID: color.MaterialID,
		CreatedAt:  color.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  color.UpdatedAt.Format(time.RFC3339),
		Image:      color.Image,
		Material:   color.Material,
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) UpdateColor(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid color ID"})
		return
	}

	var req models.ColorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate material exists
	_, err = h.materialQueries.GetMaterialByID(req.MaterialID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid material ID"})
		return
	}

	// Validate image ID if provided
	if req.ImageID != nil {
		_, err := h.imageQueries.GetImageByID(*req.ImageID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
			return
		}
	}

	// Check if name already exists for this material (excluding current color)
	exists, err := h.colorQueries.NameExistsForMaterial(req.Name, req.MaterialID, &id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check color name"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Color name already exists for this material"})
		return
	}

	color, err := h.colorQueries.UpdateColor(id, req.Name, req.ImageID, req.Custom, req.MaterialID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update color"})
		return
	}

	response := models.ColorResponse{
		ID:         color.ID,
		Name:       color.Name,
		ImageID:    color.ImageID,
		Custom:     color.Custom,
		MaterialID: color.MaterialID,
		CreatedAt:  color.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  color.UpdatedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) DeleteColor(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid color ID"})
		return
	}

	err = h.colorQueries.DeleteColor(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete color"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Color deleted successfully"})
}

// Additional Service Management

func (h *AdminHandler) ListAdditionalServices(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Parse price filter parameters
	var minPrice, maxPrice *float64

	if minPriceParam := c.Query("min_price"); minPriceParam != "" {
		if price, err := strconv.ParseFloat(minPriceParam, 64); err == nil && price >= 0 {
			minPrice = &price
		}
	}

	if maxPriceParam := c.Query("max_price"); maxPriceParam != "" {
		if price, err := strconv.ParseFloat(maxPriceParam, 64); err == nil && price >= 0 {
			maxPrice = &price
		}
	}

	services, total, err := h.additionalServiceQueries.ListAdditionalServices(page, limit, search, minPrice, maxPrice)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve additional services"})
		return
	}

	// Convert to response format
	serviceResponses := make([]models.AdditionalServiceResponse, len(services))
	for i, service := range services {
		serviceResponses[i] = models.AdditionalServiceResponse{
			ID:          service.ID,
			Name:        service.Name,
			Description: service.Description,
			Price:       service.Price,
			CreatedAt:   service.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   service.UpdatedAt.Format(time.RFC3339),
			Images:      service.Images,
		}
	}

	response := models.AdditionalServiceListResponse{
		AdditionalServices: serviceResponses,
		Total:              total,
		Page:               page,
		Limit:              limit,
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) CreateAdditionalService(c *gin.Context) {
	var req models.AdditionalServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if name already exists
	exists, err := h.additionalServiceQueries.NameExists(req.Name, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check service name"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Service name already exists"})
		return
	}

	// Validate image IDs if provided
	for _, imageID := range req.ImageIDs {
		_, err := h.imageQueries.GetImageByID(imageID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid image ID: %d", imageID)})
			return
		}
	}

	service := &models.AdditionalService{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
	}

	err = h.additionalServiceQueries.CreateAdditionalService(service)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create additional service"})
		return
	}

	// Associate images if provided
	if len(req.ImageIDs) > 0 {
		err = h.additionalServiceQueries.ReplaceImages(service.ID, req.ImageIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to associate images with service"})
			return
		}
	}

	response := models.AdditionalServiceResponse{
		ID:          service.ID,
		Name:        service.Name,
		Description: service.Description,
		Price:       service.Price,
		CreatedAt:   service.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   service.UpdatedAt.Format(time.RFC3339),
		Images:      []models.ImageResponse{}, // Will be empty for new service without images
	}

	c.JSON(http.StatusCreated, response)
}

func (h *AdminHandler) GetAdditionalService(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service ID"})
		return
	}

	service, err := h.additionalServiceQueries.GetAdditionalServiceByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Additional service not found"})
		return
	}

	response := models.AdditionalServiceResponse{
		ID:          service.ID,
		Name:        service.Name,
		Description: service.Description,
		Price:       service.Price,
		CreatedAt:   service.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   service.UpdatedAt.Format(time.RFC3339),
		Images:      service.Images,
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) UpdateAdditionalService(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service ID"})
		return
	}

	var req models.AdditionalServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if name already exists (excluding current service)
	exists, err := h.additionalServiceQueries.NameExists(req.Name, &id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check service name"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Service name already exists"})
		return
	}

	// Validate image IDs if provided
	for _, imageID := range req.ImageIDs {
		_, err := h.imageQueries.GetImageByID(imageID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid image ID: %d", imageID)})
			return
		}
	}

	service, err := h.additionalServiceQueries.UpdateAdditionalService(id, req.Name, req.Description, req.Price)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update additional service"})
		return
	}

	// Update image associations
	err = h.additionalServiceQueries.ReplaceImages(service.ID, req.ImageIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update service images"})
		return
	}

	// Get updated service with images
	updatedService, err := h.additionalServiceQueries.GetAdditionalServiceByID(service.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get updated service"})
		return
	}

	response := models.AdditionalServiceResponse{
		ID:          updatedService.ID,
		Name:        updatedService.Name,
		Description: updatedService.Description,
		Price:       updatedService.Price,
		CreatedAt:   updatedService.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   updatedService.UpdatedAt.Format(time.RFC3339),
		Images:      updatedService.Images,
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) DeleteAdditionalService(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service ID"})
		return
	}

	err = h.additionalServiceQueries.DeleteAdditionalService(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete additional service"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Additional service deleted successfully"})
}

// Helper functions

func generateUUID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}

func isValidImageType(mimeType string) bool {
	allowedTypes := []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
	}
	
	for _, t := range allowedTypes {
		if strings.EqualFold(mimeType, t) {
			return true
		}
	}
	
	return false
}

// Product Management

func (h *AdminHandler) ListProducts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")
	
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	
	var categoryID, materialID *int
	if catID := c.Query("category_id"); catID != "" && catID != "all" {
		if id, err := strconv.Atoi(catID); err == nil {
			categoryID = &id
		}
	}
	if matID := c.Query("material_id"); matID != "" && matID != "all" {
		if id, err := strconv.Atoi(matID); err == nil {
			materialID = &id
		}
	}
	
	products, total, err := h.productQueries.ListProducts(page, limit, search, categoryID, materialID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve products"})
		return
	}
	
	// Convert to response format
	var responseProducts []models.ProductResponse
	for _, product := range products {
		responseProduct := models.ProductResponse{
			ID:                 product.ID,
			Name:               product.Name,
			ShortDescription:   product.ShortDescription,
			Description:        product.Description,
			MaterialID:         product.MaterialID,
			MainImageID:        product.MainImageID,
			CategoryID:         product.CategoryID,
			CreatedAt:          product.CreatedAt.Format(time.RFC3339),
			UpdatedAt:          product.UpdatedAt.Format(time.RFC3339),
			Material:           product.Material,
			MainImage:          product.MainImage,
			Category:           product.Category,
			Images:             product.Images,
			AdditionalServices: product.AdditionalServices,
		}
		responseProducts = append(responseProducts, responseProduct)
	}
	
	response := models.ProductListResponse{
		Products: responseProducts,
		Total:    total,
		Page:     page,
		Limit:    limit,
	}
	
	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) CreateProduct(c *gin.Context) {
	var req models.ProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Validate main image exists
	if !h.validateImageExists(req.MainImageID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Main image not found"})
		return
	}
	
	// Validate all image IDs exist
	for _, imageID := range req.ImageIDs {
		if !h.validateImageExists(imageID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Image with ID %d not found", imageID)})
			return
		}
	}
	
	// Validate main image is included in images array
	mainImageIncluded := false
	for _, imageID := range req.ImageIDs {
		if imageID == req.MainImageID {
			mainImageIncluded = true
			break
		}
	}
	if !mainImageIncluded {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Main image must be included in the images list"})
		return
	}
	
	// Validate additional service IDs exist
	for _, serviceID := range req.AdditionalServiceIDs {
		if !h.validateAdditionalServiceExists(serviceID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Additional service with ID %d not found", serviceID)})
			return
		}
	}
	
	// Validate material exists if provided
	if req.MaterialID != nil && !h.validateMaterialExists(*req.MaterialID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Material not found"})
		return
	}
	
	// Validate category exists if provided
	if req.CategoryID != nil && !h.validateCategoryExists(*req.CategoryID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Category not found"})
		return
	}
	
	product := &models.Product{
		Name:             req.Name,
		ShortDescription: req.ShortDescription,
		Description:      req.Description,
		MaterialID:       req.MaterialID,
		MainImageID:      req.MainImageID,
		CategoryID:       req.CategoryID,
	}
	
	// Create product
	err := h.productQueries.CreateProduct(product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
		return
	}
	
	// Set product images
	err = h.productQueries.ReplaceImages(product.ID, req.ImageIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set product images"})
		return
	}
	
	// Set product services
	err = h.productQueries.ReplaceServices(product.ID, req.AdditionalServiceIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set product services"})
		return
	}
	
	// Return the created product with relations
	createdProduct, err := h.productQueries.GetProduct(product.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve created product"})
		return
	}
	
	response := models.ProductResponse{
		ID:                 createdProduct.ID,
		Name:               createdProduct.Name,
		ShortDescription:   createdProduct.ShortDescription,
		Description:        createdProduct.Description,
		MaterialID:         createdProduct.MaterialID,
		MainImageID:        createdProduct.MainImageID,
		CategoryID:         createdProduct.CategoryID,
		CreatedAt:          createdProduct.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          createdProduct.UpdatedAt.Format(time.RFC3339),
		Material:           createdProduct.Material,
		MainImage:          createdProduct.MainImage,
		Category:           createdProduct.Category,
		Images:             createdProduct.Images,
		AdditionalServices: createdProduct.AdditionalServices,
	}
	
	c.JSON(http.StatusCreated, response)
}

func (h *AdminHandler) GetProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}
	
	product, err := h.productQueries.GetProduct(id)
	if err != nil {
		if err.Error() == "product not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve product"})
		return
	}
	
	response := models.ProductResponse{
		ID:                 product.ID,
		Name:               product.Name,
		ShortDescription:   product.ShortDescription,
		Description:        product.Description,
		MaterialID:         product.MaterialID,
		MainImageID:        product.MainImageID,
		CategoryID:         product.CategoryID,
		CreatedAt:          product.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          product.UpdatedAt.Format(time.RFC3339),
		Material:           product.Material,
		MainImage:          product.MainImage,
		Category:           product.Category,
		Images:             product.Images,
		AdditionalServices: product.AdditionalServices,
	}
	
	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) UpdateProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}
	
	var req models.ProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Validate main image exists
	if !h.validateImageExists(req.MainImageID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Main image not found"})
		return
	}
	
	// Validate all image IDs exist
	for _, imageID := range req.ImageIDs {
		if !h.validateImageExists(imageID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Image with ID %d not found", imageID)})
			return
		}
	}
	
	// Validate main image is included in images array
	mainImageIncluded := false
	for _, imageID := range req.ImageIDs {
		if imageID == req.MainImageID {
			mainImageIncluded = true
			break
		}
	}
	if !mainImageIncluded {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Main image must be included in the images list"})
		return
	}
	
	// Validate additional service IDs exist
	for _, serviceID := range req.AdditionalServiceIDs {
		if !h.validateAdditionalServiceExists(serviceID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Additional service with ID %d not found", serviceID)})
			return
		}
	}
	
	// Validate material exists if provided
	if req.MaterialID != nil && !h.validateMaterialExists(*req.MaterialID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Material not found"})
		return
	}
	
	// Validate category exists if provided
	if req.CategoryID != nil && !h.validateCategoryExists(*req.CategoryID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Category not found"})
		return
	}
	
	product := &models.Product{
		Name:             req.Name,
		ShortDescription: req.ShortDescription,
		Description:      req.Description,
		MaterialID:       req.MaterialID,
		MainImageID:      req.MainImageID,
		CategoryID:       req.CategoryID,
	}
	
	// Update product
	err = h.productQueries.UpdateProduct(id, product)
	if err != nil {
		if err.Error() == "product not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product"})
		return
	}
	
	// Update product images
	err = h.productQueries.ReplaceImages(id, req.ImageIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product images"})
		return
	}
	
	// Update product services
	err = h.productQueries.ReplaceServices(id, req.AdditionalServiceIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product services"})
		return
	}
	
	// Return the updated product with relations
	updatedProduct, err := h.productQueries.GetProduct(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated product"})
		return
	}
	
	response := models.ProductResponse{
		ID:                 updatedProduct.ID,
		Name:               updatedProduct.Name,
		ShortDescription:   updatedProduct.ShortDescription,
		Description:        updatedProduct.Description,
		MaterialID:         updatedProduct.MaterialID,
		MainImageID:        updatedProduct.MainImageID,
		CategoryID:         updatedProduct.CategoryID,
		CreatedAt:          updatedProduct.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          updatedProduct.UpdatedAt.Format(time.RFC3339),
		Material:           updatedProduct.Material,
		MainImage:          updatedProduct.MainImage,
		Category:           updatedProduct.Category,
		Images:             updatedProduct.Images,
		AdditionalServices: updatedProduct.AdditionalServices,
	}
	
	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) DeleteProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}
	
	err = h.productQueries.DeleteProduct(id)
	if err != nil {
		if err.Error() == "product not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}

// Validation helper methods for products

func (h *AdminHandler) validateImageExists(imageID int) bool {
	query := "SELECT 1 FROM images WHERE id = $1"
	var exists int
	err := h.db.QueryRow(query, imageID).Scan(&exists)
	return err == nil
}

func (h *AdminHandler) validateMaterialExists(materialID int) bool {
	query := "SELECT 1 FROM materials WHERE id = $1"
	var exists int
	err := h.db.QueryRow(query, materialID).Scan(&exists)
	return err == nil
}

func (h *AdminHandler) validateCategoryExists(categoryID int) bool {
	query := "SELECT 1 FROM categories WHERE id = $1"
	var exists int
	err := h.db.QueryRow(query, categoryID).Scan(&exists)
	return err == nil
}

func (h *AdminHandler) validateAdditionalServiceExists(serviceID int) bool {
	query := "SELECT 1 FROM additional_services WHERE id = $1"
	var exists int
	err := h.db.QueryRow(query, serviceID).Scan(&exists)
	return err == nil
}

// Size Management

func (h *AdminHandler) ListSizes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")
	
	var productID *int
	if productIDStr := c.Query("product_id"); productIDStr != "" {
		if pid, err := strconv.Atoi(productIDStr); err == nil {
			productID = &pid
		}
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	sizes, total, err := h.sizeQueries.ListSizes(page, limit, search, productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.SizeListResponse{
		Sizes: sizes,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

func (h *AdminHandler) CreateSize(c *gin.Context) {
	var req models.SizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate product exists
	if !h.validateProductExists(req.ProductID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product not found"})
		return
	}

	size := &models.Size{
		Name:          req.Name,
		ProductID:     req.ProductID,
		BasePrice:     req.BasePrice,
		A:             req.A,
		B:             req.B,
		C:             req.C,
		D:             req.D,
		E:             req.E,
		F:             req.F,
		UseStock:      req.UseStock,
		StockQuantity: req.StockQuantity,
	}

	if err := h.sizeQueries.CreateSize(size); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Size created successfully", "id": size.ID})
}

func (h *AdminHandler) GetSize(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid size ID"})
		return
	}

	size, err := h.sizeQueries.GetSizeByID(id)
	if err != nil {
		if err.Error() == "size not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Size not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := models.SizeResponse{
		ID:        size.ID,
		Name:      size.Name,
		ProductID: size.ProductID,
		BasePrice: size.BasePrice,
		A:         size.A,
		B:         size.B,
		C:         size.C,
		D:         size.D,
		E:         size.E,
		F:         size.F,
		CreatedAt: size.CreatedAt.Format(time.RFC3339),
		UpdatedAt: size.UpdatedAt.Format(time.RFC3339),
		Product:   size.Product,
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) UpdateSize(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid size ID"})
		return
	}

	var req models.SizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate product exists
	if !h.validateProductExists(req.ProductID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product not found"})
		return
	}

	size := &models.Size{
		ID:            id,
		Name:          req.Name,
		ProductID:     req.ProductID,
		BasePrice:     req.BasePrice,
		A:             req.A,
		B:             req.B,
		C:             req.C,
		D:             req.D,
		E:             req.E,
		F:             req.F,
		UseStock:      req.UseStock,
		StockQuantity: req.StockQuantity,
	}

	if err := h.sizeQueries.UpdateSize(id, size); err != nil {
		if err.Error() == "size not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Size not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Size updated successfully"})
}

func (h *AdminHandler) DeleteSize(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid size ID"})
		return
	}

	if err := h.sizeQueries.DeleteSize(id); err != nil {
		if err.Error() == "size not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Size not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Size deleted successfully"})
}

func (h *AdminHandler) validateProductExists(productID int) bool {
	query := "SELECT 1 FROM products WHERE id = $1"
	var exists int
	err := h.db.QueryRow(query, productID).Scan(&exists)
	return err == nil
}

func (h *AdminHandler) validateColorExists(colorID int) bool {
	query := "SELECT 1 FROM colors WHERE id = $1"
	var exists int
	err := h.db.QueryRow(query, colorID).Scan(&exists)
	return err == nil
}

// ProductVariant Management

func (h *AdminHandler) ListProductVariants(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")
	
	var productID *int
	if productIDStr := c.Query("product_id"); productIDStr != "" {
		if pid, err := strconv.Atoi(productIDStr); err == nil {
			productID = &pid
		}
	}
	
	var colorID *int
	if colorIDStr := c.Query("color_id"); colorIDStr != "" {
		if cid, err := strconv.Atoi(colorIDStr); err == nil {
			colorID = &cid
		}
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	variants, total, err := h.productVariantQueries.ListProductVariants(page, limit, search, productID, colorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.ProductVariantListResponse{
		ProductVariants: variants,
		Total:           total,
		Page:            page,
		Limit:           limit,
	})
}

func (h *AdminHandler) CreateProductVariant(c *gin.Context) {
	var req models.ProductVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate product exists
	if !h.validateProductExists(req.ProductID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product not found"})
		return
	}

	// Validate color exists
	if !h.validateColorExists(req.ColorID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Color not found"})
		return
	}

	// Validate images exist
	if len(req.ImageIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one image is required"})
		return
	}

	variant := &models.ProductVariant{
		ProductID: req.ProductID,
		Name:      req.Name,
		ColorID:   req.ColorID,
		IsDefault: req.IsDefault,
	}

	if err := h.productVariantQueries.CreateProductVariant(variant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Associate images with variant
	if err := h.productVariantQueries.UpdateProductVariantImages(variant.ID, req.ImageIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to associate images"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Product variant created successfully", "id": variant.ID})
}

func (h *AdminHandler) GetProductVariant(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product variant ID"})
		return
	}

	variant, err := h.productVariantQueries.GetProductVariantByID(id)
	if err != nil {
		if err.Error() == "product variant not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product variant not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := models.ProductVariantResponse{
		ID:        variant.ID,
		ProductID: variant.ProductID,
		Name:      variant.Name,
		ColorID:   variant.ColorID,
		IsDefault: variant.IsDefault,
		CreatedAt: variant.CreatedAt.Format(time.RFC3339),
		UpdatedAt: variant.UpdatedAt.Format(time.RFC3339),
		Product:   variant.Product,
		Color:     variant.Color,
		Images:    variant.Images,
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) UpdateProductVariant(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product variant ID"})
		return
	}

	var req models.ProductVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate product exists
	if !h.validateProductExists(req.ProductID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product not found"})
		return
	}

	// Validate color exists
	if !h.validateColorExists(req.ColorID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Color not found"})
		return
	}

	// Validate images exist
	if len(req.ImageIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one image is required"})
		return
	}

	variant := &models.ProductVariant{
		ID:        id,
		ProductID: req.ProductID,
		Name:      req.Name,
		ColorID:   req.ColorID,
		IsDefault: req.IsDefault,
	}

	if err := h.productVariantQueries.UpdateProductVariant(id, variant); err != nil {
		if err.Error() == "product variant not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product variant not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update images associated with variant
	if err := h.productVariantQueries.UpdateProductVariantImages(id, req.ImageIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update images"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product variant updated successfully"})
}

func (h *AdminHandler) DeleteProductVariant(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product variant ID"})
		return
	}

	if err := h.productVariantQueries.DeleteProductVariant(id); err != nil {
		if err.Error() == "product variant not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product variant not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product variant deleted successfully"})
}

// Order Management

func (h *AdminHandler) ListOrders(c *gin.Context) {
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

func (h *AdminHandler) GetOrderDetails(c *gin.Context) {
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

	c.JSON(http.StatusOK, order)
}

func (h *AdminHandler) UpdateOrderStatus(c *gin.Context) {
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

func (h *AdminHandler) DeleteOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	err = h.orderQueries.DeleteOrder(id)
	if err != nil {
		if err.Error() == "order not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order deleted successfully"})
}

// Settings Management

func (h *AdminHandler) GetSettings(c *gin.Context) {
	settings, err := h.settingsQueries.GetAllSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get settings"})
		return
	}

	c.JSON(http.StatusOK, models.SiteSettingsResponse{Settings: settings})
}

func (h *AdminHandler) UpdateSetting(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Setting key is required"})
		return
	}

	var req models.UpdateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate maintenance_mode value
	if key == "maintenance_mode" && req.Value != "true" && req.Value != "false" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maintenance_mode must be 'true' or 'false'"})
		return
	}

	err := h.settingsQueries.UpdateSetting(key, req.Value)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Setting not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update setting"})
		return
	}

	// Get updated setting
	setting, err := h.settingsQueries.GetSettingByKey(key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get updated setting"})
		return
	}

	c.JSON(http.StatusOK, setting)
}

// Client Reviews Management

func (h *AdminHandler) ListClientReviews(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	activeOnly := c.Query("active_only") == "true"

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	reviews, total, err := h.clientReviewQueries.ListClientReviews(page, limit, activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve client reviews"})
		return
	}

	response := models.ClientReviewListResponse{
		ClientReviews: reviews,
		Total:         total,
		Page:          page,
		Limit:         limit,
	}

	c.JSON(http.StatusOK, response)
}

func (h *AdminHandler) GetClientReview(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client review ID"})
		return
	}

	review, err := h.clientReviewQueries.GetClientReviewByID(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Client review not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get client review"})
		return
	}

	c.JSON(http.StatusOK, review)
}

func (h *AdminHandler) CreateClientReview(c *gin.Context) {
	var req models.CreateClientReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify image exists
	_, err := h.imageQueries.GetImageByID(req.ImageID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image not found"})
		return
	}

	review, err := h.clientReviewQueries.CreateClientReview(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create client review"})
		return
	}

	c.JSON(http.StatusCreated, review)
}

func (h *AdminHandler) UpdateClientReview(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client review ID"})
		return
	}

	var req models.UpdateClientReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify image exists
	_, err = h.imageQueries.GetImageByID(req.ImageID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image not found"})
		return
	}

	review, err := h.clientReviewQueries.UpdateClientReview(id, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Client review not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update client review"})
		return
	}

	c.JSON(http.StatusOK, review)
}

func (h *AdminHandler) DeleteClientReview(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client review ID"})
		return
	}

	err = h.clientReviewQueries.DeleteClientReview(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Client review not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete client review"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Client review deleted successfully"})
}

func (h *AdminHandler) ReorderClientReviews(c *gin.Context) {
	var req models.ReorderClientReviewsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert to the format expected by the database function
	var orders []struct{ ID, DisplayOrder int }
	for _, order := range req.ReviewOrders {
		orders = append(orders, struct{ ID, DisplayOrder int }{
			ID:           order.ID,
			DisplayOrder: order.DisplayOrder,
		})
	}

	err := h.clientReviewQueries.ReorderClientReviews(orders)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder client reviews"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Client reviews reordered successfully"})
}
