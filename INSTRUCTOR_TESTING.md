# Per-Instructor Calendar Feature - Testing Guide

This document provides instructions for testing the new per-instructor calendar feature.

## Overview

The following functionality has been added to the Wellness & Nutrition application:

1. **Instructors Management**: Admins can create, update, and delete instructors
2. **Instructor Selection**: Users must select an instructor when creating a booking
3. **Per-Instructor Capacity**: Each instructor can have up to 2 people per time slot
4. **Calendar Filtering**: Admins can filter bookings by instructor in the calendar view
5. **Instructor Slots**: Separate tracking of bookings per instructor

## Database Changes

### New Tables

1. **instructors**: Stores instructor information
   - id (VARCHAR)
   - first_name (VARCHAR)
   - last_name (VARCHAR)
   - email (VARCHAR, UNIQUE)
   - password (TEXT, nullable)
   - created_at (TIMESTAMP)
   - updated_at (TIMESTAMP)

2. **instructor_slots**: Tracks capacity per instructor per time slot
   - instructor_id (VARCHAR)
   - starts_at (TIMESTAMP)
   - people_count (INTEGER, default 0)
   - max_capacity (INTEGER, default 2)
   - PRIMARY KEY (instructor_id, starts_at)

### Modified Tables

1. **bookings**: Added instructor_id column
   - instructor_id (VARCHAR, nullable)
   - Foreign key to instructors(id)

## Setup & Testing

### 1. Run Migrations

First, apply the database migration to add the new tables:

```bash
cd cmd/migrations
go run . -db-uri="$DATABASE_URL"
```

### 2. Seed Test Data

Run the seeder to create test data including instructors:

```bash
cd cmd/seed
go run . -seed=test -db-uri="$DATABASE_URL"
```

This will create:
- 3 instructors with credentials:
  - marco.bianchi@instructor.local / instructor123
  - giulia.ferrari@instructor.local / instructor123
  - alessandro.russo@instructor.local / instructor123
- Test bookings assigned to these instructors

### 3. Start the Server

```bash
cd cmd/server
go run . -db-uri="$DATABASE_URL" -listen-addr="localhost:3000"
```

## Testing Scenarios

### Admin Features

1. **Instructor Management**
   - Login as admin: admin@wellness.local / admin123
   - Navigate to "Istruttori" tab
   - Test creating a new instructor
   - Test editing an instructor
   - Test deleting an instructor

2. **Calendar Filtering**
   - Go to the Calendar view
   - Use the instructor dropdown to filter bookings
   - Verify that only bookings for the selected instructor are shown
   - Test switching between "Tutti" (all) and specific instructors

3. **Admin Booking Creation**
   - Try creating a booking for a user
   - Verify that instructor selection is required
   - Try creating multiple bookings for the same instructor/slot (should fail after 2)

### User Features

1. **Booking Creation with Instructor Selection**
   - Login as a test user (e.g., mario.rossi@test.local / password123)
   - Click "Prenota" to view available slots
   - Select a time slot
   - Verify that instructor selection screen appears
   - Select an instructor
   - Confirm the booking

2. **Instructor Slot Capacity**
   - Create bookings until an instructor's slot is full (2 bookings)
   - Try creating a third booking for the same instructor/slot
   - Verify that an error message appears: "This instructor's slot is full"

3. **Booking Display**
   - View your bookings in the user dashboard
   - Verify bookings are displayed correctly

## API Endpoints

### New Endpoints

- `GET /api/admin/instructors` - Get all instructors
- `POST /api/admin/instructors/create` - Create a new instructor
- `POST /api/admin/instructors/update` - Update an instructor
- `POST /api/admin/instructors/delete` - Delete an instructor
- `GET /admin/instructors` - Instructor management page

### Modified Endpoints

- `POST /api/user/bookings/create` - Now requires `instructorId` in request body
- `GET /api/admin/bookings` - Now supports `instructorId` query parameter for filtering
- `POST /api/admin/bookings/create` - Now requires `instructorId` in request body
- `GET /api/user/bookings/slots` - Now supports `instructorId` query parameter

## Expected Behavior

1. **Instructor Required**: Users cannot create a booking without selecting an instructor
2. **Capacity Management**: Each instructor can have max 2 people per slot
3. **Filtering**: Admins can view bookings for all instructors or filter by specific instructor
4. **Backward Compatibility**: Existing bookings without instructors should still work (instructor_id is nullable)

## Edge Cases to Test

1. Try creating a booking without selecting an instructor (should fail)
2. Try creating a booking when instructor slot is full (should fail)
3. Delete a booking and verify instructor slot count decrements
4. Filter calendar by instructor and verify correct bookings are shown
5. Create/edit/delete instructors and verify the changes reflect in booking creation

## Known Limitations

1. Instructors do not have a login/authentication system (they are just tags for bookings)
2. Instructors do not require email verification
3. The password field for instructors is optional and not currently used

## Troubleshooting

If you encounter issues:

1. Check that migrations have been applied successfully
2. Verify database connection string is correct
3. Check browser console for JavaScript errors
4. Review server logs for error messages
5. Ensure all instructors have been seeded properly

## Migration Path for Existing Data

If you have existing bookings without instructors:

1. The system will continue to work (instructor_id is nullable)
2. You can manually update existing bookings to assign instructors using SQL:

```sql
-- Example: Assign first instructor to all old bookings
UPDATE bookings 
SET instructor_id = (SELECT id FROM instructors LIMIT 1)
WHERE instructor_id IS NULL;
```

Or create a data migration script as needed.
