import { Prisma } from "@prisma/client";
import { TRPCError } from "@trpc/server";
import { DateTime } from "luxon";
import { z } from "zod";
import { CreateUserSchema, UpdateUserSchema } from "../../../utils/user.schema";
import { createTRPCRouter, protectedProcedure, publicProcedure } from "../trpc";
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
  getAll: publicProcedure.query(({ ctx }) => {
    return ctx.prisma.user.findMany();
  }),

  create: publicProcedure.
    input(CreateUserSchema).
    mutation(async ({ ctx, input }) => {
      try {
        const { email, id } = await ctx.prisma.user.create({
          data: input,
        })
        const { token } = await ctx.prisma.verificationToken.create({
          data: {
            expires: DateTime.now().plus({ days: 7}).toJSDate(),
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
  }),
  changePassword: publicProcedure.input(VerifyAccountSchema).mutation( async ( { ctx, input } ) => {
    try {
      const token = await ctx.prisma.verificationToken.findFirst({
        where: {
          token: input.token
        }
      });
      if (!token) throw new TRPCError( { code: 'NOT_FOUND'});
      
      if (DateTime.now().diff(DateTime.fromJSDate(token.expires)).seconds > 0) {
        throw new TRPCError( { code: 'NOT_FOUND'});
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
      console.log(error);
      throw new TRPCError({
        code: 'INTERNAL_SERVER_ERROR'
      })
      
    }
    
  })
})