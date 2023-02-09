import { Prisma } from "@prisma/client";
import { TRPCError } from "@trpc/server";
import { DateTime, Interval, } from "luxon";
import { z } from "zod";
import { AdminCreateSchema, AdminDeleteSchema, IntervalSchema } from "../../../utils/booking.schema";
import { adminProtectedProcedure, createTRPCRouter, protectedProcedure } from "../trpc";

const businessWeek = [1, 2, 3, 4, 5, 6];

function isBookeable(d: DateTime): boolean {
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
    })
  }),

  //TODO: send notification
  delete: protectedProcedure.input(z.object({
    id: z.bigint(),
    startsAt: z.date(),
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
      const resetCounter = ctx.prisma.slot.update({
        where: {
          startsAt: input.startsAt
        },
        data: {
          peopleCount: {
            increment: ctx.session.user.subType === 'SHARED' ? 1 : 2
          }
        }
      })
      const ops = input.isRefundable ? [deleteBooking, resetCounter, refund] : [deleteBooking, resetCounter]
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
    const startDate = DateTime.now().plus({ days: 1 }).startOf('day').startOf('hour')
    const allRecurrences: string[] = [];

    let nextOccurrence = null;
    do {
      nextOccurrence = nextOccurrence ? nextOccurrence.plus({ hours: 1 }) : startDate.plus({ hours: 1 })
      if (isBookeable(nextOccurrence))
        allRecurrences.push(nextOccurrence.toISO());
    } while (nextOccurrence < endDate);

    try {
      const slots = (await ctx.prisma.slot.findMany({
        where: {
          OR: [
            {
              disabled: true,
            },
            {
              AND: {
                peopleCount: {
                  gte: ctx.session.user.subType === 'SHARED' ? 2 : 0
                },
                startsAt: {
                  gte: startDate.toJSDate()
                }
              }
            }
          ]
        },
        select: {
          startsAt: true
        }
      })).map((item) => DateTime.fromJSDate(item.startsAt).toISO());

      return allRecurrences.filter((item) => !slots.includes(item));

    } catch (error) {
      console.log(error)
      throw new TRPCError({
        code: 'INTERNAL_SERVER_ERROR'
      })
    }
  }),

  create: protectedProcedure.input(z.object({
    startsAt: z.date(),
  })).mutation(async ({ ctx, input }) => {

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

    const createSlot = ctx.prisma.slot.upsert({
      create: {
        startsAt: input.startsAt,
        peopleCount: ctx.session.user.subType === 'SHARED' ? 1 : 2,
        bookings: {
          create: {
            userId: ctx.session.user.id,
          }
        }
      },
      update: {
        peopleCount: {
          increment: ctx.session.user.subType === 'SHARED' ? 1 : 2,
        },
        bookings: {
          create: {
            userId: ctx.session.user.id,
          }
        }
      },
      where: {
        startsAt: input.startsAt
      },
    });

    await ctx.prisma.$transaction([updateAccesses, createSlot])
  }),

  adminCreate: adminProtectedProcedure.input(AdminCreateSchema).mutation(async ({ ctx, input }) => {
    const h = Interval.fromDateTimes(
      DateTime.fromJSDate(input.from),
      DateTime.fromJSDate(input.to),
    ).splitBy({ hours: 1 })

    const ops = h.map((item) => ctx.prisma.slot.upsert({
      where: {
        startsAt: item.start.toJSDate(),
      },
      create: {
        startsAt: item.start.toJSDate(),
        peopleCount: 1,
        bookings: {
          create: {
            userId: input.userId || ctx.session.user.id,
          }
        },
        disabled: input.disable,
      },
      update: {
        peopleCount: {
          increment: 1
        },
        bookings: {
          create: {
            userId: input.userId || ctx.session.user.id,
          }
        },
        disabled: input.disable,
      }
    }))

    const decrementAccesses = ctx.prisma.user.update({
      where: {
        id: input.userId,
      },
      data: {
        remainingAccesses: {
          decrement: h.length,
        }
      }
    })

    const t = input.disable ? [...ops] : [...ops, decrementAccesses]
    await ctx.prisma.$transaction(t);
  }),

  adminDelete: adminProtectedProcedure.input(AdminDeleteSchema).mutation(async ({ ctx, input }) => {
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
    const updateCount = ctx.prisma.slot.update({
      where: {
        startsAt: input.startsAt
      },
      data: {
        peopleCount: {
          decrement: 1
        }
      }
    })
    const ops = input.refundAccess ? [deleteBooking, updateCount, refund] : [deleteBooking, updateCount]
    await ctx.prisma.$transaction(ops);
  }),

  //TODO: compress consecutives disabled
  getByInterval: adminProtectedProcedure.input(IntervalSchema).query(({ ctx, input }) => {
    return ctx.prisma.booking.findMany({
      where: {
        startsAt: {
          gte: input.from,
          lte: input.to
        },
      },
      include: {
        user: true,
        slot: true,
      },
    })
  }),
});