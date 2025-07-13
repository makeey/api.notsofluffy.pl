package database

import (
	"database/sql"
	"testing"
	"time"

	"notsofluffy-backend/internal/models"

	_ "github.com/lib/pq"
)

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *sql.DB {
	// Use a test database connection string
	// In a real environment, you'd use a separate test database
	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/notsofluffy?sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Ping to ensure connection works
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	return db
}

// TestOrderDiscountCreationAndRetrieval tests the complete discount workflow
func TestOrderDiscountCreationAndRetrieval(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Initialize queries
	orderQueries := NewOrderQueries(db)
	discountQueries := NewDiscountQueries(db)

	// Clean up any existing test data
	cleanupTestData(t, db)

	// Step 1: Create a discount code
	discountCodeReq := &models.DiscountCodeRequest{
		Code:           "TEST10",
		Description:    "Test 10% discount",
		DiscountType:   "percentage",
		DiscountValue:  10.0,
		MinOrderAmount: 0.0,
		UsageType:      "unlimited",
		StartDate:      time.Now(),
		EndDate:        nil, // No end date
		Active:         true,
		MaxUses:        nil, // Unlimited
	}

	// Insert discount code
	discountCodeResp, err := discountQueries.CreateDiscountCode(discountCodeReq, 1) // createdBy = 1 (test user)
	if err != nil {
		t.Fatalf("Failed to create discount code: %v", err)
	}
	// Use the created discount code response directly
	discountCodeID := discountCodeResp.ID

	// Step 2: Create test order with discount
	order := &models.Order{
		SessionID:           stringPtr("test-session-123"),
		Email:               "test@example.com",
		Phone:               "123456789",
		Status:              models.OrderStatusPending,
		TotalAmount:         90.0,  // After 10% discount on 100
		Subtotal:            100.0, // Original amount
		ShippingCost:        0.0,
		TaxAmount:           0.0,
		DiscountCodeID:      &discountCodeID,
		DiscountAmount:      10.0, // 10% of 100
		DiscountDescription: stringPtr("TEST10: Test 10% discount"),
		PaymentMethod:       stringPtr("test"),
		PaymentStatus:       models.PaymentStatusPending,
		RequiresInvoice:     false,
	}

	// Create addresses
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
		SameAsShipping: false,
	}

	// Create test order items
	items := []models.OrderItem{
		{
			ProductID:       1,
			ProductName:     "Test Product",
			VariantID:       1,
			VariantName:     "Test Variant",
			SizeID:          1,
			SizeName:        "Test Size",
			Quantity:        1,
			UnitPrice:       100.0,
			TotalPrice:      100.0,
			SizeDimensions:  map[string]interface{}{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5, "f": 6},
		},
	}

	// Step 3: Create order
	orderResponse, err := orderQueries.CreateOrder(order, shippingAddr, billingAddr, items)
	if err != nil {
		t.Fatalf("Failed to create order: %v", err)
	}

	// Step 4: Verify order was created with correct discount information
	if orderResponse.DiscountCodeID == nil {
		t.Error("Expected DiscountCodeID to be set, got nil")
	} else if *orderResponse.DiscountCodeID != discountCodeID {
		t.Errorf("Expected DiscountCodeID %d, got %d", discountCodeID, *orderResponse.DiscountCodeID)
	}

	if orderResponse.DiscountAmount != 10.0 {
		t.Errorf("Expected DiscountAmount 10.0, got %f", orderResponse.DiscountAmount)
	}

	if orderResponse.DiscountDescription == nil {
		t.Error("Expected DiscountDescription to be set, got nil")
	} else if *orderResponse.DiscountDescription != "TEST10: Test 10% discount" {
		t.Errorf("Expected DiscountDescription 'TEST10: Test 10%% discount', got '%s'", *orderResponse.DiscountDescription)
	}

	if orderResponse.Subtotal != 100.0 {
		t.Errorf("Expected Subtotal 100.0, got %f", orderResponse.Subtotal)
	}

	if orderResponse.TotalAmount != 90.0 {
		t.Errorf("Expected TotalAmount 90.0, got %f", orderResponse.TotalAmount)
	}

	// Step 5: Test GetOrderByID retrieves discount information correctly
	retrievedOrder, err := orderQueries.GetOrderByID(orderResponse.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve order by ID: %v", err)
	}

	// Verify discount information is preserved
	if retrievedOrder.DiscountCodeID == nil {
		t.Error("Retrieved order: Expected DiscountCodeID to be set, got nil")
	} else if *retrievedOrder.DiscountCodeID != discountCodeID {
		t.Errorf("Retrieved order: Expected DiscountCodeID %d, got %d", discountCodeID, *retrievedOrder.DiscountCodeID)
	}

	if retrievedOrder.DiscountAmount != 10.0 {
		t.Errorf("Retrieved order: Expected DiscountAmount 10.0, got %f", retrievedOrder.DiscountAmount)
	}

	if retrievedOrder.DiscountDescription == nil {
		t.Error("Retrieved order: Expected DiscountDescription to be set, got nil")
	}

	// Step 6: Test GetOrderByHash retrieves discount information correctly
	if orderResponse.PublicHash == nil {
		t.Fatal("Expected PublicHash to be set for hash-based retrieval")
	}

	hashOrder, err := orderQueries.GetOrderByHash(*orderResponse.PublicHash)
	if err != nil {
		t.Fatalf("Failed to retrieve order by hash: %v", err)
	}

	// Verify discount information is preserved in hash-based retrieval
	if hashOrder.DiscountCodeID == nil {
		t.Error("Hash order: Expected DiscountCodeID to be set, got nil")
	} else if *hashOrder.DiscountCodeID != discountCodeID {
		t.Errorf("Hash order: Expected DiscountCodeID %d, got %d", discountCodeID, *hashOrder.DiscountCodeID)
	}

	if hashOrder.DiscountAmount != 10.0 {
		t.Errorf("Hash order: Expected DiscountAmount 10.0, got %f", hashOrder.DiscountAmount)
	}

	if hashOrder.DiscountDescription == nil {
		t.Error("Hash order: Expected DiscountDescription to be set, got nil")
	}

	// Step 7: Test order without discount (ensure fields are properly nil/zero)
	orderNoDiscount := &models.Order{
		SessionID:       stringPtr("test-session-no-discount"),
		Email:           "test2@example.com",
		Phone:           "987654321",
		Status:          models.OrderStatusPending,
		TotalAmount:     50.0,
		Subtotal:        50.0,
		ShippingCost:    0.0,
		TaxAmount:       0.0,
		PaymentMethod:   stringPtr("test"),
		PaymentStatus:   models.PaymentStatusPending,
		RequiresInvoice: false,
	}

	shippingAddr2 := &models.ShippingAddress{
		FirstName:     "Test2",
		LastName:      "User2",
		AddressLine1:  "456 Test Ave",
		City:          "Test City",
		StateProvince: "Test State",
		PostalCode:    "54321",
		Country:       "Test Country",
		Phone:         "987654321",
	}

	billingAddr2 := &models.BillingAddress{
		FirstName:      "Test2",
		LastName:       "User2",
		AddressLine1:   "456 Test Ave",
		City:           "Test City",
		StateProvince:  "Test State",
		PostalCode:     "54321",
		Country:        "Test Country",
		Phone:          "987654321",
		SameAsShipping: false,
	}

	items2 := []models.OrderItem{
		{
			ProductID:   1,
			ProductName: "Test Product 2",
			VariantID:   1,
			VariantName: "Test Variant 2",
			SizeID:      1,
			SizeName:    "Test Size 2",
			Quantity:    1,
			UnitPrice:      50.0,
			TotalPrice:     50.0,
			SizeDimensions: map[string]interface{}{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5, "f": 6},
		},
	}

	orderResponse2, err := orderQueries.CreateOrder(orderNoDiscount, shippingAddr2, billingAddr2, items2)
	if err != nil {
		t.Fatalf("Failed to create order without discount: %v", err)
	}

	// Verify no discount fields are set
	if orderResponse2.DiscountCodeID != nil {
		t.Errorf("Expected DiscountCodeID to be nil for order without discount, got %v", orderResponse2.DiscountCodeID)
	}

	if orderResponse2.DiscountAmount != 0.0 {
		t.Errorf("Expected DiscountAmount to be 0 for order without discount, got %f", orderResponse2.DiscountAmount)
	}

	if orderResponse2.DiscountDescription != nil {
		t.Errorf("Expected DiscountDescription to be nil for order without discount, got %v", orderResponse2.DiscountDescription)
	}

	// Cleanup
	cleanupTestData(t, db)
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// cleanupTestData removes test data
func cleanupTestData(t *testing.T, db *sql.DB) {
	// Clean up in reverse order of dependencies
	queries := []string{
		"DELETE FROM order_item_services WHERE order_item_id IN (SELECT id FROM order_items WHERE order_id IN (SELECT id FROM orders WHERE email LIKE 'test%@example.com'))",
		"DELETE FROM order_items WHERE order_id IN (SELECT id FROM orders WHERE email LIKE 'test%@example.com')",
		"DELETE FROM shipping_addresses WHERE order_id IN (SELECT id FROM orders WHERE email LIKE 'test%@example.com')",
		"DELETE FROM billing_addresses WHERE order_id IN (SELECT id FROM orders WHERE email LIKE 'test%@example.com')",
		"DELETE FROM orders WHERE email LIKE 'test%@example.com'",
		"DELETE FROM discount_codes WHERE code = 'TEST10'",
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			t.Logf("Cleanup warning: %v", err)
		}
	}
}