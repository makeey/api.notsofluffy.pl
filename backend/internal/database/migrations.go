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
		`CREATE TABLE IF NOT EXISTS materials (
			id SERIAL PRIMARY KEY,
			name VARCHAR(256) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_materials_name ON materials(name);`,
		`DROP TRIGGER IF EXISTS update_materials_updated_at ON materials;`,
		`CREATE TRIGGER update_materials_updated_at
		BEFORE UPDATE ON materials
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`,
		`CREATE TABLE IF NOT EXISTS colors (
			id SERIAL PRIMARY KEY,
			name VARCHAR(256) NOT NULL,
			image_id INTEGER REFERENCES images(id) ON DELETE SET NULL,
			custom BOOLEAN NOT NULL DEFAULT false,
			material_id INTEGER NOT NULL REFERENCES materials(id) ON DELETE CASCADE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_colors_name ON colors(name);`,
		`CREATE INDEX IF NOT EXISTS idx_colors_material_id ON colors(material_id);`,
		`CREATE INDEX IF NOT EXISTS idx_colors_custom ON colors(custom);`,
		`CREATE INDEX IF NOT EXISTS idx_colors_image_id ON colors(image_id);`,
		`DROP TRIGGER IF EXISTS update_colors_updated_at ON colors;`,
		`CREATE TRIGGER update_colors_updated_at
		BEFORE UPDATE ON colors
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`,
		`CREATE TABLE IF NOT EXISTS additional_services (
			id SERIAL PRIMARY KEY,
			name VARCHAR(256) NOT NULL UNIQUE,
			description TEXT NOT NULL,
			price DECIMAL(10,2) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_additional_services_name ON additional_services(name);`,
		`CREATE INDEX IF NOT EXISTS idx_additional_services_price ON additional_services(price);`,
		`DROP TRIGGER IF EXISTS update_additional_services_updated_at ON additional_services;`,
		`CREATE TRIGGER update_additional_services_updated_at
		BEFORE UPDATE ON additional_services
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`,
		`CREATE TABLE IF NOT EXISTS additional_service_images (
			additional_service_id INTEGER NOT NULL REFERENCES additional_services(id) ON DELETE CASCADE,
			image_id INTEGER NOT NULL REFERENCES images(id) ON DELETE CASCADE,
			PRIMARY KEY (additional_service_id, image_id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_additional_service_images_service_id ON additional_service_images(additional_service_id);`,
		`CREATE INDEX IF NOT EXISTS idx_additional_service_images_image_id ON additional_service_images(image_id);`,
		`CREATE TABLE IF NOT EXISTS products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(256) NOT NULL,
			short_description VARCHAR(512) NOT NULL,
			description TEXT NOT NULL,
			material_id INTEGER REFERENCES materials(id) ON DELETE SET NULL,
			main_image_id INTEGER NOT NULL REFERENCES images(id) ON DELETE RESTRICT,
			category_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_products_name ON products(name);`,
		`CREATE INDEX IF NOT EXISTS idx_products_material_id ON products(material_id);`,
		`CREATE INDEX IF NOT EXISTS idx_products_main_image_id ON products(main_image_id);`,
		`CREATE INDEX IF NOT EXISTS idx_products_category_id ON products(category_id);`,
		`DROP TRIGGER IF EXISTS update_products_updated_at ON products;`,
		`CREATE TRIGGER update_products_updated_at
		BEFORE UPDATE ON products
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`,
		`CREATE TABLE IF NOT EXISTS product_images (
			product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			image_id INTEGER NOT NULL REFERENCES images(id) ON DELETE CASCADE,
			PRIMARY KEY (product_id, image_id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_product_images_product_id ON product_images(product_id);`,
		`CREATE INDEX IF NOT EXISTS idx_product_images_image_id ON product_images(image_id);`,
		`CREATE TABLE IF NOT EXISTS product_services (
			product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			additional_service_id INTEGER NOT NULL REFERENCES additional_services(id) ON DELETE CASCADE,
			PRIMARY KEY (product_id, additional_service_id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_product_services_product_id ON product_services(product_id);`,
		`CREATE INDEX IF NOT EXISTS idx_product_services_service_id ON product_services(additional_service_id);`,
		`CREATE TABLE IF NOT EXISTS sizes (
			id SERIAL PRIMARY KEY,
			name VARCHAR(256) NOT NULL,
			product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			base_price DECIMAL(10,2) NOT NULL,
			a DECIMAL(10,2) NOT NULL,
			b DECIMAL(10,2) NOT NULL,
			c DECIMAL(10,2) NOT NULL,
			d DECIMAL(10,2) NOT NULL,
			e DECIMAL(10,2) NOT NULL,
			f DECIMAL(10,2) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_sizes_product_id ON sizes(product_id);`,
		`CREATE INDEX IF NOT EXISTS idx_sizes_name ON sizes(name);`,
		`DROP TRIGGER IF EXISTS update_sizes_updated_at ON sizes;`,
		`CREATE TRIGGER update_sizes_updated_at
		BEFORE UPDATE ON sizes
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`,
		`CREATE TABLE IF NOT EXISTS product_variants (
			id SERIAL PRIMARY KEY,
			product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			name VARCHAR(256) NOT NULL,
			color_id INTEGER NOT NULL REFERENCES colors(id) ON DELETE CASCADE,
			is_default BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_product_variants_product_id ON product_variants(product_id);`,
		`CREATE INDEX IF NOT EXISTS idx_product_variants_color_id ON product_variants(color_id);`,
		`CREATE INDEX IF NOT EXISTS idx_product_variants_name ON product_variants(name);`,
		`CREATE INDEX IF NOT EXISTS idx_product_variants_is_default ON product_variants(is_default);`,
		`DROP TRIGGER IF EXISTS update_product_variants_updated_at ON product_variants;`,
		`CREATE TRIGGER update_product_variants_updated_at
		BEFORE UPDATE ON product_variants
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`,
		`CREATE TABLE IF NOT EXISTS product_variant_images (
			product_variant_id INTEGER NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
			image_id INTEGER NOT NULL REFERENCES images(id) ON DELETE CASCADE,
			PRIMARY KEY (product_variant_id, image_id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_product_variant_images_variant_id ON product_variant_images(product_variant_id);`,
		`CREATE INDEX IF NOT EXISTS idx_product_variant_images_image_id ON product_variant_images(image_id);`,
		
		// Cart tables
		`CREATE TABLE IF NOT EXISTS cart_sessions (
			id SERIAL PRIMARY KEY,
			session_id VARCHAR(255) UNIQUE NOT NULL,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_cart_sessions_session_id ON cart_sessions(session_id);`,
		`CREATE INDEX IF NOT EXISTS idx_cart_sessions_user_id ON cart_sessions(user_id);`,
		`DROP TRIGGER IF EXISTS update_cart_sessions_updated_at ON cart_sessions;`,
		`CREATE TRIGGER update_cart_sessions_updated_at
		BEFORE UPDATE ON cart_sessions
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`,
		
		`CREATE TABLE IF NOT EXISTS cart_items (
			id SERIAL PRIMARY KEY,
			cart_session_id INTEGER NOT NULL REFERENCES cart_sessions(id) ON DELETE CASCADE,
			product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			variant_id INTEGER NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
			size_id INTEGER NOT NULL REFERENCES sizes(id) ON DELETE CASCADE,
			quantity INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
			price_per_item DECIMAL(10, 2) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(cart_session_id, product_id, variant_id, size_id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_cart_items_cart_session_id ON cart_items(cart_session_id);`,
		`CREATE INDEX IF NOT EXISTS idx_cart_items_product_id ON cart_items(product_id);`,
		`DROP TRIGGER IF EXISTS update_cart_items_updated_at ON cart_items;`,
		`CREATE TRIGGER update_cart_items_updated_at
		BEFORE UPDATE ON cart_items
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`,
		
		`CREATE TABLE IF NOT EXISTS cart_item_services (
			cart_item_id INTEGER NOT NULL REFERENCES cart_items(id) ON DELETE CASCADE,
			additional_service_id INTEGER NOT NULL REFERENCES additional_services(id) ON DELETE CASCADE,
			PRIMARY KEY (cart_item_id, additional_service_id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_cart_item_services_cart_item_id ON cart_item_services(cart_item_id);`,
		`CREATE INDEX IF NOT EXISTS idx_cart_item_services_service_id ON cart_item_services(additional_service_id);`,
		
		// Orders tables
		`CREATE TABLE IF NOT EXISTS orders (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
			session_id VARCHAR(255),
			email VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			total_amount DECIMAL(10, 2) NOT NULL,
			subtotal DECIMAL(10, 2) NOT NULL,
			shipping_cost DECIMAL(10, 2) DEFAULT 0,
			tax_amount DECIMAL(10, 2) DEFAULT 0,
			payment_method VARCHAR(100),
			payment_status VARCHAR(50) DEFAULT 'pending',
			notes TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_orders_session_id ON orders(session_id);`,
		`CREATE INDEX IF NOT EXISTS idx_orders_email ON orders(email);`,
		`CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);`,
		`CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);`,
		
		// Add invoice fields to existing orders table
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS requires_invoice BOOLEAN DEFAULT false;`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS nip VARCHAR(20);`,
		
		// Create indexes for invoice fields after adding the columns
		`CREATE INDEX IF NOT EXISTS idx_orders_requires_invoice ON orders(requires_invoice);`,
		`CREATE INDEX IF NOT EXISTS idx_orders_nip ON orders(nip);`,
		
		`DROP TRIGGER IF EXISTS update_orders_updated_at ON orders;`,
		`CREATE TRIGGER update_orders_updated_at
		BEFORE UPDATE ON orders
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`,
		
		`CREATE TABLE IF NOT EXISTS shipping_addresses (
			id SERIAL PRIMARY KEY,
			order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
			first_name VARCHAR(100) NOT NULL,
			last_name VARCHAR(100) NOT NULL,
			company VARCHAR(100),
			address_line1 VARCHAR(255) NOT NULL,
			address_line2 VARCHAR(255),
			city VARCHAR(100) NOT NULL,
			state_province VARCHAR(100) NOT NULL,
			postal_code VARCHAR(20) NOT NULL,
			country VARCHAR(100) NOT NULL DEFAULT 'Poland',
			phone VARCHAR(50),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_shipping_addresses_order_id ON shipping_addresses(order_id);`,
		
		`CREATE TABLE IF NOT EXISTS billing_addresses (
			id SERIAL PRIMARY KEY,
			order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
			first_name VARCHAR(100) NOT NULL,
			last_name VARCHAR(100) NOT NULL,
			company VARCHAR(100),
			address_line1 VARCHAR(255) NOT NULL,
			address_line2 VARCHAR(255),
			city VARCHAR(100) NOT NULL,
			state_province VARCHAR(100) NOT NULL,
			postal_code VARCHAR(20) NOT NULL,
			country VARCHAR(100) NOT NULL DEFAULT 'Poland',
			phone VARCHAR(50),
			same_as_shipping BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_billing_addresses_order_id ON billing_addresses(order_id);`,
		
		`CREATE TABLE IF NOT EXISTS order_items (
			id SERIAL PRIMARY KEY,
			order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
			product_id INTEGER NOT NULL,
			product_name VARCHAR(255) NOT NULL,
			product_description TEXT,
			variant_id INTEGER NOT NULL,
			variant_name VARCHAR(255) NOT NULL,
			variant_color_name VARCHAR(100),
			variant_color_custom BOOLEAN DEFAULT FALSE,
			size_id INTEGER NOT NULL,
			size_name VARCHAR(100) NOT NULL,
			size_dimensions JSONB,
			quantity INTEGER NOT NULL,
			unit_price DECIMAL(10, 2) NOT NULL,
			total_price DECIMAL(10, 2) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);`,
		`CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id);`,
		
		`CREATE TABLE IF NOT EXISTS order_item_services (
			id SERIAL PRIMARY KEY,
			order_item_id INTEGER NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
			service_id INTEGER NOT NULL,
			service_name VARCHAR(255) NOT NULL,
			service_description TEXT,
			service_price DECIMAL(10, 2) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_order_item_services_order_item_id ON order_item_services(order_item_id);`,
		`CREATE INDEX IF NOT EXISTS idx_order_item_services_service_id ON order_item_services(service_id);`,
		
		// Add phone column to orders table if it doesn't exist
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS phone VARCHAR(50) NOT NULL DEFAULT '';`,
		`UPDATE orders SET phone = '' WHERE phone IS NULL;`,
		`CREATE INDEX IF NOT EXISTS idx_orders_phone ON orders(phone);`,
		
		// Add services_hash column to cart_items table for unique service combinations
		`ALTER TABLE cart_items ADD COLUMN IF NOT EXISTS services_hash VARCHAR(64) NOT NULL DEFAULT '';`,
		`UPDATE cart_items SET services_hash = '' WHERE services_hash IS NULL;`,
		// Drop old unique constraint and create new one with services_hash
		`ALTER TABLE cart_items DROP CONSTRAINT IF EXISTS cart_items_cart_session_id_product_id_variant_id_size_id_key;`,
		`ALTER TABLE cart_items DROP CONSTRAINT IF EXISTS cart_items_unique_with_services;`,
		`ALTER TABLE cart_items ADD CONSTRAINT cart_items_unique_with_services UNIQUE (cart_session_id, product_id, variant_id, size_id, services_hash);`,
		`CREATE INDEX IF NOT EXISTS idx_cart_items_services_hash ON cart_items(services_hash);`,

		// Add stock management columns to sizes table
		`ALTER TABLE sizes ADD COLUMN IF NOT EXISTS use_stock BOOLEAN NOT NULL DEFAULT FALSE;`,
		`ALTER TABLE sizes ADD COLUMN IF NOT EXISTS stock_quantity INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE sizes ADD COLUMN IF NOT EXISTS reserved_quantity INTEGER NOT NULL DEFAULT 0;`,
		`UPDATE sizes SET use_stock = FALSE, stock_quantity = 0, reserved_quantity = 0 WHERE use_stock IS NULL;`,
		`CREATE INDEX IF NOT EXISTS idx_sizes_use_stock ON sizes(use_stock);`,
		`CREATE INDEX IF NOT EXISTS idx_sizes_stock_quantity ON sizes(stock_quantity);`,

		// Add public_hash column to orders table for guest order access
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS public_hash VARCHAR(64) UNIQUE;`,
		`CREATE INDEX IF NOT EXISTS idx_orders_public_hash ON orders(public_hash);`,

		// User profiles table for extended user information
		`CREATE TABLE IF NOT EXISTS user_profiles (
			id SERIAL PRIMARY KEY,
			user_id INTEGER UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			first_name VARCHAR(100),
			last_name VARCHAR(100),
			phone VARCHAR(50),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_user_profiles_user_id ON user_profiles(user_id);`,
		`DROP TRIGGER IF EXISTS update_user_profiles_updated_at ON user_profiles;`,
		`CREATE TRIGGER update_user_profiles_updated_at
		BEFORE UPDATE ON user_profiles
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`,

		// User addresses table for saved shipping addresses
		`CREATE TABLE IF NOT EXISTS user_addresses (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			label VARCHAR(100) NOT NULL,
			first_name VARCHAR(100) NOT NULL,
			last_name VARCHAR(100) NOT NULL,
			company VARCHAR(100),
			address_line1 VARCHAR(255) NOT NULL,
			address_line2 VARCHAR(255),
			city VARCHAR(100) NOT NULL,
			state_province VARCHAR(100) NOT NULL,
			postal_code VARCHAR(20) NOT NULL,
			country VARCHAR(100) NOT NULL DEFAULT 'Poland',
			phone VARCHAR(50),
			is_default BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_user_addresses_user_id ON user_addresses(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_user_addresses_is_default ON user_addresses(is_default);`,
		`DROP TRIGGER IF EXISTS update_user_addresses_updated_at ON user_addresses;`,
		`CREATE TRIGGER update_user_addresses_updated_at
		BEFORE UPDATE ON user_addresses
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`,

		// Discount codes table
		`CREATE TABLE IF NOT EXISTS discount_codes (
			id SERIAL PRIMARY KEY,
			code VARCHAR(50) UNIQUE NOT NULL,
			description TEXT NOT NULL,
			discount_type VARCHAR(20) NOT NULL CHECK (discount_type IN ('percentage', 'fixed_amount')),
			discount_value DECIMAL(10,2) NOT NULL CHECK (discount_value > 0),
			min_order_amount DECIMAL(10,2) NOT NULL DEFAULT 0 CHECK (min_order_amount >= 0),
			usage_type VARCHAR(20) NOT NULL CHECK (usage_type IN ('one_time', 'once_per_user', 'unlimited')),
			max_uses INTEGER DEFAULT NULL CHECK (max_uses IS NULL OR max_uses > 0),
			used_count INTEGER NOT NULL DEFAULT 0 CHECK (used_count >= 0),
			active BOOLEAN NOT NULL DEFAULT true,
			start_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			end_date TIMESTAMP WITH TIME ZONE DEFAULT NULL,
			created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_discount_codes_code ON discount_codes(code);`,
		`CREATE INDEX IF NOT EXISTS idx_discount_codes_active ON discount_codes(active);`,
		`CREATE INDEX IF NOT EXISTS idx_discount_codes_usage_type ON discount_codes(usage_type);`,
		`CREATE INDEX IF NOT EXISTS idx_discount_codes_start_date ON discount_codes(start_date);`,
		`CREATE INDEX IF NOT EXISTS idx_discount_codes_end_date ON discount_codes(end_date);`,
		`DROP TRIGGER IF EXISTS update_discount_codes_updated_at ON discount_codes;`,
		`CREATE TRIGGER update_discount_codes_updated_at
		BEFORE UPDATE ON discount_codes
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`,

		// Discount code usage tracking table
		`CREATE TABLE IF NOT EXISTS discount_code_usage (
			id SERIAL PRIMARY KEY,
			discount_code_id INTEGER NOT NULL REFERENCES discount_codes(id) ON DELETE CASCADE,
			user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
			session_id VARCHAR(255),
			order_id INTEGER REFERENCES orders(id) ON DELETE SET NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_discount_code_usage_code_id ON discount_code_usage(discount_code_id);`,
		`CREATE INDEX IF NOT EXISTS idx_discount_code_usage_user_id ON discount_code_usage(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_discount_code_usage_session_id ON discount_code_usage(session_id);`,
		`CREATE INDEX IF NOT EXISTS idx_discount_code_usage_order_id ON discount_code_usage(order_id);`,

		// Add discount fields to cart_sessions table
		`ALTER TABLE cart_sessions ADD COLUMN IF NOT EXISTS applied_discount_code_id INTEGER REFERENCES discount_codes(id) ON DELETE SET NULL;`,
		`ALTER TABLE cart_sessions ADD COLUMN IF NOT EXISTS discount_amount DECIMAL(10,2) DEFAULT 0;`,
		
		// Add discount fields to orders table
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS discount_code_id INTEGER REFERENCES discount_codes(id) ON DELETE SET NULL;`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS discount_amount DECIMAL(10,2) DEFAULT 0;`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS discount_description TEXT;`,

		// Create indexes for new discount fields
		`CREATE INDEX IF NOT EXISTS idx_cart_sessions_discount_code ON cart_sessions(applied_discount_code_id);`,
		`CREATE INDEX IF NOT EXISTS idx_orders_discount_code ON orders(discount_code_id);`,

		// Site settings table for global configuration
		`CREATE TABLE IF NOT EXISTS site_settings (
			id SERIAL PRIMARY KEY,
			key VARCHAR(100) UNIQUE NOT NULL,
			value TEXT NOT NULL,
			description TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_site_settings_key ON site_settings(key);`,
		`DROP TRIGGER IF EXISTS update_site_settings_updated_at ON site_settings;`,
		`CREATE TRIGGER update_site_settings_updated_at
		BEFORE UPDATE ON site_settings
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`,

		// Insert default settings
		`INSERT INTO site_settings (key, value, description) VALUES 
		('maintenance_mode', 'false', 'Enable or disable maintenance mode for the site')
		ON CONFLICT (key) DO NOTHING;`,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("failed to run migration %d: %w", i+1, err)
		}
	}

	return nil
}