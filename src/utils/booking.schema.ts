import { SubType } from "@prisma/client";
import { z } from "zod";

export const AdminCreateSchema = z.object({
  userId: z.string().optional(),
  disable: z.boolean().default(false),
  subType: z.nativeEnum(SubType).optional(),
  from: z.date(),
  to: z.date()
})

export const AdminDeleteSchema = z.object({
  id: z.bigint(),
  refundAccess: z.boolean(),
  startsAt: z.date(),
  userId: z.string().optional(),
  isDisabled: z.boolean().default(false),
  userSubType: z.nativeEnum(SubType).optional(),
});

export const IntervalSchema = z.object({
  from: z.date(),
  to: z.date(),
});


export type AdminCreateModel = z.infer<typeof AdminCreateSchema>;
export type AdminDeleteModel = z.infer<typeof AdminDeleteSchema>
export type IntervalModel = z.infer<typeof IntervalSchema>;
