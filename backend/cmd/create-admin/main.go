package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"notsofluffy-backend/internal/auth"
	"notsofluffy-backend/internal/config"
	"notsofluffy-backend/internal/database"
	"notsofluffy-backend/internal/models"

	"golang.org/x/term"
)

func main() {
	fmt.Println("Creating Super Admin User")
	fmt.Println("========================")

	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Run migrations to ensure database is up to date
	if err := database.Migrate(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	userQueries := database.NewUserQueries(db)
	reader := bufio.NewReader(os.Stdin)

	// Get email
	fmt.Print("Enter admin email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal("Failed to read email:", err)
	}
	email = strings.TrimSpace(email)

	if email == "" {
		log.Fatal("Email cannot be empty")
	}

	// Check if user already exists
	existingUser, err := userQueries.GetUserByEmail(email)
	if err == nil && existingUser != nil {
		fmt.Printf("User with email %s already exists.\n", email)
		fmt.Print("Do you want to update this user to admin role? (y/N): ")
		confirm, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal("Failed to read confirmation:", err)
		}
		confirm = strings.TrimSpace(strings.ToLower(confirm))
		
		if confirm != "y" && confirm != "yes" {
			fmt.Println("Operation cancelled.")
			return
		}

		// Update existing user to admin
		existingUser.Role = "admin"
		if err := userQueries.UpdateUser(existingUser); err != nil {
			log.Fatal("Failed to update user role:", err)
		}
		fmt.Printf("Successfully updated user %s to admin role.\n", email)
		return
	}

	// Get password
	fmt.Print("Enter admin password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal("Failed to read password:", err)
	}
	password := string(passwordBytes)
	fmt.Println() // New line after password input

	if len(password) < 6 {
		log.Fatal("Password must be at least 6 characters long")
	}

	// Confirm password
	fmt.Print("Confirm admin password: ")
	confirmPasswordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal("Failed to read password confirmation:", err)
	}
	confirmPassword := string(confirmPasswordBytes)
	fmt.Println() // New line after password input

	if password != confirmPassword {
		log.Fatal("Passwords do not match")
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	// Create admin user
	user := &models.User{
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         "admin",
	}

	if err := userQueries.CreateUser(user); err != nil {
		log.Fatal("Failed to create admin user:", err)
	}

	fmt.Printf("Successfully created admin user: %s\n", email)
	fmt.Printf("User ID: %d\n", user.ID)
	fmt.Printf("Created at: %s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))
}