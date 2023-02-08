import { SubType } from "@prisma/client";
import { z } from "zod";

export const AdminCreateSchema = z.object({
  startsAt: z.date(),
  userId: z.string().optional(),
  disable: z.boolean(),
  subType: z.nativeEnum(SubType).optional(),
})

export type AdminCreateModel = z.infer<typeof AdminCreateSchema>;