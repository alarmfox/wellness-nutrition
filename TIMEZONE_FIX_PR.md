# Booking Timezone Display Fix

## Problem Statement
Server-rendered bookings were showing UTC time instead of being converted to the user's local timezone in the browser. This caused confusion for users in different timezones who would see booking times that didn't match their local time.

## Root Cause
The Go server was formatting booking timestamps on the server side using:
```go
StartsAtFormatted: b.StartsAt.Format("02 Jan 2006 15:04")
```

This format string doesn't include timezone information, so the time was rendered in UTC (the server's timezone) rather than being passed to the browser for local conversion.

## Solution
Changed the approach to:
1. **Server**: Send timestamps in RFC3339 format (includes timezone info: `2025-10-22T09:00:00Z`)
2. **Browser**: Use JavaScript to convert UTC timestamps to the user's local timezone
3. **Display**: Format using browser's locale settings (`toLocaleDateString('it-IT', ...)`)

## Changes

### Backend Changes (cmd/server/main.go)

#### Before:
```go
type BookingDisplay struct {
    ID                 int64
    StartsAt           string  // RFC3339 for API
    StartsAtFormatted  string  // Pre-formatted UTC time
    CreatedAtFormatted string  // Pre-formatted UTC date
}

displayBookings = append(displayBookings, BookingDisplay{
    ID:                 b.ID,
    StartsAt:           b.StartsAt.Format(time.RFC3339),
    StartsAtFormatted:  b.StartsAt.Format("02 Jan 2006 15:04"),  // UTC time!
    CreatedAtFormatted: b.CreatedAt.Format("02 Jan 2006"),
})
```

#### After:
```go
type BookingDisplay struct {
    ID        int64
    StartsAt  string  // RFC3339 with timezone
    CreatedAt string  // RFC3339 with timezone
}

displayBookings = append(displayBookings, BookingDisplay{
    ID:        b.ID,
    StartsAt:  b.StartsAt.Format(time.RFC3339),  // Includes 'Z' suffix
    CreatedAt: b.CreatedAt.Format(time.RFC3339),
})
```

### Frontend Changes (cmd/server/templates/index.html)

#### Before:
```html
<div class="list-primary">{{.StartsAtFormatted}}</div>
<div class="list-secondary">Effettuata {{.CreatedAtFormatted}}</div>
```

#### After:
```html
<div class="list-primary" data-timestamp="{{.StartsAt}}"></div>
<div class="list-secondary" data-created="{{.CreatedAt}}"></div>

<script>
function formatTimestamps() {
    document.querySelectorAll('[data-timestamp]').forEach(el => {
        const timestamp = el.getAttribute('data-timestamp');
        if (timestamp) {
            const date = new Date(timestamp);
            el.textContent = date.toLocaleDateString('it-IT', {
                day: '2-digit',
                month: 'short',
                year: 'numeric',
                hour: '2-digit',
                minute: '2-digit'
            });
        }
    });
    // ... similar for data-created attributes
}

document.addEventListener('DOMContentLoaded', formatTimestamps);
</script>
```

## Example Behavior

### For a user in UTC+2 (e.g., Rome in summer):
- **Database**: Booking at `2025-10-22 09:00:00` (UTC)
- **Server sends**: `"2025-10-22T09:00:00Z"`
- **Browser displays**: "22 ott 2025, 11:00" (converted to UTC+2)

### For a user in UTC-5 (e.g., New York):
- **Database**: Same booking at `2025-10-22 09:00:00` (UTC)
- **Server sends**: `"2025-10-22T09:00:00Z"`
- **Browser displays**: "22 Oct 2025, 04:00" (converted to UTC-5)

## Benefits
✓ Users see times in their local timezone automatically
✓ No server-side timezone conversion needed
✓ Leverages browser's built-in timezone handling
✓ Works for all timezones without code changes
✓ Consistent with best practices (store UTC, display local)

## Testing
- Built successfully with no errors
- JavaScript formatting logic validated with test page
- CodeQL security scan: 0 alerts
- No breaking changes to API or database

## Files Modified
1. `cmd/server/main.go` - Removed server-side formatting
2. `cmd/server/templates/index.html` - Added client-side formatting
3. `cmd/server/templates/events.html` - Added client-side formatting

## Backward Compatibility
✓ No database schema changes
✓ No API endpoint changes
✓ Existing data works without migration
✓ Server still stores times in UTC
