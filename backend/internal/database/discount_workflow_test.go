package database

import (
	"testing"
	"time"

	"notsofluffy-backend/internal/models"

	_ "github.com/lib/pq"
)

// TestDiscountWorkflowRaceCondition tests the complete workflow and potential race conditions
func TestDiscountWorkflowRaceCondition(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Initialize queries
	orderQueries := NewOrderQueries(db)
	discountQueries := NewDiscountQueries(db)
	cartQueries := NewCartQueries(db)

	// Clean up any existing test data
	cleanupTestData(t, db)
	_, _ = db.Exec("DELETE FROM discount_codes WHERE code = 'WORKFLOW10'")
	_, _ = db.Exec("DELETE FROM users WHERE email LIKE 'workflowuser%@example.com'")

	// Step 1: Create a "once per user" discount code
	discountCodeReq := &models.DiscountCodeRequest{
		Code:           "WORKFLOW10",
		Description:    "Test workflow 10% discount",
		DiscountType:   "percentage",
		DiscountValue:  10.0,
		MinOrderAmount: 0.0,
		UsageType:      "once_per_user",
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
		"workflowuser@example.com", "hashedpassword", "client",
	).Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	sessionID := "test-workflow-session"

	// Step 3: Test the complete workflow - Apply discount to cart
	cartSession, err := cartQueries.GetOrCreateCartSession(sessionID, &userID)
	if err != nil {
		t.Fatalf("Failed to create cart session: %v", err)
	}

	// Validate discount
	validation, err := discountQueries.ValidateDiscountCode("WORKFLOW10", 100.0, &userID, sessionID)
	if err != nil {
		t.Fatalf("Failed to validate discount code: %v", err)
	}
	if !validation.IsValid {
		t.Fatalf("Expected discount validation to succeed, got error: %s", validation.ErrorMessage)
	}

	// Apply discount to cart session
	err = discountQueries.ApplyDiscountToCartSession(cartSession.ID, discountCodeID, 10.0)
	if err != nil {
		t.Fatalf("Failed to apply discount to cart session: %v", err)
	}

	// Step 4: Try to apply the same discount again (should fail)
	validation2, err := discountQueries.ValidateDiscountCode("WORKFLOW10", 100.0, &userID, sessionID)
	if err != nil {
		t.Fatalf("Failed to validate discount code second time: %v", err)
	}
	
	// THIS IS THE KEY TEST: If the user can validate and apply the same code again,
	// it indicates the issue you're experiencing
	if validation2.IsValid {
		t.Logf("WARNING: User can validate the same 'once per user' discount code multiple times!")
		t.Logf("This indicates the bug where users can apply the same promocode multiple times")
		
		// Try to apply it again
		err = discountQueries.ApplyDiscountToCartSession(cartSession.ID, discountCodeID, 10.0)
		if err == nil {
			t.Error("CONFIRMED BUG: User can apply the same 'once per user' discount multiple times to cart!")
		}
	} else {
		t.Logf("Good: Second validation failed with message: %s", validation2.ErrorMessage)
	}

	// Step 5: Create order with the applied discount
	order := &models.Order{
		UserID:              &userID,
		SessionID:           &sessionID,
		Email:               "workflowuser@example.com",
		Phone:               "123456789",
		Status:              models.OrderStatusPending,
		TotalAmount:         90.0,
		Subtotal:            100.0,
		ShippingCost:        0.0,
		TaxAmount:           0.0,
		DiscountCodeID:      &discountCodeID,
		DiscountAmount:      10.0,
		DiscountDescription: stringPtr("WORKFLOW10: Test workflow 10% discount"),
		PaymentMethod:       stringPtr("test"),
		PaymentStatus:       models.PaymentStatusPending,
		RequiresInvoice:     false,
	}

	shippingAddr := &models.ShippingAddress{
		FirstName:     "Workflow",
		LastName:      "User",
		AddressLine1:  "123 Test St",
		City:          "Test City",
		StateProvince: "Test State",
		PostalCode:    "12345",
		Country:       "Test Country",
		Phone:         "123456789",
	}

	billingAddr := &models.BillingAddress{
		FirstName:      "Workflow",
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

	orderResponse, err := orderQueries.CreateOrder(order, shippingAddr, billingAddr, items)
	if err != nil {
		t.Fatalf("Failed to create order: %v", err)
	}

	// Step 6: Record discount usage (this should happen in order creation, but let's test it separately)
	err = discountQueries.RecordDiscountUsage(discountCodeID, &userID, sessionID, &orderResponse.ID)
	if err != nil {
		t.Errorf("Failed to record discount usage: %v", err)
		t.Logf("This could be the source of the bug - if usage recording fails silently!")
	}

	// Step 7: Now try the workflow again with a new cart session
	sessionID2 := "test-workflow-session-2"
	cartSession2, err := cartQueries.GetOrCreateCartSession(sessionID2, &userID)
	if err != nil {
		t.Fatalf("Failed to create second cart session: %v", err)
	}

	// Try to validate and apply the discount again
	validation3, err := discountQueries.ValidateDiscountCode("WORKFLOW10", 100.0, &userID, sessionID2)
	if err != nil {
		t.Fatalf("Failed to validate discount code for second session: %v", err)
	}

	if validation3.IsValid {
		t.Error("CRITICAL BUG: User can validate the same 'once per user' discount code in a new session after using it!")
		
		// Try to apply it to the new cart
		err = discountQueries.ApplyDiscountToCartSession(cartSession2.ID, discountCodeID, 10.0)
		if err == nil {
			t.Error("CRITICAL BUG CONFIRMED: User can apply the same discount to multiple cart sessions!")
		}
	} else {
		t.Logf("Good: Validation correctly failed for second session: %s", validation3.ErrorMessage)
	}

	// Cleanup
	_, _ = db.Exec("DELETE FROM discount_code_usage WHERE discount_code_id = $1", discountCodeID)
	_, _ = db.Exec("DELETE FROM cart_sessions WHERE session_id LIKE 'test-workflow%'")
	_, _ = db.Exec("DELETE FROM order_items WHERE order_id IN (SELECT id FROM orders WHERE email LIKE 'workflowuser%@example.com')")
	_, _ = db.Exec("DELETE FROM shipping_addresses WHERE order_id IN (SELECT id FROM orders WHERE email LIKE 'workflowuser%@example.com')")
	_, _ = db.Exec("DELETE FROM billing_addresses WHERE order_id IN (SELECT id FROM orders WHERE email LIKE 'workflowuser%@example.com')")
	_, _ = db.Exec("DELETE FROM orders WHERE email LIKE 'workflowuser%@example.com'")
	_, _ = db.Exec("DELETE FROM users WHERE email LIKE 'workflowuser%@example.com'")
	_, _ = db.Exec("DELETE FROM discount_codes WHERE code = 'WORKFLOW10'")
}

// TestErrorHandlingInOrderCreation tests what happens when RecordDiscountUsage fails
func TestErrorHandlingInOrderCreation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	orderQueries := NewOrderQueries(db)
	discountQueries := NewDiscountQueries(db)

	// This test simulates what happens if RecordDiscountUsage fails but the order still gets created
	// We can't easily simulate the failure without modifying the actual function,
	// but we can test that the error is properly handled

	// Create a discount code and user for testing
	cleanupTestData(t, db)
	_, _ = db.Exec("DELETE FROM discount_codes WHERE code = 'ERRORTEST10'")
	_, _ = db.Exec("DELETE FROM users WHERE email = 'erroruser@example.com'")

	discountCodeReq := &models.DiscountCodeRequest{
		Code:           "ERRORTEST10",
		Description:    "Test error handling",
		DiscountType:   "percentage",
		DiscountValue:  10.0,
		MinOrderAmount: 0.0,
		UsageType:      "once_per_user",
		StartDate:      time.Now(),
		EndDate:        nil,
		Active:         true,
		MaxUses:        nil,
	}

	discountCodeResp, err := discountQueries.CreateDiscountCode(discountCodeReq, 1)
	if err != nil {
		t.Fatalf("Failed to create discount code: %v", err)
	}

	var userID int
	err = db.QueryRow(
		"INSERT INTO users (email, password_hash, role, created_at, updated_at) VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING id",
		"erroruser@example.com", "hashedpassword", "client",
	).Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create an order with discount
	sessionID := "error-test-session"
	order := &models.Order{
		UserID:              &userID,
		SessionID:           &sessionID,
		Email:               "erroruser@example.com",
		Phone:               "123456789",
		Status:              models.OrderStatusPending,
		TotalAmount:         90.0,
		Subtotal:            100.0,
		ShippingCost:        0.0,
		TaxAmount:           0.0,
		DiscountCodeID:      &discountCodeResp.ID,
		DiscountAmount:      10.0,
		DiscountDescription: stringPtr("ERRORTEST10: Test error handling"),
		PaymentMethod:       stringPtr("test"),
		PaymentStatus:       models.PaymentStatusPending,
		RequiresInvoice:     false,
	}

	shippingAddr := &models.ShippingAddress{
		FirstName: "Error", LastName: "User", AddressLine1: "123 Test St",
		City: "Test City", StateProvince: "Test State", PostalCode: "12345",
		Country: "Test Country", Phone: "123456789",
	}

	billingAddr := &models.BillingAddress{
		FirstName: "Error", LastName: "User", AddressLine1: "123 Test St",
		City: "Test City", StateProvince: "Test State", PostalCode: "12345",
		Country: "Test Country", Phone: "123456789", SameAsShipping: true,
	}

	items := []models.OrderItem{
		{
			ProductID: 1, ProductName: "Test Product", VariantID: 1, VariantName: "Test Variant",
			SizeID: 1, SizeName: "Test Size", Quantity: 1, UnitPrice: 100.0, TotalPrice: 100.0,
			SizeDimensions: map[string]interface{}{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5, "f": 6},
		},
	}

	orderResponse, err := orderQueries.CreateOrder(order, shippingAddr, billingAddr, items)
	if err != nil {
		t.Fatalf("Failed to create order: %v", err)
	}

	// Test that RecordDiscountUsage works normally
	err = discountQueries.RecordDiscountUsage(discountCodeResp.ID, &userID, sessionID, &orderResponse.ID)
	if err != nil {
		t.Errorf("RecordDiscountUsage failed when it shouldn't: %v", err)
	}

	// Check that usage was recorded
	validation, err := discountQueries.ValidateDiscountCode("ERRORTEST10", 100.0, &userID, "new-session")
	if err != nil {
		t.Fatalf("Failed to validate discount code: %v", err)
	}
	if validation.IsValid {
		t.Error("Expected validation to fail after usage was recorded, but it succeeded")
	}

	// Cleanup
	_, _ = db.Exec("DELETE FROM discount_code_usage WHERE discount_code_id = $1", discountCodeResp.ID)
	_, _ = db.Exec("DELETE FROM order_items WHERE order_id = $1", orderResponse.ID)
	_, _ = db.Exec("DELETE FROM shipping_addresses WHERE order_id = $1", orderResponse.ID)
	_, _ = db.Exec("DELETE FROM billing_addresses WHERE order_id = $1", orderResponse.ID)
	_, _ = db.Exec("DELETE FROM orders WHERE id = $1", orderResponse.ID)
	_, _ = db.Exec("DELETE FROM users WHERE id = $1", userID)
	_, _ = db.Exec("DELETE FROM discount_codes WHERE id = $1", discountCodeResp.ID)
}