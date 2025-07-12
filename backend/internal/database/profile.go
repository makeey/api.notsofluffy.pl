package database

import (
	"database/sql"
	"fmt"

	"notsofluffy-backend/internal/models"
)

type ProfileQueries struct {
	db *sql.DB
}

func NewProfileQueries(db *sql.DB) *ProfileQueries {
	return &ProfileQueries{db: db}
}

// CreateUserProfile creates a user profile (called on user registration)
func (q *ProfileQueries) CreateUserProfile(userID int) (*models.UserProfile, error) {
	query := `
		INSERT INTO user_profiles (user_id)
		VALUES ($1)
		RETURNING id, user_id, first_name, last_name, phone, created_at, updated_at`
	
	var profile models.UserProfile
	err := q.db.QueryRow(query, userID).Scan(
		&profile.ID, &profile.UserID, &profile.FirstName, &profile.LastName, 
		&profile.Phone, &profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create user profile: %w", err)
	}
	
	return &profile, nil
}

// GetUserProfile retrieves a user's profile with addresses
func (q *ProfileQueries) GetUserProfile(userID int) (*models.UserProfileResponse, error) {
	// Get profile
	profileQuery := `
		SELECT id, user_id, first_name, last_name, phone, created_at, updated_at
		FROM user_profiles
		WHERE user_id = $1`
	
	var profile models.UserProfile
	err := q.db.QueryRow(profileQuery, userID).Scan(
		&profile.ID, &profile.UserID, &profile.FirstName, &profile.LastName,
		&profile.Phone, &profile.CreatedAt, &profile.UpdatedAt)
	if err == sql.ErrNoRows {
		// Create profile if it doesn't exist (for existing users)
		createdProfile, err := q.CreateUserProfile(userID)
		if err != nil {
			return nil, err
		}
		profile = *createdProfile
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}
	
	// Get addresses
	addresses, err := q.GetUserAddresses(userID)
	if err != nil {
		return nil, err
	}
	
	response := &models.UserProfileResponse{
		ID:        profile.ID,
		UserID:    profile.UserID,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
		Phone:     profile.Phone,
		CreatedAt: profile.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: profile.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Addresses: addresses,
	}
	
	return response, nil
}

// UpdateUserProfile updates a user's profile information
func (q *ProfileQueries) UpdateUserProfile(userID int, req *models.UserProfileRequest) (*models.UserProfileResponse, error) {
	query := `
		UPDATE user_profiles
		SET first_name = $2, last_name = $3, phone = $4
		WHERE user_id = $1
		RETURNING id, user_id, first_name, last_name, phone, created_at, updated_at`
	
	var profile models.UserProfile
	err := q.db.QueryRow(query, userID, req.FirstName, req.LastName, req.Phone).Scan(
		&profile.ID, &profile.UserID, &profile.FirstName, &profile.LastName,
		&profile.Phone, &profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}
	
	// Get addresses
	addresses, err := q.GetUserAddresses(userID)
	if err != nil {
		return nil, err
	}
	
	response := &models.UserProfileResponse{
		ID:        profile.ID,
		UserID:    profile.UserID,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
		Phone:     profile.Phone,
		CreatedAt: profile.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: profile.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Addresses: addresses,
	}
	
	return response, nil
}

// GetUserAddresses retrieves all addresses for a user
func (q *ProfileQueries) GetUserAddresses(userID int) ([]models.UserAddressResponse, error) {
	query := `
		SELECT id, user_id, label, first_name, last_name, company, address_line1, address_line2,
		       city, state_province, postal_code, country, phone, is_default, created_at, updated_at
		FROM user_addresses
		WHERE user_id = $1
		ORDER BY is_default DESC, created_at DESC`
	
	rows, err := q.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user addresses: %w", err)
	}
	defer rows.Close()
	
	var addresses []models.UserAddressResponse
	for rows.Next() {
		var addr models.UserAddress
		err := rows.Scan(&addr.ID, &addr.UserID, &addr.Label, &addr.FirstName, &addr.LastName,
			&addr.Company, &addr.AddressLine1, &addr.AddressLine2, &addr.City, &addr.StateProvince,
			&addr.PostalCode, &addr.Country, &addr.Phone, &addr.IsDefault, &addr.CreatedAt, &addr.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan address: %w", err)
		}
		
		addresses = append(addresses, models.UserAddressResponse{
			ID:           addr.ID,
			UserID:       addr.UserID,
			Label:        addr.Label,
			FirstName:    addr.FirstName,
			LastName:     addr.LastName,
			Company:      addr.Company,
			AddressLine1: addr.AddressLine1,
			AddressLine2: addr.AddressLine2,
			City:         addr.City,
			StateProvince: addr.StateProvince,
			PostalCode:   addr.PostalCode,
			Country:      addr.Country,
			Phone:        addr.Phone,
			IsDefault:    addr.IsDefault,
			CreatedAt:    addr.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:    addr.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	
	return addresses, nil
}

// CreateUserAddress creates a new address for a user
func (q *ProfileQueries) CreateUserAddress(userID int, req *models.UserAddressRequest) (*models.UserAddressResponse, error) {
	tx, err := q.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// If this is set as default, unset all other defaults for this user
	if req.IsDefault {
		_, err = tx.Exec("UPDATE user_addresses SET is_default = FALSE WHERE user_id = $1", userID)
		if err != nil {
			return nil, fmt.Errorf("failed to unset default addresses: %w", err)
		}
	}
	
	query := `
		INSERT INTO user_addresses (user_id, label, first_name, last_name, company, address_line1, address_line2,
		                           city, state_province, postal_code, country, phone, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, user_id, label, first_name, last_name, company, address_line1, address_line2,
		          city, state_province, postal_code, country, phone, is_default, created_at, updated_at`
	
	var addr models.UserAddress
	err = tx.QueryRow(query, userID, req.Label, req.FirstName, req.LastName, req.Company,
		req.AddressLine1, req.AddressLine2, req.City, req.StateProvince, req.PostalCode,
		req.Country, req.Phone, req.IsDefault).Scan(
		&addr.ID, &addr.UserID, &addr.Label, &addr.FirstName, &addr.LastName,
		&addr.Company, &addr.AddressLine1, &addr.AddressLine2, &addr.City, &addr.StateProvince,
		&addr.PostalCode, &addr.Country, &addr.Phone, &addr.IsDefault, &addr.CreatedAt, &addr.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create address: %w", err)
	}
	
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return &models.UserAddressResponse{
		ID:           addr.ID,
		UserID:       addr.UserID,
		Label:        addr.Label,
		FirstName:    addr.FirstName,
		LastName:     addr.LastName,
		Company:      addr.Company,
		AddressLine1: addr.AddressLine1,
		AddressLine2: addr.AddressLine2,
		City:         addr.City,
		StateProvince: addr.StateProvince,
		PostalCode:   addr.PostalCode,
		Country:      addr.Country,
		Phone:        addr.Phone,
		IsDefault:    addr.IsDefault,
		CreatedAt:    addr.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    addr.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// UpdateUserAddress updates an existing address
func (q *ProfileQueries) UpdateUserAddress(userID, addressID int, req *models.UserAddressRequest) (*models.UserAddressResponse, error) {
	tx, err := q.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// If this is set as default, unset all other defaults for this user
	if req.IsDefault {
		_, err = tx.Exec("UPDATE user_addresses SET is_default = FALSE WHERE user_id = $1 AND id != $2", userID, addressID)
		if err != nil {
			return nil, fmt.Errorf("failed to unset default addresses: %w", err)
		}
	}
	
	query := `
		UPDATE user_addresses
		SET label = $3, first_name = $4, last_name = $5, company = $6, address_line1 = $7, address_line2 = $8,
		    city = $9, state_province = $10, postal_code = $11, country = $12, phone = $13, is_default = $14
		WHERE user_id = $1 AND id = $2
		RETURNING id, user_id, label, first_name, last_name, company, address_line1, address_line2,
		          city, state_province, postal_code, country, phone, is_default, created_at, updated_at`
	
	var addr models.UserAddress
	err = tx.QueryRow(query, userID, addressID, req.Label, req.FirstName, req.LastName, req.Company,
		req.AddressLine1, req.AddressLine2, req.City, req.StateProvince, req.PostalCode,
		req.Country, req.Phone, req.IsDefault).Scan(
		&addr.ID, &addr.UserID, &addr.Label, &addr.FirstName, &addr.LastName,
		&addr.Company, &addr.AddressLine1, &addr.AddressLine2, &addr.City, &addr.StateProvince,
		&addr.PostalCode, &addr.Country, &addr.Phone, &addr.IsDefault, &addr.CreatedAt, &addr.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("address not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to update address: %w", err)
	}
	
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return &models.UserAddressResponse{
		ID:           addr.ID,
		UserID:       addr.UserID,
		Label:        addr.Label,
		FirstName:    addr.FirstName,
		LastName:     addr.LastName,
		Company:      addr.Company,
		AddressLine1: addr.AddressLine1,
		AddressLine2: addr.AddressLine2,
		City:         addr.City,
		StateProvince: addr.StateProvince,
		PostalCode:   addr.PostalCode,
		Country:      addr.Country,
		Phone:        addr.Phone,
		IsDefault:    addr.IsDefault,
		CreatedAt:    addr.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    addr.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// DeleteUserAddress deletes an address
func (q *ProfileQueries) DeleteUserAddress(userID, addressID int) error {
	query := `DELETE FROM user_addresses WHERE user_id = $1 AND id = $2`
	result, err := q.db.Exec(query, userID, addressID)
	if err != nil {
		return fmt.Errorf("failed to delete address: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("address not found")
	}
	
	return nil
}

// SetDefaultAddress sets an address as default (and unsets others)
func (q *ProfileQueries) SetDefaultAddress(userID, addressID int) error {
	tx, err := q.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Unset all defaults for this user
	_, err = tx.Exec("UPDATE user_addresses SET is_default = FALSE WHERE user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("failed to unset default addresses: %w", err)
	}
	
	// Set the specified address as default
	result, err := tx.Exec("UPDATE user_addresses SET is_default = TRUE WHERE user_id = $1 AND id = $2", userID, addressID)
	if err != nil {
		return fmt.Errorf("failed to set default address: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("address not found")
	}
	
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}