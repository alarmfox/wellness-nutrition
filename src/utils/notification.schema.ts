import { z } from "zod";

export const NotificationSchema = z.object({
  id: z.string(),
  firstName: z.string(),
  lastName: z.string(),
  occurredAt: z.string().datetime(),
  startsAt: z.string().datetime()
})

export type NotificationModel = z.infer<typeof NotificationSchema>;