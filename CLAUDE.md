# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Structure

This is a full-stack application with a frontend-backend split architecture:

- `frontend/` - Next.js 15 application with TypeScript, React 19, and Tailwind CSS v4
- `backend/` - Go backend service with PostgreSQL database and JWT authentication
- `docker-compose.yml` - PostgreSQL database service configuration

## Development Commands

### Database Setup
Start the PostgreSQL database with Docker Compose:
```bash
docker-compose up -d
```

### Frontend Development
Run from the `frontend/` directory:
```bash
cd frontend
npm run dev    # Start development server with Turbopack (http://localhost:3000)
npm run build  # Build production application
npm run start  # Start production server
npm run lint   # Run ESLint linter
```

### Backend Development
Run from the `backend/` directory:
```bash
cd backend
go run cmd/server/main.go  # Start Go server (http://localhost:8080)
go mod tidy               # Update dependencies
go build -o server cmd/server/main.go  # Build production binary

# Admin user management
go run cmd/create-admin/main.go  # Create admin user interactively
```

### Development Server
The frontend development server uses Turbopack (`--turbopack` flag) for faster builds and hot reloading.

## Technology Stack

### Frontend
- **Framework**: Next.js 15.3.5 with App Router
- **Language**: TypeScript 5
- **UI**: React 19 with Tailwind CSS v4
- **Styling**: PostCSS with Tailwind CSS v4
- **Fonts**: Geist Sans and Geist Mono (via next/font/google)
- **Linting**: ESLint with Next.js TypeScript configuration
- **Authentication**: JWT tokens with React Context

### Backend
- **Language**: Go 1.23.7
- **Web Framework**: Gin
- **Database**: PostgreSQL 15 with raw SQL queries
- **Authentication**: JWT tokens with bcrypt password hashing
- **ORM**: None (using pure SQL with database/sql package)
- **Middleware**: CORS, Authentication middleware

### Configuration Files
- `tsconfig.json` - TypeScript configuration with strict mode, ES2017 target
- `eslint.config.mjs` - ESLint configuration extending Next.js core web vitals and TypeScript rules
- `next.config.ts` - Next.js configuration (currently minimal)
- `postcss.config.mjs` - PostCSS configuration for Tailwind CSS

## Code Architecture

### Frontend Structure
- **App Router**: Uses Next.js App Router (app directory)
- **Authentication**: React Context pattern with JWT token management
- **API Client**: Centralized API client with automatic token handling
- **Pages**: 
  - `app/layout.tsx` - Root layout with AuthProvider wrapper
  - `app/page.tsx` - Homepage component
  - `app/login/page.tsx` - Login page with form validation
  - `app/dashboard/page.tsx` - Protected dashboard page
- **Components**: Authentication context and hooks in `contexts/` and `hooks/`
- **API**: Type-safe API client in `lib/api.ts`

### Backend Structure
- **Clean Architecture**: Separated concerns with proper layering
- **Database Layer**: Raw SQL queries with prepared statements
- **Authentication**: JWT-based auth with middleware
- **File Uploads**: Image upload handling with file storage in `uploads/images/`
- **Static File Serving**: Serves uploaded files from `/uploads` endpoint
- **Structure**:
  - `cmd/server/` - Application entry point with route definitions
  - `cmd/create-admin/` - CLI tool for creating admin users
  - `internal/config/` - Configuration management
  - `internal/database/` - Database connection, migrations, and queries
  - `internal/models/` - Data models and request/response types
  - `internal/auth/` - JWT and password hashing utilities
  - `internal/handlers/` - HTTP request handlers (auth.go, admin.go)
  - `internal/middleware/` - HTTP middleware (auth, CORS)

### Key Features
- **E-commerce Product Management**: Full CRUD for products with categories, materials, colors, and additional services
- **Image Management**: Upload, list, and delete images with database tracking
- **Relational Data Handling**: Complex queries with junction tables for many-to-many relationships
- **Admin Dashboard Support**: All endpoints designed for admin panel operations
- **File Upload Security**: Images stored with UUID filenames to prevent conflicts

### Database Schema

#### Core Tables
- **users**: id, email, password_hash, role, created_at, updated_at
  - Indexes: email (unique), role
  - Roles: client, admin

- **images**: id, filename, original_name, uploaded_by, created_at, updated_at
  - References: uploaded_by → users(id)
  - Used for storing uploaded images

- **categories**: id, name, image_id, is_active, created_at, updated_at
  - References: image_id → images(id)
  - Used for product categorization

- **materials**: id, name, created_at, updated_at
  - Indexes: name
  - Used for product materials

- **colors**: id, name, image_id, material_id, created_at, updated_at
  - References: image_id → images(id), material_id → materials(id)
  - Indexes: name, material_id

- **additional_services**: id, name, description, price, created_at, updated_at
  - Indexes: name (unique), price
  - Used for extra services that can be added to products

- **products**: id, name, short_description, description, material_id, main_image_id, category_id, created_at, updated_at
  - References: material_id → materials(id), main_image_id → images(id), category_id → categories(id)
  - Indexes: name, material_id, main_image_id, category_id

#### Junction Tables
- **additional_service_images**: additional_service_id, image_id
  - References: additional_service_id → additional_services(id), image_id → images(id)
  - Primary Key: (additional_service_id, image_id)

- **product_images**: product_id, image_id
  - References: product_id → products(id), image_id → images(id)
  - Primary Key: (product_id, image_id)

- **product_services**: product_id, additional_service_id
  - References: product_id → products(id), additional_service_id → additional_services(id)
  - Primary Key: (product_id, additional_service_id)

#### Database Features
- All tables have automatic updated_at triggers
- Cascading deletes for junction tables
- Foreign key constraints for data integrity

### Authentication Flow
1. User registers/logs in via frontend
2. Backend validates credentials and returns JWT tokens
3. Frontend stores tokens and includes in API requests
4. Backend validates tokens via middleware
5. Refresh token mechanism for seamless experience

### Path Aliases
- `@/*` maps to `./` (frontend root) for cleaner imports

## Environment Variables

### Frontend (.env.local)
- `NEXT_PUBLIC_API_URL` - Backend API URL (default: http://localhost:8080)

### Backend (.env)
- `DATABASE_URL` - PostgreSQL connection string
- `JWT_SECRET` - Secret key for JWT signing
- `PORT` - Server port (default: 8080)

## API Endpoints

### Authentication
- `POST /api/auth/register` - User registration
- `POST /api/auth/login` - User login
- `POST /api/auth/refresh` - Token refresh
- `GET /api/auth/profile` - Get user profile (protected)

### Admin Endpoints (Protected - Admin Role Required)

#### User Management
- `GET /api/admin/users` - List all users
- `POST /api/admin/users` - Create new user
- `PUT /api/admin/users/:id` - Update user
- `DELETE /api/admin/users/:id` - Delete user

#### Image Management
- `POST /api/admin/images/upload` - Upload image (multipart/form-data)
- `GET /api/admin/images` - List all images
- `DELETE /api/admin/images/:id` - Delete image

#### Category Management
- `GET /api/admin/categories` - List categories
- `POST /api/admin/categories` - Create category
- `GET /api/admin/categories/:id` - Get category details
- `PUT /api/admin/categories/:id` - Update category
- `DELETE /api/admin/categories/:id` - Delete category
- `PATCH /api/admin/categories/:id/toggle` - Toggle category active status

#### Material Management
- `GET /api/admin/materials` - List materials
- `POST /api/admin/materials` - Create material
- `GET /api/admin/materials/:id` - Get material details
- `PUT /api/admin/materials/:id` - Update material
- `DELETE /api/admin/materials/:id` - Delete material

#### Color Management
- `GET /api/admin/colors` - List colors
- `POST /api/admin/colors` - Create color
- `GET /api/admin/colors/:id` - Get color details
- `PUT /api/admin/colors/:id` - Update color
- `DELETE /api/admin/colors/:id` - Delete color

#### Additional Services Management
- `GET /api/admin/additional-services` - List additional services
- `POST /api/admin/additional-services` - Create additional service
- `GET /api/admin/additional-services/:id` - Get service details
- `PUT /api/admin/additional-services/:id` - Update service
- `DELETE /api/admin/additional-services/:id` - Delete service

#### Product Management
- `GET /api/admin/products` - List products
- `POST /api/admin/products` - Create product
- `GET /api/admin/products/:id` - Get product details with all relations
- `PUT /api/admin/products/:id` - Update product
- `DELETE /api/admin/products/:id` - Delete product

### Static Files
- `GET /uploads/*` - Serve uploaded images and files

