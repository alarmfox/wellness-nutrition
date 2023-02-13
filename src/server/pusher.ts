import Pusher from "pusher";
import { env } from "../env/server.mjs";

export const pusher = new Pusher({
  appId: env.PUSHER_APP_ID,
  key: env.PUSHER_APP_KEY,
  secret: env.PUSHER_APP_SECRET,
  cluster: env.PUSHER_APP_CLUSTER,
  useTLS: env.PUSHER_APP_USE_TLS === 'true',
  host: env.PUSHER_APP_HOST,
  port: env.PUSHER_APP_PORT,
});
