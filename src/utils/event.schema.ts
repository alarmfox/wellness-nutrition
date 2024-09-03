import { EventType } from "@prisma/client";
import { z } from "zod";

export const NotificationSchema = z.object({
  id: z.number(),
  firstName: z.string(),
  lastName: z.string(),
  occurredAt: z.string().datetime(),
  startsAt: z.string().datetime(),
  type: z.nativeEnum(EventType),
});

export const PaginationSchema = z.object({
  skip: z.number(),
  take: z.number(),
});

export type NotificationModel = z.infer<typeof NotificationSchema>;
export type PaginationModel = z.infer<typeof PaginationSchema>;

