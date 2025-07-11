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

## CRUD Implementation Architecture

This application implements a comprehensive CRUD (Create, Read, Update, Delete) system using a layered architecture pattern with raw SQL queries, strong typing, and consistent error handling across all entities.

### Backend CRUD Pattern

#### 1. Database Layer (Raw SQL Approach)
The application uses **raw SQL queries** without an ORM, providing direct control over database operations:

```go
// Example from internal/database/queries.go
func (q *Queries) GetCategories(ctx context.Context) ([]models.Category, error) {
    query := `
        SELECT c.id, c.name, c.image_id, c.is_active, c.created_at, c.updated_at,
               i.filename, i.original_name
        FROM categories c
        LEFT JOIN images i ON c.image_id = i.id
        ORDER BY c.name
    `
    // Implementation with prepared statements and proper error handling
}
```

**Key Features:**
- **Prepared Statements**: All queries use prepared statements for security
- **Transaction Support**: Complex operations use database transactions
- **Join Queries**: Efficient data retrieval with LEFT JOINs for related data
- **Contextual Queries**: All queries accept context for timeout and cancellation
- **SQL Injection Prevention**: Parameterized queries throughout

#### 2. Model Layer (Strong Typing)
Located in `internal/models/`, provides type definitions for all entities:

```go
// Example model structure
type Category struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    ImageID   *int      `json:"image_id"`
    IsActive  bool      `json:"is_active"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    // Nested related data
    Image     *Image    `json:"image,omitempty"`
}

// Request/Response models
type CreateCategoryRequest struct {
    Name     string `json:"name" binding:"required"`
    ImageID  *int   `json:"image_id"`
    IsActive bool   `json:"is_active"`
}
```

**Key Features:**
- **JSON Tags**: Proper serialization/deserialization
- **Validation Tags**: Request validation with Gin binding
- **Pointer Fields**: Optional fields use pointers for null handling
- **Nested Structures**: Related data embedded in responses
- **Request/Response Separation**: Dedicated types for API operations

#### 3. Handler Layer (HTTP Controllers)
Located in `internal/handlers/`, implements HTTP request handling:

```go
// Standard CRUD handler pattern
func (h *Handler) GetCategories(c *gin.Context) {
    categories, err := h.queries.GetCategories(c.Request.Context())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
        return
    }
    c.JSON(http.StatusOK, categories)
}

func (h *Handler) CreateCategory(c *gin.Context) {
    var req models.CreateCategoryRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    category, err := h.queries.CreateCategory(c.Request.Context(), req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
        return
    }
    
    c.JSON(http.StatusCreated, category)
}
```

**Key Features:**
- **Consistent Error Handling**: Standardized error responses
- **Request Validation**: Automatic validation with Gin binding
- **Context Propagation**: Request context passed to database layer
- **HTTP Status Codes**: Proper HTTP semantics
- **JSON Responses**: Consistent API response format

#### 4. Middleware Layer (Authentication & Authorization)
Located in `internal/middleware/`, provides cross-cutting concerns:

```go
// Authentication middleware
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := extractToken(c)
        user, err := validateToken(token)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
            c.Abort()
            return
        }
        c.Set("user", user)
        c.Next()
    }
}

// Role-based authorization
func RequireRole(role string) gin.HandlerFunc {
    return func(c *gin.Context) {
        user := c.MustGet("user").(*models.User)
        if user.Role != role {
            c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### Frontend CRUD Pattern

#### 1. API Client Layer (Type-Safe HTTP Client)
Located in `lib/api.ts`, provides centralized API communication:

```typescript
// Type-safe API client with automatic token handling
class ApiClient {
    private async request<T>(
        endpoint: string,
        options: RequestInit = {}
    ): Promise<T> {
        const token = this.getToken();
        const response = await fetch(`${this.baseUrl}${endpoint}`, {
            ...options,
            headers: {
                'Content-Type': 'application/json',
                'Authorization': token ? `Bearer ${token}` : '',
                ...options.headers,
            },
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        return response.json();
    }
    
    // CRUD operations
    async getCategories(): Promise<Category[]> {
        return this.request<Category[]>('/api/admin/categories');
    }
    
    async createCategory(data: CreateCategoryRequest): Promise<Category> {
        return this.request<Category>('/api/admin/categories', {
            method: 'POST',
            body: JSON.stringify(data),
        });
    }
}
```

**Key Features:**
- **TypeScript Types**: Full type safety for requests and responses
- **Automatic Authentication**: Token handling for all requests
- **Error Handling**: Centralized error handling with proper HTTP status codes
- **Request Interceptors**: Automatic JSON serialization
- **Response Interceptors**: Automatic JSON deserialization

#### 2. React Hooks (Data Management)
Custom hooks for CRUD operations:

```typescript
// Custom hooks for data fetching and mutations
export function useCategories() {
    const [categories, setCategories] = useState<Category[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    
    const fetchCategories = async () => {
        setLoading(true);
        try {
            const data = await apiClient.getCategories();
            setCategories(data);
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };
    
    const createCategory = async (categoryData: CreateCategoryRequest) => {
        try {
            const newCategory = await apiClient.createCategory(categoryData);
            setCategories(prev => [...prev, newCategory]);
            return newCategory;
        } catch (err) {
            setError(err.message);
            throw err;
        }
    };
    
    return { categories, loading, error, fetchCategories, createCategory };
}
```

### Entity-Specific CRUD Patterns

#### 1. Simple Entities (Materials, Categories)
- **Single Table**: Direct CRUD operations
- **Basic Relations**: Optional foreign keys (e.g., category image)
- **Validation**: Name uniqueness, required fields

#### 2. Complex Entities (Products)
- **Multiple Relations**: Foreign keys to materials, categories, images
- **Junction Tables**: Many-to-many relationships (product_images, product_services)
- **Transactional Updates**: Atomic operations for related data

#### 3. File Upload Entities (Images)
- **Multipart Upload**: Special handling for file uploads
- **UUID Filenames**: Secure filename generation
- **Metadata Storage**: Original filename, uploader tracking
- **Static File Serving**: Direct file serving with proper headers

#### 4. Junction Table Management
- **Atomic Operations**: Transaction-based updates for many-to-many relationships
- **Cascade Operations**: Proper cleanup when parent entities are deleted
- **Batch Operations**: Efficient bulk updates for large datasets

### Common CRUD Patterns

#### 1. Validation Strategy
- **Backend Validation**: Gin binding with struct tags
- **Frontend Validation**: TypeScript types + runtime validation
- **Database Constraints**: Foreign keys, unique constraints, check constraints

#### 2. Error Handling
- **Consistent Error Format**: Standardized error responses
- **HTTP Status Codes**: Proper semantic HTTP codes
- **Error Propagation**: Context-aware error handling

#### 3. Pagination
- **Offset-based Pagination**: Standard limit/offset pattern
- **Metadata**: Total count, page info in responses
- **Performance**: Indexed queries for large datasets

#### 4. Soft Deletes vs Hard Deletes
- **Hard Deletes**: Direct deletion for most entities
- **Soft Deletes**: Boolean flags for categories (is_active)
- **Cascade Rules**: Proper cleanup of related data

#### 5. Optimistic Updates
- **Frontend Optimism**: Immediate UI updates before API confirmation
- **Error Recovery**: Rollback on API failures
- **Conflict Resolution**: Proper handling of concurrent updates

### Security Considerations

#### 1. Authentication & Authorization
- **JWT Tokens**: Stateless authentication
- **Role-Based Access**: Admin-only CRUD operations
- **Token Refresh**: Automatic token renewal

#### 2. Data Validation
- **Input Sanitization**: Proper input validation
- **SQL Injection Prevention**: Parameterized queries
- **File Upload Security**: Type validation, size limits

#### 3. CORS & Security Headers
- **CORS Configuration**: Proper cross-origin handling
- **Security Headers**: Standard security headers
- **Rate Limiting**: Protection against abuse

This CRUD implementation provides a robust, scalable foundation for the e-commerce application with strong typing, proper error handling, and security best practices throughout the stack.

## Legacy Frontend Analysis (Django)

The repository contains a legacy Django application with valuable frontend patterns and design system knowledge that informs the current Next.js implementation.

### Legacy Technology Stack

#### Frontend Technologies
- **CSS Framework**: Tailwind CSS v3.4.0
- **JavaScript Libraries**:
  - Alpine.js v3.x (reactive UI framework)
  - HTMX v2.0.3 (HTML over the wire)
  - Flickity v2 (carousel/slider library)
- **Tailwind Plugins**:
  - @tailwindcss/forms v0.5.7
  - @tailwindcss/typography v0.5.10
  - @tailwindcss/aspect-ratio v0.4.2
- **Build Tools**: PostCSS with Tailwind CLI

#### Template Structure
- **Base Layout**: `loyaout/base.html` (Django template inheritance)
- **Component Structure**: Component-based templates in `components/` directory
- **Template Inheritance**: Django's `{% extends %}` and `{% block %}` pattern

### Design System & UI Patterns

#### Color Scheme
- **Primary**: Indigo (`indigo-600`, `indigo-500`)
- **Neutral**: Gray scale (`gray-900` to `gray-50`)
- **Success**: Green (`green-500`)
- **Background**: `bg-gray-900`, `bg-white`

#### Typography System
- **Headers**: `text-3xl`, `text-4xl` with `font-bold tracking-tight`
- **Body Text**: `text-sm`, `text-base`
- **Muted Text**: `text-gray-500`, `text-gray-600`

#### Spacing Patterns
- **Container Padding**: `px-4 sm:px-6 lg:px-8`
- **Section Padding**: `py-24 sm:py-32`
- **Max Width**: `max-w-7xl mx-auto`

#### Component Styling Patterns

**Buttons**:
- Primary: `bg-indigo-600 hover:bg-indigo-700 text-white`
- Secondary: `bg-white hover:bg-gray-100 text-gray-900`
- Common: `rounded-md px-8 py-3`

**Forms**:
- Using @tailwindcss/forms plugin
- Checkboxes: `rounded border-gray-300 text-indigo-600`
- Focus states: `focus:ring-indigo-500`

**Cards**:
- Structure: `border border-gray-200 bg-white rounded-lg`
- Hover states for interactive elements

### Layout Patterns

#### Navigation Structure
- **Desktop**: Flyout menus using Alpine.js
- **Mobile**: Off-canvas drawer pattern
- **Logo Placement**: Varies between mobile/desktop layouts

#### Hero Section
- **Full-width Hero**: Background image with opacity overlay
- **Conditional Display**: Based on `include_main_hero_in_header`
- **Gradient Overlays**: On category showcase images

#### Product Layout
- **Grid System**: `grid-cols-1 sm:grid-cols-2 xl:grid-cols-3`
- **Card-based Display**: Product cards with hover effects
- **Image Gallery**: Flickity carousel with thumbnail navigation

### Interactive Patterns (Alpine.js)

#### Component State Management
- **State**: `x-data` for component state
- **Events**: `@click`, `@change` for event handling
- **Conditional**: `x-show`, `x-if` for conditional rendering
- **Dynamic Classes**: `:class` for dynamic styling

#### Key Interactive Components
1. **Product Variant Selection**:
   - Color/size selection with visual feedback
   - URL parameter management
   - Dynamic price calculation

2. **Filter System**:
   - Checkbox-based filtering
   - URL parameter persistence
   - Alpine.js state management

3. **Shopping Cart**:
   - Multi-step product configuration
   - Order summary layout
   - Dynamic quantity updates

### Responsive Design Patterns

#### Breakpoint Strategy
- **Mobile-first**: Approach with `sm:`, `lg:`, `xl:` breakpoints
- **Adaptive Layouts**: Navigation and grid systems
- **Conditional Display**: Hidden/visible elements based on screen size

#### Grid Systems
- **Product Grids**: Responsive columns with consistent spacing
- **Category Showcase**: Adaptive grid layouts
- **Image Galleries**: Responsive carousel implementations

### UI/UX Patterns for Migration

#### E-commerce Specific
- **Category Showcase**: Grid layout with gradient overlays
- **Product Cards**: Price display, hover effects, image optimization
- **Shopping Cart**: Order summary, quantity controls, price calculation
- **Product Configuration**: Multi-step variant selection

#### Visual Effects
- **Backdrop Blur**: `backdrop-blur` effects
- **Gradient Overlays**: On images and hero sections
- **Hover States**: Opacity changes and transformations
- **Focus Indicators**: Ring focus states with proper colors

#### Polish Language Support
The legacy site includes Polish localization:
- "Ketegorii" (Categories)
- "Rozmiary" (Sizes)
- "Dodatkowe usługi" (Additional services)

### Migration Guidelines

#### Component Architecture
- **Template to Component**: Convert Django templates to React components
- **State Management**: Replace Alpine.js with React state/Context API
- **Styling**: Continue using Tailwind CSS (upgrade to v4)
- **Interactive Elements**: Replace HTMX with Next.js API routes

#### Design System Preservation
- **Color Scheme**: Maintain indigo primary color system
- **Typography**: Preserve font weights and sizing patterns
- **Spacing**: Continue using consistent padding/margin patterns
- **Component Styles**: Adapt button, form, and card styling patterns

#### Performance Considerations
- **Image Optimization**: Use Next.js Image component instead of standard img tags
- **Routing**: Convert Django URL patterns to Next.js file-based routing
- **Forms**: Implement React-based form handling with validation
- **Carousels**: Use modern React carousel libraries instead of Flickity

This legacy analysis provides the foundation for maintaining design consistency while modernizing the frontend architecture with Next.js and React.

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

