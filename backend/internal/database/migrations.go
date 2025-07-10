package database

import (
	"database/sql"
	"fmt"
)

func Migrate(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			role VARCHAR(50) NOT NULL DEFAULT 'client',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);`,
		`CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);`,
		`CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ language 'plpgsql';`,
		`DROP TRIGGER IF EXISTS update_users_updated_at ON users;`,
		`CREATE TRIGGER update_users_updated_at
		BEFORE UPDATE ON users
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`,
		`CREATE TABLE IF NOT EXISTS images (
			id SERIAL PRIMARY KEY,
			filename VARCHAR(255) NOT NULL,
			original_name VARCHAR(255) NOT NULL,
			path VARCHAR(500) NOT NULL,
			size_bytes BIGINT NOT NULL,
			mime_type VARCHAR(100) NOT NULL,
			uploaded_by INTEGER REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_images_uploaded_by ON images(uploaded_by);`,
		`CREATE INDEX IF NOT EXISTS idx_images_created_at ON images(created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_images_mime_type ON images(mime_type);`,
		`DROP TRIGGER IF EXISTS update_images_updated_at ON images;`,
		`CREATE TRIGGER update_images_updated_at
		BEFORE UPDATE ON images
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`,
		`CREATE TABLE IF NOT EXISTS categories (
			id SERIAL PRIMARY KEY,
			name VARCHAR(256) NOT NULL,
			slug VARCHAR(256) UNIQUE NOT NULL,
			image_id INTEGER REFERENCES images(id) ON DELETE SET NULL,
			active BOOLEAN NOT NULL DEFAULT true,
			chart_only BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug);`,
		`CREATE INDEX IF NOT EXISTS idx_categories_active ON categories(active);`,
		`CREATE INDEX IF NOT EXISTS idx_categories_chart_only ON categories(chart_only);`,
		`CREATE INDEX IF NOT EXISTS idx_categories_image_id ON categories(image_id);`,
		`DROP TRIGGER IF EXISTS update_categories_updated_at ON categories;`,
		`CREATE TRIGGER update_categories_updated_at
		BEFORE UPDATE ON categories
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("failed to run migration %d: %w", i+1, err)
		}
	}

	return nil
}