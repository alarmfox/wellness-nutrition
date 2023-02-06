import { Prisma } from "@prisma/client";
import { TRPCError } from "@trpc/server";
import { CreateUserSchema } from "../../../utils/user.schema";
import { adminProtectedProcedure, createTRPCRouter, protectedProcedure, publicProcedure } from "../trpc";

export const userRouter = createTRPCRouter({
    getCurrent: protectedProcedure.query(({ ctx }) => {
        return ctx.prisma.user.findFirst({
            where: {
                id: ctx.session.user.id
            }
        })
    }),
    getAll: publicProcedure.query(({ ctx }) => {
        return ctx.prisma.user.findMany();
    }),

    create: publicProcedure.
        input(CreateUserSchema).
        mutation(async ({ ctx, input }) => {
            try {
                return await ctx.prisma.user.create({
                    data: {
                        firstName: input.firstName,
                        lastName: input.lastName,
                        address: input.address,
                        cellphone: input.cellphone,
                        expiresAt: input.expiresAt,
                        remainingAccesses: input.remainingAccesses,
                        email: input.email
                    },
                })
            } catch (e) {
                if (e instanceof Prisma.PrismaClientKnownRequestError) {
                    // The .code property can be accessed in a type-safe manner
                    if (e.code === 'P2002') {
                        throw new TRPCError({
                            code: 'CONFLICT'
                        })
                    }
                }
                throw e;
            }


        }),
});

