import { SubType } from "@prisma/client";
import { z } from "zod";

export const CreateUserSchema = z.object({
    firstName: z.string(),
    lastName: z.string(),
    email: z.string(),
    cellphone: z.string().nullable(),
    remainingAccesses: z.number(),
    expiresAt: z.date(),
    subType: z.nativeEnum(SubType), 
    address: z.string(),
    medOk: z.boolean()
})


export const UpdateUserSchema = z.object({
    id: z.string(),
    firstName: z.string(),
    lastName: z.string(),
    cellphone: z.string().nullable(),
    remainingAccesses: z.number(),
    expiresAt: z.date(),
    subType: z.nativeEnum(SubType), 
    address: z.string(),
    medOk: z.boolean()
})
export type CreateUserModel = z.infer<typeof CreateUserSchema>;
export type UpdateUserModel = z.infer<typeof UpdateUserSchema>;