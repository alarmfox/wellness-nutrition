# Wellness & Nutrition - Go Application

This is the Go-based server-side rendered application for Wellness & Nutrition, migrated from Next.js.

## Features

- **Authentication**: JWT-based session authentication with secure signed cookies.
- **Role-based Authorization**: Strict separation between Admin and User roles.
- **Server-Side Rendering**: Fast, SEO-friendly HTML templates using Go's `html/template`.
- **Email Notifications**: Integrated mailer for welcome emails, booking notifications, and reminders.
- **WebSockets**: Real-time notifications for admin dashboard.
- **Responsive UI**: Interactive frontend built with Material UI.

## Architecture

The application is structured for simplicity and performance:
- `cmd/server`: Main entry point (Wiring, routing, and server).
- `handlers/`: Domain logic (Pages, Auth, Booking, Instructor, Survey).
- `models/`: Data structures and Repository implementations (PostgreSQL).
- `middleware/`: Security and Auth logic (RBAC, CSRF).
- `crypto/`: Centralized security primitives (Argon2id hashing, HMAC signing).
- `cmd/cleanup`: Periodic task to delete old data.
- `cmd/reminder`: Daily task to send booking reminders.

### Running with Docker

```yaml
services:
  app:
    build: .
    environment:
      - DATABASE_URL=postgres://user:pass@db:5432/wellness
      - SECRET_KEY=your-secure-key
      - ENVIRONMENT=production # Enables Secure cookies
...
```

## Security

The system is built with a "Security First" approach:

- **Password Hashing**: Argon2id (centralized in `crypto` package).
- **Session Management**: Cryptographically signed tokens using HMAC-SHA256.
- **CSRF Protection**: Double-submit cookie pattern for all state-changing operations.
- **RBAC**: Middleware enforces that Normal Users cannot access Admin routes or APIs.
- **Secure Cookies**: Controlled by `ENVIRONMENT` variable. Set to `production` to enable `Secure` flag. `SameSite` set to `Lax` for sessions and `Strict` for CSRF.
- **Safe SQL**: All database operations use parameterized queries or `lib/pq`'s safe array helpers to prevent SQL injection.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `SECRET_KEY` | Key for HMAC signing (min 32 chars) | Required |
| `ENVIRONMENT` | `production` or `development` | `development` |
| `LISTEN_ADDR` | Host and port to listen on | `localhost:3000` |
| `EMAIL_SERVER_*` | SMTP configuration for mailer | Required |

## Building and Running

### Database Migrations
```bash
go run cmd/migrations/migrate.go
```

### Database Seeding
```bash
go run cmd/seed/main.go
# Default Admin: admin@wellness.local / admin123
```

## Testing

The application features a comprehensive test suite including unit, integration, and permission-based tests.

### Running Tests
```bash
make test-unit         # Fast unit tests (no DB required)
make test-integration  # Integration tests (requires PostgreSQL)
make test              # Run all tests
```

### Test Infrastructure
- **Mocking**: Thread-safe mocks for Mailer, UserRepo, and BookingRepo in `testutil`.
- **Database**: Automatic schema management and table truncation for integration tests.
- **Permissions**: Dedicated tests in `middleware/permissions_test.go` verify RBAC integrity.

---
