import * as React from 'react';
import { styled, useTheme } from '@mui/material/styles';
import PeopleIcon from '@mui/icons-material/People';
import EventIcon from '@mui/icons-material/Event';
import { Box, CssBaseline, Toolbar, IconButton, Typography, Drawer,
    Divider, List, ListItem, ListItemButton, 
    ListItemIcon, ListItemText, Menu, MenuItem, Badge, Popover } from '@mui/material';
import type { AppBarProps as MuiAppBarProps } from '@mui/material/AppBar';
import MuiAppBar from '@mui/material/AppBar';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import MenuIcon from '@mui/icons-material/Menu';
import { useRouter } from 'next/router';
import { AccountCircle, Close, History, InfoOutlined, Logout, NotificationsRounded } from '@mui/icons-material';
import { signOut } from 'next-auth/react';
import Pusher from 'pusher-js';
import { useSnackbar } from 'notistack';
import type { NotificationModel} from '../utils/event.schema';
import { DateTime } from 'luxon';
import { api } from '../utils/api';
import Image from 'next/image';
import { env } from '../env/client.mjs';
import { formatDate } from '../utils/format.utils';

const drawerWidth = 240;
const Main = styled('main', { shouldForwardProp: (prop) => prop !== 'open' })<{
  open?: boolean;
}>(({ theme, open }) => ({
  flexGrow: 1,
  padding: theme.spacing(3),
  transition: theme.transitions.create('margin', {
    easing: theme.transitions.easing.sharp,
    duration: theme.transitions.duration.leavingScreen,
  }),
  marginLeft: `-${drawerWidth}px`,
  ...(open && {
    transition: theme.transitions.create('margin', {
      easing: theme.transitions.easing.easeOut,
      duration: theme.transitions.duration.enteringScreen,
    }),
    marginLeft: 0,
  }),
}));

interface AppBarProps extends MuiAppBarProps {
  open?: boolean;
}

const AppBar = styled(MuiAppBar, {
  shouldForwardProp: (prop) => prop !== 'open',
})<AppBarProps>(({ theme, open }) => ({
  transition: theme.transitions.create(['margin', 'width'], {
    easing: theme.transitions.easing.sharp,
    duration: theme.transitions.duration.leavingScreen,
  }),
  ...(open && {
    width: `calc(100% - ${drawerWidth}px)`,
    marginLeft: `${drawerWidth}px`,
    transition: theme.transitions.create(['margin', 'width'], {
      easing: theme.transitions.easing.easeOut,
      duration: theme.transitions.duration.enteringScreen,
    }),
  }),
}));

const DrawerHeader = styled('div')(({ theme }) => ({
  display: 'flex',
  alignItems: 'center',
  padding: theme.spacing(0, 1),
  // necessary for content to be below app bar
  ...theme.mixins.toolbar,
  justifyContent: 'flex-end',
}));

type NavigationOption = {
  name: string;
  to: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  icon: any
}

const NavigationOptions: NavigationOption[] = [
  {
    icon: <PeopleIcon />,
    name: 'Utenti',
    to: '/users'
  },
  {
    icon: <EventIcon />,
    name: 'Calendario',
    to: '/' 
  },
  {
    icon: <History />,
    name: 'Eventi',
    to: '/events'
  }
]

const pusher = new Pusher(env.NEXT_PUBLIC_PUSHER_APP_KEY, {
  cluster: env.NEXT_PUBLIC_PUSHER_APP_CLUSTER
});
const channel = pusher.subscribe('booking');

function formatNotification(n: NotificationModel): string {
  if (n.type === 'DELETED') {
    return `${n.firstName} ${n.lastName} ha cancellato la sua prenotazione per ${formatDate(n.startsAt)} `
  }
  return `${n.firstName} ${n.lastName} ha creato una prenotazione per ${formatDate(n.startsAt)}`
}

export default function AdminLayout ({ children }: React.PropsWithChildren) {
  const theme = useTheme();
  const [open, setOpen] = React.useState(true);
  const [anchorEl, setAnchorEl] = React.useState<null | HTMLElement>(null);
  const [ selected, setSelected ] = React.useState('Calendario');
  const { pathname } = useRouter();
  const [notifications, setNotifications] = React.useState<NotificationModel[]> ([]);
  const [notificatioAnchorEl, setNotificatioAnchorEl] = React.useState<HTMLButtonElement | null>(null);
  const notificationPopoverOpen = React.useMemo(() => !(notificatioAnchorEl === null), [notificatioAnchorEl]);
  const id = notificationPopoverOpen ? 'notification-popover' : undefined;

  const { enqueueSnackbar } = useSnackbar();
  const utils = api.useContext();

  const handleMenu = React.useCallback((event: React.MouseEvent<HTMLElement>) =>  setAnchorEl(event.currentTarget), [])

  const handleClose = React.useCallback(() => setAnchorEl(null), []);

  const handleDrawerOpen = React.useCallback(() => setOpen(true), []);

  const handleDrawerClose = React.useCallback(() =>  setOpen(false), []);

  const onLogout = React.useCallback(() =>  signOut({ callbackUrl: '/' }).catch(console.error), []);

  const handleNotification = React.useCallback((n: NotificationModel) => {
    enqueueSnackbar(formatNotification(n), {
      variant: n.type === 'CREATED' ? 'success' : 'warning'
    });
    setNotifications(notifications.concat(n));
    void utils.bookings.getByInterval.invalidate();
    void utils.events.getLatest.invalidate();
  } ,[notifications, setNotifications, enqueueSnackbar, utils]);

  const onDelete = React.useCallback((i: number) => {
    notifications.splice(i, 1);
    setNotifications([...notifications]);
  }, [notifications, setNotifications])

  React.useEffect(() => {
    channel.bind('user', (data: NotificationModel) => handleNotification(data));
  }, [handleNotification])

  React.useEffect(() => {
     switch(pathname) {
      case "/users":
        setSelected('Utenti');
        break;
      case "/":
        setSelected('Calendario');
        break;
     }
  }, [pathname])

  return (
    <Box sx={{ display: 'flex' }}>
      <CssBaseline />
      <AppBar position="fixed" open={open}>
        <Toolbar>
          <IconButton
            color="inherit"
            aria-label="open drawer"
            onClick={handleDrawerOpen}
            edge="start"
            sx={{ mr: 2, ...(open && { display: 'none' }) }}
          >
            <MenuIcon />
          </IconButton>
          <Typography variant="h6" component="div">
            {selected}
          </Typography>
          <Box sx={{ display: 'flex', width: '100%', justifyContent: 'end'}}>
            <IconButton color="inherit" onClick={(e) => setNotificatioAnchorEl(notificatioAnchorEl ? null : e.currentTarget)}>
              <Badge max={99} badgeContent={notifications.length}>
                <NotificationsRounded  color="inherit" />
              </Badge>
              <Popover
                id={id}
                open={notificationPopoverOpen}
                anchorEl={notificatioAnchorEl}
                onClose={() => setNotificatioAnchorEl(null)}
                anchorOrigin={{
                  vertical: 'bottom',
                  horizontal: 'left'
                }}
                transformOrigin={{
                  vertical: 'top',
                  horizontal: 'left'
                }}
              >
                {notifications.length > 0 ? 
                  <NotificationList onClose={() => setNotificatioAnchorEl(null)} onDelete={onDelete} notifications={notifications}/> :
                  <div onMouseLeave={() => setNotificatioAnchorEl(null)}>
                    <Typography variant="caption">
                      Nessuna notifica
                    </Typography>
                  </div> 
                }
              </Popover>
            </IconButton>
            <IconButton
              size="large"
              aria-label="account of current user"
              aria-controls="menu-appbar"
              aria-haspopup="true"
              onClick={handleMenu}
              color="inherit"
            >
              <AccountCircle />
            </IconButton>
            <Menu
              id="menu-appbar"
              anchorEl={anchorEl}
              anchorOrigin={{
                vertical: 'top',
                horizontal: 'right',
              }}
              keepMounted
              transformOrigin={{
                vertical: 'top',
                horizontal: 'right',
              }}
              open={Boolean(anchorEl)}
              onClose={handleClose}
            >
              {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
              <MenuItem onClick={onLogout}>Esci</MenuItem>
            </Menu>
          </Box>
        </Toolbar>
      </AppBar>
      <Drawer
        sx={{
          width: drawerWidth,
          flexShrink: 0,
          '& .MuiDrawer-paper': {
            width: drawerWidth,
            boxSizing: 'border-box',
          },
        }}
        variant="persistent"
        anchor="left"
        open={open}
      >
        <DrawerHeader>
          <Box sx={{
            display: 'flex',
            width: '100%',
          }}>
            <Image style={{ display: 'flex', justifyContent: 'flex-end'}} src="/logo.png" width="60" height="60" alt="logo"/>
            <IconButton onClick={handleDrawerClose}>
              {theme.direction === 'ltr' ? <ChevronLeftIcon /> : <ChevronRightIcon />}
            </IconButton>
          </Box>
        </DrawerHeader>
        <Divider />
        <List>
          {NavigationOptions.map((e) => (
            <ListItem key={e.name} disablePadding>
              <ListItemButton component="a" href={e.to}>
                <ListItemIcon>
                  {e.icon}
                </ListItemIcon>
                <ListItemText primary={e.name} />
              </ListItemButton>
            </ListItem>
          ))}
        </List>
        
        <Divider />
        <List>
          <ListItem key="logout" disablePadding>
            {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
            <ListItemButton onClick={onLogout}> 
              <ListItemIcon>
                <Logout />
              </ListItemIcon>
              <ListItemText primary="Esci"/>
            </ListItemButton>
          </ListItem>
        </List>
      </Drawer>
      <Main open={open}>
        <DrawerHeader />
        { children }
      </Main>
    </Box>
  );
}
interface NotificationListProps {
  notifications: NotificationModel[];
  onDelete: (index: number) => void;
  onClose: () => void;
}

function NotificationList({ notifications, onDelete, onClose}: NotificationListProps) {
  return (
    <div onMouseLeave={onClose}>
      <List  dense sx={{ width: '100%', bgcolor: 'background.paper' }}>
        {notifications.map((value, index) => {
          const labelId = `checkbox-list-secondary-label-${value.id}`;
          return (
            <ListItem
              key={value.id}
              secondaryAction={
                <IconButton onClick={() => onDelete(index)} edge="end">
                  <Close />
                </IconButton>
              }
              disablePadding
            >
              <ListItemButton>
                <ListItemIcon>
                  <InfoOutlined />
                </ListItemIcon>
                <ListItemText id={labelId}
                  secondary={DateTime.fromISO(value.occurredAt).setLocale('it').toRelativeCalendar()}
                  primary={formatNotification(value)} />
              </ListItemButton>
            </ListItem>
          );
        })}
      </List>
    </div>
  ); 
}