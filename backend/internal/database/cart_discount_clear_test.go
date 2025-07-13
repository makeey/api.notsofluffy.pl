package database

import (
	"testing"
	"time"

	"notsofluffy-backend/internal/models"

	_ "github.com/lib/pq"
)

// TestCartDiscountClearAfterOrder tests that discount information is properly cleared after order completion
func TestCartDiscountClearAfterOrder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Initialize queries
	orderQueries := NewOrderQueries(db)
	discountQueries := NewDiscountQueries(db)
	cartQueries := NewCartQueries(db)

	// Clean up any existing test data
	cleanupTestData(t, db)
	_, _ = db.Exec("DELETE FROM discount_codes WHERE code = 'CLEARCART10'")
	_, _ = db.Exec("DELETE FROM users WHERE email = 'cleartest@example.com'")

	// Step 1: Create a discount code
	discountCodeReq := &models.DiscountCodeRequest{
		Code:           "CLEARCART10",
		Description:    "Test cart clear 10% discount",
		DiscountType:   "percentage",
		DiscountValue:  10.0,
		MinOrderAmount: 0.0,
		UsageType:      "unlimited",
		StartDate:      time.Now(),
		EndDate:        nil,
		Active:         true,
		MaxUses:        nil,
	}

	discountCodeResp, err := discountQueries.CreateDiscountCode(discountCodeReq, 1)
	if err != nil {
		t.Fatalf("Failed to create discount code: %v", err)
	}
	discountCodeID := discountCodeResp.ID

	// Step 2: Create a test user
	var userID int
	err = db.QueryRow(
		"INSERT INTO users (email, password_hash, role, created_at, updated_at) VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING id",
		"cleartest@example.com", "hashedpassword", "client",
	).Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	sessionID := "test-clear-session"

	// Step 3: Create cart session and add items
	cartSession, err := cartQueries.GetOrCreateCartSession(sessionID, &userID)
	if err != nil {
		t.Fatalf("Failed to create cart session: %v", err)
	}

	// Add a test item to cart
	cartItemReq := &models.CartItemRequest{
		ProductID: 1,
		VariantID: 1,
		SizeID:    1,
		Quantity:  1,
	}
	_, err = cartQueries.AddCartItem(cartSession.ID, cartItemReq, 100.0)
	if err != nil {
		t.Fatalf("Failed to add item to cart: %v", err)
	}

	// Step 4: Apply discount to cart
	validation, err := discountQueries.ValidateDiscountCode("CLEARCART10", 100.0, &userID, sessionID)
	if err != nil {
		t.Fatalf("Failed to validate discount code: %v", err)
	}
	if !validation.IsValid {
		t.Fatalf("Expected discount validation to succeed, got error: %s", validation.ErrorMessage)
	}

	err = discountQueries.ApplyDiscountToCartSession(cartSession.ID, discountCodeID, 10.0)
	if err != nil {
		t.Fatalf("Failed to apply discount to cart session: %v", err)
	}

	// Step 5: Verify discount is applied to cart session
	cartSessionWithDiscount, err := cartQueries.GetCartSessionByID(sessionID)
	if err != nil {
		t.Fatalf("Failed to get cart session: %v", err)
	}

	if cartSessionWithDiscount.AppliedDiscountCodeID == nil {
		t.Error("Expected discount to be applied to cart session, but AppliedDiscountCodeID is nil")
	}
	if cartSessionWithDiscount.DiscountAmount == 0 {
		t.Error("Expected discount amount to be set, but it's 0")
	}

	t.Logf("Before order: AppliedDiscountCodeID=%v, DiscountAmount=%.2f", 
		*cartSessionWithDiscount.AppliedDiscountCodeID, cartSessionWithDiscount.DiscountAmount)

	// Step 6: Create order (which should clear the cart)
	order := &models.Order{
		UserID:              &userID,
		SessionID:           &sessionID,
		Email:               "cleartest@example.com",
		Phone:               "123456789",
		Status:              models.OrderStatusPending,
		TotalAmount:         90.0,
		Subtotal:            100.0,
		ShippingCost:        0.0,
		TaxAmount:           0.0,
		DiscountCodeID:      &discountCodeID,
		DiscountAmount:      10.0,
		DiscountDescription: stringPtr("CLEARCART10: Test cart clear 10% discount"),
		PaymentMethod:       stringPtr("test"),
		PaymentStatus:       models.PaymentStatusPending,
		RequiresInvoice:     false,
	}

	shippingAddr := &models.ShippingAddress{
		FirstName:     "Clear",
		LastName:      "Test",
		AddressLine1:  "123 Test St",
		City:          "Test City",
		StateProvince: "Test State",
		PostalCode:    "12345",
		Country:       "Test Country",
		Phone:         "123456789",
	}

	billingAddr := &models.BillingAddress{
		FirstName:      "Clear",
		LastName:       "Test",
		AddressLine1:   "123 Test St",
		City:           "Test City",
		StateProvince:  "Test State",
		PostalCode:     "12345",
		Country:        "Test Country",
		Phone:          "123456789",
		SameAsShipping: true,
	}

	items := []models.OrderItem{
		{
			ProductID:      1,
			ProductName:    "Test Product",
			VariantID:      1,
			VariantName:    "Test Variant",
			SizeID:         1,
			SizeName:       "Test Size",
			Quantity:       1,
			UnitPrice:      100.0,
			TotalPrice:     100.0,
			SizeDimensions: map[string]interface{}{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5, "f": 6},
		},
	}

	orderResponse, err := orderQueries.CreateOrder(order, shippingAddr, billingAddr, items)
	if err != nil {
		t.Fatalf("Failed to create order: %v", err)
	}

	// Step 7: Manually clear cart (simulating what happens in the order handler)
	err = cartQueries.ClearCart(cartSession.ID)
	if err != nil {
		t.Fatalf("Failed to clear cart: %v", err)
	}

	// Step 8: Verify discount information is cleared from cart session
	cartSessionAfterClear, err := cartQueries.GetCartSessionByID(sessionID)
	if err != nil {
		t.Fatalf("Failed to get cart session after clear: %v", err)
	}

	if cartSessionAfterClear.AppliedDiscountCodeID != nil {
		t.Errorf("Expected AppliedDiscountCodeID to be nil after cart clear, but got %v", 
			*cartSessionAfterClear.AppliedDiscountCodeID)
	}
	if cartSessionAfterClear.DiscountAmount != 0 {
		t.Errorf("Expected DiscountAmount to be 0 after cart clear, but got %.2f", 
			cartSessionAfterClear.DiscountAmount)
	}

	t.Logf("After clear: AppliedDiscountCodeID=%v, DiscountAmount=%.2f", 
		cartSessionAfterClear.AppliedDiscountCodeID, cartSessionAfterClear.DiscountAmount)

	// Step 9: Verify cart items are also cleared
	cartItems, err := cartQueries.GetCartItems(cartSession.ID)
	if err != nil {
		t.Fatalf("Failed to get cart items after clear: %v", err)
	}
	if len(cartItems) != 0 {
		t.Errorf("Expected cart to be empty after clear, but found %d items", len(cartItems))
	}

	// Step 10: Add new items to cart and verify no discount is applied
	newCartItemReq := &models.CartItemRequest{
		ProductID: 1,
		VariantID: 1,
		SizeID:    1,
		Quantity:  1,
	}
	_, err = cartQueries.AddCartItem(cartSession.ID, newCartItemReq, 100.0)
	if err != nil {
		t.Fatalf("Failed to add new item to cart: %v", err)
	}

	// Get updated cart session
	cartSessionWithNewItems, err := cartQueries.GetCartSessionByID(sessionID)
	if err != nil {
		t.Fatalf("Failed to get cart session with new items: %v", err)
	}

	// Verify no discount is automatically applied
	if cartSessionWithNewItems.AppliedDiscountCodeID != nil {
		t.Errorf("Expected no discount on new cart, but AppliedDiscountCodeID is %v", 
			*cartSessionWithNewItems.AppliedDiscountCodeID)
	}
	if cartSessionWithNewItems.DiscountAmount != 0 {
		t.Errorf("Expected no discount amount on new cart, but got %.2f", 
			cartSessionWithNewItems.DiscountAmount)
	}

	t.Logf("New cart: AppliedDiscountCodeID=%v, DiscountAmount=%.2f", 
		cartSessionWithNewItems.AppliedDiscountCodeID, cartSessionWithNewItems.DiscountAmount)

	// Cleanup
	_, _ = db.Exec("DELETE FROM discount_code_usage WHERE discount_code_id = $1", discountCodeID)
	_, _ = db.Exec("DELETE FROM cart_items WHERE cart_session_id = $1", cartSession.ID)
	_, _ = db.Exec("DELETE FROM cart_sessions WHERE session_id = $1", sessionID)
	_, _ = db.Exec("DELETE FROM order_items WHERE order_id = $1", orderResponse.ID)
	_, _ = db.Exec("DELETE FROM shipping_addresses WHERE order_id = $1", orderResponse.ID)
	_, _ = db.Exec("DELETE FROM billing_addresses WHERE order_id = $1", orderResponse.ID)
	_, _ = db.Exec("DELETE FROM orders WHERE id = $1", orderResponse.ID)
	_, _ = db.Exec("DELETE FROM users WHERE id = $1", userID)
	_, _ = db.Exec("DELETE FROM discount_codes WHERE id = $1", discountCodeID)
}

// TestCartDiscountClearGuestUser tests discount clearing for guest users
func TestCartDiscountClearGuestUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Initialize queries
	discountQueries := NewDiscountQueries(db)
	cartQueries := NewCartQueries(db)

	// Clean up any existing test data
	cleanupTestData(t, db)
	_, _ = db.Exec("DELETE FROM discount_codes WHERE code = 'GUESTCLEAR10'")

	// Step 1: Create a discount code
	discountCodeReq := &models.DiscountCodeRequest{
		Code:           "GUESTCLEAR10",
		Description:    "Test guest cart clear 10% discount",
		DiscountType:   "percentage",
		DiscountValue:  10.0,
		MinOrderAmount: 0.0,
		UsageType:      "unlimited",
		StartDate:      time.Now(),
		EndDate:        nil,
		Active:         true,
		MaxUses:        nil,
	}

	discountCodeResp, err := discountQueries.CreateDiscountCode(discountCodeReq, 1)
	if err != nil {
		t.Fatalf("Failed to create discount code: %v", err)
	}
	discountCodeID := discountCodeResp.ID

	sessionID := "test-guest-clear-session"

	// Step 2: Create guest cart session
	cartSession, err := cartQueries.GetOrCreateCartSession(sessionID, nil) // nil userID for guest
	if err != nil {
		t.Fatalf("Failed to create guest cart session: %v", err)
	}

	// Add a test item to cart
	cartItemReq := &models.CartItemRequest{
		ProductID: 1,
		VariantID: 1,
		SizeID:    1,
		Quantity:  1,
	}
	_, err = cartQueries.AddCartItem(cartSession.ID, cartItemReq, 100.0)
	if err != nil {
		t.Fatalf("Failed to add item to cart: %v", err)
	}

	// Step 3: Apply discount to guest cart
	validation, err := discountQueries.ValidateDiscountCode("GUESTCLEAR10", 100.0, nil, sessionID)
	if err != nil {
		t.Fatalf("Failed to validate discount code for guest: %v", err)
	}
	if !validation.IsValid {
		t.Fatalf("Expected discount validation to succeed for guest, got error: %s", validation.ErrorMessage)
	}

	err = discountQueries.ApplyDiscountToCartSession(cartSession.ID, discountCodeID, 10.0)
	if err != nil {
		t.Fatalf("Failed to apply discount to guest cart session: %v", err)
	}

	// Step 4: Verify discount is applied
	cartSessionWithDiscount, err := cartQueries.GetCartSessionByID(sessionID)
	if err != nil {
		t.Fatalf("Failed to get guest cart session: %v", err)
	}

	if cartSessionWithDiscount.AppliedDiscountCodeID == nil {
		t.Error("Expected discount to be applied to guest cart session")
	}

	// Step 5: Clear cart
	err = cartQueries.ClearCart(cartSession.ID)
	if err != nil {
		t.Fatalf("Failed to clear guest cart: %v", err)
	}

	// Step 6: Verify discount information is cleared
	cartSessionAfterClear, err := cartQueries.GetCartSessionByID(sessionID)
	if err != nil {
		t.Fatalf("Failed to get guest cart session after clear: %v", err)
	}

	if cartSessionAfterClear.AppliedDiscountCodeID != nil {
		t.Errorf("Expected AppliedDiscountCodeID to be nil after guest cart clear, but got %v", 
			*cartSessionAfterClear.AppliedDiscountCodeID)
	}
	if cartSessionAfterClear.DiscountAmount != 0 {
		t.Errorf("Expected DiscountAmount to be 0 after guest cart clear, but got %.2f", 
			cartSessionAfterClear.DiscountAmount)
	}

	// Cleanup
	_, _ = db.Exec("DELETE FROM cart_items WHERE cart_session_id = $1", cartSession.ID)
	_, _ = db.Exec("DELETE FROM cart_sessions WHERE session_id = $1", sessionID)
	_, _ = db.Exec("DELETE FROM discount_codes WHERE id = $1", discountCodeID)
}