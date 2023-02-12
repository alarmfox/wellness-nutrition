import * as React from 'react';
import { useSession } from 'next-auth/react';
import type { Booking } from '@prisma/client';
import { api } from '../utils/api';
import { Container, CssBaseline, Box, Typography, Button, Alert, 
  ListItemButton, 
  ListItemIcon, ListItemText, CircularProgress,
  Stack, Backdrop, 
  } from '@mui/material';
import { ResponsiveAppBar } from '../components/AppBar';
import { useConfirm } from 'material-ui-confirm';
import type { ListChildComponentProps} from 'react-window';
import { FixedSizeList } from 'react-window';
import { Delete, Event } from '@mui/icons-material';
import AdminLayout from '../components/AdminLayout';
import { formatBooking, formatDate } from '../utils/format.utils';
import { DateTime } from 'luxon';
import { Scheduler } from '../components/Scheduler';

import "react-big-calendar/lib/css/react-big-calendar.css";
import { Subscription } from '../components/Subscription';

function Home () {
  const { data: sessionData } = useSession();

  const { data: user, isLoading } = api.user.getCurrent.useQuery();
  const [ creationMode, setCreationMode ] = React.useState(false);
  const cannotCreateBooking = React.useMemo(() =>
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion, @typescript-eslint/no-non-null-asserted-optional-chain
    (user?.remainingAccesses! <= 0) || (DateTime.fromJSDate(user?.expiresAt || new Date()) < DateTime.now()), 
  [user]);
  
  React.useEffect(() => {
      // eslint-disable-next-line @typescript-eslint/no-non-null-assertion, @typescript-eslint/no-non-null-asserted-optional-chain
      if (user?.remainingAccesses! <= 0) setCreationMode(false);
  }, [user])

  if (sessionData?.user.role === 'ADMIN') {
    return (
      <AdminLayout>
        <Scheduler />
      </AdminLayout>
    )
  }

  return (
    <>
      <ResponsiveAppBar />
      <Container fixed component="main" maxWidth="xs">
        <CssBaseline />
        <Box
          sx={{
            marginTop: 3,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            overflowY: 'hidden',
            overflowX: 'hidden',
          }}
        > 
          <Subscription />
          {isLoading && <CircularProgress />}
          <Stack>
            <Typography gutterBottom variant="h6" >{creationMode ? 'Seleziona uno slot' : 'Lista prenotazioni'}</Typography>
          </Stack>
          {!creationMode ? 
            <BookingList />: <SlotList/>}
          <Button 
            disabled={cannotCreateBooking} 
            sx={{ mt: '2rem', bottom: 0, position: 'absolute', mb: '1rem' }}
            variant="contained" 
            color="primary" 
            aria-label="nuova prenotazione"
            onClick={() => setCreationMode(!creationMode)}
            >
            {creationMode ? 'Le mie prenotazioni' : 'Nuova prenotazione'}
          </Button>
        </Box>
      </Container>
    </>
  )
}

Home.auth = {
  isProtected: true,
  role: ['USER', 'ADMIN']
}

export default Home;

function RenderBooking(props: ListChildComponentProps<Booking[]>) {
  const { index, style, data } = props;
  const booking = data[index];
  const confirm = useConfirm();
  const utils = api.useContext();
  const  [error, setError] = React.useState<string | undefined>(undefined);
  
  const { mutate, isLoading } = api.bookings.delete.useMutation({
    onSuccess:  () => Promise.all([
        utils.bookings.getCurrent.invalidate(),
        utils.user.getCurrent.invalidate(),
    ]),
    onError: (err) => {
      if (err?.data?.code === 'NOT_FOUND') {
        setError('Impossibile trovare la prenotazione')
        return;
      }
      setError('Errore sconosciuto')
    }
  })
  
  const handleClick = React.useCallback(async ({ id, startsAt }: Booking) => {
    try {
      const isRefundable = DateTime.fromJSDate(startsAt).diffNow().as('hours') > 3; 
      await confirm({
        description: !isRefundable ? 
          'Sicuro di voler eliminare questa prenotazione? L\'accesso NON sarà rimborsato!' :
          'Sicuro di voler eliminare questa prenotazione?' ,
          title: 'Conferma',
          cancellationText: 'Annulla',
          confirmationText: 'Conferma',
      })
      mutate({
        id,
        startsAt,
        isRefundable,
      });
      
    } catch (error) {
      if (error) console.log(error);
    }
  }, [confirm, mutate]);
    
  if (!data) return <div>no data</div>
  if (!booking) return <div>no data</div>
  return (
      <ListItemButton divider disabled={DateTime.fromJSDate(booking.startsAt) < DateTime.now()} key={index} style={style}>
        {error ? <Alert severity="error">{error}</Alert> : <>
        <ListItemIcon sx={{ fontSize: 16 }}>
          <Event />
        </ListItemIcon>
        <ListItemText
          sx={{ my: '1rem' }}
          primary={formatBooking(booking.startsAt, undefined, DateTime.DATETIME_FULL)}
          primaryTypographyProps={{
            fontSize: 16,
            fontWeight: 'medium',
            letterSpacing: 0,
          }}
          secondary={`Effettuata ${formatDate(booking.createdAt, DateTime.DATETIME_MED)}`}
        />
        {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
        {isLoading ? <CircularProgress /> : <ListItemIcon sx={{fontSize: 18}} onClick={() => handleClick(booking)}>
          <Delete />
        </ListItemIcon>
        }
        </>
}
    </ListItemButton>
  );
}
function BookingList() {
  const { data } = api.bookings.getCurrent.useQuery();
  return (
  <Box sx={{ width: '100%', maxWidth: 360, bgcolor: 'background.paper' }}>
  {data && data.length > 0 ? 
    <FixedSizeList
      height={350}
      width={360}
      itemSize={70}
      itemCount={data.length}
      itemData={data}
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

interface CreateBookingFromSlotProps {
  slot: string;
  cb: (s: string) => Promise<void>
}


function RenderSlot(props: ListChildComponentProps<CreateBookingFromSlotProps[]>) {
  
  const { index, style, data } = props;
  if (!data) return <div>no data</div>
  
  const slot = data[index]?.slot;
  const cb = data[index]?.cb;
  
  if (!slot) return <div>no data</div>
  if (!cb) return <div>no data</div>
  return (
    <ListItemButton divider onClick={() => void cb(slot)} style={style}>
      <ListItemIcon>
        <Event />
      </ListItemIcon>
      <ListItemText
        sx={{ my: '1rem' }}
        primary={formatDate(slot, DateTime.DATE_MED_WITH_WEEKDAY)}
        primaryTypographyProps={{
          fontSize: 16,
          fontWeight: 'medium',
          letterSpacing: 0,
        }}
        secondary={`Dalle ${DateTime.fromISO(slot).toFormat('HH:mm')}
        alle ${DateTime.fromISO(slot).plus({ hours: 1 }).toFormat('HH:mm')}`}
      />
    </ListItemButton>
  )
}
function SlotList() {
    const utils = api.useContext();
    const { data, isLoading: isFetching } = api.bookings.getAvailableSlots.useQuery();
    const [error, setError] = React.useState<string | undefined>(undefined);
    
    const confirm = useConfirm();
    const { mutate, isLoading: isCreating } = api.bookings.create.useMutation({
      onSuccess:  () => Promise.all([
        utils.bookings.getCurrent.invalidate(),
        utils.user.getCurrent.invalidate(),
        utils.bookings.getAvailableSlots.invalidate(),
      ]),
      onError: (err) => {
        if (err?.data?.code === 'NOT_FOUND') {
          setError('Impossibile trovare la prenotazione')
          return;
        }
        setError('Errore sconosciuto')
      }
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
          startsAt: DateTime.fromISO(startsAt).toJSDate(),
        });
      
      } catch (error) {
        if (error) console.log(error);
      }
    }, [confirm, mutate]); 

    const isLoading = React.useMemo(() => isFetching || isCreating,  [isFetching, isCreating]);
    const rows = React.useMemo(() => {
      return data?.map((item): CreateBookingFromSlotProps => {
        return {
          slot: item,
          cb: handleClick
        }
      })
    }, [data, handleClick]);

    return (
      <Box sx={{ width: '100%', maxWidth: 360, bgcolor: 'background.paper' }}>
        <Backdrop
          sx={{color: 'darkgrey', zIndex: (theme) => theme.zIndex.drawer + 1}}
          open={isLoading}
        >
          <Stack spacing={2}>
            <CircularProgress sx={{textAlign: 'center'}} />
          </Stack>
          
        </Backdrop>
          {error && <Alert variant="filled" severity="error">{error}</Alert>}
          {rows &&
            <FixedSizeList
              height={350}
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

