package database

import (
	"database/sql"
	"fmt"
)

type StockQueries struct {
	db *sql.DB
}

func NewStockQueries(db *sql.DB) *StockQueries {
	return &StockQueries{db: db}
}

// CheckStockAvailability checks if there's enough stock available for a size
func (q *StockQueries) CheckStockAvailability(sizeID int, requestedQuantity int) (bool, int, error) {
	query := `
		SELECT use_stock, stock_quantity, reserved_quantity 
		FROM sizes 
		WHERE id = $1
	`
	var useStock bool
	var stockQuantity, reservedQuantity int
	
	err := q.db.QueryRow(query, sizeID).Scan(&useStock, &stockQuantity, &reservedQuantity)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, 0, fmt.Errorf("size not found")
		}
		return false, 0, fmt.Errorf("failed to check stock availability: %w", err)
	}
	
	// If stock management is disabled, always allow
	if !useStock {
		return true, -1, nil // -1 indicates unlimited stock
	}
	
	availableStock := stockQuantity - reservedQuantity
	return availableStock >= requestedQuantity, availableStock, nil
}

// GetStockLevel returns the current available stock level for a size
func (q *StockQueries) GetStockLevel(sizeID int) (int, error) {
	query := `
		SELECT 
			CASE 
				WHEN use_stock = false THEN -1
				ELSE stock_quantity - reserved_quantity
			END as available_stock
		FROM sizes 
		WHERE id = $1
	`
	var availableStock int
	
	err := q.db.QueryRow(query, sizeID).Scan(&availableStock)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("size not found")
		}
		return 0, fmt.Errorf("failed to get stock level: %w", err)
	}
	
	return availableStock, nil
}

// ReserveStock temporarily reserves stock for a size during checkout process
func (q *StockQueries) ReserveStock(sizeID int, quantity int) error {
	// First check if we have enough stock
	available, availableStock, err := q.CheckStockAvailability(sizeID, quantity)
	if err != nil {
		return err
	}
	
	if !available {
		return fmt.Errorf("insufficient stock: requested %d, available %d", quantity, availableStock)
	}
	
	// If stock management is disabled, do nothing
	if availableStock == -1 {
		return nil
	}
	
	// Reserve the stock
	query := `
		UPDATE sizes 
		SET reserved_quantity = reserved_quantity + $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND use_stock = true
	`
	
	result, err := q.db.Exec(query, quantity, sizeID)
	if err != nil {
		return fmt.Errorf("failed to reserve stock: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("no stock reserved: size may not exist or stock management disabled")
	}
	
	return nil
}

// ReleaseStock releases previously reserved stock (e.g., if checkout fails)
func (q *StockQueries) ReleaseStock(sizeID int, quantity int) error {
	query := `
		UPDATE sizes 
		SET reserved_quantity = GREATEST(0, reserved_quantity - $1), updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND use_stock = true
	`
	
	_, err := q.db.Exec(query, quantity, sizeID)
	if err != nil {
		return fmt.Errorf("failed to release stock: %w", err)
	}
	
	return nil
}

// DecrementStock finalizes stock reduction (converts reserved to actual reduction)
func (q *StockQueries) DecrementStock(sizeID int, quantity int) error {
	query := `
		UPDATE sizes 
		SET 
			stock_quantity = GREATEST(0, stock_quantity - $1),
			reserved_quantity = GREATEST(0, reserved_quantity - $1),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND use_stock = true
	`
	
	result, err := q.db.Exec(query, quantity, sizeID)
	if err != nil {
		return fmt.Errorf("failed to decrement stock: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("no stock decremented: size may not exist or stock management disabled")
	}
	
	return nil
}

// IncrementStock increases stock quantity (for returns, restocking, etc.)
func (q *StockQueries) IncrementStock(sizeID int, quantity int) error {
	query := `
		UPDATE sizes 
		SET stock_quantity = stock_quantity + $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND use_stock = true
	`
	
	result, err := q.db.Exec(query, quantity, sizeID)
	if err != nil {
		return fmt.Errorf("failed to increment stock: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("no stock incremented: size may not exist or stock management disabled")
	}
	
	return nil
}

// GetStockSummary returns a summary of stock levels for multiple sizes
func (q *StockQueries) GetStockSummary(sizeIDs []int) (map[int]int, error) {
	if len(sizeIDs) == 0 {
		return map[int]int{}, nil
	}
	
	query := `
		SELECT 
			id,
			CASE 
				WHEN use_stock = false THEN -1
				ELSE stock_quantity - reserved_quantity
			END as available_stock
		FROM sizes 
		WHERE id = ANY($1)
	`
	
	rows, err := q.db.Query(query, sizeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get stock summary: %w", err)
	}
	defer rows.Close()
	
	stockLevels := make(map[int]int)
	for rows.Next() {
		var sizeID, availableStock int
		if err := rows.Scan(&sizeID, &availableStock); err != nil {
			return nil, fmt.Errorf("failed to scan stock level: %w", err)
		}
		stockLevels[sizeID] = availableStock
	}
	
	return stockLevels, nil
}

// UpdateStockQuantity updates the stock quantity for a size (used by admin)
func (q *StockQueries) UpdateStockQuantity(sizeID int, newQuantity int) error {
	query := `
		UPDATE sizes 
		SET stock_quantity = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	
	result, err := q.db.Exec(query, newQuantity, sizeID)
	if err != nil {
		return fmt.Errorf("failed to update stock quantity: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("no stock updated: size may not exist")
	}
	
	return nil
}