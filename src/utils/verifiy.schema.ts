import { z } from "zod";

export const VerifyAccountSchema = z.object({
    newPassword: z.string().
        min(6, 'La password deve avere almeno 6 caratteri').
        max(50, 'La password non può avere più di 50 caratteri'),
    confirmPassword: z.string(),
    token: z.string()
})

export type VerifyAccountModel = z.infer<typeof VerifyAccountSchema>;