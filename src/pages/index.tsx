import * as React from 'react';
import { signOut, useSession } from 'next-auth/react';
import type { Booking } from '@prisma/client';
import { api } from '../utils/api';
import {
  Container, CssBaseline, Box, Typography,
  ListItemButton,
  ListItemIcon, ListItemText, CircularProgress,
  Stack, Backdrop, useMediaQuery, BottomNavigation, BottomNavigationAction, Paper,
} from '@mui/material';
import { useConfirm } from 'material-ui-confirm';
import type { ListChildComponentProps } from 'react-window';
import { FixedSizeList } from 'react-window';
import { Delete, Event, Logout, ListSharp, AddRounded } from '@mui/icons-material';
import AdminLayout from '../components/AdminLayout';
import { formatBooking, formatDate, zone } from '../utils/date.utils';
import { DateTime } from 'luxon';
import { Scheduler } from '../components/Scheduler';
import { Subscription } from '../components/Subscription';
import { useSnackbar } from 'notistack';

function Home() {
  const { data: sessionData } = useSession();

  const { data: user } = api.user.getCurrent.useQuery();
  const [expanded, setExpanded] = React.useState(false);
  const [value, setValue] = React.useState(0);
  const matches = useMediaQuery('(max-height: 600px)');

  const confirm = useConfirm();

  const onLogout = React.useCallback(async () => {
    try {
      await confirm({
        description: 'Sicuro di voler uscire dall\'applicazione?',
        title: 'Conferma',
        confirmationText: 'Conferma',
        cancellationText: 'Annulla',
      });
      await signOut({ redirect: true });
    } catch (error) {
      console.log(error);
    }
  }, [confirm]);

  const cannotCreateBooking = React.useMemo(() =>
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion, @typescript-eslint/no-non-null-asserted-optional-chain
    (user?.remainingAccesses! <= 0) || (DateTime.fromJSDate(user?.expiresAt || new Date()) < DateTime.now()),
    [user]);

  React.useEffect(() => {
    if (cannotCreateBooking) setValue(0);
  }, [cannotCreateBooking, value]);

  const height = React.useMemo(() => {
    if (matches) {
      return expanded ? 75 : 300;
    }
    return expanded ? 400 : 600;
  }, [matches, expanded]);

  if (sessionData?.user.role === 'ADMIN') {
    return (
      <AdminLayout>
        <Scheduler />
      </AdminLayout>
    )
  }
  return (
    <React.Fragment>
      <Container fixed component="main" maxWidth="xs">
        <CssBaseline />
        <Box
          sx={{
            marginTop: '.5rem',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            overflowY: 'hidden',
            overflowX: 'hidden',
          }}
        >
          <Subscription setExpanded={setExpanded} />
          <Stack>
            <Typography gutterBottom variant="h6" >{value === 0 ? 'Lista prenotazioni' : 'Seleziona uno slot'}</Typography>
          </Stack>
          {value === 1 ?
            <SlotList height={height} /> : <BookingList height={height} />}
          <Paper sx={{ position: 'fixed', bottom: 0, left: 0, right: 0 }} elevation={3}>
            <BottomNavigation
              showLabels
              value={value}
              onChange={(_event, newValue) => {
                if (cannotCreateBooking && newValue === 1) return;
                setValue(newValue);
              }}
            >
              <BottomNavigationAction
                label="Prenotazioni"
                icon={<ListSharp />}
              />
              <BottomNavigationAction
                label="Crea"
                icon={<AddRounded />}
              />
              <BottomNavigationAction
                label="Esci"
                icon={<Logout color="error" />}
                //eslint-disable-next-line @typescript-eslint/no-misused-promises
                onClick={onLogout}
              />
            </BottomNavigation>
          </Paper>
        </Box>
      </Container>
    </React.Fragment>
  )
}

Home.auth = {
  isProtected: true,
  role: ['USER', 'ADMIN']
}

export default Home;

interface BookingListProps {
  height: number;
}

function BookingList({ height }: BookingListProps) {
  const confirm = useConfirm();
  const utils = api.useContext();
  const { enqueueSnackbar } = useSnackbar();
  const { data, isLoading: isFetching } = api.bookings.getCurrent.useQuery();
  const { mutate, isLoading: isDeleting } = api.bookings.delete.useMutation({
    onSuccess: () => Promise.all([
      utils.bookings.getCurrent.invalidate(),
      utils.user.getCurrent.invalidate(),
    ]),
    onError: (err) => {
      if (err?.data?.code === 'NOT_FOUND') {
        enqueueSnackbar('Impossibile trovare la prenotazione', {
          variant: 'error'
        });
      } else {
        enqueueSnackbar('Impossibile cancellare la prenotazione. Contattare l\'amministratore del sistema', {
          variant: 'error',
        });
      }
    }
  });
  const isLoading = React.useMemo(() => isFetching || isDeleting, [isFetching, isDeleting]);
  const handleClick = React.useCallback(async ({ id, startsAt }: Booking) => {
    try {
      const isRefundable = DateTime.fromJSDate(startsAt).setZone(zone).diffNow().as('hours') > 3;
      await confirm({
        description: !isRefundable ?
          'Sicuro di voler eliminare questa prenotazione? L\'accesso NON sarà rimborsato!' :
          'Sicuro di voler eliminare questa prenotazione?',
        title: 'Conferma',
        cancellationText: 'Annulla',
        confirmationText: 'Conferma',
      })
      mutate({
        id,
        startsAt: DateTime.fromJSDate(startsAt).setZone(zone).toJSDate(),
      });

    } catch (error) {
      if (error) console.log(error);
    }
  }, [confirm, mutate]);

  const rows = React.useMemo(() => {
    return data?.map((item): BookingActionProps => {
      return {
        booking: item,
        cb: handleClick
      }
    })
  }, [data, handleClick]);
  return (
    <Box sx={{ width: '100%', maxWidth: 360, bgcolor: 'background.paper' }}>
      <Backdrop
        sx={{ color: 'darkgrey', zIndex: (theme) => theme.zIndex.drawer + 1 }}
        open={isLoading}
      >
        <CircularProgress sx={{ textAlign: 'center' }} />
      </Backdrop>
      {data && data.length > 0 ?
        <FixedSizeList
          height={height}
          width={360}
          itemSize={70}
          itemCount={data.length}
          itemData={rows}
        >
          {RenderBooking}
        </FixedSizeList>
        :
        <Typography variant="caption" color="gray">
          Nessuna prenotazione
        </Typography>
      }
    </Box>
  );
}

interface BookingActionProps {
  booking: Booking;
  cb: (b: Booking) => Promise<void>
}
function RenderBooking(props: ListChildComponentProps<BookingActionProps[]>) {
  const { index, style, data } = props;
  const prop = data.at(index);
  if (!prop) return <div>no data</div>
  const { booking, cb } = prop;

  return (
    <ListItemButton divider disabled={DateTime.fromJSDate(booking.startsAt) < DateTime.now()} key={index} style={style}>
      <ListItemIcon sx={{ fontSize: 16 }}>
        <Event />
      </ListItemIcon>
      <ListItemText
        sx={{ my: '.5rem' }}
        primary={formatBooking(booking.startsAt, undefined, DateTime.DATETIME_FULL)}
        primaryTypographyProps={{
          fontSize: 16,
          fontWeight: 'medium',
          letterSpacing: 0,
        }}
        secondary={`Effettuata ${formatDate(booking.createdAt, DateTime.DATETIME_MED)}`}
      />
      {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
      <ListItemIcon sx={{ fontSize: 18 }} onClick={() => void cb(booking)}>
        <Delete />
      </ListItemIcon>
    </ListItemButton>
  );
}

interface SlotListProps {
  height: number;
}

function SlotList({ height }: SlotListProps) {
  const utils = api.useContext();
  const { data, isLoading: isFetching } = api.bookings.getAvailableSlots.useQuery();
  const { data: bookings } = api.bookings.getCurrent.useQuery();
  const { enqueueSnackbar } = useSnackbar();
  const confirm = useConfirm();

  const { mutate, isLoading: isCreating } = api.bookings.create.useMutation({
    onSuccess: () => Promise.all([
      utils.bookings.getCurrent.invalidate(),
      utils.user.getCurrent.invalidate(),
      utils.bookings.getAvailableSlots.invalidate(),
    ]),
    onError: async (err) => {
      switch (err?.data?.code) {
        case "BAD_REQUEST":
          enqueueSnackbar('Lo slot è stato disabilitato dall\'amministratore', { variant: 'error' });
          break;
        case "CONFLICT":
          enqueueSnackbar('Lo slot risulta già prenotato.', { variant: 'error' });
          break;
        case "UNAUTHORIZED":
          enqueueSnackbar('Al momento, lo stato del tuo abbonamento non ti consente di effettuare prenotazioni', { variant: 'error' });
          break;
        default:
          enqueueSnackbar('Impossibile creare la prenotazione. Contattare l\'amministratore', { variant: 'error' });
      }
      await Promise.all([
        utils.bookings.getAvailableSlots.invalidate(),
        utils.bookings.getCurrent.invalidate(),
      ]);
    },
  });

  const handleClick = React.useCallback(async (startsAt: string) => {
    try {
      await confirm({
        description: `Confermi la prenotazione per il giorno:
          ${formatBooking(startsAt, undefined, DateTime.DATETIME_FULL)}`,
        title: 'Conferma',
        cancellationText: 'Annulla',
        confirmationText: 'Conferma',
      });
      mutate({
        startsAt: DateTime.fromISO(startsAt).setZone(zone).toJSDate(),
      });

    } catch (error) {
      if (error) console.log(error);
    }
  }, [confirm, mutate]);

  const isLoading = React.useMemo(() => isFetching || isCreating, [isFetching, isCreating]);
  const rows = React.useMemo(() => {
    return data?.map((item): CreateBookingFromSlotProps => {
      return {
        slot: DateTime.fromISO(item),
        cb: handleClick,
        bookedDays: bookings ?
          new Set(bookings.map((item) => DateTime.fromJSDate(item.startsAt).startOf('day').toSeconds()))
          :
          new Set([]),
      }
    })
  }, [data, handleClick, bookings]);

  return (
    <Box sx={{ width: '100%', bgcolor: 'background.paper', overflowY: 'hidden' }}>
      <Backdrop
        sx={{ color: 'darkgrey', zIndex: (theme) => theme.zIndex.drawer + 1 }}
        open={isLoading}
      >
        <CircularProgress sx={{ textAlign: 'center' }} />
      </Backdrop>
      {rows &&
        <FixedSizeList
          height={height}
          width={360}
          itemSize={70}
          itemCount={rows.length}
          itemData={rows}
        >
          {RenderSlot}
        </FixedSizeList>
      }
    </Box>
  );
}

interface CreateBookingFromSlotProps {
  slot: DateTime;
  cb: (s: string) => Promise<void>;
  bookedDays: Set<number>;
}

function RenderSlot(props: ListChildComponentProps<CreateBookingFromSlotProps[]>) {

  const { index, style, data } = props;

  const prop = data.at(index);
  if (!prop) return <div>no data</div>
  const { slot, cb, bookedDays } = prop;

  return (
    <ListItemButton divider onClick={() => void cb(slot.toISO())} style={style}>
      <ListItemIcon>
        <Event color={bookedDays.has(slot.startOf('day').toSeconds()) ? 'warning' : 'success'} />
      </ListItemIcon>
      <ListItemText
        sx={{ my: '.5rem' }}
        primary={formatDate(slot.toJSDate(), DateTime.DATE_MED_WITH_WEEKDAY)}
        primaryTypographyProps={{
          fontSize: 16,
          fontWeight: 'medium',
          letterSpacing: 0,
        }}
        secondary={`Dalle ${slot.toFormat('HH:mm')}
        alle ${slot.plus({ hours: 1 }).toFormat('HH:mm')}`}
      />
    </ListItemButton>
  )
}
