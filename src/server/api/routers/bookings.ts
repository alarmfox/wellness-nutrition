import { Prisma, SubType } from "@prisma/client";
import { TRPCError } from "@trpc/server";
import { DateTime, } from "luxon";
import { z } from "zod";
import { env } from "../../../env/server.mjs";
import { createTRPCRouter, protectedProcedure } from "../trpc";

const businessWeek = [1, 2, 3, 4, 5, 6];

function isBookeable (d: DateTime): boolean {
  return businessWeek.includes(d.weekday) && d.hour > 8 && d.hour < 22;
}

export const bookingRouter = createTRPCRouter({
  getCurrent: protectedProcedure.query(({ ctx }) => {
    return ctx.prisma.booking.findMany({
      where: {
        userId: ctx.session.user.id
      },
      orderBy: {
        startsAt: 'desc',
      },
      distinct: ['userId', 'startsAt']
    })
  }),

  //TODO: send notification
  delete: protectedProcedure.input(z.object({
    id: z.bigint(),
    isRefundable: z.boolean(),
  })).mutation(async ({ ctx, input }) => {
    try {
      const deleteBooking = ctx.prisma.booking.delete({
        where: {
          id: input.id
        }
      })
      const refund = ctx.prisma.user.update({
        where: {
          id: ctx.session.user.id,
        },
        data: {
          remainingAccesses: {
            increment: 1
          }
        }
      })
      const ops = input.isRefundable ? [deleteBooking, refund] : [deleteBooking]
      await ctx.prisma.$transaction(ops);

    } catch (error) {
      if (error instanceof Prisma.PrismaClientKnownRequestError) {
        if (error.code === 'P2025') {
          throw new TRPCError({
            code: 'NOT_FOUND'
          })
        }
      }
    }
  }),
  getAvailableSlots: protectedProcedure.query(async ({ ctx }) => {
    const endDate = DateTime.now().endOf('month');
    const startDate = DateTime.now().plus( { days: 1 }).startOf('day').startOf('hour')
    const allRecurrences: Date[] = [];
    const maxUserPerSlot = +env.MAX_USERS_PER_SLOT;

    let nextOccurrence = null;
    do {
      nextOccurrence = nextOccurrence ? nextOccurrence.plus({ hours: 1 }) : startDate.plus({ hours: 1 })
      if (isBookeable(nextOccurrence)) 
        allRecurrences.push(nextOccurrence.toJSDate());
    } while(nextOccurrence < endDate);

    try {
      const bookings = await ctx.prisma.booking.groupBy({
        _count: {
          startsAt: true,
        },
        by: ['startsAt', 'userId']
      });

      const toExclude = bookings.filter(
        (item) => ctx.session.user.subType === SubType.SHARED ? 
          maxUserPerSlot - item._count.startsAt < 1 : maxUserPerSlot - item._count.startsAt < 2 
      ).map((item) => item.startsAt)

      return allRecurrences.filter((item) => !toExclude.includes(item));
      
    } catch (error) {
      console.log(error)
      throw new TRPCError({
        code: 'INTERNAL_SERVER_ERROR'
      })
    }
  }),

  create: protectedProcedure.input(z.object({
    startsAt: z.date(),
  })).mutation(async({ctx, input}) => {
    const booking = {
      startsAt: input.startsAt,
      userId: ctx.session.user.id,
      createdAt: new Date(),
    };
    const createBooking = ctx.prisma.booking.createMany({
      data: ctx.session.user.subType === SubType.SHARED ? [booking] : [booking, booking]
    });

    const updateAccesses = ctx.prisma.user.update({
      where: {
        id: ctx.session.user.id
      },
      data: {
        remainingAccesses: {
          decrement: 1
        }
      }
    });
    await ctx.prisma.$transaction([createBooking, updateAccesses])
  })
});