# Security Features

This document describes the security features implemented in the Wellness & Nutrition application.

## Overview

The application now includes comprehensive security measures to protect against common web vulnerabilities:

1. **Signed Cookies and Tokens** - All cookies and tokens are signed using HMAC-SHA256 to prevent tampering and replay attacks
2. **CSRF Protection** - Cross-Site Request Forgery protection for all state-changing operations
3. **Secure Session Management** - Sessions are cryptographically signed and verified on each request

## Signed Cookies and Tokens

### How it Works

All sensitive tokens (session cookies, email verification tokens, password reset tokens) are signed using HMAC-SHA256 with a secret key. The signature is appended to the token value:

```
Format: <data>.<signature>
Example: abc123def456.7f8e9d10a11b12c13d14e15f16
```

For tokens with expiration, the format includes a timestamp:

```
Format: <data>|<unix_timestamp>.<signature>
Example: abc123def456|1698765432.7f8e9d10a11b12c13d14e15f16
```

### Benefits

- **Tampering Prevention**: Any modification to the token invalidates the signature
- **Replay Attack Prevention**: Timed tokens expire after a specified duration
- **Integrity Verification**: The server can verify that the token was issued by the application

### Implementation Details

- Session tokens: 30 days expiration
- Email verification tokens: 7 days expiration  
- Password reset tokens: 1 hour expiration

## CSRF Protection

### How it Works

The application implements a double-submit cookie pattern for CSRF protection:

1. On GET requests, a CSRF token is generated and stored in a cookie
2. On POST/PUT/DELETE/PATCH requests, the token must be sent in either:
   - The `X-CSRF-Token` HTTP header (recommended for AJAX)
   - The `csrf_token` form field (for traditional form submissions)
3. The middleware validates that the token matches the cookie value

### Protected Routes

CSRF protection is applied to:

- All authentication endpoints (login, password reset, account verification)
- All authenticated user actions (bookings, profile updates)
- All admin actions (user management, instructor management, etc.)
- Survey submissions

### Exempted Routes

- Static file serving
- WebSocket connections (uses different authentication)

### JavaScript Integration

All fetch requests in the templates include the CSRF token:

```javascript
const csrfToken = getCookie('csrf_token');
fetch('/api/endpoint', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken,
    },
    body: JSON.stringify(data),
});
```

**Note**: The CSRF token cookie is NOT HttpOnly so JavaScript can read it for AJAX requests. This is safe because:
- CSRF tokens don't grant authentication privileges
- The cookie is protected by SameSite=Strict
- The token itself is cryptographically signed to prevent tampering

## Configuration

### Required Environment Variables

```bash
# Secret key for signing tokens and cookies
# Generate with: openssl rand -hex 32
SECRET_KEY=your-secret-key-here
```

**Important**: The SECRET_KEY must be:
- At least 32 characters long
- Randomly generated
- Kept secret and never committed to version control
- Changed if compromised

### Production Recommendations

1. **Use HTTPS**: Set `Secure` flag on cookies to true
2. **Rotate Secret Keys**: Periodically rotate the SECRET_KEY (will invalidate existing sessions)
3. **Monitor Failed Validations**: Log and monitor CSRF validation failures
4. **Rate Limiting**: Implement rate limiting on authentication endpoints

## Testing

The security features include comprehensive test coverage:

```bash
# Test crypto package
go test ./crypto -v

# Test CSRF middleware
go test ./middleware -v

# Run all tests
go test ./... -v
```

## Migration Notes

When deploying this update:

1. Set the `SECRET_KEY` environment variable before starting the application
2. Existing sessions will be invalidated (users will need to log in again)
3. Any pending email verification or password reset links will need to be resent

## Security Considerations

### What This Protects Against

- ✅ Session hijacking through cookie tampering
- ✅ Replay attacks using expired tokens
- ✅ Cross-Site Request Forgery (CSRF)
- ✅ Token forgery and manipulation

### What This Does NOT Protect Against

- ❌ XSS attacks (implement Content Security Policy)
- ❌ SQL injection (use parameterized queries - already implemented)
- ❌ Brute force attacks (implement rate limiting)
- ❌ Man-in-the-middle attacks (use HTTPS)

## Troubleshooting

### "CSRF token missing" or "Invalid CSRF token"

- Ensure cookies are enabled in the browser
- Check that the CSRF token cookie is being set on GET requests
- Verify JavaScript is correctly reading and sending the token

### "Invalid or expired verification token"

- Token may have expired (check expiration times)
- Token may have been tampered with
- SECRET_KEY may have changed since token was generated

### All users logged out after deployment

- This is expected when SECRET_KEY changes
- Users need to log in again to get new signed session tokens

## Code References

- Crypto package: `/crypto/crypto.go`
- CSRF middleware: `/middleware/csrf.go`
- Session management: `/models/session.go`
- Tests: `/crypto/crypto_test.go`, `/middleware/csrf_test.go`
