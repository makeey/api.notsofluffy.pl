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
- **Structure**:
  - `cmd/server/` - Application entry point
  - `cmd/create-admin/` - CLI tool for creating admin users
  - `internal/config/` - Configuration management
  - `internal/database/` - Database connection, migrations, and queries
  - `internal/models/` - Data models and request/response types
  - `internal/auth/` - JWT and password hashing utilities
  - `internal/handlers/` - HTTP request handlers
  - `internal/middleware/` - HTTP middleware (auth, CORS)

### Database Schema
- **Users Table**: id, email, password_hash, role, created_at, updated_at
- **Indexes**: email (unique), role
- **Triggers**: Automatic updated_at timestamp
- **Roles**: client, admin

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