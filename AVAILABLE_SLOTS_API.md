# GetAvailableSlots API Documentation

## Endpoint
`GET /api/user/bookings/slots`

## Authentication
Requires authenticated user session (cookie-based authentication)

## Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| instructorId | integer | Yes | The ID of the instructor to get available slots for |
| now | string (RFC3339) | No | Current time in RFC3339 format for timezone handling. If not provided, server time is used |

## Response

Returns a JSON object with available time slots as an array of RFC3339 timestamps:

```json
{
  "slots": [
    "2024-01-15T07:00:00Z",
    "2024-01-15T08:00:00Z",
    "2024-01-15T09:00:00Z"
  ]
}
```

### Response Fields

- `slots` (array of strings): Array of RFC3339 timestamp strings representing available time slots

## Business Logic

### Slot Generation
- Generates hourly slots from **7:00 AM to 9:00 PM** (inclusive)
- Only includes **Monday through Saturday** (no Sunday slots)
- Generates slots for **1 month** from the current time, or until the user's subscription expires (whichever comes first)

### Filtering Rules

A slot is **NOT included** in the response if:

1. **Capacity reached**: There are already 2 SIMPLE bookings for this instructor in this slot
2. **Blocked by admin**: There is a DISABLE, APPOINTMENT, or MASSAGE booking in this slot
3. **Single plan restriction**: The user has a SINGLE (dedicated) subscription plan and there is already 1 SIMPLE booking in this slot

### User Subscription Types

- **SHARED**: Users can book slots that have 0 or 1 existing booking (max 2 people per slot)
- **SINGLE**: Users can only book completely empty slots (dedicated access, no sharing)

## Example Requests

### Get slots for instructor ID 1
```bash
curl -X GET "http://localhost:3000/api/user/bookings/slots?instructorId=1" \
  -H "Cookie: session=xxx"
```

### Get slots with timezone-aware current time
```bash
curl -X GET "http://localhost:3000/api/user/bookings/slots?instructorId=1&now=2024-01-15T10:30:00Z" \
  -H "Cookie: session=xxx"
```

## Error Responses

### Missing instructor ID
```json
{
  "error": "instructorId is required"
}
```
Status: 400 Bad Request

### Invalid instructor ID
```json
{
  "error": "Invalid instructorId"
}
```
Status: 400 Bad Request

### Instructor not found
```json
{
  "error": "Instructor not found"
}
```
Status: 404 Not Found

### Unauthorized
```json
{
  "error": "Unauthorized"
}
```
Status: 401 Unauthorized

## Implementation Notes

1. All times are handled in UTC internally
2. The `now` parameter is provided by the frontend to ensure proper timezone handling
3. Slots are generated in memory (not stored in database)
4. The endpoint queries all bookings for the instructor in the date range to determine availability
5. Empty slots array is returned if no slots are available (not an error condition)
6. Response is kept simple - just an array of time strings, no additional metadata
