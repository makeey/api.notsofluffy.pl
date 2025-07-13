package database

import (
	"testing"
	"time"

	"notsofluffy-backend/internal/models"

	_ "github.com/lib/pq"
)

// TestOncePerUserDiscountUsage tests that once per user discount codes can only be used once
func TestOncePerUserDiscountUsage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Initialize queries
	orderQueries := NewOrderQueries(db)
	discountQueries := NewDiscountQueries(db)

	// Clean up any existing test data
	cleanupTestData(t, db)
	
	// Additional cleanup for this specific test
	_, _ = db.Exec("DELETE FROM discount_codes WHERE code = 'ONCEUSER10'")
	_, _ = db.Exec("DELETE FROM users WHERE email LIKE 'testuser%@example.com'")

	// Step 1: Create a "once per user" discount code
	discountCodeReq := &models.DiscountCodeRequest{
		Code:           "ONCEUSER10",
		Description:    "Test once per user 10% discount",
		DiscountType:   "percentage",
		DiscountValue:  10.0,
		MinOrderAmount: 0.0,
		UsageType:      "once_per_user",
		StartDate:      time.Now(),
		EndDate:        nil,
		Active:         true,
		MaxUses:        nil,
	}

	// Insert discount code
	discountCodeResp, err := discountQueries.CreateDiscountCode(discountCodeReq, 1) // createdBy = 1 (test user)
	if err != nil {
		t.Fatalf("Failed to create discount code: %v", err)
	}
	discountCodeID := discountCodeResp.ID

	// Step 2: Create a test user first
	var userID int
	err = db.QueryRow(
		"INSERT INTO users (email, password_hash, role, created_at, updated_at) VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING id",
		"testuser100@example.com", "hashedpassword", "client",
	).Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	sessionID := "test-session-user-100"

	// First validation should succeed
	validation1, err := discountQueries.ValidateDiscountCode("ONCEUSER10", 100.0, &userID, sessionID)
	if err != nil {
		t.Fatalf("Failed to validate discount code: %v", err)
	}
	if !validation1.IsValid {
		t.Errorf("Expected first validation to be valid, got error: %s", validation1.ErrorMessage)
	}

	// Step 3: Create first order with the discount
	order1 := &models.Order{
		UserID:              &userID,
		SessionID:           &sessionID,
		Email:               "user100@example.com",
		Phone:               "123456789",
		Status:              models.OrderStatusPending,
		TotalAmount:         90.0,  // After 10% discount
		Subtotal:            100.0,
		ShippingCost:        0.0,
		TaxAmount:           0.0,
		DiscountCodeID:      &discountCodeID,
		DiscountAmount:      10.0,
		DiscountDescription: stringPtr("ONCEUSER10: Test once per user 10% discount"),
		PaymentMethod:       stringPtr("test"),
		PaymentStatus:       models.PaymentStatusPending,
		RequiresInvoice:     false,
	}

	shippingAddr := &models.ShippingAddress{
		FirstName:     "Test",
		LastName:      "User",
		AddressLine1:  "123 Test St",
		City:          "Test City",
		StateProvince: "Test State",
		PostalCode:    "12345",
		Country:       "Test Country",
		Phone:         "123456789",
	}

	billingAddr := &models.BillingAddress{
		FirstName:      "Test",
		LastName:       "User",
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

	orderResponse1, err := orderQueries.CreateOrder(order1, shippingAddr, billingAddr, items)
	if err != nil {
		t.Fatalf("Failed to create first order: %v", err)
	}

	// Step 4: Record discount usage (simulating what happens after order creation)
	err = discountQueries.RecordDiscountUsage(discountCodeID, &userID, sessionID, &orderResponse1.ID)
	if err != nil {
		t.Fatalf("Failed to record discount usage: %v", err)
	}

	// Step 5: Try to validate the same code again for the same user
	validation2, err := discountQueries.ValidateDiscountCode("ONCEUSER10", 100.0, &userID, sessionID)
	if err != nil {
		t.Fatalf("Failed to validate discount code second time: %v", err)
	}
	if validation2.IsValid {
		t.Error("Expected second validation to fail for same user, but it succeeded")
	}
	if validation2.ErrorMessage != "You have already used this discount code" {
		t.Errorf("Expected error message 'You have already used this discount code', got '%s'", validation2.ErrorMessage)
	}

	// Step 6: Create another test user
	var userID2 int
	err = db.QueryRow(
		"INSERT INTO users (email, password_hash, role, created_at, updated_at) VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING id",
		"testuser101@example.com", "hashedpassword", "client",
	).Scan(&userID2)
	if err != nil {
		t.Fatalf("Failed to create second test user: %v", err)
	}
	sessionID2 := "test-session-user-101"
	
	validation3, err := discountQueries.ValidateDiscountCode("ONCEUSER10", 100.0, &userID2, sessionID2)
	if err != nil {
		t.Fatalf("Failed to validate discount code for different user: %v", err)
	}
	if !validation3.IsValid {
		t.Errorf("Expected validation for different user to succeed, got error: %s", validation3.ErrorMessage)
	}

	// Step 7: Test guest user scenario (no user ID)
	sessionID3 := "test-session-guest-1"
	
	validationGuest1, err := discountQueries.ValidateDiscountCode("ONCEUSER10", 100.0, nil, sessionID3)
	if err != nil {
		t.Fatalf("Failed to validate discount code for guest: %v", err)
	}
	if !validationGuest1.IsValid {
		t.Errorf("Expected first guest validation to succeed, got error: %s", validationGuest1.ErrorMessage)
	}

	// Record guest usage
	err = discountQueries.RecordDiscountUsage(discountCodeID, nil, sessionID3, &orderResponse1.ID)
	if err != nil {
		t.Fatalf("Failed to record guest discount usage: %v", err)
	}

	// Try again with same guest session
	validationGuest2, err := discountQueries.ValidateDiscountCode("ONCEUSER10", 100.0, nil, sessionID3)
	if err != nil {
		t.Fatalf("Failed to validate discount code for guest second time: %v", err)
	}
	if validationGuest2.IsValid {
		t.Error("Expected second guest validation to fail, but it succeeded")
	}

	// Cleanup
	_, _ = db.Exec("DELETE FROM discount_code_usage WHERE discount_code_id = $1", discountCodeID)
	_, _ = db.Exec("DELETE FROM order_items WHERE order_id IN (SELECT id FROM orders WHERE email LIKE 'testuser%@example.com')")
	_, _ = db.Exec("DELETE FROM shipping_addresses WHERE order_id IN (SELECT id FROM orders WHERE email LIKE 'testuser%@example.com')")
	_, _ = db.Exec("DELETE FROM billing_addresses WHERE order_id IN (SELECT id FROM orders WHERE email LIKE 'testuser%@example.com')")
	_, _ = db.Exec("DELETE FROM orders WHERE email LIKE 'testuser%@example.com'")
	_, _ = db.Exec("DELETE FROM users WHERE email LIKE 'testuser%@example.com'")
	_, _ = db.Exec("DELETE FROM discount_codes WHERE code = 'ONCEUSER10'")
}