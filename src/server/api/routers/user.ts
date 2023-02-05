import { createTRPCRouter, protectedProcedure } from "../trpc";

export const userRouter = createTRPCRouter({
    getCurrent: protectedProcedure.query(({ ctx }) => {
        return ctx.prisma.user.findFirst({
            where: {
                id: ctx.session.user.id
            }
        })
    })
});

