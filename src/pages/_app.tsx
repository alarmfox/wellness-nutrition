import type { AppProps} from "next/app";
import { type AppType } from "next/app";
import { type Session } from "next-auth";
import { SessionProvider, useSession } from "next-auth/react";

import { api } from "../utils/api";

import { CircularProgress, createTheme, Stack, ThemeProvider, Typography } from "@mui/material";
import { ConfirmProvider } from "material-ui-confirm";
import { useRouter } from "next/router";
import type { NextComponentType } from "next";
import type { Role } from "@prisma/client";
import { SnackbarProvider } from "notistack";

const theme = createTheme();

export type AuthProps = {
  isProtected: boolean;
  role: Role[]
}

export type CustomAppProps = AppProps & {
  Component: NextComponentType & {auth?: AuthProps} // add auth type
}
const MyApp: AppType<{ session: Session | null } > = ({
  Component,
  pageProps: { session, ...pageProps },
}: CustomAppProps) => {
  return (
    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
    <SessionProvider session={session}>
      <ThemeProvider theme={theme}>
        <ConfirmProvider>
          <SnackbarProvider preventDuplicate autoHideDuration={5000}>
            {
              Component.auth ? (
                <Auth auth={Component.auth}>
                  <Component {...pageProps} />
                </Auth>
              ):
              <Component {...pageProps} />
            }
          </SnackbarProvider>
        </ConfirmProvider>
      </ThemeProvider>
    </SessionProvider>
  );
};
export default api.withTRPC(MyApp);

function Auth({ children, auth }: { children: JSX.Element, auth: AuthProps}) {
  const router = useRouter();
  const { data, status } = useSession({ 
    required: true, 
    // eslint-disable-next-line @typescript-eslint/no-misused-promises
    onUnauthenticated: () => router.replace('/signin').catch(console.error) 
  })
  if (status === "loading") {
    return (
     <div style={{ minHeight: '100%', display: 'flex', justifyContent: 'center', alignItems: 'center'}} >
       <Stack>
         <CircularProgress />
         <Typography color="gray" variant="caption"> Caricamento in corso...</Typography>
       </Stack>
     </div>
    )
  }

  if (!auth.role.includes(data.user.role)) {
    return <div>Forbidden</div>
  }
  return children;
}

