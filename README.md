# Wellness & Nutrition
React app to manage booking, time slots and users for a local gym.

## Description
The app is used by an admin to manage users and the calendar. An admin can:
- book slots for other users;
- mark slots as unvailable;
- create/update/delete users;
- receive notifications when clients make a booking or delete them;

Users can:
- view their plan informations;
- check their bookings;
- make a new booking according to available slots;

Plan can be SINGLE or SHARED. On a shared plan, users can share their slots with another one. Instead, when on single, slots is dedicated.

Users are registered with credentials (Email and Password) and the email is verified through an activation link which is sent when the admin registers a new client. 
This allows users to perform first access, verify their email and set their password on their own.

When a user performs an action (aka DELETE or CREATE a booking), the admin is notified with an in-app notification (delivered through a websocket) and an email. Also, the
event is logged in the database to be viewed in the events page.

### App
The app is scaffolded with [T3 Stack](https://create.t3.gg/) and uses the following modules. 
- [Next.js](https://nextjs.org)
- [NextAuth.js](https://next-auth.js.org)
- [Prisma](https://prisma.io)
- [tRPC](https://trpc.io)

Styling is done using Material UI.

### Cleanup
The project also contains a go module called `cleanup` which contains programs to cleanup database obsolete data periodically.

### Events
Events are sent using a [Soketi](https://docs.soketi.app/) instance with the Pusher SDK.

## Deploy
The app is designed to be deploy on a VPS using Docker and Traefik v2 as reverse proxy. A docker-compose.example.yml is provided
for a basic setup.

### Preview
A preview of the app is hosted on [Vercel](https://wellness-nutrition.vercel.app/) using:
- a free instance of a cloud Postgres deployment hosted on [ElephantSQL](https://elephantsql.com);
- sandbox plan on [Pusher](https://pusher.com);
