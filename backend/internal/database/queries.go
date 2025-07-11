package database

import (
	"database/sql"
	"fmt"
	"time"
	"notsofluffy-backend/internal/auth"
	"notsofluffy-backend/internal/models"
	"github.com/lib/pq"
)

type UserQueries struct {
	db *sql.DB
}

func NewUserQueries(db *sql.DB) *UserQueries {
	return &UserQueries{db: db}
}

func (q *UserQueries) CreateUser(user *models.User) error {
	query := `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	err := q.db.QueryRow(query, user.Email, user.PasswordHash, user.Role).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (q *UserQueries) GetUserByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	user := &models.User{}
	err := q.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func (q *UserQueries) GetUserByID(id int) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	user := &models.User{}
	err := q.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}


func (q *UserQueries) DeleteUser(id int) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := q.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	
	return nil
}

func (q *UserQueries) EmailExists(email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	var exists bool
	err := q.db.QueryRow(query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}
	return exists, nil
}

// Admin user management methods

func (q *UserQueries) ListUsers(page, limit int, search string) ([]models.User, int, error) {
	offset := (page - 1) * limit
	var users []models.User
	var total int

	// Count total users
	countQuery := `SELECT COUNT(*) FROM users`
	countArgs := []interface{}{}
	
	if search != "" {
		countQuery += ` WHERE email ILIKE $1`
		countArgs = append(countArgs, "%"+search+"%")
	}
	
	err := q.db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Get users
	query := `
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM users
	`
	args := []interface{}{}
	
	if search != "" {
		query += ` WHERE email ILIKE $1`
		args = append(args, "%"+search+"%")
	}
	
	query += ` ORDER BY created_at DESC LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
	args = append(args, limit, offset)

	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, total, nil
}

func (q *UserQueries) CreateAdminUser(email, password, role string) (*models.User, error) {
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         role,
	}

	query := `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	err = q.db.QueryRow(query, user.Email, user.PasswordHash, user.Role).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (q *UserQueries) UpdateUser(id int, email, password, role string) (*models.User, error) {
	user, err := q.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	user.Email = email
	user.Role = role

	if password != "" {
		hashedPassword, err := auth.HashPassword(password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		user.PasswordHash = hashedPassword
	}

	query := `
		UPDATE users
		SET email = $1, password_hash = $2, role = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
		RETURNING updated_at
	`
	err = q.db.QueryRow(query, user.Email, user.PasswordHash, user.Role, user.ID).Scan(
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// Image Queries

type ImageQueries struct {
	db *sql.DB
}

func NewImageQueries(db *sql.DB) *ImageQueries {
	return &ImageQueries{db: db}
}

func (q *ImageQueries) CreateImage(image *models.Image) error {
	query := `
		INSERT INTO images (filename, original_name, path, size_bytes, mime_type, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	err := q.db.QueryRow(query, 
		image.Filename, 
		image.OriginalName, 
		image.Path, 
		image.SizeBytes, 
		image.MimeType, 
		image.UploadedBy,
	).Scan(
		&image.ID,
		&image.CreatedAt,
		&image.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create image: %w", err)
	}
	return nil
}

func (q *ImageQueries) GetImageByID(id int) (*models.Image, error) {
	query := `
		SELECT id, filename, original_name, path, size_bytes, mime_type, uploaded_by, created_at, updated_at
		FROM images
		WHERE id = $1
	`
	image := &models.Image{}
	err := q.db.QueryRow(query, id).Scan(
		&image.ID,
		&image.Filename,
		&image.OriginalName,
		&image.Path,
		&image.SizeBytes,
		&image.MimeType,
		&image.UploadedBy,
		&image.CreatedAt,
		&image.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("image not found")
		}
		return nil, fmt.Errorf("failed to get image: %w", err)
	}
	return image, nil
}

func (q *ImageQueries) ListImages(page, limit int) ([]models.Image, int, error) {
	offset := (page - 1) * limit
	var images []models.Image
	var total int

	// Count total images
	countQuery := `SELECT COUNT(*) FROM images`
	err := q.db.QueryRow(countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count images: %w", err)
	}

	// Get images
	query := `
		SELECT id, filename, original_name, path, size_bytes, mime_type, uploaded_by, created_at, updated_at
		FROM images
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := q.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list images: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var image models.Image
		err := rows.Scan(
			&image.ID,
			&image.Filename,
			&image.OriginalName,
			&image.Path,
			&image.SizeBytes,
			&image.MimeType,
			&image.UploadedBy,
			&image.CreatedAt,
			&image.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan image: %w", err)
		}
		images = append(images, image)
	}

	return images, total, nil
}

func (q *ImageQueries) DeleteImage(id int) error {
	query := `DELETE FROM images WHERE id = $1`
	result, err := q.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("image not found")
	}
	
	return nil
}

// Category Queries

type CategoryQueries struct {
	db *sql.DB
}

func NewCategoryQueries(db *sql.DB) *CategoryQueries {
	return &CategoryQueries{db: db}
}

func (q *CategoryQueries) CreateCategory(category *models.Category) error {
	query := `
		INSERT INTO categories (name, slug, image_id, active, chart_only)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	err := q.db.QueryRow(query, 
		category.Name, 
		category.Slug, 
		category.ImageID, 
		category.Active, 
		category.ChartOnly,
	).Scan(
		&category.ID,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}
	return nil
}

func (q *CategoryQueries) GetCategoryByID(id int) (*models.CategoryWithImage, error) {
	query := `
		SELECT 
			c.id, c.name, c.slug, c.image_id, c.active, c.chart_only, c.created_at, c.updated_at,
			i.id, i.filename, i.original_name, i.path, i.size_bytes, i.mime_type, i.uploaded_by, i.created_at, i.updated_at
		FROM categories c
		LEFT JOIN images i ON c.image_id = i.id
		WHERE c.id = $1
	`
	category := &models.CategoryWithImage{}
	var image models.Image
	var imageCreatedAt, imageUpdatedAt sql.NullTime
	
	err := q.db.QueryRow(query, id).Scan(
		&category.ID,
		&category.Name,
		&category.Slug,
		&category.ImageID,
		&category.Active,
		&category.ChartOnly,
		&category.CreatedAt,
		&category.UpdatedAt,
		&image.ID,
		&image.Filename,
		&image.OriginalName,
		&image.Path,
		&image.SizeBytes,
		&image.MimeType,
		&image.UploadedBy,
		&imageCreatedAt,
		&imageUpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("category not found")
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}
	
	// Add image if it exists
	if category.ImageID != nil && image.ID != 0 {
		if imageCreatedAt.Valid {
			image.CreatedAt = imageCreatedAt.Time
		}
		if imageUpdatedAt.Valid {
			image.UpdatedAt = imageUpdatedAt.Time
		}
		category.Image = &models.ImageResponse{
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
	}
	
	return category, nil
}

func (q *CategoryQueries) ListCategories(page, limit int, search string, activeOnly *bool, chartOnly *bool) ([]models.CategoryWithImage, int, error) {
	offset := (page - 1) * limit
	var categories []models.CategoryWithImage
	var total int

	// Build WHERE clause
	whereConditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if search != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("(c.name ILIKE $%d OR c.slug ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+search+"%")
		argIndex++
	}

	if activeOnly != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("c.active = $%d", argIndex))
		args = append(args, *activeOnly)
		argIndex++
	}

	if chartOnly != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("c.chart_only = $%d", argIndex))
		args = append(args, *chartOnly)
		argIndex++
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + fmt.Sprintf("(%s)", whereConditions[0])
		for i := 1; i < len(whereConditions); i++ {
			whereClause += " AND " + fmt.Sprintf("(%s)", whereConditions[i])
		}
	}

	// Count total categories
	countQuery := `SELECT COUNT(*) FROM categories c ` + whereClause
	err := q.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count categories: %w", err)
	}

	// Get categories with images
	query := `
		SELECT 
			c.id, c.name, c.slug, c.image_id, c.active, c.chart_only, c.created_at, c.updated_at,
			i.id, i.filename, i.original_name, i.path, i.size_bytes, i.mime_type, i.uploaded_by, i.created_at, i.updated_at
		FROM categories c
		LEFT JOIN images i ON c.image_id = i.id
		` + whereClause + `
		ORDER BY c.created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argIndex) + ` OFFSET $` + fmt.Sprintf("%d", argIndex+1)
	
	args = append(args, limit, offset)
	
	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list categories: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var category models.CategoryWithImage
		var image models.Image
		var imageCreatedAt, imageUpdatedAt sql.NullTime
		var imageID, imageSizeBytes, imageUploadedBy sql.NullInt64
		var imageFilename, imageOriginalName, imagePath, imageMimeType sql.NullString

		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Slug,
			&category.ImageID,
			&category.Active,
			&category.ChartOnly,
			&category.CreatedAt,
			&category.UpdatedAt,
			&imageID,
			&imageFilename,
			&imageOriginalName,
			&imagePath,
			&imageSizeBytes,
			&imageMimeType,
			&imageUploadedBy,
			&imageCreatedAt,
			&imageUpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan category: %w", err)
		}

		// Add image if it exists
		if category.ImageID != nil && imageID.Valid {
			image.ID = int(imageID.Int64)
			image.Filename = imageFilename.String
			image.OriginalName = imageOriginalName.String
			image.Path = imagePath.String
			image.SizeBytes = imageSizeBytes.Int64
			image.MimeType = imageMimeType.String
			image.UploadedBy = int(imageUploadedBy.Int64)
			
			if imageCreatedAt.Valid {
				image.CreatedAt = imageCreatedAt.Time
			}
			if imageUpdatedAt.Valid {
				image.UpdatedAt = imageUpdatedAt.Time
			}
			
			category.Image = &models.ImageResponse{
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
		}

		categories = append(categories, category)
	}

	return categories, total, nil
}

// GetActiveCategories returns all active categories with their images
func (q *CategoryQueries) GetActiveCategories() ([]models.CategoryWithImage, error) {
	query := `
		SELECT 
			c.id, c.name, c.slug, c.image_id, c.active, c.chart_only, c.created_at, c.updated_at,
			i.id, i.filename, i.original_name, i.path, i.size_bytes, i.mime_type, i.uploaded_by, i.created_at, i.updated_at
		FROM categories c
		LEFT JOIN images i ON c.image_id = i.id
		WHERE c.active = true
		ORDER BY c.name
	`

	rows, err := q.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active categories: %w", err)
	}
	defer rows.Close()

	var categories []models.CategoryWithImage
	for rows.Next() {
		var category models.CategoryWithImage
		var image models.Image
		var imageCreatedAt, imageUpdatedAt sql.NullTime
		var imageID, imageSizeBytes, imageUploadedBy sql.NullInt64
		var imageFilename, imageOriginalName, imagePath, imageMimeType sql.NullString

		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Slug,
			&category.ImageID,
			&category.Active,
			&category.ChartOnly,
			&category.CreatedAt,
			&category.UpdatedAt,
			&imageID,
			&imageFilename,
			&imageOriginalName,
			&imagePath,
			&imageSizeBytes,
			&imageMimeType,
			&imageUploadedBy,
			&imageCreatedAt,
			&imageUpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}

		// Add image if it exists
		if category.ImageID != nil && imageID.Valid {
			image.ID = int(imageID.Int64)
			image.Filename = imageFilename.String
			image.OriginalName = imageOriginalName.String
			image.Path = imagePath.String
			image.SizeBytes = imageSizeBytes.Int64
			image.MimeType = imageMimeType.String
			image.UploadedBy = int(imageUploadedBy.Int64)
			
			if imageCreatedAt.Valid {
				image.CreatedAt = imageCreatedAt.Time
			}
			if imageUpdatedAt.Valid {
				image.UpdatedAt = imageUpdatedAt.Time
			}
			
			category.Image = &models.ImageResponse{
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
		}

		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating categories: %w", err)
	}

	return categories, nil
}

func (q *CategoryQueries) UpdateCategory(id int, name, slug string, imageID *int, active, chartOnly bool) (*models.Category, error) {
	category := &models.Category{
		ID:        id,
		Name:      name,
		Slug:      slug,
		ImageID:   imageID,
		Active:    active,
		ChartOnly: chartOnly,
	}

	query := `
		UPDATE categories
		SET name = $1, slug = $2, image_id = $3, active = $4, chart_only = $5, updated_at = CURRENT_TIMESTAMP
		WHERE id = $6
		RETURNING created_at, updated_at
	`
	err := q.db.QueryRow(query, name, slug, imageID, active, chartOnly, id).Scan(
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	return category, nil
}

func (q *CategoryQueries) DeleteCategory(id int) error {
	query := `DELETE FROM categories WHERE id = $1`
	result, err := q.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("category not found")
	}
	
	return nil
}

func (q *CategoryQueries) SlugExists(slug string, excludeID *int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM categories WHERE slug = $1`
	args := []interface{}{slug}
	
	if excludeID != nil {
		query += ` AND id != $2`
		args = append(args, *excludeID)
	}
	
	query += `)`
	
	var exists bool
	err := q.db.QueryRow(query, args...).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check slug existence: %w", err)
	}
	return exists, nil
}

func (q *CategoryQueries) ToggleActive(id int) error {
	query := `UPDATE categories SET active = NOT active, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	result, err := q.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to toggle category active status: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("category not found")
	}
	
	return nil
}

// Material Queries

type MaterialQueries struct {
	db *sql.DB
}

func NewMaterialQueries(db *sql.DB) *MaterialQueries {
	return &MaterialQueries{db: db}
}

func (q *MaterialQueries) CreateMaterial(material *models.Material) error {
	query := `
		INSERT INTO materials (name)
		VALUES ($1)
		RETURNING id, created_at, updated_at
	`
	err := q.db.QueryRow(query, material.Name).Scan(
		&material.ID,
		&material.CreatedAt,
		&material.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create material: %w", err)
	}
	return nil
}

func (q *MaterialQueries) GetMaterialByID(id int) (*models.Material, error) {
	query := `
		SELECT id, name, created_at, updated_at
		FROM materials
		WHERE id = $1
	`
	material := &models.Material{}
	err := q.db.QueryRow(query, id).Scan(
		&material.ID,
		&material.Name,
		&material.CreatedAt,
		&material.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("material not found")
		}
		return nil, fmt.Errorf("failed to get material: %w", err)
	}
	return material, nil
}

func (q *MaterialQueries) ListMaterials(page, limit int, search string) ([]models.Material, int, error) {
	offset := (page - 1) * limit
	var materials []models.Material
	var total int

	// Build WHERE clause
	whereClause := ""
	args := []interface{}{}
	argIndex := 1

	if search != "" {
		whereClause = "WHERE name ILIKE $1"
		args = append(args, "%"+search+"%")
		argIndex++
	}

	// Count total materials
	countQuery := `SELECT COUNT(*) FROM materials ` + whereClause
	err := q.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count materials: %w", err)
	}

	// Get materials
	query := `
		SELECT id, name, created_at, updated_at
		FROM materials
		` + whereClause + `
		ORDER BY name ASC
		LIMIT $` + fmt.Sprintf("%d", argIndex) + ` OFFSET $` + fmt.Sprintf("%d", argIndex+1)
	
	args = append(args, limit, offset)
	
	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list materials: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var material models.Material
		err := rows.Scan(
			&material.ID,
			&material.Name,
			&material.CreatedAt,
			&material.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan material: %w", err)
		}
		materials = append(materials, material)
	}

	return materials, total, nil
}

func (q *MaterialQueries) UpdateMaterial(id int, name string) (*models.Material, error) {
	material := &models.Material{
		ID:   id,
		Name: name,
	}

	query := `
		UPDATE materials
		SET name = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
		RETURNING created_at, updated_at
	`
	err := q.db.QueryRow(query, name, id).Scan(
		&material.CreatedAt,
		&material.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update material: %w", err)
	}

	return material, nil
}

func (q *MaterialQueries) DeleteMaterial(id int) error {
	query := `DELETE FROM materials WHERE id = $1`
	result, err := q.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete material: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("material not found")
	}
	
	return nil
}

func (q *MaterialQueries) NameExists(name string, excludeID *int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM materials WHERE name = $1`
	args := []interface{}{name}
	
	if excludeID != nil {
		query += ` AND id != $2`
		args = append(args, *excludeID)
	}
	
	query += `)`
	
	var exists bool
	err := q.db.QueryRow(query, args...).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check material name existence: %w", err)
	}
	return exists, nil
}

// Color Queries

type ColorQueries struct {
	db *sql.DB
}

func NewColorQueries(db *sql.DB) *ColorQueries {
	return &ColorQueries{db: db}
}

func (q *ColorQueries) CreateColor(color *models.Color) error {
	query := `
		INSERT INTO colors (name, image_id, custom, material_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	err := q.db.QueryRow(query, 
		color.Name, 
		color.ImageID, 
		color.Custom, 
		color.MaterialID,
	).Scan(
		&color.ID,
		&color.CreatedAt,
		&color.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create color: %w", err)
	}
	return nil
}

func (q *ColorQueries) GetColorByID(id int) (*models.ColorWithRelations, error) {
	query := `
		SELECT 
			c.id, c.name, c.image_id, c.custom, c.material_id, c.created_at, c.updated_at,
			i.id, i.filename, i.original_name, i.path, i.size_bytes, i.mime_type, i.uploaded_by, i.created_at, i.updated_at,
			m.id, m.name, m.created_at, m.updated_at
		FROM colors c
		LEFT JOIN images i ON c.image_id = i.id
		INNER JOIN materials m ON c.material_id = m.id
		WHERE c.id = $1
	`
	color := &models.ColorWithRelations{}
	var image models.Image
	var material models.Material
	var imageCreatedAt, imageUpdatedAt sql.NullTime
	var imageID, imageSizeBytes, imageUploadedBy sql.NullInt64
	var imageFilename, imageOriginalName, imagePath, imageMimeType sql.NullString
	
	err := q.db.QueryRow(query, id).Scan(
		&color.ID,
		&color.Name,
		&color.ImageID,
		&color.Custom,
		&color.MaterialID,
		&color.CreatedAt,
		&color.UpdatedAt,
		&imageID,
		&imageFilename,
		&imageOriginalName,
		&imagePath,
		&imageSizeBytes,
		&imageMimeType,
		&imageUploadedBy,
		&imageCreatedAt,
		&imageUpdatedAt,
		&material.ID,
		&material.Name,
		&material.CreatedAt,
		&material.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("color not found")
		}
		return nil, fmt.Errorf("failed to get color: %w", err)
	}
	
	// Add material
	color.Material = &models.MaterialResponse{
		ID:        material.ID,
		Name:      material.Name,
		CreatedAt: material.CreatedAt.Format(time.RFC3339),
		UpdatedAt: material.UpdatedAt.Format(time.RFC3339),
	}
	
	// Add image if it exists
	if color.ImageID != nil && imageID.Valid {
		image.ID = int(imageID.Int64)
		image.Filename = imageFilename.String
		image.OriginalName = imageOriginalName.String
		image.Path = imagePath.String
		image.SizeBytes = imageSizeBytes.Int64
		image.MimeType = imageMimeType.String
		image.UploadedBy = int(imageUploadedBy.Int64)
		
		if imageCreatedAt.Valid {
			image.CreatedAt = imageCreatedAt.Time
		}
		if imageUpdatedAt.Valid {
			image.UpdatedAt = imageUpdatedAt.Time
		}
		
		color.Image = &models.ImageResponse{
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
	}
	
	return color, nil
}

func (q *ColorQueries) ListColors(page, limit int, search string, materialID *int, customOnly *bool) ([]models.ColorWithRelations, int, error) {
	offset := (page - 1) * limit
	var colors []models.ColorWithRelations
	var total int

	// Build WHERE clause
	whereConditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if search != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("c.name ILIKE $%d", argIndex))
		args = append(args, "%"+search+"%")
		argIndex++
	}

	if materialID != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("c.material_id = $%d", argIndex))
		args = append(args, *materialID)
		argIndex++
	}

	if customOnly != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("c.custom = $%d", argIndex))
		args = append(args, *customOnly)
		argIndex++
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + fmt.Sprintf("(%s)", whereConditions[0])
		for i := 1; i < len(whereConditions); i++ {
			whereClause += " AND " + fmt.Sprintf("(%s)", whereConditions[i])
		}
	}

	// Count total colors
	countQuery := `
		SELECT COUNT(*) 
		FROM colors c 
		INNER JOIN materials m ON c.material_id = m.id 
		` + whereClause
	err := q.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count colors: %w", err)
	}

	// Get colors with relations
	query := `
		SELECT 
			c.id, c.name, c.image_id, c.custom, c.material_id, c.created_at, c.updated_at,
			i.id, i.filename, i.original_name, i.path, i.size_bytes, i.mime_type, i.uploaded_by, i.created_at, i.updated_at,
			m.id, m.name, m.created_at, m.updated_at
		FROM colors c
		LEFT JOIN images i ON c.image_id = i.id
		INNER JOIN materials m ON c.material_id = m.id
		` + whereClause + `
		ORDER BY c.name ASC
		LIMIT $` + fmt.Sprintf("%d", argIndex) + ` OFFSET $` + fmt.Sprintf("%d", argIndex+1)
	
	args = append(args, limit, offset)
	
	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list colors: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var color models.ColorWithRelations
		var image models.Image
		var material models.Material
		var imageCreatedAt, imageUpdatedAt sql.NullTime
		var imageID, imageSizeBytes, imageUploadedBy sql.NullInt64
		var imageFilename, imageOriginalName, imagePath, imageMimeType sql.NullString

		err := rows.Scan(
			&color.ID,
			&color.Name,
			&color.ImageID,
			&color.Custom,
			&color.MaterialID,
			&color.CreatedAt,
			&color.UpdatedAt,
			&imageID,
			&imageFilename,
			&imageOriginalName,
			&imagePath,
			&imageSizeBytes,
			&imageMimeType,
			&imageUploadedBy,
			&imageCreatedAt,
			&imageUpdatedAt,
			&material.ID,
			&material.Name,
			&material.CreatedAt,
			&material.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan color: %w", err)
		}

		// Add material
		color.Material = &models.MaterialResponse{
			ID:        material.ID,
			Name:      material.Name,
			CreatedAt: material.CreatedAt.Format(time.RFC3339),
			UpdatedAt: material.UpdatedAt.Format(time.RFC3339),
		}

		// Add image if it exists
		if color.ImageID != nil && imageID.Valid {
			image.ID = int(imageID.Int64)
			image.Filename = imageFilename.String
			image.OriginalName = imageOriginalName.String
			image.Path = imagePath.String
			image.SizeBytes = imageSizeBytes.Int64
			image.MimeType = imageMimeType.String
			image.UploadedBy = int(imageUploadedBy.Int64)
			
			if imageCreatedAt.Valid {
				image.CreatedAt = imageCreatedAt.Time
			}
			if imageUpdatedAt.Valid {
				image.UpdatedAt = imageUpdatedAt.Time
			}
			
			color.Image = &models.ImageResponse{
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
		}

		colors = append(colors, color)
	}

	return colors, total, nil
}

func (q *ColorQueries) UpdateColor(id int, name string, imageID *int, custom bool, materialID int) (*models.Color, error) {
	color := &models.Color{
		ID:         id,
		Name:       name,
		ImageID:    imageID,
		Custom:     custom,
		MaterialID: materialID,
	}

	query := `
		UPDATE colors
		SET name = $1, image_id = $2, custom = $3, material_id = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
		RETURNING created_at, updated_at
	`
	err := q.db.QueryRow(query, name, imageID, custom, materialID, id).Scan(
		&color.CreatedAt,
		&color.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update color: %w", err)
	}

	return color, nil
}

func (q *ColorQueries) DeleteColor(id int) error {
	query := `DELETE FROM colors WHERE id = $1`
	result, err := q.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete color: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("color not found")
	}
	
	return nil
}

func (q *ColorQueries) NameExistsForMaterial(name string, materialID int, excludeID *int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM colors WHERE name = $1 AND material_id = $2`
	args := []interface{}{name, materialID}
	
	if excludeID != nil {
		query += ` AND id != $3`
		args = append(args, *excludeID)
	}
	
	query += `)`
	
	var exists bool
	err := q.db.QueryRow(query, args...).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check color name existence for material: %w", err)
	}
	return exists, nil
}

// AdditionalService Queries

type AdditionalServiceQueries struct {
	db *sql.DB
}

func NewAdditionalServiceQueries(db *sql.DB) *AdditionalServiceQueries {
	return &AdditionalServiceQueries{db: db}
}

func (q *AdditionalServiceQueries) CreateAdditionalService(service *models.AdditionalService) error {
	query := `
		INSERT INTO additional_services (name, description, price)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	err := q.db.QueryRow(query, 
		service.Name, 
		service.Description, 
		service.Price,
	).Scan(
		&service.ID,
		&service.CreatedAt,
		&service.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create additional service: %w", err)
	}
	return nil
}

func (q *AdditionalServiceQueries) GetAdditionalServiceByID(id int) (*models.AdditionalServiceWithImages, error) {
	// First get the service
	serviceQuery := `
		SELECT id, name, description, price, created_at, updated_at
		FROM additional_services
		WHERE id = $1
	`
	service := &models.AdditionalServiceWithImages{}
	err := q.db.QueryRow(serviceQuery, id).Scan(
		&service.ID,
		&service.Name,
		&service.Description,
		&service.Price,
		&service.CreatedAt,
		&service.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("additional service not found")
		}
		return nil, fmt.Errorf("failed to get additional service: %w", err)
	}

	// Then get associated images
	imagesQuery := `
		SELECT i.id, i.filename, i.original_name, i.path, i.size_bytes, i.mime_type, i.uploaded_by, i.created_at, i.updated_at
		FROM images i
		INNER JOIN additional_service_images asi ON i.id = asi.image_id
		WHERE asi.additional_service_id = $1
		ORDER BY i.created_at ASC
	`
	rows, err := q.db.Query(imagesQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get service images: %w", err)
	}
	defer rows.Close()

	var images []models.ImageResponse
	for rows.Next() {
		var image models.Image
		err := rows.Scan(
			&image.ID,
			&image.Filename,
			&image.OriginalName,
			&image.Path,
			&image.SizeBytes,
			&image.MimeType,
			&image.UploadedBy,
			&image.CreatedAt,
			&image.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan service image: %w", err)
		}

		images = append(images, models.ImageResponse{
			ID:           image.ID,
			Filename:     image.Filename,
			OriginalName: image.OriginalName,
			Path:         image.Path,
			SizeBytes:    image.SizeBytes,
			MimeType:     image.MimeType,
			UploadedBy:   image.UploadedBy,
			CreatedAt:    image.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    image.UpdatedAt.Format(time.RFC3339),
		})
	}

	service.Images = images
	return service, nil
}

func (q *AdditionalServiceQueries) ListAdditionalServices(page, limit int, search string, minPrice, maxPrice *float64) ([]models.AdditionalServiceWithImages, int, error) {
	offset := (page - 1) * limit
	var services []models.AdditionalServiceWithImages
	var total int

	// Build WHERE clause
	whereConditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if search != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("(s.name ILIKE $%d OR s.description ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+search+"%")
		argIndex++
	}

	if minPrice != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("s.price >= $%d", argIndex))
		args = append(args, *minPrice)
		argIndex++
	}

	if maxPrice != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("s.price <= $%d", argIndex))
		args = append(args, *maxPrice)
		argIndex++
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + fmt.Sprintf("(%s)", whereConditions[0])
		for i := 1; i < len(whereConditions); i++ {
			whereClause += " AND " + fmt.Sprintf("(%s)", whereConditions[i])
		}
	}

	// Count total services
	countQuery := `SELECT COUNT(*) FROM additional_services s ` + whereClause
	err := q.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count additional services: %w", err)
	}

	// Get services
	servicesQuery := `
		SELECT s.id, s.name, s.description, s.price, s.created_at, s.updated_at
		FROM additional_services s
		` + whereClause + `
		ORDER BY s.name ASC
		LIMIT $` + fmt.Sprintf("%d", argIndex) + ` OFFSET $` + fmt.Sprintf("%d", argIndex+1)
	
	args = append(args, limit, offset)
	
	rows, err := q.db.Query(servicesQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list additional services: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var service models.AdditionalServiceWithImages
		err := rows.Scan(
			&service.ID,
			&service.Name,
			&service.Description,
			&service.Price,
			&service.CreatedAt,
			&service.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan additional service: %w", err)
		}

		// Get images for this service
		imagesQuery := `
			SELECT i.id, i.filename, i.original_name, i.path, i.size_bytes, i.mime_type, i.uploaded_by, i.created_at, i.updated_at
			FROM images i
			INNER JOIN additional_service_images asi ON i.id = asi.image_id
			WHERE asi.additional_service_id = $1
			ORDER BY i.created_at ASC
		`
		imageRows, err := q.db.Query(imagesQuery, service.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get images for service %d: %w", service.ID, err)
		}

		var images []models.ImageResponse
		for imageRows.Next() {
			var image models.Image
			err := imageRows.Scan(
				&image.ID,
				&image.Filename,
				&image.OriginalName,
				&image.Path,
				&image.SizeBytes,
				&image.MimeType,
				&image.UploadedBy,
				&image.CreatedAt,
				&image.UpdatedAt,
			)
			if err != nil {
				imageRows.Close()
				return nil, 0, fmt.Errorf("failed to scan service image: %w", err)
			}

			images = append(images, models.ImageResponse{
				ID:           image.ID,
				Filename:     image.Filename,
				OriginalName: image.OriginalName,
				Path:         image.Path,
				SizeBytes:    image.SizeBytes,
				MimeType:     image.MimeType,
				UploadedBy:   image.UploadedBy,
				CreatedAt:    image.CreatedAt.Format(time.RFC3339),
				UpdatedAt:    image.UpdatedAt.Format(time.RFC3339),
			})
		}
		imageRows.Close()

		service.Images = images
		services = append(services, service)
	}

	return services, total, nil
}

func (q *AdditionalServiceQueries) UpdateAdditionalService(id int, name, description string, price float64) (*models.AdditionalService, error) {
	service := &models.AdditionalService{
		ID:          id,
		Name:        name,
		Description: description,
		Price:       price,
	}

	query := `
		UPDATE additional_services
		SET name = $1, description = $2, price = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
		RETURNING created_at, updated_at
	`
	err := q.db.QueryRow(query, name, description, price, id).Scan(
		&service.CreatedAt,
		&service.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update additional service: %w", err)
	}

	return service, nil
}

func (q *AdditionalServiceQueries) DeleteAdditionalService(id int) error {
	query := `DELETE FROM additional_services WHERE id = $1`
	result, err := q.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete additional service: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("additional service not found")
	}
	
	return nil
}

func (q *AdditionalServiceQueries) NameExists(name string, excludeID *int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM additional_services WHERE name = $1`
	args := []interface{}{name}
	
	if excludeID != nil {
		query += ` AND id != $2`
		args = append(args, *excludeID)
	}
	
	query += `)`
	
	var exists bool
	err := q.db.QueryRow(query, args...).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check additional service name existence: %w", err)
	}
	return exists, nil
}

// ManyToMany image management methods

func (q *AdditionalServiceQueries) ReplaceImages(serviceID int, imageIDs []int) error {
	// Start transaction
	tx, err := q.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing associations
	_, err = tx.Exec("DELETE FROM additional_service_images WHERE additional_service_id = $1", serviceID)
	if err != nil {
		return fmt.Errorf("failed to delete existing image associations: %w", err)
	}

	// Add new associations
	for _, imageID := range imageIDs {
		_, err = tx.Exec("INSERT INTO additional_service_images (additional_service_id, image_id) VALUES ($1, $2)", serviceID, imageID)
		if err != nil {
			return fmt.Errorf("failed to add image association: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (q *AdditionalServiceQueries) AddImages(serviceID int, imageIDs []int) error {
	for _, imageID := range imageIDs {
		_, err := q.db.Exec("INSERT INTO additional_service_images (additional_service_id, image_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", serviceID, imageID)
		if err != nil {
			return fmt.Errorf("failed to add image association: %w", err)
		}
	}
	return nil
}

func (q *AdditionalServiceQueries) RemoveImages(serviceID int, imageIDs []int) error {
	for _, imageID := range imageIDs {
		_, err := q.db.Exec("DELETE FROM additional_service_images WHERE additional_service_id = $1 AND image_id = $2", serviceID, imageID)
		if err != nil {
			return fmt.Errorf("failed to remove image association: %w", err)
		}
	}
	return nil
}

// Product Queries

type ProductQueries struct {
	db *sql.DB
}

func NewProductQueries(db *sql.DB) *ProductQueries {
	return &ProductQueries{db: db}
}

func (q *ProductQueries) ListProducts(page, limit int, search string, categoryID, materialID *int) ([]models.ProductWithRelations, int, error) {
	offset := (page - 1) * limit
	
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argCount := 0
	
	if search != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND (p.name ILIKE $%d OR p.short_description ILIKE $%d OR p.description ILIKE $%d)", argCount, argCount, argCount)
		args = append(args, "%"+search+"%")
	}
	
	if categoryID != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND p.category_id = $%d", argCount)
		args = append(args, *categoryID)
	}
	
	if materialID != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND p.material_id = $%d", argCount)
		args = append(args, *materialID)
	}
	
	// First get total count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM products p
		%s
	`, whereClause)
	
	var total int
	err := q.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}
	
	// Then get paginated results with all relations
	argCount++
	limitArg := argCount
	argCount++
	offsetArg := argCount
	
	query := fmt.Sprintf(`
		SELECT 
			p.id, p.name, p.short_description, p.description, p.material_id, p.main_image_id, p.category_id, p.created_at, p.updated_at,
			mi.id, mi.filename, mi.original_name, mi.path, mi.size_bytes, mi.mime_type, mi.uploaded_by, mi.created_at, mi.updated_at,
			m.id, m.name, m.created_at, m.updated_at,
			c.id, c.name, c.slug, c.image_id, c.active, c.chart_only, c.created_at, c.updated_at
		FROM products p
		JOIN images mi ON p.main_image_id = mi.id
		LEFT JOIN materials m ON p.material_id = m.id
		LEFT JOIN categories c ON p.category_id = c.id
		%s
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, limitArg, offsetArg)
	
	args = append(args, limit, offset)
	
	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list products: %w", err)
	}
	defer rows.Close()
	
	var products []models.ProductWithRelations
	
	for rows.Next() {
		var product models.ProductWithRelations
		var mainImage models.ImageResponse
		var material models.MaterialResponse
		var category models.CategoryResponse
		var materialID, categoryID sql.NullInt64
		var materialName, materialCreatedAt, materialUpdatedAt sql.NullString
		var categoryName, categorySlug, categoryCreatedAt, categoryUpdatedAt sql.NullString
		var categoryImageID sql.NullInt64
		var categoryActive, categoryChartOnly sql.NullBool
		
		err := rows.Scan(
			&product.ID, &product.Name, &product.ShortDescription, &product.Description,
			&product.MaterialID, &product.MainImageID, &product.CategoryID, &product.CreatedAt, &product.UpdatedAt,
			&mainImage.ID, &mainImage.Filename, &mainImage.OriginalName, &mainImage.Path,
			&mainImage.SizeBytes, &mainImage.MimeType, &mainImage.UploadedBy, &mainImage.CreatedAt, &mainImage.UpdatedAt,
			&materialID, &materialName, &materialCreatedAt, &materialUpdatedAt,
			&categoryID, &categoryName, &categorySlug, &categoryImageID, &categoryActive, &categoryChartOnly, &categoryCreatedAt, &categoryUpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan product: %w", err)
		}
		
		product.MainImage = mainImage
		
		// Add material if exists
		if materialID.Valid {
			material.ID = int(materialID.Int64)
			material.Name = materialName.String
			material.CreatedAt = materialCreatedAt.String
			material.UpdatedAt = materialUpdatedAt.String
			product.Material = &material
		}
		
		// Add category if exists
		if categoryID.Valid {
			category.ID = int(categoryID.Int64)
			category.Name = categoryName.String
			category.Slug = categorySlug.String
			if categoryImageID.Valid {
				imageID := int(categoryImageID.Int64)
				category.ImageID = &imageID
			}
			category.Active = categoryActive.Bool
			category.ChartOnly = categoryChartOnly.Bool
			category.CreatedAt = categoryCreatedAt.String
			category.UpdatedAt = categoryUpdatedAt.String
			product.Category = &category
		}
		
		// Get product images
		images, err := q.getProductImages(product.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get product images: %w", err)
		}
		product.Images = images
		
		// Get product services
		services, err := q.getProductServices(product.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get product services: %w", err)
		}
		product.AdditionalServices = services
		
		products = append(products, product)
	}
	
	return products, total, nil
}

func (q *ProductQueries) getProductImages(productID int) ([]models.ImageResponse, error) {
	query := `
		SELECT i.id, i.filename, i.original_name, i.path, i.size_bytes, i.mime_type, i.uploaded_by, i.created_at, i.updated_at
		FROM images i
		JOIN product_images pi ON i.id = pi.image_id
		WHERE pi.product_id = $1
		ORDER BY i.created_at ASC
	`
	
	rows, err := q.db.Query(query, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product images: %w", err)
	}
	defer rows.Close()
	
	var images []models.ImageResponse
	for rows.Next() {
		var image models.ImageResponse
		err := rows.Scan(
			&image.ID, &image.Filename, &image.OriginalName, &image.Path,
			&image.SizeBytes, &image.MimeType, &image.UploadedBy, &image.CreatedAt, &image.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan image: %w", err)
		}
		images = append(images, image)
	}
	
	return images, nil
}

func (q *ProductQueries) getProductServices(productID int) ([]models.AdditionalServiceResponse, error) {
	query := `
		SELECT a.id, a.name, a.description, a.price, a.created_at, a.updated_at
		FROM additional_services a
		JOIN product_services ps ON a.id = ps.additional_service_id
		WHERE ps.product_id = $1
		ORDER BY a.name ASC
	`
	
	rows, err := q.db.Query(query, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product services: %w", err)
	}
	defer rows.Close()
	
	var services []models.AdditionalServiceResponse
	for rows.Next() {
		var service models.AdditionalServiceResponse
		err := rows.Scan(
			&service.ID, &service.Name, &service.Description, &service.Price, &service.CreatedAt, &service.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan service: %w", err)
		}
		
		// For each service, get its images (empty for now, but maintaining structure)
		service.Images = []models.ImageResponse{}
		
		services = append(services, service)
	}
	
	return services, nil
}

func (q *ProductQueries) CreateProduct(product *models.Product) error {
	query := `
		INSERT INTO products (name, short_description, description, material_id, main_image_id, category_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	
	err := q.db.QueryRow(query, product.Name, product.ShortDescription, product.Description, 
		product.MaterialID, product.MainImageID, product.CategoryID).Scan(
		&product.ID, &product.CreatedAt, &product.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}
	
	return nil
}

func (q *ProductQueries) GetProduct(id int) (*models.ProductWithRelations, error) {
	query := `
		SELECT 
			p.id, p.name, p.short_description, p.description, p.material_id, p.main_image_id, p.category_id, p.created_at, p.updated_at,
			mi.id, mi.filename, mi.original_name, mi.path, mi.size_bytes, mi.mime_type, mi.uploaded_by, mi.created_at, mi.updated_at,
			m.id, m.name, m.created_at, m.updated_at,
			c.id, c.name, c.slug, c.image_id, c.active, c.chart_only, c.created_at, c.updated_at
		FROM products p
		JOIN images mi ON p.main_image_id = mi.id
		LEFT JOIN materials m ON p.material_id = m.id
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE p.id = $1
	`
	
	var product models.ProductWithRelations
	var mainImage models.ImageResponse
	var material models.MaterialResponse
	var category models.CategoryResponse
	var materialID, categoryID sql.NullInt64
	var materialName, materialCreatedAt, materialUpdatedAt sql.NullString
	var categoryName, categorySlug, categoryCreatedAt, categoryUpdatedAt sql.NullString
	var categoryImageID sql.NullInt64
	var categoryActive, categoryChartOnly sql.NullBool
	
	err := q.db.QueryRow(query, id).Scan(
		&product.ID, &product.Name, &product.ShortDescription, &product.Description,
		&product.MaterialID, &product.MainImageID, &product.CategoryID, &product.CreatedAt, &product.UpdatedAt,
		&mainImage.ID, &mainImage.Filename, &mainImage.OriginalName, &mainImage.Path,
		&mainImage.SizeBytes, &mainImage.MimeType, &mainImage.UploadedBy, &mainImage.CreatedAt, &mainImage.UpdatedAt,
		&materialID, &materialName, &materialCreatedAt, &materialUpdatedAt,
		&categoryID, &categoryName, &categorySlug, &categoryImageID, &categoryActive, &categoryChartOnly, &categoryCreatedAt, &categoryUpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("failed to get product: %w", err)
	}
	
	product.MainImage = mainImage
	
	// Add material if exists
	if materialID.Valid {
		material.ID = int(materialID.Int64)
		material.Name = materialName.String
		material.CreatedAt = materialCreatedAt.String
		material.UpdatedAt = materialUpdatedAt.String
		product.Material = &material
	}
	
	// Add category if exists
	if categoryID.Valid {
		category.ID = int(categoryID.Int64)
		category.Name = categoryName.String
		category.Slug = categorySlug.String
		if categoryImageID.Valid {
			imageID := int(categoryImageID.Int64)
			category.ImageID = &imageID
		}
		category.Active = categoryActive.Bool
		category.ChartOnly = categoryChartOnly.Bool
		category.CreatedAt = categoryCreatedAt.String
		category.UpdatedAt = categoryUpdatedAt.String
		product.Category = &category
	}
	
	// Get product images
	images, err := q.getProductImages(product.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product images: %w", err)
	}
	product.Images = images
	
	// Get product services
	services, err := q.getProductServices(product.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product services: %w", err)
	}
	product.AdditionalServices = services
	
	return &product, nil
}

func (q *ProductQueries) UpdateProduct(id int, product *models.Product) error {
	query := `
		UPDATE products 
		SET name = $1, short_description = $2, description = $3, material_id = $4, main_image_id = $5, category_id = $6
		WHERE id = $7
		RETURNING updated_at
	`
	
	err := q.db.QueryRow(query, product.Name, product.ShortDescription, product.Description,
		product.MaterialID, product.MainImageID, product.CategoryID, id).Scan(&product.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("product not found")
		}
		return fmt.Errorf("failed to update product: %w", err)
	}
	
	return nil
}

func (q *ProductQueries) DeleteProduct(id int) error {
	// Delete product (this will cascade delete images and services associations)
	result, err := q.db.Exec("DELETE FROM products WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("product not found")
	}
	
	return nil
}

// ManyToMany operations for product images
func (q *ProductQueries) ReplaceImages(productID int, imageIDs []int) error {
	tx, err := q.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Delete existing associations
	_, err = tx.Exec("DELETE FROM product_images WHERE product_id = $1", productID)
	if err != nil {
		return fmt.Errorf("failed to delete existing image associations: %w", err)
	}
	
	// Add new associations
	for _, imageID := range imageIDs {
		_, err = tx.Exec("INSERT INTO product_images (product_id, image_id) VALUES ($1, $2)", productID, imageID)
		if err != nil {
			return fmt.Errorf("failed to add image association: %w", err)
		}
	}
	
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// ManyToMany operations for product services
func (q *ProductQueries) ReplaceServices(productID int, serviceIDs []int) error {
	tx, err := q.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Delete existing associations
	_, err = tx.Exec("DELETE FROM product_services WHERE product_id = $1", productID)
	if err != nil {
		return fmt.Errorf("failed to delete existing service associations: %w", err)
	}
	
	// Add new associations
	for _, serviceID := range serviceIDs {
		_, err = tx.Exec("INSERT INTO product_services (product_id, additional_service_id) VALUES ($1, $2)", productID, serviceID)
		if err != nil {
			return fmt.Errorf("failed to add service association: %w", err)
		}
	}
	
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// GetPublicProducts returns products for public access with filtering and pagination
func (q *ProductQueries) GetPublicProducts(page, limit int, search string, categoryIDs []int) ([]models.ProductWithRelations, error) {
	offset := (page - 1) * limit
	
	whereClause := "WHERE (c.active = true OR c.id IS NULL)"
	args := []interface{}{}
	argCount := 0
	
	if search != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND (p.name ILIKE $%d OR p.short_description ILIKE $%d OR p.description ILIKE $%d)", argCount, argCount, argCount)
		args = append(args, "%"+search+"%")
	}
	
	if len(categoryIDs) > 0 {
		argCount++
		whereClause += fmt.Sprintf(" AND p.category_id = ANY($%d)", argCount)
		args = append(args, pq.Array(categoryIDs))
	}
	
	// Get paginated results with all relations
	argCount++
	limitArg := argCount
	argCount++
	offsetArg := argCount
	
	query := fmt.Sprintf(`
		SELECT 
			p.id, p.name, p.short_description, p.description, p.material_id, p.main_image_id, p.category_id, p.created_at, p.updated_at,
			mi.id, mi.filename, mi.original_name, mi.path, mi.size_bytes, mi.mime_type, mi.uploaded_by, mi.created_at, mi.updated_at,
			m.id, m.name, m.created_at, m.updated_at,
			c.id, c.name, c.slug, c.image_id, c.active, c.chart_only, c.created_at, c.updated_at,
			COALESCE(MIN(s.base_price), 0) as min_price
		FROM products p
		JOIN images mi ON p.main_image_id = mi.id
		LEFT JOIN materials m ON p.material_id = m.id
		LEFT JOIN categories c ON p.category_id = c.id
		LEFT JOIN sizes s ON p.id = s.product_id
		%s
		GROUP BY p.id, p.name, p.short_description, p.description, p.material_id, p.main_image_id, p.category_id, p.created_at, p.updated_at,
			mi.id, mi.filename, mi.original_name, mi.path, mi.size_bytes, mi.mime_type, mi.uploaded_by, mi.created_at, mi.updated_at,
			m.id, m.name, m.created_at, m.updated_at,
			c.id, c.name, c.slug, c.image_id, c.active, c.chart_only, c.created_at, c.updated_at
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, limitArg, offsetArg)
	
	args = append(args, limit, offset)
	
	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get public products - query: %s, args: %v, error: %w", query, args, err)
	}
	defer rows.Close()
	
	var products []models.ProductWithRelations
	
	for rows.Next() {
		var product models.ProductWithRelations
		var mainImage models.ImageResponse
		var material models.MaterialResponse
		var category models.CategoryResponse
		var materialID, categoryID sql.NullInt64
		var materialName, materialCreatedAt, materialUpdatedAt sql.NullString
		var categoryName, categorySlug, categoryCreatedAt, categoryUpdatedAt sql.NullString
		var categoryImageID sql.NullInt64
		var categoryActive, categoryChartOnly sql.NullBool
		var minPrice sql.NullFloat64
		
		err := rows.Scan(
			&product.ID, &product.Name, &product.ShortDescription, &product.Description,
			&product.MaterialID, &product.MainImageID, &product.CategoryID, &product.CreatedAt, &product.UpdatedAt,
			&mainImage.ID, &mainImage.Filename, &mainImage.OriginalName, &mainImage.Path,
			&mainImage.SizeBytes, &mainImage.MimeType, &mainImage.UploadedBy, &mainImage.CreatedAt, &mainImage.UpdatedAt,
			&materialID, &materialName, &materialCreatedAt, &materialUpdatedAt,
			&categoryID, &categoryName, &categorySlug, &categoryImageID, &categoryActive, &categoryChartOnly, &categoryCreatedAt, &categoryUpdatedAt,
			&minPrice,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		
		product.MainImage = mainImage
		
		// Handle optional material
		if materialID.Valid {
			material.ID = int(materialID.Int64)
			material.Name = materialName.String
			material.CreatedAt = materialCreatedAt.String
			material.UpdatedAt = materialUpdatedAt.String
			product.Material = &material
		}
		
		// Handle optional category
		if categoryID.Valid {
			category.ID = int(categoryID.Int64)
			category.Name = categoryName.String
			category.Slug = categorySlug.String
			if categoryImageID.Valid {
				imageID := int(categoryImageID.Int64)
				category.ImageID = &imageID
			}
			category.Active = categoryActive.Bool
			category.ChartOnly = categoryChartOnly.Bool
			category.CreatedAt = categoryCreatedAt.String
			category.UpdatedAt = categoryUpdatedAt.String
			product.Category = &category
		}
		
		// Set minimum price
		if minPrice.Valid {
			product.MinPrice = minPrice.Float64
		}
		
		// Get product images
		images, err := q.getProductImages(product.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get product images: %w", err)
		}
		product.Images = images
		
		// Get additional services
		services, err := q.getProductServices(product.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get product services: %w", err)
		}
		product.AdditionalServices = services
		
		products = append(products, product)
	}
	
	return products, nil
}

// GetPublicProductsCount returns the count of products for public access with filtering
func (q *ProductQueries) GetPublicProductsCount(search string, categoryIDs []int) (int, error) {
	whereClause := "WHERE (c.active = true OR c.id IS NULL)"
	args := []interface{}{}
	argCount := 0
	
	if search != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND (p.name ILIKE $%d OR p.short_description ILIKE $%d OR p.description ILIKE $%d)", argCount, argCount, argCount)
		args = append(args, "%"+search+"%")
	}
	
	if len(categoryIDs) > 0 {
		argCount++
		whereClause += fmt.Sprintf(" AND p.category_id = ANY($%d)", argCount)
		args = append(args, pq.Array(categoryIDs))
	}
	
	query := fmt.Sprintf(`
		SELECT COUNT(DISTINCT p.id)
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		%s
	`, whereClause)
	
	var count int
	err := q.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count public products: %w", err)
	}
	
	return count, nil
}

// GetProductVariants returns all variants for a specific product
func (q *ProductQueries) GetProductVariants(productID int) ([]models.ProductVariantResponse, error) {
	variantQueries := NewProductVariantQueries(q.db)
	variants, _, err := variantQueries.ListProductVariants(1, 1000, "", &productID, nil)
	return variants, err
}

// GetProductSizes returns all sizes for a specific product
func (q *ProductQueries) GetProductSizes(productID int) ([]models.SizeResponse, error) {
	query := `
		SELECT s.id, s.name, s.a, s.b, s.c, s.d, s.e, s.f, 
			   s.base_price, s.product_id, s.created_at, s.updated_at
		FROM sizes s
		WHERE s.product_id = $1
		ORDER BY s.base_price ASC
	`
	
	rows, err := q.db.Query(query, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product sizes: %w", err)
	}
	defer rows.Close()
	
	var sizes []models.SizeResponse
	for rows.Next() {
		var size models.SizeResponse
		var createdAt, updatedAt time.Time
		err := rows.Scan(
			&size.ID, &size.Name, &size.A, &size.B, &size.C, 
			&size.D, &size.E, &size.F, &size.BasePrice,
			&size.ProductID, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan size: %w", err)
		}
		
		size.CreatedAt = createdAt.Format(time.RFC3339)
		size.UpdatedAt = updatedAt.Format(time.RFC3339)
		sizes = append(sizes, size)
	}
	
	return sizes, nil
}

// Size queries
type SizeQueries struct {
	db *sql.DB
}

func NewSizeQueries(db *sql.DB) *SizeQueries {
	return &SizeQueries{db: db}
}

func (q *SizeQueries) CreateSize(size *models.Size) error {
	query := `
		INSERT INTO sizes (name, product_id, base_price, a, b, c, d, e, f)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`
	
	err := q.db.QueryRow(query, size.Name, size.ProductID, size.BasePrice, 
		size.A, size.B, size.C, size.D, size.E, size.F).Scan(&size.ID, &size.CreatedAt, &size.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create size: %w", err)
	}
	
	return nil
}

func (q *SizeQueries) GetSizeByID(id int) (*models.SizeWithProduct, error) {
	query := `
		SELECT s.id, s.name, s.product_id, s.base_price, s.a, s.b, s.c, s.d, s.e, s.f, s.created_at, s.updated_at,
			   p.id, p.name, p.short_description, p.description, p.material_id, p.main_image_id, p.category_id, p.created_at, p.updated_at
		FROM sizes s
		JOIN products p ON s.product_id = p.id
		WHERE s.id = $1
	`
	
	var size models.SizeWithProduct
	var product models.Product
	
	err := q.db.QueryRow(query, id).Scan(
		&size.ID, &size.Name, &size.ProductID, &size.BasePrice, &size.A, &size.B, &size.C, &size.D, &size.E, &size.F, &size.CreatedAt, &size.UpdatedAt,
		&product.ID, &product.Name, &product.ShortDescription, &product.Description, &product.MaterialID, &product.MainImageID, &product.CategoryID, &product.CreatedAt, &product.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("size not found")
		}
		return nil, fmt.Errorf("failed to get size: %w", err)
	}
	
	// Convert product to response format
	size.Product = models.ProductResponse{
		ID:               product.ID,
		Name:             product.Name,
		ShortDescription: product.ShortDescription,
		Description:      product.Description,
		MaterialID:       product.MaterialID,
		MainImageID:      product.MainImageID,
		CategoryID:       product.CategoryID,
		CreatedAt:        product.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
	}
	
	return &size, nil
}

func (q *SizeQueries) ListSizes(page, limit int, search string, productID *int) ([]models.SizeResponse, int, error) {
	offset := (page - 1) * limit
	
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1
	
	if search != "" {
		whereClause += fmt.Sprintf(" AND s.name ILIKE $%d", argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}
	
	if productID != nil {
		whereClause += fmt.Sprintf(" AND s.product_id = $%d", argIndex)
		args = append(args, *productID)
		argIndex++
	}
	
	// Count total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM sizes s
		JOIN products p ON s.product_id = p.id
		%s
	`, whereClause)
	
	var total int
	err := q.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count sizes: %w", err)
	}
	
	// Get sizes
	query := fmt.Sprintf(`
		SELECT s.id, s.name, s.product_id, s.base_price, s.a, s.b, s.c, s.d, s.e, s.f, s.created_at, s.updated_at,
			   p.id, p.name, p.short_description, p.description, p.material_id, p.main_image_id, p.category_id, p.created_at, p.updated_at
		FROM sizes s
		JOIN products p ON s.product_id = p.id
		%s
		ORDER BY s.name
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)
	
	args = append(args, limit, offset)
	
	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query sizes: %w", err)
	}
	defer rows.Close()
	
	var sizes []models.SizeResponse
	for rows.Next() {
		var size models.SizeResponse
		var product models.Product
		
		err := rows.Scan(
			&size.ID, &size.Name, &size.ProductID, &size.BasePrice, &size.A, &size.B, &size.C, &size.D, &size.E, &size.F, &size.CreatedAt, &size.UpdatedAt,
			&product.ID, &product.Name, &product.ShortDescription, &product.Description, &product.MaterialID, &product.MainImageID, &product.CategoryID, &product.CreatedAt, &product.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan size: %w", err)
		}
		
		size.CreatedAt = size.CreatedAt[:19] // Format timestamp
		size.UpdatedAt = size.UpdatedAt[:19]
		
		size.Product = models.ProductResponse{
			ID:               product.ID,
			Name:             product.Name,
			ShortDescription: product.ShortDescription,
			Description:      product.Description,
			MaterialID:       product.MaterialID,
			MainImageID:      product.MainImageID,
			CategoryID:       product.CategoryID,
			CreatedAt:        product.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
		}
		
		sizes = append(sizes, size)
	}
	
	return sizes, total, nil
}

func (q *SizeQueries) UpdateSize(id int, size *models.Size) error {
	query := `
		UPDATE sizes 
		SET name = $1, product_id = $2, base_price = $3, a = $4, b = $5, c = $6, d = $7, e = $8, f = $9
		WHERE id = $10
		RETURNING updated_at
	`
	
	err := q.db.QueryRow(query, size.Name, size.ProductID, size.BasePrice,
		size.A, size.B, size.C, size.D, size.E, size.F, id).Scan(&size.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("size not found")
		}
		return fmt.Errorf("failed to update size: %w", err)
	}
	
	return nil
}

func (q *SizeQueries) DeleteSize(id int) error {
	query := `DELETE FROM sizes WHERE id = $1`
	
	result, err := q.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete size: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("size not found")
	}
	
	return nil
}


// ProductVariant queries
type ProductVariantQueries struct {
	db *sql.DB
}

func NewProductVariantQueries(db *sql.DB) *ProductVariantQueries {
	return &ProductVariantQueries{db: db}
}

func (q *ProductVariantQueries) CreateProductVariant(variant *models.ProductVariant) error {
	// If this variant is being set as default, clear other defaults for this product
	if variant.IsDefault {
		if err := q.ensureOnlyOneDefaultVariant(variant.ProductID, nil); err != nil {
			return err
		}
	}
	
	query := `
		INSERT INTO product_variants (product_id, name, color_id, is_default)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	
	err := q.db.QueryRow(query, variant.ProductID, variant.Name, variant.ColorID, variant.IsDefault).Scan(&variant.ID, &variant.CreatedAt, &variant.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create product variant: %w", err)
	}
	
	return nil
}

func (q *ProductVariantQueries) GetProductVariantByID(id int) (*models.ProductVariantWithRelations, error) {
	query := `
		SELECT pv.id, pv.product_id, pv.name, pv.color_id, pv.is_default, pv.created_at, pv.updated_at,
			   p.id, p.name, p.short_description, p.description, p.material_id, p.main_image_id, p.category_id, p.created_at, p.updated_at,
			   c.id, c.name, c.custom, c.material_id, c.created_at, c.updated_at
		FROM product_variants pv
		JOIN products p ON pv.product_id = p.id
		JOIN colors c ON pv.color_id = c.id
		WHERE pv.id = $1
	`
	
	var variant models.ProductVariantWithRelations
	var product models.Product
	var color models.Color
	
	err := q.db.QueryRow(query, id).Scan(
		&variant.ID, &variant.ProductID, &variant.Name, &variant.ColorID, &variant.IsDefault, &variant.CreatedAt, &variant.UpdatedAt,
		&product.ID, &product.Name, &product.ShortDescription, &product.Description, &product.MaterialID, &product.MainImageID, &product.CategoryID, &product.CreatedAt, &product.UpdatedAt,
		&color.ID, &color.Name, &color.Custom, &color.MaterialID, &color.CreatedAt, &color.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product variant not found")
		}
		return nil, fmt.Errorf("failed to get product variant: %w", err)
	}
	
	// Convert to response format
	variant.Product = models.ProductResponse{
		ID:               product.ID,
		Name:             product.Name,
		ShortDescription: product.ShortDescription,
		Description:      product.Description,
		MaterialID:       product.MaterialID,
		MainImageID:      product.MainImageID,
		CategoryID:       product.CategoryID,
		CreatedAt:        product.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
	}
	
	variant.Color = models.ColorResponse{
		ID:         color.ID,
		Name:       color.Name,
		Custom:     color.Custom,
		MaterialID: color.MaterialID,
		CreatedAt:  color.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  color.UpdatedAt.Format(time.RFC3339),
	}
	
	// Get variant images
	images, err := q.getProductVariantImages(variant.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get variant images: %w", err)
	}
	variant.Images = images
	
	return &variant, nil
}

func (q *ProductVariantQueries) UpdateProductVariantImages(variantID int, imageIDs []int) error {
	tx, err := q.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Delete existing associations
	_, err = tx.Exec("DELETE FROM product_variant_images WHERE product_variant_id = $1", variantID)
	if err != nil {
		return fmt.Errorf("failed to delete existing image associations: %w", err)
	}
	
	// Add new associations
	for _, imageID := range imageIDs {
		_, err = tx.Exec("INSERT INTO product_variant_images (product_variant_id, image_id) VALUES ($1, $2)", variantID, imageID)
		if err != nil {
			return fmt.Errorf("failed to add image association: %w", err)
		}
	}
	
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}


func (q *ProductVariantQueries) ListProductVariants(page, limit int, search string, productID, colorID *int) ([]models.ProductVariantResponse, int, error) {
	offset := (page - 1) * limit
	
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1
	
	if search != "" {
		whereClause += fmt.Sprintf(" AND pv.name ILIKE $%d", argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}
	
	if productID != nil {
		whereClause += fmt.Sprintf(" AND pv.product_id = $%d", argIndex)
		args = append(args, *productID)
		argIndex++
	}
	
	if colorID != nil {
		whereClause += fmt.Sprintf(" AND pv.color_id = $%d", argIndex)
		args = append(args, *colorID)
		argIndex++
	}
	
	// Count total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM product_variants pv
		JOIN products p ON pv.product_id = p.id
		JOIN colors c ON pv.color_id = c.id
		%s
	`, whereClause)
	
	var total int
	err := q.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count product variants: %w", err)
	}
	
	// Get variants
	query := fmt.Sprintf(`
		SELECT pv.id, pv.product_id, pv.name, pv.color_id, pv.is_default, pv.created_at, pv.updated_at,
			   p.id, p.name, p.short_description, p.description, p.material_id, p.main_image_id, p.category_id, p.created_at, p.updated_at,
			   c.id, c.name, c.custom, c.material_id, c.created_at, c.updated_at
		FROM product_variants pv
		JOIN products p ON pv.product_id = p.id
		JOIN colors c ON pv.color_id = c.id
		%s
		ORDER BY pv.product_id, pv.is_default DESC, pv.name
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)
	
	args = append(args, limit, offset)
	
	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query product variants: %w", err)
	}
	defer rows.Close()
	
	var variants []models.ProductVariantResponse
	for rows.Next() {
		var variant models.ProductVariantResponse
		var product models.Product
		var color models.Color
		
		err := rows.Scan(
			&variant.ID, &variant.ProductID, &variant.Name, &variant.ColorID, &variant.IsDefault, &variant.CreatedAt, &variant.UpdatedAt,
			&product.ID, &product.Name, &product.ShortDescription, &product.Description, &product.MaterialID, &product.MainImageID, &product.CategoryID, &product.CreatedAt, &product.UpdatedAt,
			&color.ID, &color.Name, &color.Custom, &color.MaterialID, &color.CreatedAt, &color.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan product variant: %w", err)
		}
		
		variant.CreatedAt = variant.CreatedAt[:19] // Format timestamp
		variant.UpdatedAt = variant.UpdatedAt[:19]
		
		variant.Product = models.ProductResponse{
			ID:               product.ID,
			Name:             product.Name,
			ShortDescription: product.ShortDescription,
			Description:      product.Description,
			MaterialID:       product.MaterialID,
			MainImageID:      product.MainImageID,
			CategoryID:       product.CategoryID,
			CreatedAt:        product.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
		}
		
		variant.Color = models.ColorResponse{
			ID:         color.ID,
			Name:       color.Name,
			Custom:     color.Custom,
			MaterialID: color.MaterialID,
			CreatedAt:  color.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  color.UpdatedAt.Format(time.RFC3339),
		}
		
		// Get variant images
		images, err := q.getProductVariantImages(variant.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get variant images: %w", err)
		}
		variant.Images = images
		
		variants = append(variants, variant)
	}
	
	return variants, total, nil
}

func (q *ProductVariantQueries) UpdateProductVariant(id int, variant *models.ProductVariant) error {
	// If this variant is being set as default, clear other defaults for this product
	if variant.IsDefault {
		if err := q.ensureOnlyOneDefaultVariant(variant.ProductID, &id); err != nil {
			return err
		}
	}
	
	query := `
		UPDATE product_variants 
		SET product_id = $1, name = $2, color_id = $3, is_default = $4
		WHERE id = $5
		RETURNING updated_at
	`
	
	err := q.db.QueryRow(query, variant.ProductID, variant.Name, variant.ColorID, variant.IsDefault, id).Scan(&variant.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("product variant not found")
		}
		return fmt.Errorf("failed to update product variant: %w", err)
	}
	
	return nil
}

func (q *ProductVariantQueries) DeleteProductVariant(id int) error {
	query := `DELETE FROM product_variants WHERE id = $1`
	
	result, err := q.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete product variant: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("product variant not found")
	}
	
	return nil
}

func (q *ProductVariantQueries) getProductVariantImages(variantID int) ([]models.ImageResponse, error) {
	query := `
		SELECT i.id, i.filename, i.original_name, i.path, i.size_bytes, i.mime_type, i.uploaded_by, i.created_at, i.updated_at
		FROM product_variant_images pvi
		JOIN images i ON pvi.image_id = i.id
		WHERE pvi.product_variant_id = $1
		ORDER BY i.id
	`
	
	rows, err := q.db.Query(query, variantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query variant images: %w", err)
	}
	defer rows.Close()
	
	var images []models.ImageResponse
	for rows.Next() {
		var image models.ImageResponse
		
		err := rows.Scan(&image.ID, &image.Filename, &image.OriginalName, &image.Path, &image.SizeBytes, &image.MimeType, &image.UploadedBy, &image.CreatedAt, &image.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan image: %w", err)
		}
		
		image.CreatedAt = image.CreatedAt[:19] // Format timestamp
		image.UpdatedAt = image.UpdatedAt[:19]
		
		images = append(images, image)
	}
	
	return images, nil
}

// Helper method to ensure only one default variant per product
func (q *ProductVariantQueries) ensureOnlyOneDefaultVariant(productID int, excludeVariantID *int) error {
	query := `UPDATE product_variants SET is_default = FALSE WHERE product_id = $1`
	args := []interface{}{productID}
	
	if excludeVariantID != nil {
		query += ` AND id != $2`
		args = append(args, *excludeVariantID)
	}
	
	_, err := q.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to clear default variants: %w", err)
	}
	
	return nil
}
