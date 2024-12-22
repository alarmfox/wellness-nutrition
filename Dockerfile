FROM node:lts-alpine AS deps

WORKDIR /app

ENV TZ=Europe/Rome

# https://github.com/prisma/prisma/discussions/19341
RUN apk add --no-cache openssl tzdata

COPY prisma ./

# Install dependencies based on the preferred package manager

COPY package.json yarn.lock* package-lock.json* pnpm-lock.yaml\* ./

RUN \
  if [ -f yarn.lock ]; then yarn --frozen-lockfile; \
  elif [ -f package-lock.json ]; then npm ci; \
  elif [ -f pnpm-lock.yaml ]; then yarn global add pnpm && pnpm i; \
  else echo "Lockfile not found." && exit 1; \
  fi

##### BUILDER

FROM node:lts-alpine AS builder

ENV TZ=Europe/Rome
# https://github.com/prisma/prisma/discussions/19341
RUN apk add --no-cache openssl tzdata

# client var
ARG NEXT_PUBLIC_PUSHER_APP_HOST
ARG NEXT_PUBLIC_PUSHER_APP_PORT
ARG NEXT_PUBLIC_PUSHER_APP_KEY
ARG NEXT_PUBLIC_PUSHER_APP_CLUSTER
ARG NEXT_PUBLIC_PUSHER_APP_USE_TLS

WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

ENV NEXT_TELEMETRY_DISABLED 1
ENV IS_DOCKER 1

RUN npx prisma generate

RUN \
  if [ -f yarn.lock ]; then SKIP_ENV_VALIDATION=1 yarn build; \
  elif [ -f package-lock.json ]; then SKIP_ENV_VALIDATION=1 npm run build; \
  elif [ -f pnpm-lock.yaml ]; then yarn global add pnpm && SKIP_ENV_VALIDATION=1 pnpm run build; \
  else echo "Lockfile not found." && exit 1; \
  fi

##### RUNNER
FROM node:lts-alpine AS runner
WORKDIR /app

ENV TZ=Europe/Rome
# https://github.com/prisma/prisma/discussions/19341
RUN apk add --no-cache openssl tzdata

ENV NODE_ENV production

ENV NEXT_TELEMETRY_DISABLED 1

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

COPY --from=builder /app/next.config.mjs ./
COPY --from=builder /app/public ./public
COPY --from=builder /app/package.json ./package.json

COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

COPY --chown=nextjs:nodejs prisma ./prisma

USER nextjs
EXPOSE 3000
ENV PORT 3000

CMD ["node", "server.js"]
