package database

import (
	"database/sql"
	"fmt"
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