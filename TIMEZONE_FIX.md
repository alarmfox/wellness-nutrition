# Timezone Handling Fix

## Problem
Time slots created by the seed script at 7 AM were appearing as 9 AM in the UI for users in GMT+2 timezone.

## Root Cause
The seed script was using `time.Local` to create slots, which used the server's local timezone. When the server ran in UTC (GMT+0) and created slots at 7 AM UTC, browsers in GMT+2 displayed them as 9 AM local time, causing a 2-hour offset.

## Solution
Changed all time creation in the seed script to explicitly use UTC (`time.UTC`) instead of `time.Local`. This ensures:

1. **Consistent Storage**: All slots are created at explicit UTC times (7 AM UTC, 8 AM UTC, etc.)
2. **Proper Conversion**: When browsers send booking requests in their local timezone (e.g., GMT+2), Go's `time.Parse(time.RFC3339, ...)` correctly converts them to UTC
3. **Correct Display**: When slots are returned to the browser in RFC3339 format, the browser automatically displays them in the user's local timezone

## Changes Made

### Files Modified
- `cmd/seed/main.go`: Changed all `time.Local` to `time.UTC` in both `seedTest()` and `seedSlot()` functions
- `cmd/seed/timezone_test.go`: Added comprehensive tests to verify timezone handling

### Code Changes
```go
// Before (incorrect):
now := time.Now()
startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
slotTime := time.Date(..., hour, 0, 0, 0, time.Local)

// After (correct):
now := time.Now().UTC()
startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
slotTime := time.Date(..., hour, 0, 0, 0, time.UTC)
```

## How It Works

### Time Flow
1. **Seed**: Creates slots at 7 AM UTC, 8 AM UTC, etc.
2. **Database**: PostgreSQL stores these as TIMESTAMP (without timezone)
3. **Client Request**: Browser in GMT+2 wants to book 9 AM local time, sends `"2024-01-15T09:00:00+02:00"`
4. **Server Parse**: `time.Parse(time.RFC3339, ...)` converts this to 7 AM UTC
5. **Database Query**: Finds the slot at 7 AM UTC
6. **Response**: Server returns `"2024-01-15T07:00:00Z"` (UTC)
7. **Client Display**: Browser converts to local timezone and shows 9 AM

### Example Scenario
- Server creates slot at 7 AM UTC
- User in GMT+2 sees it as 9 AM in their browser
- User in GMT-5 sees it as 2 AM in their browser
- Both users are booking the same absolute time, just displayed differently

## Testing
Run the timezone tests:
```bash
cd cmd/seed
go test -v .
```

All tests verify that:
- Times are created in UTC
- Timezone conversions work correctly
- Browser times are properly handled

## No Breaking Changes
This fix does not require changes to:
- Database schema (still uses TIMESTAMP)
- API endpoints (still use RFC3339)
- Client code (still sends/receives RFC3339)

The only change is that new slots created by the seed script will be at the correct UTC times. Existing slots in the database are not affected and will continue to work as-is.

## Migration Path
To fix existing slots in a production database:

1. **Option 1**: Delete and recreate slots using the updated seed script
2. **Option 2**: Run a SQL migration to adjust existing slot times (if needed)

For new installations, simply run the seed script and slots will be created correctly from the start.
