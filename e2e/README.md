# End-to-End Tests

This directory contains end-to-end tests for the wellness-nutrition application.

## Overview

These tests verify the integration between different components of the application, including:
- Authentication and authorization flows
- CSRF protection
- Middleware behavior
- Session management

## Running E2E Tests

E2E tests require a test database to be running.

### Setup Test Database

```bash
# Start test database
make test-docker-up

# Set environment variable
export DATABASE_URL="postgresql://postgres:test123@localhost:5433/test_db?sslmode=disable"
```

### Run E2E Tests

```bash
# Run with build tag
go test -tags=e2e ./e2e/...

# Or use make target
make test-e2e
```

### Clean Up

```bash
# Stop test database
make test-docker-down
```

## Test Structure

### TestLoginEndpointExists
Tests that the login endpoint is accessible and properly protected with CSRF.

### TestAuthenticationMiddleware
Tests that:
- Authenticated requests with valid session cookies succeed
- Unauthenticated requests are rejected
- User context is properly set in authenticated requests

### TestAdminAuthenticationMiddleware
Tests that:
- Admin users can access admin-only endpoints
- Regular users cannot access admin-only endpoints
- Authentication and authorization are properly enforced

### TestCSRFProtection
Tests that:
- Unsafe methods (POST, PUT, DELETE) without CSRF tokens are rejected
- Safe methods (GET) succeed and set CSRF cookies
- Requests with valid CSRF tokens succeed

## Build Tags

These tests use the `e2e` build tag to prevent them from running during regular test runs. This is because:
1. They require a database connection
2. They may take longer to execute
3. They test integration rather than individual units

To include them in your test runs, use the `-tags=e2e` flag.

## Writing E2E Tests

When writing new e2e tests:

1. **Use the e2e build tag** at the top of the file:
   ```go
   // +build e2e
   ```

2. **Check for short mode**:
   ```go
   if testing.Short() {
       t.Skip("Skipping e2e test in short mode")
   }
   ```

3. **Set up test database**:
   ```go
   db := testutil.SetupTestDB(t)
   defer testutil.CleanupTestDB(t, db)
   
   testutil.CreateTestSchema(t, db)
   defer testutil.DropTestSchema(t, db)
   ```

4. **Use httptest for HTTP testing**:
   ```go
   req := httptest.NewRequest("GET", "/endpoint", nil)
   w := httptest.NewRecorder()
   
   handler.ServeHTTP(w, req)
   
   if w.Code != http.StatusOK {
       t.Errorf("Expected 200, got %d", w.Code)
   }
   ```

## Best Practices

- **Test realistic scenarios**: E2E tests should mimic actual user interactions
- **Use subtests**: Organize related tests using `t.Run()`
- **Clean up between tests**: Use `defer` and cleanup functions
- **Test error cases**: Don't just test the happy path
- **Keep tests independent**: Tests should not depend on each other
- **Use descriptive test names**: Make it clear what each test is verifying

## Examples

### Testing a Protected Endpoint

```go
func TestProtectedEndpoint(t *testing.T) {
    // Setup
    db := testutil.SetupTestDB(t)
    defer testutil.CleanupTestDB(t, db)
    
    testutil.CreateTestSchema(t, db)
    defer testutil.DropTestSchema(t, db)
    
    userRepo := models.NewUserRepository(db)
    sessionStore := models.NewSessionStore(db)
    
    // Create test user
    user := &models.User{...}
    userRepo.Create(user)
    
    // Create session
    token, _ := sessionStore.CreateSession(user.ID)
    
    // Test
    req := httptest.NewRequest("GET", "/protected", nil)
    req.AddCookie(&http.Cookie{Name: "session", Value: token})
    w := httptest.NewRecorder()
    
    handler := middleware.Auth(sessionStore, userRepo)(protectedHandler)
    handler.ServeHTTP(w, req)
    
    // Verify
    if w.Code != http.StatusOK {
        t.Errorf("Expected 200, got %d", w.Code)
    }
}
```

### Testing CSRF Protection

```go
func TestCSRFProtection(t *testing.T) {
    // Get CSRF token
    getReq := httptest.NewRequest("GET", "/page", nil)
    getW := httptest.NewRecorder()
    
    csrfHandler := middleware.CSRF(handler)
    csrfHandler.ServeHTTP(getW, getReq)
    
    // Extract token from cookie
    var token string
    for _, c := range getW.Result().Cookies() {
        if c.Name == middleware.CSRFCookieName {
            token = c.Value
        }
    }
    
    // Use token in POST request
    postReq := httptest.NewRequest("POST", "/action", nil)
    postReq.Header.Set(middleware.CSRFHeaderName, token)
    postReq.AddCookie(&http.Cookie{
        Name: middleware.CSRFCookieName,
        Value: token,
    })
    postW := httptest.NewRecorder()
    
    csrfHandler.ServeHTTP(postW, postReq)
    
    // Should succeed with valid token
    if postW.Code != http.StatusOK {
        t.Errorf("Expected 200, got %d", postW.Code)
    }
}
```
