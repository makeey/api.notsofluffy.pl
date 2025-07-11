package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"notsofluffy-backend/internal/models"
)

type OrderQueries struct {
	db *sql.DB
}

func NewOrderQueries(db *sql.DB) *OrderQueries {
	return &OrderQueries{db: db}
}

// CreateOrder creates a new order with addresses and items in a transaction
func (q *OrderQueries) CreateOrder(order *models.Order, shippingAddr *models.ShippingAddress, billingAddr *models.BillingAddress, items []models.OrderItem) (*models.OrderResponse, error) {
	tx, err := q.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert order
	orderQuery := `
		INSERT INTO orders (user_id, session_id, email, status, total_amount, subtotal, shipping_cost, tax_amount, payment_method, payment_status, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`
	
	err = tx.QueryRow(orderQuery, order.UserID, order.SessionID, order.Email, order.Status, order.TotalAmount, order.Subtotal, order.ShippingCost, order.TaxAmount, order.PaymentMethod, order.PaymentStatus, order.Notes).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert order: %w", err)
	}

	// Insert shipping address
	shippingQuery := `
		INSERT INTO shipping_addresses (order_id, first_name, last_name, company, address_line1, address_line2, city, state_province, postal_code, country, phone)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at`
	
	err = tx.QueryRow(shippingQuery, order.ID, shippingAddr.FirstName, shippingAddr.LastName, shippingAddr.Company, shippingAddr.AddressLine1, shippingAddr.AddressLine2, shippingAddr.City, shippingAddr.StateProvince, shippingAddr.PostalCode, shippingAddr.Country, shippingAddr.Phone).Scan(&shippingAddr.ID, &shippingAddr.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert shipping address: %w", err)
	}
	shippingAddr.OrderID = order.ID

	// Insert billing address
	billingQuery := `
		INSERT INTO billing_addresses (order_id, first_name, last_name, company, address_line1, address_line2, city, state_province, postal_code, country, phone, same_as_shipping)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at`
	
	err = tx.QueryRow(billingQuery, order.ID, billingAddr.FirstName, billingAddr.LastName, billingAddr.Company, billingAddr.AddressLine1, billingAddr.AddressLine2, billingAddr.City, billingAddr.StateProvince, billingAddr.PostalCode, billingAddr.Country, billingAddr.Phone, billingAddr.SameAsShipping).Scan(&billingAddr.ID, &billingAddr.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert billing address: %w", err)
	}
	billingAddr.OrderID = order.ID

	// Insert order items
	for i := range items {
		item := &items[i]
		
		// Convert size dimensions to JSON
		var dimensionsJSON []byte
		if item.SizeDimensions != nil {
			dimensionsJSON, err = json.Marshal(item.SizeDimensions)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal size dimensions: %w", err)
			}
		}

		itemQuery := `
			INSERT INTO order_items (order_id, product_id, product_name, product_description, variant_id, variant_name, variant_color_name, variant_color_custom, size_id, size_name, size_dimensions, quantity, unit_price, total_price)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			RETURNING id, created_at`
		
		err = tx.QueryRow(itemQuery, order.ID, item.ProductID, item.ProductName, item.ProductDescription, item.VariantID, item.VariantName, item.VariantColorName, item.VariantColorCustom, item.SizeID, item.SizeName, dimensionsJSON, item.Quantity, item.UnitPrice, item.TotalPrice).Scan(&item.ID, &item.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to insert order item: %w", err)
		}
		item.OrderID = order.ID

		// Insert order item services
		for j := range item.Services {
			service := &item.Services[j]
			serviceQuery := `
				INSERT INTO order_item_services (order_item_id, service_id, service_name, service_description, service_price)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id, created_at`
			
			err = tx.QueryRow(serviceQuery, item.ID, service.ServiceID, service.ServiceName, service.ServiceDescription, service.ServicePrice).Scan(&service.ID, &service.CreatedAt)
			if err != nil {
				return nil, fmt.Errorf("failed to insert order item service: %w", err)
			}
			service.OrderItemID = item.ID
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return order response
	return &models.OrderResponse{
		ID:              order.ID,
		UserID:          order.UserID,
		SessionID:       order.SessionID,
		Email:           order.Email,
		Status:          order.Status,
		TotalAmount:     order.TotalAmount,
		Subtotal:        order.Subtotal,
		ShippingCost:    order.ShippingCost,
		TaxAmount:       order.TaxAmount,
		PaymentMethod:   order.PaymentMethod,
		PaymentStatus:   order.PaymentStatus,
		Notes:           order.Notes,
		ShippingAddress: shippingAddr,
		BillingAddress:  billingAddr,
		Items:           items,
		CreatedAt:       order.CreatedAt,
		UpdatedAt:       order.UpdatedAt,
	}, nil
}

// GetOrderByID retrieves an order by ID with all related data
func (q *OrderQueries) GetOrderByID(id int) (*models.OrderResponse, error) {
	// Get order
	orderQuery := `
		SELECT id, user_id, session_id, email, status, total_amount, subtotal, shipping_cost, tax_amount, payment_method, payment_status, notes, created_at, updated_at
		FROM orders
		WHERE id = $1`
	
	var order models.Order
	err := q.db.QueryRow(orderQuery, id).Scan(&order.ID, &order.UserID, &order.SessionID, &order.Email, &order.Status, &order.TotalAmount, &order.Subtotal, &order.ShippingCost, &order.TaxAmount, &order.PaymentMethod, &order.PaymentStatus, &order.Notes, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Get shipping address
	shippingQuery := `
		SELECT id, first_name, last_name, company, address_line1, address_line2, city, state_province, postal_code, country, phone, created_at
		FROM shipping_addresses
		WHERE order_id = $1`
	
	var shippingAddr models.ShippingAddress
	err = q.db.QueryRow(shippingQuery, id).Scan(&shippingAddr.ID, &shippingAddr.FirstName, &shippingAddr.LastName, &shippingAddr.Company, &shippingAddr.AddressLine1, &shippingAddr.AddressLine2, &shippingAddr.City, &shippingAddr.StateProvince, &shippingAddr.PostalCode, &shippingAddr.Country, &shippingAddr.Phone, &shippingAddr.CreatedAt)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get shipping address: %w", err)
	}
	shippingAddr.OrderID = id

	// Get billing address
	billingQuery := `
		SELECT id, first_name, last_name, company, address_line1, address_line2, city, state_province, postal_code, country, phone, same_as_shipping, created_at
		FROM billing_addresses
		WHERE order_id = $1`
	
	var billingAddr models.BillingAddress
	err = q.db.QueryRow(billingQuery, id).Scan(&billingAddr.ID, &billingAddr.FirstName, &billingAddr.LastName, &billingAddr.Company, &billingAddr.AddressLine1, &billingAddr.AddressLine2, &billingAddr.City, &billingAddr.StateProvince, &billingAddr.PostalCode, &billingAddr.Country, &billingAddr.Phone, &billingAddr.SameAsShipping, &billingAddr.CreatedAt)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get billing address: %w", err)
	}
	billingAddr.OrderID = id

	// Get order items
	itemsQuery := `
		SELECT id, product_id, product_name, product_description, variant_id, variant_name, variant_color_name, variant_color_custom, size_id, size_name, size_dimensions, quantity, unit_price, total_price, created_at
		FROM order_items
		WHERE order_id = $1
		ORDER BY id`
	
	rows, err := q.db.Query(itemsQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		var dimensionsJSON []byte
		
		err := rows.Scan(&item.ID, &item.ProductID, &item.ProductName, &item.ProductDescription, &item.VariantID, &item.VariantName, &item.VariantColorName, &item.VariantColorCustom, &item.SizeID, &item.SizeName, &dimensionsJSON, &item.Quantity, &item.UnitPrice, &item.TotalPrice, &item.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}
		
		// Parse size dimensions
		if dimensionsJSON != nil {
			err = json.Unmarshal(dimensionsJSON, &item.SizeDimensions)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal size dimensions: %w", err)
			}
		}
		
		item.OrderID = id
		items = append(items, item)
	}

	// Get services for each item
	for i := range items {
		servicesQuery := `
			SELECT id, service_id, service_name, service_description, service_price, created_at
			FROM order_item_services
			WHERE order_item_id = $1
			ORDER BY id`
		
		serviceRows, err := q.db.Query(servicesQuery, items[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order item services: %w", err)
		}
		defer serviceRows.Close()

		var services []models.OrderItemService
		for serviceRows.Next() {
			var service models.OrderItemService
			err := serviceRows.Scan(&service.ID, &service.ServiceID, &service.ServiceName, &service.ServiceDescription, &service.ServicePrice, &service.CreatedAt)
			if err != nil {
				return nil, fmt.Errorf("failed to scan order item service: %w", err)
			}
			service.OrderItemID = items[i].ID
			services = append(services, service)
		}
		items[i].Services = services
	}

	return &models.OrderResponse{
		ID:              order.ID,
		UserID:          order.UserID,
		SessionID:       order.SessionID,
		Email:           order.Email,
		Status:          order.Status,
		TotalAmount:     order.TotalAmount,
		Subtotal:        order.Subtotal,
		ShippingCost:    order.ShippingCost,
		TaxAmount:       order.TaxAmount,
		PaymentMethod:   order.PaymentMethod,
		PaymentStatus:   order.PaymentStatus,
		Notes:           order.Notes,
		ShippingAddress: &shippingAddr,
		BillingAddress:  &billingAddr,
		Items:           items,
		CreatedAt:       order.CreatedAt,
		UpdatedAt:       order.UpdatedAt,
	}, nil
}

// ListOrders retrieves orders with pagination and filtering
func (q *OrderQueries) ListOrders(page, limit int, userID *int, email, status string) (*models.OrderListResponse, error) {
	offset := (page - 1) * limit
	
	var conditions []string
	var args []interface{}
	argIndex := 1

	if userID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, *userID)
		argIndex++
	}

	if email != "" {
		conditions = append(conditions, fmt.Sprintf("email ILIKE $%d", argIndex))
		args = append(args, "%"+email+"%")
		argIndex++
	}

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, status)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total orders
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM orders %s", whereClause)
	var total int
	err := q.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count orders: %w", err)
	}

	// Get orders
	ordersQuery := fmt.Sprintf(`
		SELECT id, user_id, session_id, email, status, total_amount, subtotal, shipping_cost, tax_amount, payment_method, payment_status, notes, created_at, updated_at
		FROM orders
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)
	
	args = append(args, limit, offset)
	
	rows, err := q.db.Query(ordersQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}
	defer rows.Close()

	var orders []models.OrderResponse
	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.ID, &order.UserID, &order.SessionID, &order.Email, &order.Status, &order.TotalAmount, &order.Subtotal, &order.ShippingCost, &order.TaxAmount, &order.PaymentMethod, &order.PaymentStatus, &order.Notes, &order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		
		orders = append(orders, models.OrderResponse{
			ID:            order.ID,
			UserID:        order.UserID,
			SessionID:     order.SessionID,
			Email:         order.Email,
			Status:        order.Status,
			TotalAmount:   order.TotalAmount,
			Subtotal:      order.Subtotal,
			ShippingCost:  order.ShippingCost,
			TaxAmount:     order.TaxAmount,
			PaymentMethod: order.PaymentMethod,
			PaymentStatus: order.PaymentStatus,
			Notes:         order.Notes,
			CreatedAt:     order.CreatedAt,
			UpdatedAt:     order.UpdatedAt,
		})
	}

	return &models.OrderListResponse{
		Orders: orders,
		Total:  total,
		Page:   page,
		Limit:  limit,
	}, nil
}

// UpdateOrderStatus updates an order's status
func (q *OrderQueries) UpdateOrderStatus(id int, status string) error {
	query := `UPDATE orders SET status = $1 WHERE id = $2`
	result, err := q.db.Exec(query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("order not found")
	}
	
	return nil
}

// GetOrdersByUserID retrieves orders for a specific user
func (q *OrderQueries) GetOrdersByUserID(userID int, page, limit int) (*models.OrderListResponse, error) {
	return q.ListOrders(page, limit, &userID, "", "")
}

// DeleteOrder deletes an order and all related data
func (q *OrderQueries) DeleteOrder(id int) error {
	tx, err := q.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete order item services first
	_, err = tx.Exec(`
		DELETE FROM order_item_services 
		WHERE order_item_id IN (
			SELECT id FROM order_items WHERE order_id = $1
		)`, id)
	if err != nil {
		return fmt.Errorf("failed to delete order item services: %w", err)
	}

	// Delete order items
	_, err = tx.Exec("DELETE FROM order_items WHERE order_id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete order items: %w", err)
	}

	// Delete shipping address
	_, err = tx.Exec("DELETE FROM shipping_addresses WHERE order_id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete shipping address: %w", err)
	}

	// Delete billing address
	_, err = tx.Exec("DELETE FROM billing_addresses WHERE order_id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete billing address: %w", err)
	}

	// Delete order
	result, err := tx.Exec("DELETE FROM orders WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("order not found")
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}