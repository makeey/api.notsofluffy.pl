package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"notsofluffy-backend/internal/models"
)

type DiscountQueries struct {
	db *sql.DB
}

func NewDiscountQueries(db *sql.DB) *DiscountQueries {
	return &DiscountQueries{db: db}
}

// ValidateDiscountCode validates a discount code and returns validation result
func (q *DiscountQueries) ValidateDiscountCode(code string, cartTotal float64, userID *int, sessionID string) (*models.DiscountValidationResult, error) {
	// Get discount code
	discountCode, err := q.GetDiscountCodeByCode(code)
	if err != nil {
		if err.Error() == "discount code not found" {
			return &models.DiscountValidationResult{
				IsValid:      false,
				ErrorMessage: "Invalid discount code",
			}, nil
		}
		return nil, fmt.Errorf("failed to get discount code: %w", err)
	}

	// Check if code is active
	if !discountCode.Active {
		return &models.DiscountValidationResult{
			IsValid:      false,
			ErrorMessage: "Discount code is not active",
		}, nil
	}

	// Check date validity
	now := time.Now()
	if now.Before(discountCode.StartDate) {
		return &models.DiscountValidationResult{
			IsValid:      false,
			ErrorMessage: "Discount code is not yet valid",
		}, nil
	}

	if discountCode.EndDate != nil && now.After(*discountCode.EndDate) {
		return &models.DiscountValidationResult{
			IsValid:      false,
			ErrorMessage: "Discount code has expired",
		}, nil
	}

	// Check minimum order amount
	if cartTotal < discountCode.MinOrderAmount {
		return &models.DiscountValidationResult{
			IsValid:      false,
			ErrorMessage: fmt.Sprintf("Minimum order amount of %.2f required", discountCode.MinOrderAmount),
		}, nil
	}

	// Check usage limits
	usageValid, err := q.validateUsageLimits(discountCode, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate usage limits: %w", err)
	}
	if !usageValid.IsValid {
		return usageValid, nil
	}

	// Calculate discount amount
	var discountAmount float64
	if discountCode.DiscountType == models.DiscountTypePercentage {
		discountAmount = cartTotal * (discountCode.DiscountValue / 100)
	} else {
		discountAmount = discountCode.DiscountValue
		// Don't discount more than the cart total
		if discountAmount > cartTotal {
			discountAmount = cartTotal
		}
	}

	return &models.DiscountValidationResult{
		IsValid:        true,
		DiscountAmount: discountAmount,
		DiscountCode:   discountCode,
	}, nil
}

// validateUsageLimits checks if the discount code can be used based on usage type
func (q *DiscountQueries) validateUsageLimits(discountCode *models.DiscountCode, userID *int, sessionID string) (*models.DiscountValidationResult, error) {
	switch discountCode.UsageType {
	case models.UsageTypeOneTime:
		// Check if the code has been used at all
		if discountCode.UsedCount > 0 {
			return &models.DiscountValidationResult{
				IsValid:      false,
				ErrorMessage: "Discount code has already been used",
			}, nil
		}

	case models.UsageTypeOncePerUser:
		// Check if this user has used the code before (completed orders)
		if userID != nil {
			hasUsed, err := q.hasUserUsedCode(discountCode.ID, *userID)
			if err != nil {
				return nil, fmt.Errorf("failed to check user usage: %w", err)
			}
			if hasUsed {
				return &models.DiscountValidationResult{
					IsValid:      false,
					ErrorMessage: "You have already used this discount code",
				}, nil
			}

			// Also check if the code is currently applied to any active cart sessions for this user
			// (including the current session)
			hasActiveCart, err := q.hasUserActiveCartWithCodeIncludingCurrent(discountCode.ID, *userID)
			if err != nil {
				return nil, fmt.Errorf("failed to check active cart usage: %w", err)
			}
			if hasActiveCart {
				return &models.DiscountValidationResult{
					IsValid:      false,
					ErrorMessage: "This discount code is already applied to your cart",
				}, nil
			}
		} else {
			// For guest users, check by session
			hasUsed, err := q.hasSessionUsedCode(discountCode.ID, sessionID)
			if err != nil {
				return nil, fmt.Errorf("failed to check session usage: %w", err)
			}
			if hasUsed {
				return &models.DiscountValidationResult{
					IsValid:      false,
					ErrorMessage: "This discount code has already been used",
				}, nil
			}

			// For guest users, also check if code is applied to any active sessions
			// (including the current session)
			hasActiveSession, err := q.hasActiveSessionWithCodeIncludingCurrent(discountCode.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to check active session usage: %w", err)
			}
			if hasActiveSession {
				return &models.DiscountValidationResult{
					IsValid:      false,
					ErrorMessage: "This discount code is already applied to a cart session",
				}, nil
			}
		}

	case models.UsageTypeUnlimited:
		// Check max uses if specified
		if discountCode.MaxUses != nil && discountCode.UsedCount >= *discountCode.MaxUses {
			return &models.DiscountValidationResult{
				IsValid:      false,
				ErrorMessage: "Discount code usage limit reached",
			}, nil
		}
	}

	return &models.DiscountValidationResult{IsValid: true}, nil
}

// hasUserUsedCode checks if a user has used a specific discount code
func (q *DiscountQueries) hasUserUsedCode(discountCodeID, userID int) (bool, error) {
	var count int
	err := q.db.QueryRow(
		"SELECT COUNT(*) FROM discount_code_usage WHERE discount_code_id = $1 AND user_id = $2",
		discountCodeID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check user usage: %w", err)
	}
	return count > 0, nil
}

// hasSessionUsedCode checks if a session has used a specific discount code
func (q *DiscountQueries) hasSessionUsedCode(discountCodeID int, sessionID string) (bool, error) {
	var count int
	err := q.db.QueryRow(
		"SELECT COUNT(*) FROM discount_code_usage WHERE discount_code_id = $1 AND session_id = $2",
		discountCodeID, sessionID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check session usage: %w", err)
	}
	return count > 0, nil
}

// hasUserActiveCartWithCode checks if a user has the discount code applied to any active cart sessions
// (excluding the current session)
func (q *DiscountQueries) hasUserActiveCartWithCode(discountCodeID, userID int, currentSessionID string) (bool, error) {
	var count int
	err := q.db.QueryRow(
		"SELECT COUNT(*) FROM cart_sessions WHERE applied_discount_code_id = $1 AND user_id = $2 AND session_id != $3",
		discountCodeID, userID, currentSessionID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check active cart usage: %w", err)
	}
	return count > 0, nil
}

// hasActiveSessionWithCode checks if any other session has the discount code applied
// (excluding the current session)
func (q *DiscountQueries) hasActiveSessionWithCode(discountCodeID int, currentSessionID string) (bool, error) {
	var count int
	err := q.db.QueryRow(
		"SELECT COUNT(*) FROM cart_sessions WHERE applied_discount_code_id = $1 AND session_id != $2",
		discountCodeID, currentSessionID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check active session usage: %w", err)
	}
	return count > 0, nil
}

// hasUserActiveCartWithCodeIncludingCurrent checks if a user has the discount code applied to any active cart sessions
// (including the current session - this prevents double application)
func (q *DiscountQueries) hasUserActiveCartWithCodeIncludingCurrent(discountCodeID, userID int) (bool, error) {
	var count int
	err := q.db.QueryRow(
		"SELECT COUNT(*) FROM cart_sessions WHERE applied_discount_code_id = $1 AND user_id = $2",
		discountCodeID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check active cart usage: %w", err)
	}
	return count > 0, nil
}

// hasActiveSessionWithCodeIncludingCurrent checks if any session has the discount code applied
// (including the current session - this prevents double application)
func (q *DiscountQueries) hasActiveSessionWithCodeIncludingCurrent(discountCodeID int) (bool, error) {
	var count int
	err := q.db.QueryRow(
		"SELECT COUNT(*) FROM cart_sessions WHERE applied_discount_code_id = $1",
		discountCodeID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check active session usage: %w", err)
	}
	return count > 0, nil
}

// ApplyDiscountToCartSession applies a discount to a cart session
func (q *DiscountQueries) ApplyDiscountToCartSession(cartSessionID, discountCodeID int, discountAmount float64) error {
	_, err := q.db.Exec(
		"UPDATE cart_sessions SET applied_discount_code_id = $1, discount_amount = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3",
		discountCodeID, discountAmount, cartSessionID,
	)
	if err != nil {
		return fmt.Errorf("failed to apply discount to cart session: %w", err)
	}
	return nil
}

// RemoveDiscountFromCartSession removes discount from a cart session
func (q *DiscountQueries) RemoveDiscountFromCartSession(cartSessionID int) error {
	_, err := q.db.Exec(
		"UPDATE cart_sessions SET applied_discount_code_id = NULL, discount_amount = 0, updated_at = CURRENT_TIMESTAMP WHERE id = $1",
		cartSessionID,
	)
	if err != nil {
		return fmt.Errorf("failed to remove discount from cart session: %w", err)
	}
	return nil
}

// RecordDiscountUsage records usage of a discount code
func (q *DiscountQueries) RecordDiscountUsage(discountCodeID int, userID *int, sessionID string, orderID *int) error {
	tx, err := q.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert usage record
	_, err = tx.Exec(
		"INSERT INTO discount_code_usage (discount_code_id, user_id, session_id, order_id) VALUES ($1, $2, $3, $4)",
		discountCodeID, userID, sessionID, orderID,
	)
	if err != nil {
		return fmt.Errorf("failed to record discount usage: %w", err)
	}

	// Increment used count
	_, err = tx.Exec(
		"UPDATE discount_codes SET used_count = used_count + 1, updated_at = CURRENT_TIMESTAMP WHERE id = $1",
		discountCodeID,
	)
	if err != nil {
		return fmt.Errorf("failed to increment usage count: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetDiscountCodeByCode gets a discount code by its code string
func (q *DiscountQueries) GetDiscountCodeByCode(code string) (*models.DiscountCode, error) {
	var dc models.DiscountCode
	err := q.db.QueryRow(
		`SELECT id, code, description, discount_type, discount_value, min_order_amount, 
		 usage_type, max_uses, used_count, active, start_date, end_date, created_by, created_at, updated_at
		 FROM discount_codes WHERE code = $1`,
		code,
	).Scan(
		&dc.ID, &dc.Code, &dc.Description, &dc.DiscountType, &dc.DiscountValue,
		&dc.MinOrderAmount, &dc.UsageType, &dc.MaxUses, &dc.UsedCount, &dc.Active,
		&dc.StartDate, &dc.EndDate, &dc.CreatedBy, &dc.CreatedAt, &dc.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("discount code not found")
		}
		return nil, fmt.Errorf("failed to get discount code: %w", err)
	}
	return &dc, nil
}

// Admin methods below

// CreateDiscountCode creates a new discount code
func (q *DiscountQueries) CreateDiscountCode(req *models.DiscountCodeRequest, createdBy int) (*models.DiscountCodeResponse, error) {
	var dc models.DiscountCode
	err := q.db.QueryRow(
		`INSERT INTO discount_codes (code, description, discount_type, discount_value, min_order_amount, 
		 usage_type, max_uses, active, start_date, end_date, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id, code, description, discount_type, discount_value, min_order_amount, 
		 usage_type, max_uses, used_count, active, start_date, end_date, created_by, created_at, updated_at`,
		req.Code, req.Description, req.DiscountType, req.DiscountValue, req.MinOrderAmount,
		req.UsageType, req.MaxUses, req.Active, req.StartDate, req.EndDate, createdBy,
	).Scan(
		&dc.ID, &dc.Code, &dc.Description, &dc.DiscountType, &dc.DiscountValue,
		&dc.MinOrderAmount, &dc.UsageType, &dc.MaxUses, &dc.UsedCount, &dc.Active,
		&dc.StartDate, &dc.EndDate, &dc.CreatedBy, &dc.CreatedAt, &dc.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create discount code: %w", err)
	}

	return q.buildDiscountCodeResponse(&dc), nil
}

// GetDiscountCodes gets a paginated list of discount codes
func (q *DiscountQueries) GetDiscountCodes(page, limit int, activeFilter *bool) (*models.DiscountCodeListResponse, error) {
	offset := (page - 1) * limit

	var conditions []string
	var args []interface{}
	argIndex := 1

	if activeFilter != nil {
		conditions = append(conditions, fmt.Sprintf("active = $%d", argIndex))
		args = append(args, *activeFilter)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM discount_codes %s", whereClause)
	var total int
	err := q.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count discount codes: %w", err)
	}

	// Get discount codes
	query := fmt.Sprintf(`
		SELECT id, code, description, discount_type, discount_value, min_order_amount, 
		       usage_type, max_uses, used_count, active, start_date, end_date, created_by, created_at, updated_at
		FROM discount_codes %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get discount codes: %w", err)
	}
	defer rows.Close()

	var discountCodes []models.DiscountCodeResponse
	for rows.Next() {
		var dc models.DiscountCode
		err := rows.Scan(
			&dc.ID, &dc.Code, &dc.Description, &dc.DiscountType, &dc.DiscountValue,
			&dc.MinOrderAmount, &dc.UsageType, &dc.MaxUses, &dc.UsedCount, &dc.Active,
			&dc.StartDate, &dc.EndDate, &dc.CreatedBy, &dc.CreatedAt, &dc.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan discount code: %w", err)
		}
		discountCodes = append(discountCodes, *q.buildDiscountCodeResponse(&dc))
	}

	return &models.DiscountCodeListResponse{
		DiscountCodes: discountCodes,
		Total:         total,
		Page:          page,
		Limit:         limit,
	}, nil
}

// GetDiscountCodeByID gets a discount code by ID
func (q *DiscountQueries) GetDiscountCodeByID(id int) (*models.DiscountCodeResponse, error) {
	var dc models.DiscountCode
	err := q.db.QueryRow(
		`SELECT id, code, description, discount_type, discount_value, min_order_amount, 
		 usage_type, max_uses, used_count, active, start_date, end_date, created_by, created_at, updated_at
		 FROM discount_codes WHERE id = $1`,
		id,
	).Scan(
		&dc.ID, &dc.Code, &dc.Description, &dc.DiscountType, &dc.DiscountValue,
		&dc.MinOrderAmount, &dc.UsageType, &dc.MaxUses, &dc.UsedCount, &dc.Active,
		&dc.StartDate, &dc.EndDate, &dc.CreatedBy, &dc.CreatedAt, &dc.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("discount code not found")
		}
		return nil, fmt.Errorf("failed to get discount code: %w", err)
	}

	return q.buildDiscountCodeResponse(&dc), nil
}

// UpdateDiscountCode updates a discount code
func (q *DiscountQueries) UpdateDiscountCode(id int, req *models.DiscountCodeRequest) (*models.DiscountCodeResponse, error) {
	var dc models.DiscountCode
	err := q.db.QueryRow(
		`UPDATE discount_codes SET 
		 code = $1, description = $2, discount_type = $3, discount_value = $4, min_order_amount = $5,
		 usage_type = $6, max_uses = $7, active = $8, start_date = $9, end_date = $10, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $11
		 RETURNING id, code, description, discount_type, discount_value, min_order_amount, 
		 usage_type, max_uses, used_count, active, start_date, end_date, created_by, created_at, updated_at`,
		req.Code, req.Description, req.DiscountType, req.DiscountValue, req.MinOrderAmount,
		req.UsageType, req.MaxUses, req.Active, req.StartDate, req.EndDate, id,
	).Scan(
		&dc.ID, &dc.Code, &dc.Description, &dc.DiscountType, &dc.DiscountValue,
		&dc.MinOrderAmount, &dc.UsageType, &dc.MaxUses, &dc.UsedCount, &dc.Active,
		&dc.StartDate, &dc.EndDate, &dc.CreatedBy, &dc.CreatedAt, &dc.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("discount code not found")
		}
		return nil, fmt.Errorf("failed to update discount code: %w", err)
	}

	return q.buildDiscountCodeResponse(&dc), nil
}

// DeleteDiscountCode deletes a discount code
func (q *DiscountQueries) DeleteDiscountCode(id int) error {
	result, err := q.db.Exec("DELETE FROM discount_codes WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete discount code: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("discount code not found")
	}

	return nil
}

// GetDiscountCodeUsage gets usage statistics for a discount code
func (q *DiscountQueries) GetDiscountCodeUsage(id int) ([]models.DiscountCodeUsage, error) {
	// First check if discount code exists
	var exists bool
	err := q.db.QueryRow("SELECT EXISTS(SELECT 1 FROM discount_codes WHERE id = $1)", id).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check if discount code exists: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("discount code not found")
	}

	rows, err := q.db.Query(
		`SELECT id, discount_code_id, user_id, session_id, order_id, created_at
		 FROM discount_code_usage WHERE discount_code_id = $1 ORDER BY created_at DESC`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get discount code usage: %w", err)
	}
	defer rows.Close()

	var usage []models.DiscountCodeUsage
	for rows.Next() {
		var u models.DiscountCodeUsage
		err := rows.Scan(&u.ID, &u.DiscountCodeID, &u.UserID, &u.SessionID, &u.OrderID, &u.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan usage record: %w", err)
		}
		usage = append(usage, u)
	}

	return usage, nil
}

// buildDiscountCodeResponse builds a response with additional calculated fields
func (q *DiscountQueries) buildDiscountCodeResponse(dc *models.DiscountCode) *models.DiscountCodeResponse {
	now := time.Now()
	isExpired := dc.EndDate != nil && now.After(*dc.EndDate)
	isUsageExceeded := dc.MaxUses != nil && dc.UsedCount >= *dc.MaxUses

	return &models.DiscountCodeResponse{
		ID:              dc.ID,
		Code:            dc.Code,
		Description:     dc.Description,
		DiscountType:    dc.DiscountType,
		DiscountValue:   dc.DiscountValue,
		MinOrderAmount:  dc.MinOrderAmount,
		UsageType:       dc.UsageType,
		MaxUses:         dc.MaxUses,
		UsedCount:       dc.UsedCount,
		Active:          dc.Active,
		StartDate:       dc.StartDate,
		EndDate:         dc.EndDate,
		CreatedBy:       dc.CreatedBy,
		CreatedAt:       dc.CreatedAt,
		UpdatedAt:       dc.UpdatedAt,
		IsExpired:       isExpired,
		IsUsageExceeded: isUsageExceeded,
	}
}