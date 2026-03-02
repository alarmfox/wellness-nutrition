# Wellness & Nutrition - Go Application

This is the Go-based server-side rendered application for Wellness & Nutrition, migrated from Next.js.

## Features

- **Authentication**: JWT-based session authentication with secure cookies
- **Role-based Authorization**: Users cannot access admin API endpoints
- **Server-Side Rendering**: HTML templates using Go's `html/template`
- **Email Notifications**: Mail template system for user notifications (welcome emails, booking notifications)
- **Material UI**: Frontend styling using Material UI CDN
- **API Endpoints**: RESTful API for user management, bookings, and events
- **Admin Calendar View**: Interactive calendar showing all bookings (week view)
- **User Dashboard**: Personalized view showing only user's own bookings
- **Database Seeder**: Tool to populate database with test data

## Architecture

Application has a centralized database:
- `cmd/server`: main server application
- `cmd/migrations`: database migrations
- `cmd/cleanup`: application to be run periodically to delete old bookings and events
- `cmd/reminder`: application to send email day by day to users to remind their bookings. To be run periodically
- `cmd/seed`: test application to populate database with test data

The application can be run with `docker compose`:

```yaml
services:
  app:
    build: .
    environment:
      - DATABASE_URL=
      - EMAIL_SERVER_HOST=
      - EMAIL_SERVER_PORT=
      - EMAIL_SERVER_USER=
      - EMAIL_SERVER_PASSWORD=
      - EMAIL_SERVER_FROM=
      - EMAIL_NOTIFY_ADDRESS=
      - AUTH_URL=
      - LISTEN_ADDR=
    depends_on:
      db:
        condition: service_healthy
        restart: true

  db:
    image: postgres:16.10-alpine
    environment:
      - POSTGRES_PASSWORD=
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d postgres"]
      interval: 10s
      retries: 5
      start_period: 30s
      timeout: 10s

  cron:
    build: .
    environment:
      - DATABASE_URL=
      - EMAIL_SERVER_HOST=
      - EMAIL_SERVER_PORT=
      - EMAIL_SERVER_USER=
      - EMAIL_SERVER_PASSWORD=
      - EMAIL_SERVER_FROM=
      - EMAIL_NOTIFY_ADDRESS=
    depends_on:
        db:
          condition: service_healthy
    command: >
      sh -c "echo '0 2 * * * /app/cleanup >> /var/log/cron.log 2>&1' > /etc/crontabs/root &&
             echo '0 0 1/1 * * /app/reminder >> /var/log/cron.log 2>&1' >> /etc/crontabs/root &&
             crond -f -l 2"
```



## Environment Variables

Required environment variables:

```bash
# Database
DATABASE_URL=postgresql://user:password@localhost:5432/dbname

# Email Server
EMAIL_SERVER_HOST=smtp.example.com
EMAIL_SERVER_PORT=587
EMAIL_SERVER_USER=user@example.com
EMAIL_SERVER_PASSWORD=password
EMAIL_FROM=noreply@example.com
EMAIL_NOTIFY_ADDRESS=admin@example.com

# Application
AUTH_URL=http://localhost:3000
SECRET_KEY=secret
LISTEN_ADDR=localhost:3000
```

## Building and Running

### Database Migrations

**First, run the migrations to create the database schema:**

```bash
go run cmd/migrations/migrate.go
```

This creates all required tables using **lowercase snake_case naming** with idempotent `CREATE TABLE IF NOT EXISTS` statements. Safe to run multiple times.

### Database Seeding (for Testing)

After running migrations, seed the database with test data:

```bash
go run cmd/seed/main.go
```

This creates:
- 1 admin user: `admin@wellness.local` / `admin123`
- 5 regular users: `*.@test.local` / `password123`
- 30 days of time slots (7 AM - 9 PM UTC, Mon-Sat)
- 12-15 sample bookings

**Note**: Time slots are created in UTC. Browsers automatically display them in the user's local timezone.

### Docker

Build a docker image:
```bash
docker build -t wellness-nutrition .
```

Run the application with `docker compose`:
```bash
docker compose up -d
```

## Testing

The application includes a comprehensive test infrastructure with unit tests, integration tests, and mock implementations.

### Running Tests

```bash
# Run all tests (unit + integration)
make test

# Run unit tests only (no database required)
make test-unit

# Run integration tests (requires database)
make test-integration

# Run tests with coverage report
make test-coverage

# Run tests with Docker database
make test-docker
```

### Test Structure

- **Unit Tests**: Use mock implementations for database and mail
- **Integration Tests**: Use a real PostgreSQL database
- **Mock Implementations**: Available in `testutil` package
  - `MockMailer`: Test email functionality without SMTP
  - `MockUserRepository`: Test user logic without database
  - `MockBookingRepository`: Test booking logic without database

### Setting Up Test Database

```bash
# Start test database
make test-docker-up

# Set environment variable
export DATABASE_URL="postgresql://postgres:test123@localhost:5433/test_db?sslmode=disable"

# Run tests
make test

# Stop test database
make test-docker-down
```

For more details on testing utilities and best practices, see [testutil/README.md](testutil/README.md).

## Security

- Passwords are hashed using Argon2id
- Sessions use cryptographically secure random tokens
- HTTP-only cookies prevent XSS attacks
- Admin endpoints are protected by role-based middleware
- SQL queries use parameterized statements to prevent SQL injection
- CSRF token in all forms and API
- Singed cookies with HMAC-SHA256

## Migration from Next.js

This application replaces the Next.js stack with:

- Go `net/http` server instead of Next.js
- Go `html/template` instead of React/JSX
- Custom authentication instead of NextAuth.js
- Direct PostgreSQL access instead of Prisma Client
- Native Go email instead of Nodemailer

The UI maintains the same Material UI styling using CDN links, and the database schema remains unchanged.
