import * as React from 'react';
import { AppBar, Container, Toolbar, Box, IconButton, Menu, MenuItem, Typography, Button, Tooltip } from '@mui/material';
import MenuIcon from '@mui/icons-material/Menu';
import { signOut } from 'next-auth/react';
import { AccountCircle } from '@mui/icons-material';
import { useSnackbar } from 'notistack';
import { api } from '../utils/api';
import { env } from '../env/client.mjs';
import Pusher from 'pusher-js';

type Page = {
  name: string;
  id: number;
  path: string;
}


const pages: Page[] = [
  {
    name: 'Home',
    id: 1,
    path: '/',
  },
]

type Action = {
  name: string;
  id: number;
  callback: () => void
};

const actions: Action[] = [
  {
    id: 1,
    name: 'Esci',
    // eslint-disable-next-line @typescript-eslint/no-misused-promises
    callback: () => signOut({ callbackUrl: '/' })
  }
];

export function ResponsiveAppBar() {

  const [anchorElNav, setAnchorElNav] = React.useState<null | HTMLElement>(null);
  const [anchorElUser, setAnchorElUser] = React.useState<null | HTMLElement>(null);

  const { data: user } = api.user.getCurrent.useQuery();

  const { enqueueSnackbar } = useSnackbar();
  const [remainingAccesses, setRemainingAccesses] = React.useState(user?.remainingAccesses);

  const handleOpenNavMenu = React.useCallback((event: React.MouseEvent<HTMLElement>) => {
    setAnchorElNav(event.currentTarget);
  }, []);

  const handleOpenUserMenu = React.useCallback((event: React.MouseEvent<HTMLElement>) => {
    setAnchorElUser(event.currentTarget);
  }, []);

  const handleCloseNavMenu = React.useCallback(() => {
    setAnchorElNav(null);
  }, []);

  const handleCloseUserMenu = React.useCallback(() => {
    setAnchorElUser(null);
  }, []);

  const utils = api.useContext();

  const handleNotifications = React.useCallback(() => Promise.all([
    utils.bookings.getAvailableSlots.invalidate(),
    utils.bookings.getCurrent.invalidate(),
  ]), [utils]);

  React.useEffect(() => {
    if (user?.remainingAccesses === undefined) return;
    if (user.remainingAccesses === remainingAccesses) return;

    setRemainingAccesses(remainingAccesses);

    enqueueSnackbar(`Accessi rimasti: ${user?.remainingAccesses}`, {
      variant: user?.remainingAccesses <= 0 ? 'warning' : 'success',
      anchorOrigin: {
        vertical: 'top',
        horizontal: 'center',
      },
    });
  }, [user?.remainingAccesses, enqueueSnackbar, setRemainingAccesses, remainingAccesses]);

  React.useEffect(() => {
    const pusher = new Pusher(env.NEXT_PUBLIC_PUSHER_APP_KEY, {
      cluster: env.NEXT_PUBLIC_PUSHER_APP_CLUSTER,
      wsHost: env.NEXT_PUBLIC_PUSHER_APP_HOST,
      wsPort: env.NEXT_PUBLIC_PUSHER_APP_PORT ? parseInt(env.NEXT_PUBLIC_PUSHER_APP_PORT) : undefined,
      forceTLS: env.NEXT_PUBLIC_PUSHER_APP_USE_TLS === 'true',
      enabledTransports: env.NEXT_PUBLIC_PUSHER_APP_HOST ? ['ws', 'wss'] : undefined,
    });

    const channel = pusher.subscribe('booking');
    channel.bind('refresh', handleNotifications);

    return () => {
      pusher.disconnect();
    }
  }, [handleNotifications]);

  return (
    <AppBar position="static">
      <Container maxWidth="xl">
        <Toolbar disableGutters>
          <Box sx={{ flexGrow: 1, display: { xs: 'flex', md: 'none' } }}>
            <IconButton
              size="large"
              aria-label="account of current user"
              aria-controls="menu-appbar"
              aria-haspopup="true"
              onClick={handleOpenNavMenu}
              color="inherit"
            >
              <MenuIcon />
            </IconButton>
            <Menu
              id="menu-appbar"
              anchorEl={anchorElNav}
              anchorOrigin={{
                vertical: 'bottom',
                horizontal: 'left',
              }}
              keepMounted
              transformOrigin={{
                vertical: 'top',
                horizontal: 'left',
              }}
              open={Boolean(anchorElNav)}
              onClose={handleCloseNavMenu}
              sx={{
                display: { xs: 'block', md: 'none' },
              }}
            >
              {pages.map((page) => (
                <MenuItem component="a" href={page.path} key={page.id}>
                  <Typography textAlign="center">{page.name}</Typography>
                </MenuItem>
              ))}
            </Menu>
          </Box>
          <Box sx={{ flexGrow: 1, display: { xs: 'none', md: 'flex' } }}>
            {pages.map((page) => (
              <Button
                key={page.id}
                href={page.path}
                sx={{ my: 2, color: 'white', display: 'block' }}
              >
                {page.name}
              </Button>
            ))}
          </Box>

          <Box sx={{ flexGrow: 0 }}>
            <Tooltip title="Open settings">
              <IconButton color="inherit" onClick={handleOpenUserMenu} sx={{ p: 0 }}>
                <AccountCircle />
              </IconButton>
            </Tooltip>
            <Menu
              sx={{ mt: '45px' }}
              id="menu-appbar"
              anchorEl={anchorElUser}
              anchorOrigin={{
                vertical: 'top',
                horizontal: 'right',
              }}

              keepMounted
              transformOrigin={{
                vertical: 'top',
                horizontal: 'right',
              }}
              open={Boolean(anchorElUser)}
              onClose={handleCloseUserMenu}
            >
              {actions.map((action) => (
                // eslint-disable-next-line @typescript-eslint/no-misused-promises, @typescript-eslint/no-unsafe-assignment
                <MenuItem key={action.id} onClick={action.callback}>
                  <Typography textAlign="center">{action.name}</Typography>
                </MenuItem>
              ))}
            </Menu>
          </Box>
        </Toolbar>
      </Container>
    </AppBar>
  );
}
