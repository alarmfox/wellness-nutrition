# Wellness & Nutrition - Go Application

This is the Go-based server-side rendered application for Wellness & Nutrition, migrated from Next.js.

## Features

- **Authentication**: JWT-based session authentication with secure cookies
- **Role-based Authorization**: Users cannot access admin API endpoints
- **Server-Side Rendering**: HTML templates using Go's `html/template`
- **Email Notifications**: Mail template system for user notifications (welcome emails, booking notifications)
- **Material UI**: Frontend styling using Material UI CDN
- **API Endpoints**: RESTful API for user management, bookings, and events

## Architecture

### Packages

- **`main.go`**: Entry point, HTTP server setup, and route configuration
- **`models/`**: Database models and repositories (User, Booking, Slot, Event)
- **`handlers/`**: HTTP request handlers for authentication, users, and bookings
- **`middleware/`**: Authentication and authorization middleware
- **`mail/`**: Email template system for sending notifications
- **`templates/`**: HTML templates for server-side rendering
- **`static/`**: Static assets (CSS, JavaScript)

### Database

The application uses PostgreSQL with the existing Prisma schema. All table names and column names match the Prisma schema exactly.

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
  - `/api/admin/users`
  - `/api/admin/users/create`

Admin API endpoints return a 403 Forbidden error if accessed by non-admin users.

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
```

## Building and Running

### Development

```bash
cd app
go run . -db-uri="postgresql://..." -listen-addr="localhost:3000"
```

### Production

```bash
cd app
go build -o wellness-nutrition .
./wellness-nutrition -db-uri="$DATABASE_URL" -listen-addr=":3000"
```

### Docker

```bash
docker build -t wellness-nutrition -f Dockerfile.app .
docker run -p 3000:3000 --env-file .env wellness-nutrition
```

## API Endpoints

### Authentication

- `POST /api/auth/login` - Login with email and password
- `POST /api/auth/logout` - Logout and clear session

### User (Protected)

- `GET /api/user/current` - Get current user information

### Bookings (Protected)

- `GET /api/bookings/current` - Get user's bookings
- `POST /api/bookings/create` - Create a new booking
- `POST /api/bookings/delete` - Delete a booking
- `GET /api/bookings/slots` - Get available time slots

### Admin (Admin Only)

- `GET /api/admin/users` - Get all users
- `POST /api/admin/users/create` - Create a new user

## Email Templates

The mail system sends HTML emails with a consistent design:

- **Welcome Email**: Sent when a user is created with account verification link
- **Reset Password Email**: Sent when a user requests password reset
- **New Booking Notification**: Sent to admin when a user creates a booking
- **Delete Booking Notification**: Sent to admin when a user deletes a booking

## Security

- Passwords are hashed using Argon2id
- Sessions use cryptographically secure random tokens
- HTTP-only cookies prevent XSS attacks
- Admin endpoints are protected by role-based middleware
- SQL queries use parameterized statements to prevent SQL injection

## Migration from Next.js

This application replaces the Next.js stack with:

- Go `net/http` server instead of Next.js
- Go `html/template` instead of React/JSX
- Custom authentication instead of NextAuth.js
- Direct PostgreSQL access instead of Prisma Client
- Native Go email instead of Nodemailer

The UI maintains the same Material UI styling using CDN links, and the database schema remains unchanged.
