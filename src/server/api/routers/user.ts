import { Prisma } from "@prisma/client";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { CreateUserSchema, UpdateUserSchema } from "../../../utils/user.schema";
import { createTRPCRouter, protectedProcedure, publicProcedure } from "../trpc";

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
          data: input,
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
  delete: publicProcedure.input(z.array(z.string())).mutation(({ ctx, input}) => {
    return ctx.prisma.user.deleteMany({
      where: {
        id: {
          in: input
        },
      }
    })
  }),
  update: publicProcedure.input(UpdateUserSchema).mutation(({ ctx, input } ) => {
    return ctx.prisma.user.update({
      where: {
        id: input.id
      },
      data: input
    })
  })


})