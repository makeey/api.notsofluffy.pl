package database

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"notsofluffy-backend/internal/models"
)

type CartQueries struct {
	db *sql.DB
}

func NewCartQueries(db *sql.DB) *CartQueries {
	return &CartQueries{db: db}
}

// generateServicesHash creates a consistent hash from service IDs
func generateServicesHash(serviceIDs []int) string {
	if len(serviceIDs) == 0 {
		return ""
	}

	// Sort IDs to ensure consistent hash for same services
	sortedIDs := make([]int, len(serviceIDs))
	copy(sortedIDs, serviceIDs)
	sort.Ints(sortedIDs)

	// Create string representation
	idStrings := make([]string, len(sortedIDs))
	for i, id := range sortedIDs {
		idStrings[i] = strconv.Itoa(id)
	}
	idsString := strings.Join(idStrings, ",")

	// Generate MD5 hash
	hash := md5.Sum([]byte(idsString))
	return fmt.Sprintf("%x", hash)
}

// GetOrCreateCartSession gets an existing cart session or creates a new one
func (q *CartQueries) GetOrCreateCartSession(sessionID string, userID *int) (*models.CartSession, error) {
	// First try to get existing session
	session, err := q.GetCartSessionByID(sessionID)
	if err == nil {
		// If user is provided and session doesn't have a user, update it
		if userID != nil && session.UserID == nil {
			session.UserID = userID
			err = q.UpdateCartSessionUser(session.ID, *userID)
			if err != nil {
				return nil, fmt.Errorf("failed to update cart session user: %w", err)
			}
		}
		return session, nil
	}

	// Create new session if not found
	return q.CreateCartSession(sessionID, userID)
}

// GetCartSessionByID gets a cart session by session ID
func (q *CartQueries) GetCartSessionByID(sessionID string) (*models.CartSession, error) {
	query := `
		SELECT id, session_id, user_id, applied_discount_code_id, discount_amount, created_at, updated_at
		FROM cart_sessions
		WHERE session_id = $1
	`
	session := &models.CartSession{}
	err := q.db.QueryRow(query, sessionID).Scan(
		&session.ID,
		&session.SessionID,
		&session.UserID,
		&session.AppliedDiscountCodeID,
		&session.DiscountAmount,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("cart session not found")
		}
		return nil, fmt.Errorf("failed to get cart session: %w", err)
	}
	return session, nil
}

// CreateCartSession creates a new cart session
func (q *CartQueries) CreateCartSession(sessionID string, userID *int) (*models.CartSession, error) {
	session := &models.CartSession{
		SessionID:      sessionID,
		UserID:         userID,
		DiscountAmount: 0,
	}

	query := `
		INSERT INTO cart_sessions (session_id, user_id, applied_discount_code_id, discount_amount)
		VALUES ($1, $2, NULL, 0)
		RETURNING id, created_at, updated_at
	`
	err := q.db.QueryRow(query, session.SessionID, session.UserID).Scan(
		&session.ID,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cart session: %w", err)
	}
	return session, nil
}

// UpdateCartSessionUser updates the user_id for a cart session
func (q *CartQueries) UpdateCartSessionUser(cartSessionID int, userID int) error {
	query := `UPDATE cart_sessions SET user_id = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	_, err := q.db.Exec(query, userID, cartSessionID)
	if err != nil {
		return fmt.Errorf("failed to update cart session user: %w", err)
	}
	return nil
}

// AddCartItem adds an item to the cart or updates quantity if it exists
func (q *CartQueries) AddCartItem(cartSessionID int, item *models.CartItemRequest, pricePerItem float64) (*models.CartItem, error) {
	// Generate services hash
	servicesHash := generateServicesHash(item.AdditionalServiceIDs)

	// Check if item already exists with same services
	existing, err := q.GetCartItemByDetailsWithServices(cartSessionID, item.ProductID, item.VariantID, item.SizeID, servicesHash)
	if err == nil {
		// Item exists with same services, update quantity
		existing.Quantity += item.Quantity
		return q.UpdateCartItemQuantity(existing.ID, existing.Quantity)
	}

	// Create new cart item
	cartItem := &models.CartItem{
		CartSessionID: cartSessionID,
		ProductID:     item.ProductID,
		VariantID:     item.VariantID,
		SizeID:        item.SizeID,
		Quantity:      item.Quantity,
		PricePerItem:  pricePerItem,
		ServicesHash:  servicesHash,
	}

	query := `
		INSERT INTO cart_items (cart_session_id, product_id, variant_id, size_id, quantity, price_per_item, services_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	err = q.db.QueryRow(query, cartItem.CartSessionID, cartItem.ProductID, cartItem.VariantID,
		cartItem.SizeID, cartItem.Quantity, cartItem.PricePerItem, cartItem.ServicesHash).Scan(
		&cartItem.ID,
		&cartItem.CreatedAt,
		&cartItem.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cart item: %w", err)
	}

	// Add additional services
	if len(item.AdditionalServiceIDs) > 0 {
		err = q.AddCartItemServices(cartItem.ID, item.AdditionalServiceIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to add cart item services: %w", err)
		}
	}

	return cartItem, nil
}

// GetCartItemByDetails gets a cart item by its details (without considering services)
func (q *CartQueries) GetCartItemByDetails(cartSessionID, productID, variantID, sizeID int) (*models.CartItem, error) {
	query := `
		SELECT id, cart_session_id, product_id, variant_id, size_id, quantity, price_per_item, services_hash, created_at, updated_at
		FROM cart_items
		WHERE cart_session_id = $1 AND product_id = $2 AND variant_id = $3 AND size_id = $4
	`
	item := &models.CartItem{}
	err := q.db.QueryRow(query, cartSessionID, productID, variantID, sizeID).Scan(
		&item.ID,
		&item.CartSessionID,
		&item.ProductID,
		&item.VariantID,
		&item.SizeID,
		&item.Quantity,
		&item.PricePerItem,
		&item.ServicesHash,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("cart item not found")
		}
		return nil, fmt.Errorf("failed to get cart item: %w", err)
	}
	return item, nil
}

// GetCartItemByDetailsWithServices gets a cart item by its details including services hash
func (q *CartQueries) GetCartItemByDetailsWithServices(cartSessionID, productID, variantID, sizeID int, servicesHash string) (*models.CartItem, error) {
	query := `
		SELECT id, cart_session_id, product_id, variant_id, size_id, quantity, price_per_item, services_hash, created_at, updated_at
		FROM cart_items
		WHERE cart_session_id = $1 AND product_id = $2 AND variant_id = $3 AND size_id = $4 AND services_hash = $5
	`
	item := &models.CartItem{}
	err := q.db.QueryRow(query, cartSessionID, productID, variantID, sizeID, servicesHash).Scan(
		&item.ID,
		&item.CartSessionID,
		&item.ProductID,
		&item.VariantID,
		&item.SizeID,
		&item.Quantity,
		&item.PricePerItem,
		&item.ServicesHash,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("cart item not found")
		}
		return nil, fmt.Errorf("failed to get cart item: %w", err)
	}
	return item, nil
}

// UpdateCartItemQuantity updates the quantity of a cart item
func (q *CartQueries) UpdateCartItemQuantity(cartItemID, quantity int) (*models.CartItem, error) {
	query := `
		UPDATE cart_items 
		SET quantity = $1, updated_at = CURRENT_TIMESTAMP 
		WHERE id = $2
		RETURNING id, cart_session_id, product_id, variant_id, size_id, quantity, price_per_item, services_hash, created_at, updated_at
	`
	item := &models.CartItem{}
	err := q.db.QueryRow(query, quantity, cartItemID).Scan(
		&item.ID,
		&item.CartSessionID,
		&item.ProductID,
		&item.VariantID,
		&item.SizeID,
		&item.Quantity,
		&item.PricePerItem,
		&item.ServicesHash,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update cart item: %w", err)
	}
	return item, nil
}

// RemoveCartItem removes an item from the cart
func (q *CartQueries) RemoveCartItem(cartItemID int) error {
	query := `DELETE FROM cart_items WHERE id = $1`
	result, err := q.db.Exec(query, cartItemID)
	if err != nil {
		return fmt.Errorf("failed to remove cart item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("cart item not found")
	}

	return nil
}

// ClearCart removes all items from a cart session and clears discount information
func (q *CartQueries) ClearCart(cartSessionID int) error {
	tx, err := q.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete all cart items
	_, err = tx.Exec(`DELETE FROM cart_items WHERE cart_session_id = $1`, cartSessionID)
	if err != nil {
		return fmt.Errorf("failed to clear cart items: %w", err)
	}

	// Clear discount information from cart session
	_, err = tx.Exec(`
		UPDATE cart_sessions 
		SET applied_discount_code_id = NULL, 
		    discount_amount = 0, 
		    updated_at = CURRENT_TIMESTAMP 
		WHERE id = $1`, cartSessionID)
	if err != nil {
		return fmt.Errorf("failed to clear discount information: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetCartItems gets all items in a cart with full details
func (q *CartQueries) GetCartItems(cartSessionID int) ([]models.CartItemResponse, error) {
	query := `
		SELECT 
			ci.id, ci.product_id, ci.variant_id, ci.size_id, ci.quantity, ci.price_per_item, ci.created_at, ci.updated_at,
			p.id, p.name, p.short_description, p.description, p.material_id, p.main_image_id, p.category_id, p.created_at, p.updated_at,
			mi.id, mi.filename, mi.original_name, mi.path, mi.size_bytes, mi.mime_type, mi.uploaded_by, mi.created_at, mi.updated_at,
			pv.id, pv.product_id, pv.name, pv.color_id, pv.is_default, pv.created_at, pv.updated_at,
			c.id, c.name, c.image_id, c.custom, c.material_id, c.created_at, c.updated_at,
			s.id, s.name, s.product_id, s.base_price, s.a, s.b, s.c, s.d, s.e, s.f, s.created_at, s.updated_at
		FROM cart_items ci
		JOIN products p ON ci.product_id = p.id
		JOIN images mi ON p.main_image_id = mi.id
		JOIN product_variants pv ON ci.variant_id = pv.id
		JOIN colors c ON pv.color_id = c.id
		JOIN sizes s ON ci.size_id = s.id
		WHERE ci.cart_session_id = $1
		ORDER BY ci.created_at DESC
	`

	rows, err := q.db.Query(query, cartSessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart items: %w", err)
	}
	defer rows.Close()

	var items []models.CartItemResponse
	for rows.Next() {
		var item models.CartItemResponse
		var product models.Product
		var mainImage models.Image
		var variant models.ProductVariant
		var color models.Color
		var size models.Size
		var itemCreatedAt, itemUpdatedAt time.Time

		err := rows.Scan(
			&item.ID, &item.ProductID, &item.VariantID, &item.SizeID, &item.Quantity, &item.PricePerItem, &itemCreatedAt, &itemUpdatedAt,
			&product.ID, &product.Name, &product.ShortDescription, &product.Description, &product.MaterialID, &product.MainImageID, &product.CategoryID, &product.CreatedAt, &product.UpdatedAt,
			&mainImage.ID, &mainImage.Filename, &mainImage.OriginalName, &mainImage.Path, &mainImage.SizeBytes, &mainImage.MimeType, &mainImage.UploadedBy, &mainImage.CreatedAt, &mainImage.UpdatedAt,
			&variant.ID, &variant.ProductID, &variant.Name, &variant.ColorID, &variant.IsDefault, &variant.CreatedAt, &variant.UpdatedAt,
			&color.ID, &color.Name, &color.ImageID, &color.Custom, &color.MaterialID, &color.CreatedAt, &color.UpdatedAt,
			&size.ID, &size.Name, &size.ProductID, &size.BasePrice, &size.A, &size.B, &size.C, &size.D, &size.E, &size.F, &size.CreatedAt, &size.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan cart item: %w", err)
		}

		// Build response structures
		item.Product = models.ProductResponse{
			ID:               product.ID,
			Name:             product.Name,
			ShortDescription: product.ShortDescription,
			Description:      product.Description,
			MaterialID:       product.MaterialID,
			MainImageID:      product.MainImageID,
			CategoryID:       product.CategoryID,
			CreatedAt:        product.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
			MainImage: models.ImageResponse{
				ID:           mainImage.ID,
				Filename:     mainImage.Filename,
				OriginalName: mainImage.OriginalName,
				Path:         mainImage.Path,
				SizeBytes:    mainImage.SizeBytes,
				MimeType:     mainImage.MimeType,
				UploadedBy:   mainImage.UploadedBy,
				CreatedAt:    mainImage.CreatedAt.Format(time.RFC3339),
				UpdatedAt:    mainImage.UpdatedAt.Format(time.RFC3339),
			},
		}

		item.Variant = models.ProductVariantResponse{
			ID:        variant.ID,
			ProductID: variant.ProductID,
			Name:      variant.Name,
			ColorID:   variant.ColorID,
			IsDefault: variant.IsDefault,
			CreatedAt: variant.CreatedAt.Format(time.RFC3339),
			UpdatedAt: variant.UpdatedAt.Format(time.RFC3339),
			Color: models.ColorResponse{
				ID:         color.ID,
				Name:       color.Name,
				ImageID:    color.ImageID,
				Custom:     color.Custom,
				MaterialID: color.MaterialID,
				CreatedAt:  color.CreatedAt.Format(time.RFC3339),
				UpdatedAt:  color.UpdatedAt.Format(time.RFC3339),
			},
		}

		item.Size = models.SizeResponse{
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
		}

		item.CreatedAt = itemCreatedAt.Format(time.RFC3339)
		item.UpdatedAt = itemUpdatedAt.Format(time.RFC3339)
		item.TotalPrice = item.PricePerItem * float64(item.Quantity)

		// Get additional services for this item
		services, err := q.GetCartItemServices(item.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get cart item services: %w", err)
		}
		item.AdditionalServices = services

		items = append(items, item)
	}

	return items, nil
}

// GetCartItemServices gets additional services for a cart item
func (q *CartQueries) GetCartItemServices(cartItemID int) ([]models.AdditionalServiceResponse, error) {
	query := `
		SELECT a.id, a.name, a.description, a.price, a.created_at, a.updated_at
		FROM additional_services a
		JOIN cart_item_services cis ON a.id = cis.additional_service_id
		WHERE cis.cart_item_id = $1
		ORDER BY a.name
	`

	rows, err := q.db.Query(query, cartItemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart item services: %w", err)
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
		
		service.Images = []models.ImageResponse{} // Initialize empty slice
		services = append(services, service)
	}

	return services, nil
}

// AddCartItemServices adds additional services to a cart item
func (q *CartQueries) AddCartItemServices(cartItemID int, serviceIDs []int) error {
	if len(serviceIDs) == 0 {
		return nil
	}

	// First remove existing services
	_, err := q.db.Exec("DELETE FROM cart_item_services WHERE cart_item_id = $1", cartItemID)
	if err != nil {
		return fmt.Errorf("failed to remove existing services: %w", err)
	}

	// Add new services
	for _, serviceID := range serviceIDs {
		_, err := q.db.Exec("INSERT INTO cart_item_services (cart_item_id, additional_service_id) VALUES ($1, $2)",
			cartItemID, serviceID)
		if err != nil {
			return fmt.Errorf("failed to add service %d: %w", serviceID, err)
		}
	}

	return nil
}

// GetCartItemCount gets the total number of items in a cart
func (q *CartQueries) GetCartItemCount(cartSessionID int) (int, error) {
	query := `SELECT COALESCE(SUM(quantity), 0) FROM cart_items WHERE cart_session_id = $1`
	var count int
	err := q.db.QueryRow(query, cartSessionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get cart item count: %w", err)
	}
	return count, nil
}