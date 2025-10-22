# Browser Notification System

## Overview

The application now includes a browser notification system that sends reminders to users 6 hours before their scheduled bookings. This feature uses the browser's native Notification API to deliver timely reminders.

## Features

- **Automatic Permission Request**: When users access their dashboard, they are prompted to allow browser notifications
- **6-Hour Advance Notifications**: Users receive a notification exactly 6 hours before their booking time
- **Persistent Scheduling**: Bookings are stored in localStorage, allowing notifications to work even if the user closes and reopens the browser
- **Smart Tracking**: The system tracks which notifications have been sent to avoid duplicates
- **Automatic Cleanup**: Past bookings and their notification records are automatically cleaned up

## How It Works

### 1. Permission Request
When a user loads the dashboard (`/user`), the system automatically requests notification permission if it hasn't been granted yet.

### 2. Booking Synchronization
All future bookings are synchronized to localStorage under the key `wellness_bookings_notifications`. This data includes:
- Booking ID
- Start time (ISO 8601 format)
- Formatted start time for display

### 3. Periodic Checking
The system checks for upcoming bookings every 60 seconds (1 minute). This interval is configurable via the `CHECK_INTERVAL` constant.

### 4. Notification Trigger
A notification is sent when:
- The booking time is in the future
- The time until the booking is between 6 hours and 5 hours 59 minutes
- A notification hasn't already been sent for this booking

### 5. Notification Tracking
Sent notifications are tracked in localStorage under the key `wellness_notified_bookings` to prevent duplicate notifications.

## Technical Details

### Constants
```javascript
STORAGE_KEY: 'wellness_bookings_notifications'
CHECK_INTERVAL: 60000 // 1 minute in milliseconds
NOTIFICATION_ADVANCE_MS: 6 * 60 * 60 * 1000 // 6 hours in milliseconds
```

### Notification Content
- **Title**: "Promemoria Prenotazione" (Booking Reminder)
- **Body**: "La tua prenotazione è tra 6 ore: [formatted date/time]"
- **Icon**: Material Icons event icon
- **Tag**: Unique tag per booking to allow replacement if needed

### LocalStorage Schema

#### Bookings Storage (`wellness_bookings_notifications`)
```json
[
  {
    "id": "123",
    "startsAt": "2025-10-22T06:00:00Z",
    "startsAtFormatted": "22 Oct 2025 08:00"
  }
]
```

#### Notification Tracking (`wellness_notified_bookings`)
```json
["123", "456", "789"]
```

## Browser Compatibility

The notification system works in all modern browsers that support:
- `Notification` API
- `localStorage`
- ES6 JavaScript (arrow functions, template literals)

Supported browsers include:
- Chrome 22+
- Firefox 22+
- Safari 7+
- Edge 14+
- Opera 25+

## User Experience

### First Visit
1. User logs into the dashboard
2. Browser prompts for notification permission
3. User can allow or deny permissions

### Permission Granted
- Notifications will be sent 6 hours before bookings
- User can test notifications using the browser's developer tools

### Permission Denied
- No notifications will be sent
- User can manually enable notifications in browser settings
- All other features continue to work normally

## Privacy & Security

- **No Server Communication**: All notification scheduling happens client-side
- **No Personal Data Sharing**: Booking data is only stored in the user's browser
- **User Control**: Users can revoke notification permission at any time via browser settings
- **Auto Cleanup**: Notification data is automatically removed after bookings pass

## Maintenance

### Adjusting Notification Timing
To change the advance notification time, modify the `NOTIFICATION_ADVANCE_MS` constant in `/cmd/server/templates/index.html`:

```javascript
NOTIFICATION_ADVANCE_MS: 6 * 60 * 60 * 1000, // Change the 6 to desired hours
```

### Adjusting Check Frequency
To change how often the system checks for upcoming bookings, modify the `CHECK_INTERVAL` constant:

```javascript
CHECK_INTERVAL: 60000, // Change to desired milliseconds
```

**Note**: More frequent checks consume more battery on mobile devices.

## Testing

To test the notification system:

1. **Grant Permission**: Visit the user dashboard and allow notifications when prompted
2. **Create a Test Booking**: Create a booking 6 hours in the future
3. **Wait for Notification**: The notification should appear within 1 minute of the 6-hour mark
4. **Manual Test**: Use browser developer console:
   ```javascript
   NotificationManager.sendNotification({
     id: 'test',
     startsAtFormatted: 'Test Booking Time'
   });
   ```

## Troubleshooting

### Notifications Not Appearing
1. Check browser notification permission settings
2. Verify that notifications are not blocked by OS settings
3. Check browser console for JavaScript errors
4. Verify that bookings exist in localStorage: `localStorage.getItem('wellness_bookings_notifications')`

### Duplicate Notifications
1. Clear notification tracking: `localStorage.removeItem('wellness_notified_bookings')`
2. Refresh the page

### Testing Without Waiting 6 Hours
For development/testing, you can temporarily modify the `NOTIFICATION_ADVANCE_MS` to a shorter duration (e.g., 5 minutes = 5 * 60 * 1000).

## Future Enhancements

Potential improvements for future versions:
- Multiple notification times (e.g., 24 hours, 6 hours, 1 hour)
- User-configurable notification preferences
- Different notification sounds
- Rich notifications with action buttons (e.g., "View Details", "Cancel Booking")
- Service Worker for background notifications
- Push notifications for server-initiated alerts
