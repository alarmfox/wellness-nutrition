import { Prisma, Role } from "@prisma/client";
import { TRPCError } from "@trpc/server";
import { DateTime } from "luxon";
import { z } from "zod";
import { CreateUserSchema, UpdateUserSchema } from "../../../utils/user.schema";
import { adminProtectedProcedure, createTRPCRouter, protectedProcedure, publicProcedure } from "../trpc";
import { randomBytes } from "crypto";
import { env } from "../../../env/server.mjs";
import { sendWelcomeEmail, sendResetEmail } from "../../mail";
import { VerifyAccountSchema } from "../../../utils/verifiy.schema";
import argon2 from "argon2";


export const userRouter = createTRPCRouter({
  getCurrent: protectedProcedure.query(async ({ ctx }) => {
    const user = await ctx.prisma.user.findUnique({
      where: {
        id: ctx.session.user.id,
      },
    });
    if (user) user.password = null;
    return user;
  }),
  getAll: adminProtectedProcedure.query(({ ctx }) => {
    return ctx.prisma.user.findMany({
      where: {
        role: Role.USER,
      }
    });
  }),

  create: adminProtectedProcedure.
    input(CreateUserSchema).
    mutation(async ({ ctx, input }) => {
      try {
        const { goals, ...rest } = input;
        const token = randomBytes(48).toString('base64url')
        const user = await ctx.prisma.user.create({
          data: {
            ...rest,
            goals: goals?.join('-'),
            verificationToken: token,
            verificationTokenExpiresIn: DateTime.now().plus({ days: 7 }).toJSDate()
          },
        })
        const url = `${env.NEXTAUTH_URL}/verify?token=${token}`
        await sendWelcomeEmail(user, url)
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
    const { goals, ...rest } = input;
    return ctx.prisma.user.update({
      where: {
        id: input.id
      },
      data: {
        ...rest,
        goals: goals?.join('-'),
      }
    })
  }),
  changePassword: publicProcedure.input(VerifyAccountSchema).mutation(async ({ ctx, input }) => {
    try {
      const user = await ctx.prisma.user.updateMany({
        where: {
          AND: {
            verificationToken: input.token,
            verificationTokenExpiresIn: {
              gt: new Date()
            },
          }
        },
        data: {
          emailVerified: DateTime.now().toJSDate(),
          password: await argon2.hash(input.newPassword),
          verificationToken: null,
          verificationTokenExpiresIn: null
        }
      })
      if (user.count === 0)
        throw new TRPCError({
          code: 'NOT_FOUND'
        })
    } catch (error) {
      if (error instanceof TRPCError) throw error;
      console.log(error);
      throw new TRPCError({
        code: 'INTERNAL_SERVER_ERROR'
      })
    }
  }),
  resetPassword: publicProcedure.input(z.string()).mutation(async ({ ctx, input }) => {
    const token = randomBytes(48).toString('base64url');
    try {
      const user = await ctx.prisma.user.update({
        where: {
          email: input
        },
        data: {
          verificationToken: token,
          verificationTokenExpiresIn: DateTime.now().plus({ days: 7 }).toJSDate()
        }
      });

      const url = `${env.NEXTAUTH_URL}/verify?token=${token}`
      await sendResetEmail(user, url);

    } catch (error) {
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

  getActive: adminProtectedProcedure.query(({ ctx }) => {
    return ctx.prisma.user.findMany({
      where: {
        AND: {
          expiresAt: {
            gt: new Date(),
          },
          remainingAccesses: {
            gt: 0
          },
          emailVerified: {
            not: null
          },
          role: Role.USER,
        }
      }
    })
  })
})
