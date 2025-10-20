# Database Seeder

This tool seeds the database with test data for development and testing purposes.

## What it Creates

### Users
- **1 Admin User**
  - Email: `admin@wellness.local`
  - Password: `admin123`
  - Role: ADMIN
  - Full access to all features

- **5 Regular Users**
  - Email: `*.@test.local` (e.g., `mario.rossi@test.local`)
  - Password: `password123`
  - Role: USER
  - Mix of SHARED and SINGLE subscription types
  - Different remaining accesses (5-15)

### Time Slots
- **30 days** of time slots starting from today
- **Hours**: 9:00 AM to 8:00 PM (every hour)
- **Days**: Monday through Saturday (no Sundays)
- **Total**: ~348 slots (29 days × 12 hours)

### Bookings
- **12-15 sample bookings** spread across the next week
- Each user has 2-3 bookings
- Bookings at different times to show variety
- People count automatically updated for each slot

## Usage

### Prerequisites
- PostgreSQL database running
- Database connection string

### Running the Seeder

```bash
# Using environment variable
export DATABASE_URL="postgresql://user:password@localhost:5432/wellness"
cd app/cmd/seed
go run .

# Using command line flag
cd app/cmd/seed
go run . -db-uri="postgresql://user:password@localhost:5432/wellness"

# Using compiled binary
cd app/cmd/seed
go build -o seed
./seed -db-uri="$DATABASE_URL"
```

### Output

The seeder provides clear output showing what was created:

```
Seeding database...
✓ Created admin user (email: admin@wellness.local, password: admin123)
✓ Created user Mario Rossi (email: mario.rossi@test.local, password: password123)
✓ Created user Laura Bianchi (email: laura.bianchi@test.local, password: password123)
...
✓ Created 348 time slots for the next 30 days
✓ Created 13 bookings for test users

=== Seeding Complete ===

Test Accounts:
  Admin: admin@wellness.local / admin123
  Users: *.@test.local / password123
    - mario.rossi@test.local
    - laura.bianchi@test.local
    - giuseppe.verdi@test.local
    - anna.romano@test.local
    - francesco.ferrari@test.local
```

## Safety

- Uses `ON CONFLICT DO NOTHING` for users and slots
- Won't create duplicate data if run multiple times
- Safe to run on existing databases

## Test Data Details

### Admin User
- Can access admin calendar view
- Can manage all users
- Can view all events
- Can see all bookings in calendar

### Regular Users
1. **Mario Rossi** - SHARED subscription, 10 accesses
2. **Laura Bianchi** - SINGLE subscription, 8 accesses
3. **Giuseppe Verdi** - SHARED subscription, 12 accesses
4. **Anna Romano** - SINGLE subscription, 5 accesses
5. **Francesco Ferrari** - SHARED subscription, 15 accesses

### Subscription Types
- **SHARED**: Users can share time slots (purple in calendar)
- **SINGLE**: Dedicated slots for individual user (blue in calendar)

## Testing Scenarios

After seeding, you can test:

1. **Admin Login**: Use `admin@wellness.local` / `admin123`
   - View calendar with all bookings
   - Week and month views
   - Different subscription types
   - Manage users
   - View events

2. **User Login**: Use any `*.@test.local` / `password123`
   - View own bookings only
   - Create new bookings
   - Delete own bookings
   - Cannot modify other users' bookings

3. **Authorization**: Try accessing admin endpoints as regular user
   - Should receive 403 Forbidden

4. **Booking Management**: Test booking creation and deletion
   - Updates people count
   - Creates events
   - Sends email notifications

## Cleanup

To remove test data:

```sql
-- Remove test users (will cascade delete bookings)
DELETE FROM "User" WHERE email LIKE '%@test.local' OR email = 'admin@wellness.local';

-- Remove all slots (will cascade delete bookings)
DELETE FROM "Slot";

-- Remove all events
DELETE FROM "Event";

-- Remove sessions
DELETE FROM sessions;
```

Or use a fresh database for a clean slate.

## Notes

- Passwords are hashed with Argon2id for security
- User IDs are randomly generated
- Bookings are spread across different days and times
- All users have verified email addresses
- Subscription expiry dates are set 6 months in the future for regular users
- Admin subscription is valid for 1 year
