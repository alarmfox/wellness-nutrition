import type { Event, User } from "@prisma/client";
import { Prisma } from "@prisma/client";
import { TRPCError } from "@trpc/server";
import { DateTime, Interval, } from "luxon";
import { z } from "zod";
import { AdminCreateSchema, AdminDeleteSchema, IntervalSchema } from "../../../utils/booking.schema";
import type { NotificationModel } from "../../../utils/event.schema";
import { zone } from "../../../utils/format.utils";
import { sendOnDeleteBooking, sendOnNewBooking } from "../../mail";
import { pusher } from "../../pusher";
import { adminProtectedProcedure, createTRPCRouter, protectedProcedure } from "../trpc";

const businessWeek = [1, 2, 3, 4, 5];

function isBookeable(d: DateTime): boolean {
  if (d.weekday === 6) return d.hour >= 7 && d.hour <= 11;
  return businessWeek.includes(d.weekday) && d.hour >= 7 && d.hour <= 21;
}

function createNotification(
  { user: { firstName, lastName },
    id,
    occurredAt,
    startsAt,
    type
  }: Event & { user: User }): NotificationModel {
  return {
    firstName,
    lastName,
    id,
    occurredAt: occurredAt.toISOString(),
    startsAt: startsAt.toISOString(),
    type
  }
}

export const bookingRouter = createTRPCRouter({
  getCurrent: protectedProcedure.query(({ ctx }) => {
    return ctx.prisma.booking.findMany({
      where: {
        AND: {
          userId: ctx.session.user.id,
          startsAt: {
            gt: DateTime.now().startOf('month').toJSDate(),
          }
        }
      },
      orderBy: {
        startsAt: 'desc',
      },
    })
  }),

  delete: protectedProcedure.input(z.object({
    id: z.bigint(),
    startsAt: z.date(),
    isRefundable: z.boolean(),
  })).mutation(async ({ ctx, input }) => {
    try {
      const deleteBooking = ctx.prisma.booking.delete({
        where: {
          id: input.id,
        },
        include: {
          user: true,
        }
      });
      const refund = ctx.prisma.user.update({
        where: {
          id: ctx.session.user.id,
        },
        data: {
          remainingAccesses: {
            increment: 1,
          }
        }
      });
      const resetCounter = ctx.prisma.slot.update({
        where: {
          startsAt: input.startsAt,
        },
        data: {
          peopleCount: {
            decrement: ctx.session.user.subType === 'SHARED' ? 1 : 2,
          }
        }
      });
      const logEvent = ctx.prisma.event.create({
        data: {
          startsAt: input.startsAt,
          type: 'DELETED',
          userId: ctx.session.user.id,
        },
        include: {
          user: true,
        }
      })
      const ops = input.isRefundable ? [deleteBooking, resetCounter, refund, logEvent] : [deleteBooking, resetCounter, logEvent]
      const res = await ctx.prisma.$transaction(ops);

      const createdEvent = (input.isRefundable ? res[3] : res[2]) as Event & { user: User };

      await Promise.all([
        pusher.trigger('booking', 'user', createNotification(createdEvent)),
        sendOnDeleteBooking(createdEvent.user, input.startsAt),
        pusher.trigger('booking', 'refresh', { startsAt: input.startsAt }),
      ]);

    } catch (error) {
      if (error instanceof Prisma.PrismaClientKnownRequestError) {
        if (error.code === 'P2025') {
          throw new TRPCError({
            code: 'NOT_FOUND',
          })
        }
      }
      console.log(error);
    }
  }),
  getAvailableSlots: protectedProcedure.query(async ({ ctx }) => {
    const now = DateTime.now().setZone(zone);
    const isLastWeekOfMonth = now.endOf('month').weekNumber - 1 === now.weekNumber;
    const endDate = isLastWeekOfMonth ? now.plus({ months: 1 }).endOf('month') : now.endOf('month');
    const startDate = now.hour >= 17 ?
      now.plus({ days: 2 }).startOf('day').startOf('hour') :
      now.plus({ days: 1 }).startOf('day').startOf('hour');

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
                  gte: ctx.session.user.subType === 'SHARED' ? 2 : 1,
                },
                startsAt: {
                  gte: startDate.toJSDate(),
                }
              }
            }
          ]
        },
        select: {
          startsAt: true,
        }
      })).map((item) => DateTime.fromJSDate(item.startsAt).setZone(zone).toISO());

      return allRecurrences.filter((item) => !slots.includes(item));

    } catch (error) {
      console.log(error);
      throw new TRPCError({
        code: 'INTERNAL_SERVER_ERROR',
      });
    }
  }),

  create: protectedProcedure.input(z.object({
    startsAt: z.date(),
  })).mutation(async ({ ctx, input }) => {
    const slot = await ctx.prisma.slot.findUnique({
      where: {
        startsAt: input.startsAt,
      }
    });

    if (slot?.disabled) {
      throw new TRPCError({
        code: 'BAD_REQUEST',
        message: 'Slot disabled'
      });
    }

    const updateAccesses = ctx.prisma.user.update({
      where: {
        id: ctx.session.user.id,
      },
      data: {
        remainingAccesses: {
          decrement: 1,
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
        startsAt: input.startsAt,
      },
    });

    const logEvent = ctx.prisma.event.create({
      data: {
        type: 'CREATED',
        startsAt: input.startsAt,
        userId: ctx.session.user.id,
      },
      include: {
        user: true,
      }
    });
    const res = await ctx.prisma.$transaction([updateAccesses, createSlot, logEvent]);
    await Promise.all([
      pusher.trigger('booking', 'user', createNotification(res[2])),
      sendOnNewBooking(res[2].user, input.startsAt),
      pusher.trigger('booking', 'refresh', { startsAt: input.startsAt }),
    ]);
  }),

  adminCreate: adminProtectedProcedure.input(AdminCreateSchema).mutation(async ({ ctx, input }) => {
    const h = Interval.fromDateTimes(
      DateTime.fromJSDate(input.from).startOf('hour').startOf('second'),
      DateTime.fromJSDate(input.to).startOf('hour').startOf('second'),
    ).splitBy({ hours: 1 });

    const ops = h.map((item) => ctx.prisma.slot.upsert({
      where: {
        startsAt: item.start.toJSDate(),
      },
      create: {
        startsAt: item.start.toJSDate(),
        peopleCount: input.subType === 'SHARED' ? 1 : 2,
        bookings: {
          create: {
            userId: input.userId || ctx.session.user.id,
          }
        },
        disabled: input.disable,
      },
      update: {
        peopleCount: {
          increment: input.subType === 'SHARED' ? 1 : 2,
        },
        bookings: {
          create: {
            userId: input.userId || ctx.session.user.id,
          }
        },
        disabled: input.disable,
      }
    }));

    const decrementAccesses = ctx.prisma.user.update({
      where: {
        id: input.userId,
      },
      data: {
        remainingAccesses: {
          decrement: h.length,
        }
      }
    });

    const t = input.disable ? [...ops] : [...ops, decrementAccesses]
    await ctx.prisma.$transaction(t),
      await pusher.trigger('booking', 'refresh', {});
  }),

  adminDelete: adminProtectedProcedure.input(AdminDeleteSchema).mutation(async ({ ctx, input }) => {
    if (input.isDisabled) {
      return ctx.prisma.slot.delete({
        where: {
          startsAt: input.startsAt,
        }
      });
    }
    const deleteBooking = ctx.prisma.booking.delete({
      where: {
        id: input.id,
      }
    });
    const refund = ctx.prisma.user.update({
      where: {
        id: input.userId || ctx.session.user.id,
      },
      data: {
        remainingAccesses: {
          increment: 1,
        }
      }
    });
    const updateCount = ctx.prisma.slot.update({
      where: {
        startsAt: input.startsAt,
      },
      data: {
        peopleCount: {
          decrement: input.userSubType === 'SHARED' ? 1 : 2,
        }
      }
    });
    const ops = input.refundAccess ? [deleteBooking, updateCount, refund] : [deleteBooking, updateCount]
    await ctx.prisma.$transaction(ops);
    await pusher.trigger('booking', 'refresh', { startsAt: input.startsAt });
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
