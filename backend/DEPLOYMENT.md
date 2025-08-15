# NotSoFluffy Backend Deployment Guide

This guide covers deploying the NotSoFluffy backend using Docker with an external PostgreSQL database.

## Quick Start

```bash
# 1. Clone and navigate to backend directory
cd backend

# 2. Configure environment
cp .env.example .env
# Edit .env with your database and domain settings

# 3. Build and start
docker-compose up -d

# 4. Create admin user (optional)
docker-compose exec backend ./create-admin
```

Your backend will be available at `http://localhost:8080`

## Prerequisites

- Docker and Docker Compose installed
- External PostgreSQL database (AWS RDS, DigitalOcean, Google Cloud SQL, etc.)
- Domain configured (for production)
- SSL certificates (if using Nginx reverse proxy)

## Environment Configuration

### 1. Create Environment File

```bash
cp .env.example .env
```

### 2. Configure Required Variables

Edit `.env` and set these essential variables:

```env
# Database (REQUIRED)
DATABASE_URL=postgres://username:password@your-db-host:5432/notsofluffy?sslmode=require

# Security (REQUIRED)
JWT_SECRET=your-super-secret-jwt-key

# CORS (REQUIRED for production)
ALLOWED_ORIGINS=https://notsofluffy.pl,https://www.notsofluffy.com

# Domain (REQUIRED for production)
DOMAIN=notsofluffy.pl
```

### 3. Database SSL Configuration

Choose appropriate SSL mode based on your database provider:

#### Option A: Basic SSL (Recommended for most managed databases)
```env
DB_SSL_MODE=require
```

#### Option B: SSL with CA Verification
```env
DB_SSL_MODE=verify-ca
DB_SSL_ROOT_CERT=/app/ssl/certs/ca-certificate.crt
```

#### Option C: Full SSL Verification with Client Certificates
```env
DB_SSL_MODE=verify-full
DB_SSL_ROOT_CERT=/app/ssl/certs/ca-certificate.crt
DB_SSL_CERT=/app/ssl/certs/client-certificate.crt
DB_SSL_KEY=/app/ssl/certs/client-key.key
```

## Database Provider Examples

### AWS RDS
```env
DATABASE_URL=postgres://username:password@your-rds-endpoint.region.rds.amazonaws.com:5432/notsofluffy?sslmode=require
DB_SSL_MODE=require
```

### DigitalOcean Managed Database
```env
DATABASE_URL=postgres://username:password@your-db-cluster.db.ondigitalocean.com:25060/notsofluffy?sslmode=require
DB_SSL_MODE=require
```

### Google Cloud SQL
```env
DATABASE_URL=postgres://username:password@your-instance-ip:5432/notsofluffy?sslmode=require
DB_SSL_MODE=require
```

### Azure Database for PostgreSQL
```env
DATABASE_URL=postgres://username@servername:password@servername.postgres.database.azure.com:5432/notsofluffy?sslmode=require
DB_SSL_MODE=require
```

## Deployment Methods

### Method 1: Docker Compose (Recommended)

```bash
# Development
docker-compose up -d

# Production with custom configuration
docker-compose -f docker-compose.yml up -d

# View logs
docker-compose logs -f backend

# Stop services
docker-compose down
```

### Method 2: Direct Docker Run

```bash
# Build image
docker build -t notsofluffy-backend .

# Run container
docker run -d \
  --name notsofluffy-backend \
  --env-file .env \
  -p 8080:8080 \
  -v notsofluffy-uploads:/app/uploads \
  notsofluffy-backend
```

### Method 3: Production with Nginx Reverse Proxy

```bash
# Start backend
docker-compose up -d

# Configure Nginx (see ../nginx/README.md)
# Backend will be available at http://localhost:8080
# Nginx will proxy requests from https://yourdomain.com to the backend
```

## SSL Certificate Management (for Database)

### Mounting SSL Certificates

If using `verify-ca` or `verify-full` SSL modes, mount certificate files:

```yaml
# In docker-compose.yml
volumes:
  - uploads_data:/app/uploads
  - ./ssl/certs:/app/ssl/certs:ro  # Mount SSL certificates
```

Directory structure:
```
backend/
├── ssl/
│   └── certs/
│       ├── ca-certificate.crt
│       ├── client-certificate.crt
│       └── client-key.key
└── docker-compose.yml
```

### Obtaining SSL Certificates

#### AWS RDS
1. Download from: https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.SSL.html
2. Save as `ssl/certs/ca-certificate.crt`

#### DigitalOcean
1. Download from DigitalOcean Control Panel → Databases → Settings → SSL
2. Save certificates in `ssl/certs/`

#### Google Cloud SQL
1. Download from Google Cloud Console → SQL → Connections → SSL certificates
2. Save certificates in `ssl/certs/`

## Health Checks and Monitoring

### Health Check Endpoint

```bash
# Check if backend is healthy
curl http://localhost:8080/api/health

# Expected response:
{
  "status": "healthy",
  "timestamp": 1642345678,
  "service": "notsofluffy-api"
}
```

### Container Health Status

```bash
# Check container health
docker-compose ps

# View health check logs
docker inspect notsofluffy-backend | grep -A 10 Health
```

### Logging

```bash
# View real-time logs
docker-compose logs -f backend

# View last 100 lines
docker-compose logs --tail=100 backend

# Export logs
docker-compose logs backend > backend.log
```

## Admin User Management

### Create Admin User

```bash
# Interactive admin creation
docker-compose exec backend ./create-admin

# Follow prompts to enter email and password
```

### Alternative: Direct Database Access

If you need to create an admin user directly in the database:

```sql
-- Connect to your PostgreSQL database
-- Hash password with bcrypt (cost 12)
INSERT INTO users (email, password_hash, role, created_at, updated_at)
VALUES ('admin@yourdomain.com', '$2a$12$hashed_password_here', 'admin', NOW(), NOW());
```

## Production Considerations

### 1. Security

- **Change JWT Secret**: Generate with `openssl rand -base64 32`
- **Database SSL**: Always use `require` or higher in production
- **CORS Origins**: Limit to your actual frontend domains
- **Firewall**: Restrict access to port 8080 (use reverse proxy)

### 2. Performance

- **Resource Limits**: Adjust memory/CPU limits in docker-compose.yml
- **Database Connections**: Configure connection pooling
- **File Storage**: Consider external storage for uploads (AWS S3, etc.)

### 3. Backup Strategy

- **Database**: Use your cloud provider's backup features
- **Uploads**: Regular backup of uploads volume
- **Configuration**: Backup .env and certificate files

### 4. Monitoring

- **Health Checks**: Monitor `/api/health` endpoint
- **Log Aggregation**: Send logs to centralized logging (ELK, Splunk)
- **Metrics**: Monitor container resources and database connections

## Integration with Nginx SSL

This backend setup works seamlessly with the Nginx SSL configuration:

1. **Deploy backend** with Docker (this guide)
2. **Configure Nginx** with SSL (see `../nginx/README.md`)
3. **Update environment**:
   ```env
   ALLOWED_ORIGINS=https://yourdomain.com
   ENABLE_HTTPS=false  # Nginx handles SSL
   ```

## Troubleshooting

### Backend Won't Start

```bash
# Check logs
docker-compose logs backend

# Common issues:
# 1. Database connection failed
# 2. Invalid environment variables
# 3. Port already in use
```

### Database Connection Issues

```bash
# Test database connectivity
docker run --rm postgres:15 psql "postgres://username:password@host:5432/database" -c "SELECT 1;"

# Check SSL configuration
docker-compose exec backend ./server --test-db-connection
```

### SSL Certificate Issues

```bash
# Verify certificate files
ls -la ssl/certs/

# Check certificate validity
openssl x509 -in ssl/certs/ca-certificate.crt -text -noout
```

### Upload Issues

```bash
# Check uploads volume
docker volume inspect notsofluffy_uploads_data

# Fix permissions
docker-compose exec backend chown -R appuser:appgroup /app/uploads
```

### Performance Issues

```bash
# Monitor container resources
docker stats notsofluffy-backend

# Check database connections
docker-compose exec backend ./server --debug-db-connections
```

## Environment Variables Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | - | PostgreSQL connection string |
| `JWT_SECRET` | Yes | - | Secret key for JWT tokens |
| `ALLOWED_ORIGINS` | Yes | - | CORS allowed origins (comma-separated) |
| `DOMAIN` | Prod | localhost | Your domain name |
| `PORT` | No | 8080 | Server port |
| `GIN_MODE` | No | release | Gin framework mode |
| `DEVELOPMENT` | No | false | Enable development features |
| `ENABLE_HTTPS` | No | false | Enable direct HTTPS (use false with Nginx) |
| `DB_SSL_MODE` | No | require | Database SSL mode |
| `DB_SSL_CERT` | No | - | Client certificate file path |
| `DB_SSL_KEY` | No | - | Client key file path |
| `DB_SSL_ROOT_CERT` | No | - | CA certificate file path |

## Support

For additional help:

1. Check logs: `docker-compose logs backend`
2. Review environment configuration in `.env`
3. Verify database connectivity
4. Check SSL certificate configuration
5. Review nginx configuration (if using reverse proxy)

## Updates and Maintenance

### Updating the Backend

```bash
# Pull latest changes
git pull

# Rebuild and restart
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

### Database Migrations

```bash
# Migrations run automatically on startup
# To run manually:
docker-compose exec backend ./server --migrate-only
```

### Backup Uploads

```bash
# Create backup
docker run --rm -v notsofluffy_uploads_data:/data -v $(pwd):/backup alpine tar czf /backup/uploads-backup.tar.gz -C /data .

# Restore backup
docker run --rm -v notsofluffy_uploads_data:/data -v $(pwd):/backup alpine tar xzf /backup/uploads-backup.tar.gz -C /data
```
