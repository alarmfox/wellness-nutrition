import type { GetServerSidePropsContext } from "next";
import {
  getServerSession,
  type NextAuthOptions,
  type DefaultSession,
} from "next-auth";
import { PrismaAdapter } from "@next-auth/prisma-adapter";
import { prisma } from "./db";
import CredentialsProvider from 'next-auth/providers/credentials';
import type { Role, SubType } from "@prisma/client";
import argon2 from "argon2";

/**
 * Module augmentation for `next-auth` types.
 * Allows us to add custom properties to the `session` object and keep type
 * safety.
 *
 * @see https://next-auth.js.org/getting-started/typescript#module-augmentation
 **/
declare module "next-auth" {
  interface Session extends DefaultSession {
    user: {
      id: string;
      // ...other properties
      role: Role;
      subType: SubType;
    } & DefaultSession["user"];
  }

  interface User {
    // ...other properties
    role: Role;
    subType: SubType;
  }
}

/**
 * Options for NextAuth.js used to configure adapters, providers, callbacks,
 * etc.
 *
 * @see https://next-auth.js.org/configuration/options
 **/
export const authOptions: NextAuthOptions = {
  callbacks: {
    jwt({ token, user }) {
      if (user) {
        token.role = user.role;
        token.subType = user.subType;
      }
      return token
    },
    session({ session, token }) {
      if (session.user) {
        session.user.id = token.sub || '';
        session.user.role = token.role as Role;
        session.user.subType = token.subType as SubType;
      }
      return session;
    },
    redirect({ baseUrl }) {
      return baseUrl;

    }
  },

  session: {
    strategy: 'jwt',
    // Seconds - How long until an idle session expires and is no longer valid.
    maxAge: 30 * 24 * 60 * 60, // 30 days

    // Seconds - Throttle how frequently to write to database to extend a session.
    // Use it to limit write operations. Set to 0 to always update the database.
    // Note: This option is ignored if using JSON Web Tokens

    updateAge: 24 * 60 * 60, // 24 hours
  },

  adapter: PrismaAdapter(prisma),

  providers: [
    CredentialsProvider({
      name: 'Credentials',
      credentials: {
        email: {
          label: "Email", type: "email", placeholder: "Indirizzo email",
        },
        password: {
          label: "Password", type: "password",
        }
      },
      async authorize(credentials) {
        const user = await prisma.user.findUnique({
          where: {
            email: credentials?.email
          }
        })

        if (!user || !user.emailVerified || !user.password)
          return null;

        if (await argon2.verify(user.password, credentials?.password || '')) {
          return user;
        }
        return null

      },
    })
    /**
     * ...add more providers here
     *
     * Most other providers require a bit more work than the Discord provider.
     * For example, the GitHub provider requires you to add the
     * `refresh_token_expires_in` field to the Account model. Refer to the
     * NextAuth.js docs for the provider you want to use. Example:
     * @see https://next-auth.js.org/providers/github
     **/
  ],
  pages: {
    signIn: '/signin',
    error: '/signin',
  }
};

/**
 * Wrapper for `getServerSession` so that you don't need to import the
 * `authOptions` in every file.
 *
 * @see https://next-auth.js.org/configuration/nextjs
 **/
export const getServerAuthSession = (ctx: {
  req: GetServerSidePropsContext["req"];
  res: GetServerSidePropsContext["res"];
}) => {
  return getServerSession(ctx.req, ctx.res, authOptions);
};
