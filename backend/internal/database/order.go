package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"notsofluffy-backend/internal/models"
)

type OrderQueries struct {
	db *sql.DB
}

// generatePublicHash generates a secure random hash for public order access
func generatePublicHash() (string, error) {
	bytes := make([]byte, 16) // 128 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
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

	// Generate public hash for guest order access
	publicHash, err := generatePublicHash()
	if err != nil {
		return nil, fmt.Errorf("failed to generate public hash: %w", err)
	}
	order.PublicHash = &publicHash

	// Insert order
	orderQuery := `
		INSERT INTO orders (user_id, session_id, public_hash, email, phone, status, total_amount, subtotal, shipping_cost, tax_amount, discount_code_id, discount_amount, discount_description, payment_method, payment_status, notes, requires_invoice, nip)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id, created_at, updated_at`
	
	err = tx.QueryRow(orderQuery, order.UserID, order.SessionID, order.PublicHash, order.Email, order.Phone, order.Status, order.TotalAmount, order.Subtotal, order.ShippingCost, order.TaxAmount, order.DiscountCodeID, order.DiscountAmount, order.DiscountDescription, order.PaymentMethod, order.PaymentStatus, order.Notes, order.RequiresInvoice, order.NIP).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
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
		ID:                 order.ID,
		UserID:             order.UserID,
		SessionID:          order.SessionID,
		PublicHash:         order.PublicHash,
		Email:              order.Email,
		Phone:              order.Phone,
		Status:             order.Status,
		TotalAmount:        order.TotalAmount,
		Subtotal:           order.Subtotal,
		ShippingCost:       order.ShippingCost,
		TaxAmount:          order.TaxAmount,
		DiscountCodeID:     order.DiscountCodeID,
		DiscountAmount:     order.DiscountAmount,
		DiscountDescription: order.DiscountDescription,
		PaymentMethod:      order.PaymentMethod,
		PaymentStatus:      order.PaymentStatus,
		Notes:              order.Notes,
		RequiresInvoice:    order.RequiresInvoice,
		NIP:                order.NIP,
		ShippingAddress:    shippingAddr,
		BillingAddress:     billingAddr,
		Items:              items,
		CreatedAt:          order.CreatedAt,
		UpdatedAt:          order.UpdatedAt,
	}, nil
}

// GetOrderByID retrieves an order by ID with all related data
func (q *OrderQueries) GetOrderByID(id int) (*models.OrderResponse, error) {
	// Get order
	orderQuery := `
		SELECT id, user_id, session_id, public_hash, email, phone, status, total_amount, subtotal, shipping_cost, tax_amount, discount_code_id, discount_amount, discount_description, payment_method, payment_status, notes, requires_invoice, nip, created_at, updated_at
		FROM orders
		WHERE id = $1`
	
	var order models.Order
	err := q.db.QueryRow(orderQuery, id).Scan(&order.ID, &order.UserID, &order.SessionID, &order.PublicHash, &order.Email, &order.Phone, &order.Status, &order.TotalAmount, &order.Subtotal, &order.ShippingCost, &order.TaxAmount, &order.DiscountCodeID, &order.DiscountAmount, &order.DiscountDescription, &order.PaymentMethod, &order.PaymentStatus, &order.Notes, &order.RequiresInvoice, &order.NIP, &order.CreatedAt, &order.UpdatedAt)
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

	// Get order items with product images
	itemsQuery := `
		SELECT oi.id, oi.product_id, oi.product_name, oi.product_description, oi.variant_id, oi.variant_name, oi.variant_color_name, oi.variant_color_custom, oi.size_id, oi.size_name, oi.size_dimensions, oi.quantity, oi.unit_price, oi.total_price, oi.created_at,
		       mi.id as main_image_id, mi.filename as main_image_filename, mi.original_name as main_image_original_name, mi.path as main_image_path, mi.size_bytes as main_image_size_bytes, mi.mime_type as main_image_mime_type, mi.uploaded_by as main_image_uploaded_by, mi.created_at as main_image_created_at, mi.updated_at as main_image_updated_at
		FROM order_items oi
		LEFT JOIN products p ON oi.product_id = p.id
		LEFT JOIN images mi ON p.main_image_id = mi.id
		WHERE oi.order_id = $1
		ORDER BY oi.id`
	
	rows, err := q.db.Query(itemsQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		var dimensionsJSON []byte
		var mainImageID sql.NullInt64
		var mainImageFilename, mainImageOriginalName, mainImagePath, mainImageMimeType sql.NullString
		var mainImageSizeBytes sql.NullInt64
		var mainImageUploadedBy sql.NullInt64
		var mainImageCreatedAt, mainImageUpdatedAt sql.NullTime
		
		err := rows.Scan(&item.ID, &item.ProductID, &item.ProductName, &item.ProductDescription, &item.VariantID, &item.VariantName, &item.VariantColorName, &item.VariantColorCustom, &item.SizeID, &item.SizeName, &dimensionsJSON, &item.Quantity, &item.UnitPrice, &item.TotalPrice, &item.CreatedAt,
			&mainImageID, &mainImageFilename, &mainImageOriginalName, &mainImagePath, &mainImageSizeBytes, &mainImageMimeType, &mainImageUploadedBy, &mainImageCreatedAt, &mainImageUpdatedAt)
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
		
		// Add main image if available
		if mainImageID.Valid {
			item.MainImage = &models.ImageResponse{
				ID:           int(mainImageID.Int64),
				Filename:     mainImageFilename.String,
				OriginalName: mainImageOriginalName.String,
				Path:         mainImagePath.String,
				SizeBytes:    mainImageSizeBytes.Int64,
				MimeType:     mainImageMimeType.String,
				UploadedBy:   int(mainImageUploadedBy.Int64),
				CreatedAt:    mainImageCreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:    mainImageUpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
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
		ID:                 order.ID,
		UserID:             order.UserID,
		SessionID:          order.SessionID,
		PublicHash:         order.PublicHash,
		Email:              order.Email,
		Phone:              order.Phone,
		Status:             order.Status,
		TotalAmount:        order.TotalAmount,
		Subtotal:           order.Subtotal,
		ShippingCost:       order.ShippingCost,
		TaxAmount:          order.TaxAmount,
		DiscountCodeID:     order.DiscountCodeID,
		DiscountAmount:     order.DiscountAmount,
		DiscountDescription: order.DiscountDescription,
		PaymentMethod:      order.PaymentMethod,
		PaymentStatus:      order.PaymentStatus,
		Notes:              order.Notes,
		RequiresInvoice:    order.RequiresInvoice,
		NIP:                order.NIP,
		ShippingAddress:    &shippingAddr,
		BillingAddress:     &billingAddr,
		Items:              items,
		CreatedAt:          order.CreatedAt,
		UpdatedAt:          order.UpdatedAt,
	}, nil
}

// GetOrderByHash retrieves an order by public hash for guest access
func (q *OrderQueries) GetOrderByHash(hash string) (*models.OrderResponse, error) {
	// Get order
	orderQuery := `
		SELECT id, user_id, session_id, public_hash, email, phone, status, total_amount, subtotal, shipping_cost, tax_amount, discount_code_id, discount_amount, discount_description, payment_method, payment_status, notes, requires_invoice, nip, created_at, updated_at
		FROM orders
		WHERE public_hash = $1`
	
	var order models.Order
	err := q.db.QueryRow(orderQuery, hash).Scan(&order.ID, &order.UserID, &order.SessionID, &order.PublicHash, &order.Email, &order.Phone, &order.Status, &order.TotalAmount, &order.Subtotal, &order.ShippingCost, &order.TaxAmount, &order.DiscountCodeID, &order.DiscountAmount, &order.DiscountDescription, &order.PaymentMethod, &order.PaymentStatus, &order.Notes, &order.RequiresInvoice, &order.NIP, &order.CreatedAt, &order.UpdatedAt)
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
	err = q.db.QueryRow(shippingQuery, order.ID).Scan(&shippingAddr.ID, &shippingAddr.FirstName, &shippingAddr.LastName, &shippingAddr.Company, &shippingAddr.AddressLine1, &shippingAddr.AddressLine2, &shippingAddr.City, &shippingAddr.StateProvince, &shippingAddr.PostalCode, &shippingAddr.Country, &shippingAddr.Phone, &shippingAddr.CreatedAt)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get shipping address: %w", err)
	}
	shippingAddr.OrderID = order.ID

	// Get billing address
	billingQuery := `
		SELECT id, first_name, last_name, company, address_line1, address_line2, city, state_province, postal_code, country, phone, same_as_shipping, created_at
		FROM billing_addresses
		WHERE order_id = $1`
	
	var billingAddr models.BillingAddress
	err = q.db.QueryRow(billingQuery, order.ID).Scan(&billingAddr.ID, &billingAddr.FirstName, &billingAddr.LastName, &billingAddr.Company, &billingAddr.AddressLine1, &billingAddr.AddressLine2, &billingAddr.City, &billingAddr.StateProvince, &billingAddr.PostalCode, &billingAddr.Country, &billingAddr.Phone, &billingAddr.SameAsShipping, &billingAddr.CreatedAt)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get billing address: %w", err)
	}
	billingAddr.OrderID = order.ID

	// Get order items with product images
	itemsQuery := `
		SELECT oi.id, oi.product_id, oi.product_name, oi.product_description, oi.variant_id, oi.variant_name, oi.variant_color_name, oi.variant_color_custom, oi.size_id, oi.size_name, oi.size_dimensions, oi.quantity, oi.unit_price, oi.total_price, oi.created_at,
		       mi.id as main_image_id, mi.filename as main_image_filename, mi.original_name as main_image_original_name, mi.path as main_image_path, mi.size_bytes as main_image_size_bytes, mi.mime_type as main_image_mime_type, mi.uploaded_by as main_image_uploaded_by, mi.created_at as main_image_created_at, mi.updated_at as main_image_updated_at
		FROM order_items oi
		LEFT JOIN products p ON oi.product_id = p.id
		LEFT JOIN images mi ON p.main_image_id = mi.id
		WHERE oi.order_id = $1
		ORDER BY oi.id`
	
	rows, err := q.db.Query(itemsQuery, order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		var dimensionsJSON []byte
		var mainImageID sql.NullInt64
		var mainImageFilename, mainImageOriginalName, mainImagePath, mainImageMimeType sql.NullString
		var mainImageSizeBytes sql.NullInt64
		var mainImageUploadedBy sql.NullInt64
		var mainImageCreatedAt, mainImageUpdatedAt sql.NullTime
		
		err := rows.Scan(&item.ID, &item.ProductID, &item.ProductName, &item.ProductDescription, &item.VariantID, &item.VariantName, &item.VariantColorName, &item.VariantColorCustom, &item.SizeID, &item.SizeName, &dimensionsJSON, &item.Quantity, &item.UnitPrice, &item.TotalPrice, &item.CreatedAt,
			&mainImageID, &mainImageFilename, &mainImageOriginalName, &mainImagePath, &mainImageSizeBytes, &mainImageMimeType, &mainImageUploadedBy, &mainImageCreatedAt, &mainImageUpdatedAt)
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
		
		// Add main image if available
		if mainImageID.Valid {
			item.MainImage = &models.ImageResponse{
				ID:           int(mainImageID.Int64),
				Filename:     mainImageFilename.String,
				OriginalName: mainImageOriginalName.String,
				Path:         mainImagePath.String,
				SizeBytes:    mainImageSizeBytes.Int64,
				MimeType:     mainImageMimeType.String,
				UploadedBy:   int(mainImageUploadedBy.Int64),
				CreatedAt:    mainImageCreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:    mainImageUpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
			}
		}
		
		item.OrderID = order.ID
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
		ID:                 order.ID,
		UserID:             order.UserID,
		SessionID:          order.SessionID,
		PublicHash:         order.PublicHash,
		Email:              order.Email,
		Phone:              order.Phone,
		Status:             order.Status,
		TotalAmount:        order.TotalAmount,
		Subtotal:           order.Subtotal,
		ShippingCost:       order.ShippingCost,
		TaxAmount:          order.TaxAmount,
		DiscountCodeID:     order.DiscountCodeID,
		DiscountAmount:     order.DiscountAmount,
		DiscountDescription: order.DiscountDescription,
		PaymentMethod:      order.PaymentMethod,
		PaymentStatus:      order.PaymentStatus,
		Notes:              order.Notes,
		RequiresInvoice:    order.RequiresInvoice,
		NIP:                order.NIP,
		ShippingAddress:    &shippingAddr,
		BillingAddress:     &billingAddr,
		Items:              items,
		CreatedAt:          order.CreatedAt,
		UpdatedAt:          order.UpdatedAt,
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
		SELECT id, user_id, session_id, email, phone, status, total_amount, subtotal, shipping_cost, tax_amount, payment_method, payment_status, notes, requires_invoice, nip, created_at, updated_at
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
		err := rows.Scan(&order.ID, &order.UserID, &order.SessionID, &order.Email, &order.Phone, &order.Status, &order.TotalAmount, &order.Subtotal, &order.ShippingCost, &order.TaxAmount, &order.PaymentMethod, &order.PaymentStatus, &order.Notes, &order.RequiresInvoice, &order.NIP, &order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		
		orders = append(orders, models.OrderResponse{
			ID:              order.ID,
			UserID:          order.UserID,
			SessionID:       order.SessionID,
			Email:           order.Email,
			Phone:           order.Phone,
			Status:          order.Status,
			TotalAmount:     order.TotalAmount,
			Subtotal:        order.Subtotal,
			ShippingCost:    order.ShippingCost,
			TaxAmount:       order.TaxAmount,
			PaymentMethod:   order.PaymentMethod,
			PaymentStatus:   order.PaymentStatus,
			Notes:           order.Notes,
			RequiresInvoice: order.RequiresInvoice,
			NIP:             order.NIP,
			CreatedAt:       order.CreatedAt,
			UpdatedAt:       order.UpdatedAt,
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

// GetOrdersByUserIDWithItems retrieves orders for a specific user with full order items, addresses and services
func (q *OrderQueries) GetOrdersByUserIDWithItems(userID int, page, limit int) (*models.OrderListResponse, error) {
	offset := (page - 1) * limit
	
	// Count total orders for the user
	countQuery := "SELECT COUNT(*) FROM orders WHERE user_id = $1"
	var total int
	err := q.db.QueryRow(countQuery, userID).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count orders: %w", err)
	}

	// Get basic order information with pagination
	ordersQuery := `
		SELECT id, user_id, session_id, email, phone, status, total_amount, subtotal, shipping_cost, tax_amount, payment_method, payment_status, notes, requires_invoice, nip, created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`
	
	rows, err := q.db.Query(ordersQuery, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}
	defer rows.Close()

	var orders []models.OrderResponse
	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.ID, &order.UserID, &order.SessionID, &order.Email, &order.Phone, &order.Status, &order.TotalAmount, &order.Subtotal, &order.ShippingCost, &order.TaxAmount, &order.PaymentMethod, &order.PaymentStatus, &order.Notes, &order.RequiresInvoice, &order.NIP, &order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		// Get shipping address for this order
		var shippingAddr *models.ShippingAddress
		shippingQuery := `
			SELECT id, first_name, last_name, company, address_line1, address_line2, city, state_province, postal_code, country, phone, created_at
			FROM shipping_addresses
			WHERE order_id = $1`
		
		var addr models.ShippingAddress
		err = q.db.QueryRow(shippingQuery, order.ID).Scan(&addr.ID, &addr.FirstName, &addr.LastName, &addr.Company, &addr.AddressLine1, &addr.AddressLine2, &addr.City, &addr.StateProvince, &addr.PostalCode, &addr.Country, &addr.Phone, &addr.CreatedAt)
		if err == nil {
			addr.OrderID = order.ID
			shippingAddr = &addr
		} else if err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to get shipping address: %w", err)
		}

		// Get billing address for this order
		var billingAddr *models.BillingAddress
		billingQuery := `
			SELECT id, first_name, last_name, company, address_line1, address_line2, city, state_province, postal_code, country, phone, same_as_shipping, created_at
			FROM billing_addresses
			WHERE order_id = $1`
		
		var bAddr models.BillingAddress
		err = q.db.QueryRow(billingQuery, order.ID).Scan(&bAddr.ID, &bAddr.FirstName, &bAddr.LastName, &bAddr.Company, &bAddr.AddressLine1, &bAddr.AddressLine2, &bAddr.City, &bAddr.StateProvince, &bAddr.PostalCode, &bAddr.Country, &bAddr.Phone, &bAddr.SameAsShipping, &bAddr.CreatedAt)
		if err == nil {
			bAddr.OrderID = order.ID
			billingAddr = &bAddr
		} else if err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to get billing address: %w", err)
		}

		// Get order items for this order with product images
		itemsQuery := `
			SELECT oi.id, oi.product_id, oi.product_name, oi.product_description, oi.variant_id, oi.variant_name, oi.variant_color_name, oi.variant_color_custom, oi.size_id, oi.size_name, oi.size_dimensions, oi.quantity, oi.unit_price, oi.total_price, oi.created_at,
			       mi.id as main_image_id, mi.filename as main_image_filename, mi.original_name as main_image_original_name, mi.path as main_image_path, mi.size_bytes as main_image_size_bytes, mi.mime_type as main_image_mime_type, mi.uploaded_by as main_image_uploaded_by, mi.created_at as main_image_created_at, mi.updated_at as main_image_updated_at
			FROM order_items oi
			LEFT JOIN products p ON oi.product_id = p.id
			LEFT JOIN images mi ON p.main_image_id = mi.id
			WHERE oi.order_id = $1
			ORDER BY oi.id`
		
		itemRows, err := q.db.Query(itemsQuery, order.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order items: %w", err)
		}

		var items []models.OrderItem
		for itemRows.Next() {
			var item models.OrderItem
			var dimensionsJSON []byte
			var mainImageID sql.NullInt64
			var mainImageFilename, mainImageOriginalName, mainImagePath, mainImageMimeType sql.NullString
			var mainImageSizeBytes sql.NullInt64
			var mainImageUploadedBy sql.NullInt64
			var mainImageCreatedAt, mainImageUpdatedAt sql.NullTime
			
			err := itemRows.Scan(&item.ID, &item.ProductID, &item.ProductName, &item.ProductDescription, &item.VariantID, &item.VariantName, &item.VariantColorName, &item.VariantColorCustom, &item.SizeID, &item.SizeName, &dimensionsJSON, &item.Quantity, &item.UnitPrice, &item.TotalPrice, &item.CreatedAt,
				&mainImageID, &mainImageFilename, &mainImageOriginalName, &mainImagePath, &mainImageSizeBytes, &mainImageMimeType, &mainImageUploadedBy, &mainImageCreatedAt, &mainImageUpdatedAt)
			if err != nil {
				itemRows.Close()
				return nil, fmt.Errorf("failed to scan order item: %w", err)
			}
			
			// Parse size dimensions
			if dimensionsJSON != nil {
				err = json.Unmarshal(dimensionsJSON, &item.SizeDimensions)
				if err != nil {
					itemRows.Close()
					return nil, fmt.Errorf("failed to unmarshal size dimensions: %w", err)
				}
			}
			
			// Add main image if available
			if mainImageID.Valid {
				item.MainImage = &models.ImageResponse{
					ID:           int(mainImageID.Int64),
					Filename:     mainImageFilename.String,
					OriginalName: mainImageOriginalName.String,
					Path:         mainImagePath.String,
					SizeBytes:    mainImageSizeBytes.Int64,
					MimeType:     mainImageMimeType.String,
					UploadedBy:   int(mainImageUploadedBy.Int64),
					CreatedAt:    mainImageCreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
					UpdatedAt:    mainImageUpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
				}
			}
			
			item.OrderID = order.ID

			// Get services for this item
			servicesQuery := `
				SELECT id, service_id, service_name, service_description, service_price, created_at
				FROM order_item_services
				WHERE order_item_id = $1
				ORDER BY id`
			
			serviceRows, err := q.db.Query(servicesQuery, item.ID)
			if err != nil {
				itemRows.Close()
				return nil, fmt.Errorf("failed to get order item services: %w", err)
			}

			var services []models.OrderItemService
			for serviceRows.Next() {
				var service models.OrderItemService
				err := serviceRows.Scan(&service.ID, &service.ServiceID, &service.ServiceName, &service.ServiceDescription, &service.ServicePrice, &service.CreatedAt)
				if err != nil {
					serviceRows.Close()
					itemRows.Close()
					return nil, fmt.Errorf("failed to scan order item service: %w", err)
				}
				service.OrderItemID = item.ID
				services = append(services, service)
			}
			serviceRows.Close()
			
			item.Services = services
			items = append(items, item)
		}
		itemRows.Close()

		// Create order response with all related data
		orderResponse := models.OrderResponse{
			ID:              order.ID,
			UserID:          order.UserID,
			SessionID:       order.SessionID,
			Email:           order.Email,
			Phone:           order.Phone,
			Status:          order.Status,
			TotalAmount:     order.TotalAmount,
			Subtotal:        order.Subtotal,
			ShippingCost:    order.ShippingCost,
			TaxAmount:       order.TaxAmount,
			PaymentMethod:   order.PaymentMethod,
			PaymentStatus:   order.PaymentStatus,
			Notes:           order.Notes,
			RequiresInvoice: order.RequiresInvoice,
			NIP:             order.NIP,
			ShippingAddress: shippingAddr,
			BillingAddress:  billingAddr,
			Items:           items,
			CreatedAt:       order.CreatedAt,
			UpdatedAt:       order.UpdatedAt,
		}

		orders = append(orders, orderResponse)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating orders: %w", err)
	}

	return &models.OrderListResponse{
		Orders: orders,
		Total:  total,
		Page:   page,
		Limit:  limit,
	}, nil
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