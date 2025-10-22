# User View Simulation Feature

## Overview
This feature allows administrators to see the User POV (user page) from a navigation link in the admin panel. The page rendered displays a crafted/simulated user with mock data. All bookings and actions are mocked and no real operations are performed in the database.

## Implementation Details

### 1. New Route
- **URL**: `/admin/user-view-simulation`
- **Access**: Admin only (requires admin authentication)
- **Method**: GET

### 2. Navigation Link
A new "Vista Utente" (User View) link has been added to all admin pages:
- Calendar page (`/admin/calendar`)
- Users page (`/admin/users`)
- Events page (`/admin/events`)
- Survey Results page (`/admin/survey/results`)
- Survey Questions page (`/admin/survey/questions`)

### 3. Simulated User Data
The simulation creates a mock user with the following characteristics:
- **Name**: Demo Utente
- **Email**: demo@example.com
- **Subscription Type**: SHARED
- **Remaining Accesses**: 8
- **Expiration Date**: 3 months from current date
- **Goals**: "Migliorare il benessere generale e la forma fisica"
- **Mock Bookings**: 3 upcoming appointments at different dates and times

### 4. Mocked Operations

#### Booking Creation
- When clicking on available slots, a confirmation dialog appears with "SIMULAZIONE" prefix
- Instead of calling the real API (`/api/user/bookings/create`), the action is simulated
- A success message is shown indicating it's a simulation
- An alert explains that no real action was performed

#### Booking Deletion
- When clicking delete on a booking, a confirmation dialog appears with "SIMULAZIONE" prefix
- Instead of calling the real API (`/api/user/bookings/delete`), the action is simulated
- A success message is shown indicating it's a simulation
- An alert explains that no real action was performed

#### Slot Fetching
- Instead of fetching real slots from the API, mock slots are generated
- Creates 21 slots (7 days × 3 time slots per day)
- Time slots: 9:00, 14:00, 17:00
- Random occupancy for SHARED subscription type

### 5. Visual Indicators

#### Simulation Banner
- A prominent purple gradient banner at the top of the page
- Contains:
  - "Back to Admin" link with arrow icon
  - "MODALITÀ SIMULAZIONE - Nessuna azione verrà effettuata" message
  - Eye icon to indicate viewing mode

#### Modified Bottom Navigation
- The "Esci" (Logout) button is replaced with "Admin" (Back to Admin)
- Uses blue color and back arrow icon
- Returns user to admin calendar view

### 6. Disabled Features in Simulation Mode
- Browser notifications system is disabled (no need to notify about simulated bookings)
- No real API calls are made
- No database operations are performed

## User Experience Flow

1. Admin navigates to any admin page
2. Clicks on "Vista Utente" link in the navigation
3. Views the simulated user dashboard with:
   - User information (name, subscription details, expiration date)
   - List of mock bookings
   - Ability to view available slots
4. Can interact with the UI:
   - View bookings list
   - Click "Crea" to see mock available slots
   - Click on slots to trigger mock booking creation
   - Click delete icon on bookings to trigger mock deletion
5. All actions show clear "SIMULAZIONE" indicators
6. Can return to admin area via:
   - Top banner "Torna all'Admin" link
   - Bottom navigation "Admin" button

## Technical Implementation

### Files Modified
1. `cmd/server/main.go`
   - Added `serveUserViewSimulation` handler function
   - Added route mapping to admin middleware

2. Navigation templates (all updated with "Vista Utente" link):
   - `cmd/server/templates/calendar.html`
   - `cmd/server/templates/users.html`
   - `cmd/server/templates/events.html`
   - `cmd/server/templates/survey-results.html`
   - `cmd/server/templates/survey-questions.html`

3. New template:
   - `cmd/server/templates/user-view-simulation.html`
   - Based on `index.html` (user dashboard)
   - Modified to show simulation banner
   - Modified JavaScript to mock all API calls
   - Disabled notification system
   - Changed bottom navigation

### Security Considerations
- Route is protected by `adminMiddleware` - only admins can access
- No actual database operations are performed
- No real bookings are created or deleted
- No real user data is exposed (all data is crafted)

## Benefits
- Allows admins to preview the user experience without affecting real data
- Useful for training new administrators
- Helps admins understand what users see
- Safe environment to test UI/UX changes
- No risk of accidentally modifying production data

## Future Enhancements
Could add:
- Ability to simulate different subscription types (SHARED vs SINGLE)
- Different user scenarios (expired subscription, no remaining accesses, etc.)
- Toggle between different mock user profiles
- Simulation of error states
