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
	userQueries  *database.UserQueries
	imageQueries *database.ImageQueries
}

func NewAdminHandler(db *sql.DB) *AdminHandler {
	return &AdminHandler{
		userQueries:  database.NewUserQueries(db),
		imageQueries: database.NewImageQueries(db),
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