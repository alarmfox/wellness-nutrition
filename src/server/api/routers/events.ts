import { DateTime } from "luxon";
import { adminProtectedProcedure, createTRPCRouter } from "../trpc";

export const eventsRouter = createTRPCRouter({
  getLatest: adminProtectedProcedure.query(async ({ ctx }) => {
    return ctx.prisma.event.findMany({
      where: {
        occurredAt: {
          gte: DateTime.now().startOf('week').toJSDate()
        }
      },
      orderBy: {
        occurredAt: 'asc'
      },
      include: {
        user: true,
      }
    })
  })
})
