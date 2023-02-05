import { createTRPCRouter, protectedProcedure } from "../trpc";

export const bookingRouter = createTRPCRouter({
    getCurrent: protectedProcedure.query(({ ctx }) => {
        return ctx.prisma.booking.findMany({
            where: {
               Subcription: {
                userId: ctx.session.user.id
               } 
            }
        })
    })
});

