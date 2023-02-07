import { Prisma, Role } from "@prisma/client";
import { TRPCError } from "@trpc/server";
import { DateTime } from "luxon";
import { z } from "zod";
import { CreateUserSchema, UpdateUserSchema } from "../../../utils/user.schema";
import { adminProtectedProcedure, createTRPCRouter, protectedProcedure, publicProcedure } from "../trpc";
import { randomBytes } from "crypto";
import { env } from "../../../env/server.mjs";
import { sendVerificationEmail } from "../../mail";
import { VerifyAccountSchema } from "../../../utils/verifiy.schema";
import argon2 from "argon2";

export const userRouter = createTRPCRouter({
  getCurrent: protectedProcedure.query(({ ctx }) => {
    return ctx.prisma.user.findFirst({
      where: {
        id: ctx.session.user.id
      }
    })
  }),
  getAll: adminProtectedProcedure.query(({ ctx }) => {
    return ctx.prisma.user.findMany({
      where: {
        role: Role.USER
      }
    });
  }),

  create: adminProtectedProcedure.
    input(CreateUserSchema).
    mutation(async ({ ctx, input }) => {
      try {
        const { email, id } = await ctx.prisma.user.create({
          data: input,
        })
        const { token } = await ctx.prisma.verificationToken.create({
          data: {
            expires: DateTime.now().plus({ days: 7 }).toJSDate(),
            identifier: id,
            token: randomBytes(48).toString('base64url')
          }
        })
        const url = `${env.NEXTAUTH_URL}/verify?token=${token}`
        sendVerificationEmail(email, url)
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
  delete: adminProtectedProcedure.input(z.array(z.string())).mutation(({ ctx, input }) => {
    return ctx.prisma.user.deleteMany({
      where: {
        id: {
          in: input
        },
      }
    })
  }),
  update: adminProtectedProcedure.input(UpdateUserSchema).mutation(({ ctx, input }) => {
    return ctx.prisma.user.update({
      where: {
        id: input.id
      },
      data: input
    })
  }),
  changePassword: publicProcedure.input(VerifyAccountSchema).mutation(async ({ ctx, input }) => {
    try {
      const token = await ctx.prisma.verificationToken.delete({
        where: {
          token: input.token
        }
      });

      if (DateTime.fromJSDate(token.expires) < DateTime.now()) {
        throw new TRPCError({ code: 'NOT_FOUND' });
      }

      return await ctx.prisma.user.update({
        where: {
          id: token.identifier
        },
        data: {
          emailVerified: DateTime.now().toJSDate(),
          password: await argon2.hash(input.newPassword)
        }
      })
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      if (error instanceof Prisma.PrismaClientKnownRequestError) {
        if (error.code === 'P2025') {
            throw new TRPCError({
              code: 'NOT_FOUND'
            })
        }
      }
      console.log(error);
      throw new TRPCError({
        code: 'INTERNAL_SERVER_ERROR'
      })
    }
  }),
  resetPassword: publicProcedure.input(z.string()).mutation(async ({ ctx, input }) => {
    try {
      const user = await ctx.prisma.user.findFirst({
        where: {
          email: input
        }
      });
      if (!user) throw new TRPCError({
        code: 'NOT_FOUND'
      })
      const { token } = await ctx.prisma.verificationToken.create({
        data: {
          expires: DateTime.now().plus({ days: 7 }).toJSDate(),
          identifier: user.id,
          token: randomBytes(48).toString('base64url')
        }
      })
      const url = `${env.NEXTAUTH_URL}/verify?token=${token}`
      sendVerificationEmail(user.email, url)

    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      console.log(error);
      throw new TRPCError({
        code: 'INTERNAL_SERVER_ERROR'
      })

    }
  })
})