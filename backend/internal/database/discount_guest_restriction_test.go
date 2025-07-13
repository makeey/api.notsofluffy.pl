package database

import (
	"testing"
	"time"

	"notsofluffy-backend/internal/models"

	_ "github.com/lib/pq"
)

// TestGuestUserOncePerUserRestriction tests that guest users cannot use "once_per_user" discount codes
func TestGuestUserOncePerUserRestriction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Initialize queries
	discountQueries := NewDiscountQueries(db)

	// Clean up any existing test data
	cleanupTestData(t, db)
	_, _ = db.Exec("DELETE FROM discount_codes WHERE code = 'GUESTBLOCK10'")

	// Step 1: Create a "once_per_user" discount code
	discountCodeReq := &models.DiscountCodeRequest{
		Code:           "GUESTBLOCK10",
		Description:    "Test guest blocking for once per user discount",
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

	// Step 2: Test guest user validation (should fail)
	sessionID := "test-guest-restriction-session"
	
	validation, err := discountQueries.ValidateDiscountCode("GUESTBLOCK10", 100.0, nil, sessionID)
	if err != nil {
		t.Fatalf("Failed to validate discount code for guest: %v", err)
	}

	// Verify that guest user is blocked
	if validation.IsValid {
		t.Error("Expected guest validation to fail for 'once_per_user' discount, but it succeeded")
	}
	
	expectedErrorMessage := "This discount code requires you to be logged in. Please sign in to use this discount."
	if validation.ErrorMessage != expectedErrorMessage {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrorMessage, validation.ErrorMessage)
	}

	// Step 3: Create a registered user to verify the discount still works for them
	var userID int
	err = db.QueryRow(
		"INSERT INTO users (email, password_hash, role, created_at, updated_at) VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING id",
		"restricttest@example.com", "hashedpassword", "client",
	).Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Step 4: Test that registered user can still use the discount
	validation2, err := discountQueries.ValidateDiscountCode("GUESTBLOCK10", 100.0, &userID, sessionID)
	if err != nil {
		t.Fatalf("Failed to validate discount code for registered user: %v", err)
	}

	if !validation2.IsValid {
		t.Errorf("Expected registered user validation to succeed, got error: %s", validation2.ErrorMessage)
	}

	// Cleanup
	_, _ = db.Exec("DELETE FROM users WHERE id = $1", userID)
	_, _ = db.Exec("DELETE FROM discount_codes WHERE id = $1", discountCodeID)
}

// TestGuestUserUnlimitedDiscount tests that guest users can still use unlimited and one_time discounts
func TestGuestUserUnlimitedDiscount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Initialize queries
	discountQueries := NewDiscountQueries(db)

	// Clean up any existing test data
	cleanupTestData(t, db)
	_, _ = db.Exec("DELETE FROM discount_codes WHERE code LIKE 'GUESTOK%'")

	// Test 1: Unlimited discount
	unlimitedDiscountReq := &models.DiscountCodeRequest{
		Code:           "GUESTOK_UNLIMITED",
		Description:    "Test unlimited discount for guest",
		DiscountType:   "percentage",
		DiscountValue:  5.0,
		MinOrderAmount: 0.0,
		UsageType:      "unlimited",
		StartDate:      time.Now(),
		EndDate:        nil,
		Active:         true,
		MaxUses:        nil,
	}

	unlimitedDiscountResp, err := discountQueries.CreateDiscountCode(unlimitedDiscountReq, 1)
	if err != nil {
		t.Fatalf("Failed to create unlimited discount code: %v", err)
	}

	// Test guest user with unlimited discount
	sessionID1 := "test-guest-unlimited-session"
	validation1, err := discountQueries.ValidateDiscountCode("GUESTOK_UNLIMITED", 100.0, nil, sessionID1)
	if err != nil {
		t.Fatalf("Failed to validate unlimited discount code for guest: %v", err)
	}

	if !validation1.IsValid {
		t.Errorf("Expected guest validation to succeed for unlimited discount, got error: %s", validation1.ErrorMessage)
	}

	// Test 2: One time discount
	oneTimeDiscountReq := &models.DiscountCodeRequest{
		Code:           "GUESTOK_ONETIME",
		Description:    "Test one time discount for guest",
		DiscountType:   "fixed_amount",
		DiscountValue:  10.0,
		MinOrderAmount: 0.0,
		UsageType:      "one_time",
		StartDate:      time.Now(),
		EndDate:        nil,
		Active:         true,
		MaxUses:        nil,
	}

	oneTimeDiscountResp, err := discountQueries.CreateDiscountCode(oneTimeDiscountReq, 1)
	if err != nil {
		t.Fatalf("Failed to create one time discount code: %v", err)
	}

	// Test guest user with one time discount
	sessionID2 := "test-guest-onetime-session"
	validation2, err := discountQueries.ValidateDiscountCode("GUESTOK_ONETIME", 100.0, nil, sessionID2)
	if err != nil {
		t.Fatalf("Failed to validate one time discount code for guest: %v", err)
	}

	if !validation2.IsValid {
		t.Errorf("Expected guest validation to succeed for one time discount, got error: %s", validation2.ErrorMessage)
	}

	// Cleanup
	_, _ = db.Exec("DELETE FROM discount_codes WHERE id IN ($1, $2)", unlimitedDiscountResp.ID, oneTimeDiscountResp.ID)
}