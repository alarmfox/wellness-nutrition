import * as React from 'react';
import { useSession } from 'next-auth/react';
import type { Booking } from '@prisma/client';
import { api } from '../utils/api';
import { Container, CssBaseline, Box, Typography, Button, Alert, Card, 
  CardContent, Grid, ListItemButton, 
  ListItemIcon, ListItemText, CircularProgress, 
  } from '@mui/material';
import { ResponsiveAppBar } from '../components/AppBar';
import { DateTime } from 'luxon';
import { useConfirm } from 'material-ui-confirm';
import type { ListChildComponentProps} from 'react-window';
import { FixedSizeList } from 'react-window';
import { Delete, Event } from '@mui/icons-material';
import AdminLayout from '../components/AdminLayout';
import { Calendar, luxonLocalizer } from 'react-big-calendar'
import "react-big-calendar/lib/css/react-big-calendar.css";
import type { IntervalModel } from '../utils/booking.schema';

function Home () {
  const { data: sessionData } = useSession();

  const { data: user, isLoading } = api.user.getCurrent.useQuery();
  const [ creationMode, setCreationMode ] = React.useState(false);
  
  React.useEffect(() => {
      if (user?.remainingAccesses === 0) setCreationMode(false);
  }, [user])

  if (sessionData?.user.role === 'ADMIN') return <Admin />


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
        > {isLoading && <CircularProgress />}
          <SubscriptionInfo />
          {!creationMode ? <BookingList />: <SlotList/>}
          <Button 
            disabled={user?.remainingAccesses === 0 || DateTime.fromJSDate(user?.expiresAt || new Date()) < DateTime.now()} 
            sx={{ mt: '2rem' }}
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

function SubscriptionInfo() {
  const { data } = api.user.getCurrent.useQuery();
  
  
  return (
    <Card sx={{ maxWidth: 345 }} variant={'outlined'}>
      {data && 
        <CardContent>
          <Grid container>
            <Grid item xs={12} >
              <Typography gutterBottom variant="h4">{data.firstName} {data.lastName} </Typography>
            </Grid>
            <Grid item xs={12} >
              <Typography variant="body1" color="text.secondary">{data.email} </Typography>
            </Grid>
            <Grid item xs={6} sx={{display: 'flex', alignItems: 'center'}}> 
              <Typography gutterBottom color="grey" variant="h6">
                Accessi 
              </Typography>
            </Grid>
            <Grid item xs={6} >
              <Typography align="right" color={data.remainingAccesses > 0 ? 'green': 'red'} gutterBottom variant="h5">{data.remainingAccesses}
              </Typography>
            </Grid>
            <Grid item xs={6}>
              <Typography color="grey" gutterBottom variant="h6">
                Abb.
              </Typography>
            </Grid>
            <Grid item xs={6}>
              <Typography align="right" gutterBottom variant="h5" >
                {data.subType === 'SHARED' ? 'Condiviso' : 'Singolo'}
              </Typography>
            </Grid>
            <Grid item xs={6} >
              <Typography gutterBottom variant="h6" color="text.secondary"  >
                Scadenza
              </Typography>
            </Grid>
            <Grid item xs={6}>
              <Typography color={DateTime.fromJSDate(data.expiresAt) < DateTime.now() ? 'red' : 'green'} align="right" variant="h5">
                {formatDate(data.expiresAt.toISOString(), DateTime.DATE_SHORT)}
              </Typography>
            </Grid>
          </Grid>
        </CardContent>
    }
    </Card> 
  );
}


function formatDate(s: string, format: Intl.DateTimeFormatOptions = DateTime.DATETIME_FULL): string {
  return DateTime.fromISO(s).setLocale('it').toLocaleString(format)
}

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
        isRefundable,
        id,
        startsAt,
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
        <ListItemIcon sx={{ fontSize: 18 }}>
          <Event />
        </ListItemIcon>
        <ListItemText
          sx={{ my: '1rem' }}
          primary={formatDate(booking.startsAt.toISOString())}
          primaryTypographyProps={{
            fontSize: 16,
            fontWeight: 'medium',
            letterSpacing: 0,
          }}
          secondary={`Effettuata ${formatDate(booking.createdAt.toISOString(), DateTime.DATETIME_MED)}`}
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
    : <Typography variant="caption" color="gray">Nessuna prenotazione</Typography>
  }
  </Box>
  );
}


function RenderSlot(props: ListChildComponentProps<string[]>) {
  
  const { index, style, data } = props;
  const slot = data[index];
  const confirm = useConfirm();
  const utils = api.useContext();
  const  [error, setError] = React.useState<string | undefined>(undefined);
  
  const { mutate } = api.bookings.create.useMutation({
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
  })
  
  const handleClick = React.useCallback(async (startsAt: string) => {
    try {
      await confirm({
        description: `Confermi la prenotazione per il giorno:
         ${DateTime.fromISO(startsAt).setLocale('it').toLocaleString(DateTime.DATETIME_FULL)}` ,
          title: 'Conferma',
          cancellationText: 'Annulla',
          confirmationText: 'Conferma',
      })
      mutate({
        startsAt: DateTime.fromISO(startsAt).toJSDate(),
      });
      
    } catch (error) {
      if (error) console.log(error);
    }
  }, [confirm, mutate]);
    
  if (!data) return <div>no data</div>
  if (!slot) return <div>no data</div>
  return (
      // eslint-disable-next-line @typescript-eslint/no-misused-promises
      <ListItemButton onClick={() => handleClick(slot)} divider key={index} style={style}>
        {error ? <Alert severity="error">{error}</Alert> : <>
        <ListItemIcon sx={{ fontSize: 18 }}>
          <Event />
        </ListItemIcon>
        <ListItemText
          sx={{ my: '1rem' }}
          primary={DateTime.fromISO(slot).setLocale('it').toLocaleString(DateTime.DATE_MED_WITH_WEEKDAY)}
          primaryTypographyProps={{
            fontSize: 16,
            fontWeight: 'medium',
            letterSpacing: 0,
          }}
          secondary={`Dalle ${DateTime.fromISO(slot).toFormat('HH:mm')}
          alle ${DateTime.fromISO(slot).plus({hours: 1}).toFormat('HH:mm')}`}
        />
        </>
    }
    </ListItemButton>
  );
}
function SlotList() {
    const { data, isLoading } = api.bookings.getAvailableSlots.useQuery();
    return (
    <Box sx={{ width: '100%', maxWidth: 360, bgcolor: 'background.paper' }}>
      {isLoading && <CircularProgress/>}
      {data &&
        <FixedSizeList
          height={350}
          width={360}
          itemSize={70}
          itemCount={data.length}
          itemData={data}
        >
          {RenderSlot}
        </FixedSizeList>
      }
    </Box>
  );
}

function Admin() {
  const [input, setInput] = React.useState<IntervalModel>({
    from: DateTime.now().startOf('week').toJSDate(),
    to: DateTime.now().endOf('week').toJSDate(),
  })
  const { data, isLoading } = api.bookings.getByInterval.useQuery({
    from: input.from,
    to: input.to 
  });
   return (
    <AdminLayout>
      {isLoading && <CircularProgress />}
      <Calendar
      localizer={luxonLocalizer(DateTime)}
      startAccessor="startsAt"
      endAccessor="endsAt"
      titleAccessor="user"
      selectable={false}
      culture="it"
      messages={{
        month: 'Mese',
        week: 'Settimana',
        today: 'Oggi',
        day: 'Giorno',
        previous: 'Prec.',
        next: 'Succ.'
      }}
      style={{ height: '100%' }}
      defaultView="week"
      views={['week']}
      
      onNavigate={(d) => setInput({
        from: DateTime.fromJSDate(d).startOf('week').toJSDate(),
        to: DateTime.fromJSDate(d).endOf('week').toJSDate(),
      })}
      events={data?.map((item) => {
        return {
          startsAt: item.startsAt,
          endsAt: DateTime.fromJSDate(item.startsAt).plus({hours: 1}).toJSDate(),
          user: `${item.User.lastName}`
        }
      })}

      min={DateTime.now().set({hour: 8, minute: 0, second: 0}).toJSDate()}
      max={DateTime.now().set({hour: 22, minute: 0, second: 0}).toJSDate()}
    />
    </AdminLayout>
  )
}