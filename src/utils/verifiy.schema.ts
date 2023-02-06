import { z } from "zod";

export const VerifyAccountSchema = z.object({
    newPassword: z.string().min(6).max(50),
    confirmPassword: z.string(),
    token: z.string()
})

export type VerifyAccountModel = z.infer<typeof VerifyAccountSchema>;