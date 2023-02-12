import Pusher from "pusher";
import { env } from "../env/server.mjs";

export const pusher = new Pusher({
  appId: env.PUSHER_APP_ID,
  key: env.PUSHER_APP_KEY,
  secret: env.PUSHER_APP_SECRET,
  cluster: env.PUSHER_APP_CLUSTER,
  useTLS: env.PUSHER_APP_USE_TLS === 'true',
});

/* const pusher = new Pusher({
  appId: "1552406",
  key: "a017f5abd9769da5b770",
  secret: "17d3e9411e7a65ce6e15",
  cluster: "eu",
  useTLS: true
});
 */