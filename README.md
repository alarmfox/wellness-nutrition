# Wellness & Nutrition
A React application designed to manage bookings, time slots, and users for a local gym.

## ⚠️ Migration Notice

This application has been migrated from Next.js to a **Golang-based server-side rendered application**. The new Go application is located in the `app/` directory.

### New Stack
- **Backend**: Go (net/http)
- **Templates**: Go html/template with Material UI styling
- **Database**: PostgreSQL (existing schema unchanged)
- **Authentication**: JWT-based sessions with secure cookies
- **Email**: Native Go SMTP with HTML templates

See [`app/README.md`](app/README.md) for complete documentation of the Go application.

---

## Original Description
The app is used by an admin to manage users and the calendar. An admin can:
- Book slots for other users
- Mark slots as unavailable
- Create, update, or delete users
- Receive notifications when clients make or delete a booking

Users can:
- View their plan information
- Check their bookings
- Make new bookings according to available slots

Plans can be SINGLE or SHARED. On a shared plan, users can share their slots with another user. On a single plan, slots are dedicated to the individual user.
Users are registered with credentials (Email and Password).

The email is verified through an activation link sent when the admin registers a new client. This allows users to perform their first access, verify their email, and set their password on their own.

When a user performs an action (e.g., DELETE or CREATE a booking), the admin is notified with an in-app notification (delivered through a websocket) and an email. The event is also logged in the database to be viewed on the events page.

### Legacy App (Next.js)
The legacy Next.js application is scaffolded with [T3 Stack](https://create.t3.gg/) and uses the following modules:
- [Next.js](https://nextjs.org)
- [NextAuth.js](https://next-auth.js.org)
- [Prisma](https://prisma.io)
- [tRPC](https://trpc.io)
Styling is done using Material UI.

### Go Application (Current)
The current Go application provides:
- Server-side rendered HTML pages using Go templates
- RESTful API endpoints with role-based authorization
- Session-based authentication with secure cookies
- Email notifications with HTML templates
- Material UI styling via CDN

**Key Features:**
- ✅ Users cannot access admin API endpoints (403 Forbidden for non-admins)
- ✅ UI reproduced using Go templates with Material UI
- ✅ Mail template system similar to nodemailer

### Cleanup
The project also contains a Go module called `cleanup`, which contains programs to periodically clean up obsolete data in the database.

### Events
Events are sent using a [Soketi](https://docs.soketi.app/) instance with the Pusher SDK.
