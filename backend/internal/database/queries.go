package database

import (
	"database/sql"
	"fmt"
	"time"
	"notsofluffy-backend/internal/auth"
	"notsofluffy-backend/internal/models"
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