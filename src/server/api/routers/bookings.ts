import type { Event, PrismaClient, User } from "@prisma/client";
import { Prisma } from "@prisma/client";
import { TRPCError } from "@trpc/server";
import { DateTime, Interval } from "luxon";
import { z } from "zod";
import {
  AdminCreateSchema,
  AdminDeleteSchema,
  IntervalSchema,
} from "../../../utils/booking.schema";
import type { NotificationModel } from "../../../utils/event.schema";
import { isBookeable, zone } from "../../../utils/date.utils";
import { sendOnDeleteBooking, sendOnNewBooking } from "../../mail";
import { pusher } from "../../pusher";
import {
  adminProtectedProcedure,
  createTRPCRouter,
  protectedProcedure,
} from "../trpc";

function createNotification({
  user: { firstName, lastName },
  id,
  occurredAt,
  startsAt,
  type,
}: Event & { user: User }): NotificationModel {
  return {
    firstName,
    lastName,
    id,
    occurredAt: occurredAt.toISOString(),
    startsAt: startsAt.toISOString(),
    type,
  };
}

async function userValidOrThrow(
  prisma: PrismaClient,
  id: string,
): Promise<User> {
  const user = await prisma.user.findUnique({
    where: {
      id,
    },
  });
  if (!user) {
    throw new TRPCError({
      code: "UNAUTHORIZED",
    });
  }

  if (
    DateTime.now() > DateTime.fromJSDate(user.expiresAt) ||
    user.remainingAccesses <= 0
  ) {
    throw new TRPCError({
      code: "UNAUTHORIZED",
    });
  }

  return user;
}

export const bookingRouter = createTRPCRouter({
  getCurrent: protectedProcedure.query(({ ctx }) => {
    return ctx.prisma.booking.findMany({
      where: {
        AND: {
          userId: ctx.session.user.id,
          startsAt: {
            gt: DateTime.now().startOf("month").toJSDate(),
          },
        },
      },
      orderBy: {
        startsAt: "desc",
      },
    });
  }),

  delete: protectedProcedure
    .input(
      z.object({
        id: z.bigint(),
        startsAt: z.date(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      try {
        const deleteBooking = ctx.prisma.booking.delete({
          where: {
            id: input.id,
          },
          include: {
            user: {
              select: {
                emailVerified: true,
                cellphone: true,
                firstName: true,
                lastName: true,
                id: true,
                subType: true,
                expiresAt: true,
                email: true,
                address: true,
                remainingAccesses: true,
                medOk: true,
                role: true,
                goals: true,
              },
            },
          },
        });
        const refund = ctx.prisma.user.update({
          where: {
            id: ctx.session.user.id,
          },
          data: {
            remainingAccesses: {
              increment: 1,
            },
          },
        });
        const resetCounter = ctx.prisma.slot.update({
          where: {
            startsAt: input.startsAt,
          },
          data: {
            peopleCount: {
              decrement: ctx.session.user.subType === "SHARED" ? 1 : 2,
            },
          },
        });
        const logEvent = ctx.prisma.event.create({
          data: {
            startsAt: input.startsAt,
            type: "DELETED",
            userId: ctx.session.user.id,
          },
          include: {
            user: {
              select: {
                emailVerified: true,
                cellphone: true,
                firstName: true,
                lastName: true,
                id: true,
                subType: true,
                expiresAt: true,
                email: true,
                address: true,
                remainingAccesses: true,
                medOk: true,
                role: true,
                goals: true,
              },
            },
          },
        });

        const isRefundable =
          DateTime.fromJSDate(input.startsAt).diffNow().as("hours") > 3;
        const ops = isRefundable
          ? [deleteBooking, resetCounter, refund, logEvent]
          : [deleteBooking, resetCounter, logEvent];
        const res = await ctx.prisma.$transaction(ops);

        const createdEvent = (isRefundable ? res[3] : res[2]) as Event & {
          user: User;
        };

        await Promise.all([
          pusher.trigger("booking", "user", createNotification(createdEvent)),
          sendOnDeleteBooking(
            createdEvent.user,
            DateTime.fromJSDate(input.startsAt).setZone(zone).toJSDate(),
          ),
          pusher.trigger("booking", "refresh", {}),
        ]);
      } catch (error) {
        if (error instanceof Prisma.PrismaClientKnownRequestError) {
          if (error.code === "P2025") {
            throw new TRPCError({
              code: "NOT_FOUND",
            });
          }
        }
        console.log(error);
      }
    }),
  getAvailableSlots: protectedProcedure.query(async ({ ctx }) => {
    const user = await userValidOrThrow(ctx.prisma, ctx.session.user.id);

    const now = DateTime.now().setZone(zone);
    const endOfMonth = now.endOf("month").day;
    const isLastWeekOfMonth =
      now.day <= endOfMonth && now.day >= endOfMonth - 7;
    const endDate = isLastWeekOfMonth
      ? now.plus({ months: 1 }).endOf("month")
      : now.endOf("month");

    const startDate =
      now.hour >= 17
        ? now.plus({ days: 2 }).startOf("day").startOf("hour")
        : now.plus({ days: 1 }).startOf("day").startOf("hour");

    const allRecurrences: number[] = [];
    const expiresAt = DateTime.fromJSDate(user.expiresAt);

    let nextOccurrence = null;
    do {
      nextOccurrence = nextOccurrence
        ? nextOccurrence.plus({ hours: 1 })
        : startDate.plus({ hours: 1 });
      if (isBookeable(nextOccurrence))
        allRecurrences.push(nextOccurrence.toSeconds());
    } while (nextOccurrence < endDate && nextOccurrence < expiresAt);

    try {
      const slots = (
        await ctx.prisma.slot.findMany({
          where: {
            OR: [
              {
                disabled: true,
              },
              {
                AND: {
                  peopleCount: {
                    gte: ctx.session.user.subType === "SHARED" ? 2 : 1,
                  },
                  startsAt: {
                    gte: startDate.toJSDate(),
                  },
                },
              },
              {
                bookings: {
                  some: {
                    userId: ctx.session.user.id,
                  },
                },
              },
            ],
          },
          select: {
            startsAt: true,
          },
        })
      ).map((item) =>
        DateTime.fromJSDate(item.startsAt).setZone(zone).toSeconds(),
      );

      return allRecurrences.filter((item) => !slots.includes(item));
    } catch (error) {
      if (error instanceof TRPCError) throw error;
      console.log(error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
      });
    }
  }),

  create: protectedProcedure
    .input(
      z.object({
        startsAt: z.date(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      await userValidOrThrow(ctx.prisma, ctx.session.user.id);
      const slot = await ctx.prisma.slot.findUnique({
        where: {
          startsAt: input.startsAt,
        },
      });

      if (slot?.disabled) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Slot disabled",
        });
      }
      const {
        user: { subType, id },
      } = ctx.session;

      if (slot) {
        if (
          (subType === "SHARED" && slot?.peopleCount >= 2) ||
          (subType === "SINGLE" && slot?.peopleCount >= 1)
        ) {
          throw new TRPCError({
            code: "CONFLICT",
            message: "Slot full",
          });
        }
      }

      const updateAccesses = ctx.prisma.user.update({
        where: {
          id,
        },
        data: {
          remainingAccesses: {
            decrement: 1,
          },
        },
      });

      const createSlot = ctx.prisma.slot.upsert({
        create: {
          startsAt: input.startsAt,
          peopleCount: subType === "SHARED" ? 1 : 2,
          bookings: {
            create: {
              userId: id,
            },
          },
        },
        update: {
          peopleCount: {
            increment: subType === "SHARED" ? 1 : 2,
          },
          bookings: {
            create: {
              userId: id,
            },
          },
        },
        where: {
          startsAt: input.startsAt,
        },
      });

      const logEvent = ctx.prisma.event.create({
        data: {
          type: "CREATED",
          startsAt: input.startsAt,
          userId: id,
        },
        include: {
          user: true,
        },
      });
      const res = await ctx.prisma.$transaction([
        updateAccesses,
        createSlot,
        logEvent,
      ]);
      await Promise.all([
        pusher.trigger("booking", "user", createNotification(res[2])),
        sendOnNewBooking(
          res[2].user,
          DateTime.fromJSDate(input.startsAt).setZone(zone).toJSDate(),
        ),
        pusher.trigger("booking", "refresh", {}),
      ]);
    }),

  adminCreate: adminProtectedProcedure
    .input(AdminCreateSchema)
    .mutation(async ({ ctx, input }) => {
      const h = Interval.fromDateTimes(
        DateTime.fromJSDate(input.from).startOf("hour").startOf("second"),
        DateTime.fromJSDate(input.to).startOf("hour").startOf("second"),
      ).splitBy({ hours: 1 });

      const ops = h.map((item) =>
        ctx.prisma.slot.upsert({
          where: {
            startsAt: item.start.toJSDate(),
          },
          create: {
            startsAt: item.start.toJSDate(),
            peopleCount: input.subType === "SHARED" ? 1 : 2,
            bookings: {
              create: {
                userId: input.userId || ctx.session.user.id,
              },
            },
            disabled: input.disable,
          },
          update: {
            peopleCount: {
              increment: input.subType === "SHARED" ? 1 : 2,
            },
            bookings: {
              create: {
                userId: input.userId || ctx.session.user.id,
              },
            },
            disabled: input.disable,
          },
        }),
      );

      const decrementAccesses = ctx.prisma.user.update({
        where: {
          id: input.userId,
        },
        data: {
          remainingAccesses: {
            decrement: h.length,
          },
        },
      });

      const t = input.disable ? [...ops] : [...ops, decrementAccesses];
      await Promise.all([
        ctx.prisma.$transaction(t),
        pusher.trigger("booking", "refresh", {}),
      ]);
    }),

  adminDelete: adminProtectedProcedure
    .input(AdminDeleteSchema)
    .mutation(async ({ ctx, input }) => {
      if (input.isDisabled) {
        return ctx.prisma.slot.delete({
          where: {
            startsAt: input.startsAt,
          },
        });
      }
      const deleteBooking = ctx.prisma.booking.delete({
        where: {
          id: input.id,
        },
      });
      const refund = ctx.prisma.user.update({
        where: {
          id: input.userId || ctx.session.user.id,
        },
        data: {
          remainingAccesses: {
            increment: 1,
          },
        },
      });
      const updateCount = ctx.prisma.slot.update({
        where: {
          startsAt: input.startsAt,
        },
        data: {
          peopleCount: {
            decrement: input.userSubType === "SHARED" ? 1 : 2,
          },
        },
      });
      const ops = input.refundAccess
        ? [deleteBooking, updateCount, refund]
        : [deleteBooking, updateCount];
      await Promise.all([
        ctx.prisma.$transaction(ops),
        pusher.trigger("booking", "refresh", { startsAt: input.startsAt }),
      ]);
    }),

  //TODO: compress consecutives disabled
  getByInterval: adminProtectedProcedure
    .input(IntervalSchema)
    .query(({ ctx, input }) => {
      return ctx.prisma.booking.findMany({
        where: {
          startsAt: {
            gte: input.from,
            lte: input.to,
          },
        },
        include: {
          user: {
            select: {
              emailVerified: true,
              cellphone: true,
              firstName: true,
              lastName: true,
              id: true,
              subType: true,
              expiresAt: true,
              email: true,
              address: true,
              remainingAccesses: true,
              medOk: true,
              role: true,
              goals: true,
            },
          },
          slot: true,
        },
      });
    }),
});
