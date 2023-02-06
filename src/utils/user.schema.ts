import { SubType } from "@prisma/client";
import { z } from "zod";

export const CreateUserSchema = z.object({
    firstName: z.string(),
    lastName: z.string(),
    email: z.string(),
    cellphone: z.string(),
    remainingAccesses: z.number(),
    expiresAt: z.date(),
    subType: z.nativeEnum(SubType), 
    address: z.string()
})

export type CreateUserModel = z.infer<typeof CreateUserSchema>;