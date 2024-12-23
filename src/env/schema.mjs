// @ts-check
import { z } from "zod";

/**
 * Specify your server-side environment variables schema here.
 * This way you can ensure the app isn't built with invalid env vars.
 */
export const serverSchema = z.object({
  DATABASE_URL: z.string().url(),
  NODE_ENV: z.enum(["development", "test", "production"]),
  NEXTAUTH_SECRET:
    process.env.NODE_ENV === "production"
      ? z.string().min(1)
      : z.string().min(1).optional(),
  NEXTAUTH_URL: z.preprocess(
    // This makes Vercel deployments not fail if you don't set NEXTAUTH_URL
    // Since NextAuth.js automatically uses the VERCEL_URL if present.
    (str) => process.env.VERCEL_URL ?? str,
    // VERCEL_URL doesn't include `https` so it cant be validated as a URL
    process.env.VERCEL ? z.string() : z.string().url(),
  ),
  EMAIL_SERVER_PORT: z.string(),
  EMAIL_SERVER_HOST: z.string(),
  EMAIL_SERVER_USER: z.string(),
  EMAIL_SERVER_PASSWORD: z.string(),
  EMAIL_FROM: z.string().email(),
  EMAIL_NOTIFY_ADDRESS: z.string().email(),
  PUSHER_APP_HOST: z.string().optional(),
  PUSHER_APP_PORT: z.string().optional(),
  PUSHER_APP_ID: z.string(),
  PUSHER_APP_KEY: z.string(),
  PUSHER_APP_SECRET: z.string(),
  PUSHER_APP_CLUSTER: z.string(),
  PUSHER_APP_USE_TLS: z.string(),
});

/**
 * You can't destruct `process.env` as a regular object in the Next.js
 * middleware, so you have to do it manually here.
 * @type {{ [k in keyof z.input<typeof serverSchema>]: string | undefined }}
 */
export const serverEnv = {
  DATABASE_URL: process.env.DATABASE_URL,
  NODE_ENV: process.env.NODE_ENV,
  NEXTAUTH_SECRET: process.env.NEXTAUTH_SECRET,
  NEXTAUTH_URL: process.env.NEXTAUTH_URL,
  EMAIL_FROM: process.env.EMAIL_FROM,
  EMAIL_SERVER_HOST: process.env.EMAIL_SERVER_HOST,
  EMAIL_SERVER_USER: process.env.EMAIL_SERVER_USER,
  EMAIL_SERVER_PORT: process.env.EMAIL_SERVER_PORT,
  EMAIL_SERVER_PASSWORD: process.env.EMAIL_SERVER_PASSWORD,
  EMAIL_NOTIFY_ADDRESS: process.env.EMAIL_NOTIFY_ADDRESS,
  PUSHER_APP_HOST: process.env.PUSHER_APP_HOST,
  PUSHER_APP_PORT: process.env.PUSHER_APP_PORT,
  PUSHER_APP_ID: process.env.PUSHER_APP_ID,
  PUSHER_APP_KEY: process.env.PUSHER_APP_KEY,
  PUSHER_APP_SECRET: process.env.PUSHER_APP_SECRET,
  PUSHER_APP_USE_TLS: process.env.PUSHER_APP_USE_TLS,
  PUSHER_APP_CLUSTER: process.env.PUSHER_APP_CLUSTER,
};

/**
 * Specify your client-side environment variables schema here.
 * This way you can ensure the app isn't built with invalid env vars.
 * To expose them to the client, prefix them with `NEXT_PUBLIC_`.
 */
export const clientSchema = z.object({
  NEXT_PUBLIC_PUSHER_APP_HOST: z.string().optional(),
  NEXT_PUBLIC_PUSHER_APP_PORT: z.string().optional(),
  NEXT_PUBLIC_PUSHER_APP_KEY: z.string(),
  NEXT_PUBLIC_PUSHER_APP_CLUSTER: z.string(),
  NEXT_PUBLIC_PUSHER_APP_USE_TLS: z.string().default("false"),
});

/**
 * You can't destruct `process.env` as a regular object, so you have to do
 * it manually here. This is because Next.js evaluates this at build time,
 * and only used environment variables are included in the build.
 * @type {{ [k in keyof z.input<typeof clientSchema>]: string | undefined }}
 */
export const clientEnv = {
  NEXT_PUBLIC_PUSHER_APP_HOST: process.env.NEXT_PUBLIC_PUSHER_APP_HOST,
  NEXT_PUBLIC_PUSHER_APP_PORT: process.env.NEXT_PUBLIC_PUSHER_APP_PORT,
  NEXT_PUBLIC_PUSHER_APP_KEY: process.env.NEXT_PUBLIC_PUSHER_APP_KEY,
  NEXT_PUBLIC_PUSHER_APP_CLUSTER: process.env.NEXT_PUBLIC_PUSHER_APP_CLUSTER,
  NEXT_PUBLIC_PUSHER_APP_USE_TLS: process.env.NEXT_PUBLIC_PUSHER_APP_USE_TLS,
};
