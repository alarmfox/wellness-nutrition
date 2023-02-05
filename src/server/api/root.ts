import { createTRPCRouter } from "./trpc";
import { bookingRouter } from "./routers/bookings";
import { userRouter } from "./routers/user";

/**
 * This is the primary router for your server.
 *
 * All routers added in /api/routers should be manually added here
 */
export const appRouter = createTRPCRouter({
  user: userRouter,
  bookings: bookingRouter
});

// export type definition of API
export type AppRouter = typeof appRouter;
