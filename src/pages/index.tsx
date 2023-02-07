import * as React from 'react';
import { useSession } from 'next-auth/react';
import type { Booking} from '@prisma/client';
import { Role, SubType } from '@prisma/client';
import { api } from '../utils/api';
import { Container, CssBaseline, Box, Typography, Button, Alert, Card, 
  CardContent, Grid, ListItemButton, 
  ListItemIcon, ListItemText, CircularProgress } from '@mui/material';
import { ResponsiveAppBar } from '../components/AppBar';
import { DateTime } from 'luxon';
import { useConfirm } from 'material-ui-confirm';
import type { ListChildComponentProps} from 'react-window';
import { FixedSizeList } from 'react-window';
import { Delete, Event } from '@mui/icons-material';
import AdminLayout from '../components/AdminLayout';

function Home () {
  const { data: sessionData } = useSession();
   
  if (sessionData?.user.role === 'ADMIN') return <Admin />

  const { data, isLoading } = api.bookings.getCurrent.useQuery();
  const { data: user, isLoading: userLoading } = api.user.getCurrent.useQuery();

  if (!user) return <div>Caricamento in corso...</div>

  return (
    <><ResponsiveAppBar />
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
      > {isLoading || userLoading && <CircularProgress />}
        <SubscriptionInfo />
        {data && data.length > 0 ? <Box sx={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center'
        }}>
          <Typography gutterBottom variant="h6">Prenotazioni</Typography>
          <BookingList bookings={data} />
        </Box> : <Typography color="gray"> Nessuna prenotazione </Typography>}
        <Button 
          disabled={user.remainingAccesses === 0 || DateTime.fromJSDate(user.expiresAt) < DateTime.now()} 
          component="a" href="/new" sx={{ mt: '2rem' }} variant="contained" color="primary" aria-label="nuova prenotazione">
          Nuova prenotazione
        </Button>
      </Box>
    </Container></>
  )
}

Home.auth = {
  isProtected: true,
  role: [Role.USER, Role.ADMIN]
}

export default Home;

function SubscriptionInfo() {
  const { data, isLoading } = api.user.getCurrent.useQuery();
  
  if (isLoading) return <CircularProgress />
  
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
                {data.subType === SubType.SHARED ? 'Condiviso' : 'Singolo'}
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

function RenderRow(props: ListChildComponentProps<Booking[]>) {
  
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
  
  const handleClick = React.useCallback(async ({ id, startsAt}: Booking) => {
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
function BookingList({ bookings }: { bookings: Booking[]}) {
    return (
    <Box
    sx={{ width: '100%', maxWidth: 360, bgcolor: 'background.paper' }}
    >
      <FixedSizeList
        height={350}
        width={360}
        itemSize={70}
        itemCount={bookings.length}
        itemData={bookings}
        >
          {RenderRow}
      </FixedSizeList>
    </Box>
  );
}

function Admin() {
  return (
    <AdminLayout>
     <div>hello world</div> 
    </AdminLayout>
  )
}