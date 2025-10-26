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
- `cmd/remind`: application to send email day by day to users to remind their bookings. To be run periodically
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
      - EMAIL_SERVER_HOST=
      - EMAIL_SERVER_PORT=
      - EMAIL_SERVER_USER=
      - EMAIL_SERVER_PASSWORD=
      - EMAIL_SERVER_FROM=
      - EMAIL_NOTIFY_ADDRESS=
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d postgres"]
      interval: 10s
      retries: 5
      start_period: 30s
      timeout: 10s
 
  cron:
    build: .
    depends_on:
        db:
          condition: service_healthy
    command: >
      sh -c "echo '0 2 * * * /app/cleanup >> /var/log/cron.log 2>&1' > /etc/crontabs/root &&
             echo '0 0 1/1 * * /app/reminder >> /var/log/cron.log 2>&1' >> /etc/crontabs/root &&
             crond -f -l 2"
```

### Database

The application uses PostgreSQL with **lowercase snake_case table and column names**. All migrations are idempotent using `CREATE TABLE IF NOT EXISTS` statements.

**Tables:**
- `users` - User accounts and subscriptions
- `bookings` - User bookings linked to slots
- `events` - Event log for booking actions
- `sessions` - Session management for authentication

**Key Differences from Prisma:**
- Prisma uses PascalCase table names (`User`, `Booking`) and camelCase columns (`firstName`, `startsAt`)
- Go app uses lowercase snake_case (`users`, `bookings`, `first_name`, `starts_at`)
- Go app removes the `slots` table

**Running Migrations:**
```bash
cd app/migrations
go run .
```

### Authentication

- Sessions are stored in a `sessions` table
- Passwords are hashed using Argon2
- Session tokens are stored in secure HTTP-only cookies
- Session duration: 30 days

### Authorization

- **Public routes**: `/signin`, `/api/auth/login`
- **User routes**: Require authentication, accessible to both users and admins
  - `/` (home page)
  - `/api/user/current`
  - `/api/bookings/*`
- **Admin routes**: Require admin role
  - `/admin` (redirects to calendar)
  - `/admin/calendar` (admin calendar view with all bookings)
  - `/admin/users` (user management page)
  - `/admin/events` (event log page)
  - `/api/admin/users` (user CRUD endpoints)
  - `/api/admin/bookings` (get all bookings for calendar)

Admin API endpoints return a 403 Forbidden error if accessed by non-admin users.

### Views

The application provides two distinct views based on user role:

**Admin View** (`/admin/*`):
- Redirected to `/admin/calendar` on login
- `/admin/calendar` - Interactive calendar showing all bookings from all users
- `/admin/users` - User management page
- `/admin/events` - Event log page
- Week and month calendar views available
- Color-coded by subscription type (SHARED vs SINGLE)
- Access to user management and event logs
- Can view bookings owned by any user

**User View** (`/user`):
- Dashboard at `/user` showing only their own bookings
- Can create new bookings from available slots
- Can delete their own bookings
- Cannot view or modify other users' bookings
- Bottom navigation for mobile-friendly experience

The root path `/` automatically redirects users to their appropriate view based on role.

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
NEXTAUTH_URL=http://localhost:3000
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
