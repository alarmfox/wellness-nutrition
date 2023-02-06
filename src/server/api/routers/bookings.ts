import { createTRPCRouter, protectedProcedure } from "../trpc";

export const bookingRouter = createTRPCRouter({
    getCurrent: protectedProcedure.query(({ ctx }) => {
        return ctx.prisma.booking.findMany({
            where: {
                usesrId: ctx.session.user.id
            }
        })
    })
});

