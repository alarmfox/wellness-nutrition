import * as React from 'react';
import { useSession } from 'next-auth/react';
import type { Booking, Slot, User } from '@prisma/client';
import { api } from '../utils/api';
import { Container, CssBaseline, Box, Typography, Button, Alert, Card, 
  CardContent, Grid, ListItemButton, 
  ListItemIcon, ListItemText, CircularProgress, useTheme, Dialog, DialogActions, DialogContent, DialogContentText, DialogTitle, 
  Checkbox, FormControlLabel, FormGroup, FormControl, Autocomplete, TextField, 
  } from '@mui/material';
import { ResponsiveAppBar } from '../components/AppBar';
import { DateTime, Interval } from 'luxon';
import { useConfirm } from 'material-ui-confirm';
import type { ListChildComponentProps} from 'react-window';
import { FixedSizeList } from 'react-window';
import { Delete, Event } from '@mui/icons-material';
import AdminLayout from '../components/AdminLayout';
import type { SlotInfo } from 'react-big-calendar';
import { Calendar, luxonLocalizer } from 'react-big-calendar'
import "react-big-calendar/lib/css/react-big-calendar.css";
import type { AdminDeleteModel, IntervalModel } from '../utils/booking.schema';
import { AdminDeleteSchema } from '../utils/booking.schema';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';

function Home () {
  const { data: sessionData } = useSession();

  const { data: user, isLoading: getCurrentLoading } = api.user.getCurrent.useQuery();
  const { isLoading: creatingLoading } = api.bookings.create.useMutation();
  const [ creationMode, setCreationMode ] = React.useState(false);
  const isLoading = React.useMemo(() => creatingLoading || getCurrentLoading, [getCurrentLoading, creatingLoading])
  
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
        > 
          <SubscriptionInfo />
          {!creationMode ? <BookingList />: <SlotList/>}
          {isLoading && <CircularProgress />}
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

function getTooltipInfo({firstName, lastName, subType }: User, slot: Slot): string {
  if (slot.disabled) return 'Slot disabilitato';
  return `${lastName} ${firstName} - ${subType === 'SHARED' ? 'Condiviso' : 'Singolo'}`
}

function getBookingInfo(booking: FullBooking): string {
  if(booking.slot.disabled) 
    return `Slot disabilitato ${DateTime.fromJSDate(booking.createdAt).setLocale('it').toLocaleString(DateTime.DATETIME_FULL)}`;
  return `Prenotazione di ${booking.user.lastName} ${booking.user.firstName} 
  (Abb. ${booking.user.subType === 'SHARED' ? 'Condiviso' : 'Singolo'}), 
  effettuata ${DateTime.fromJSDate(booking.createdAt).setLocale('it').toLocaleString(DateTime.DATETIME_FULL)})`;
}

type FullBooking = Booking &  {
  user: User, 
  slot: Slot, 
  info: string, 
  title: string
}

function Admin() {
  const theme = useTheme();
  const [input, setInput] = React.useState<IntervalModel>({
    from: DateTime.now().startOf('week').toJSDate(),
    to: DateTime.now().endOf('week').toJSDate(),
  })
  
  const { data, isLoading } = api.bookings.getByInterval.useQuery(input);

  const [selected, setSelected] = React.useState<FullBooking | null>(null);
  const [slotInfo, setSlotInfo] = React.useState<SlotInfo | null>(null);

  const showEventDialog = React.useMemo(() => !(selected === null), [selected]);
  const showCreateDialog = React.useMemo(() => !(slotInfo === null), [slotInfo]);
  
  const closeEventDialog = React.useCallback(() => {
    setSelected(null);
  }, []);

  const closeCreateDialog = React.useCallback(() => {
    setSlotInfo(null);
  }, [])
  
  const handleSelectEvent = React.useCallback((b: FullBooking) => {
    if (DateTime.fromJSDate(b.startsAt).startOf('day') < DateTime.now().startOf('day')) return;
    setSelected(b);
  }, []);

  const handleSelectSlot = React.useCallback((s: SlotInfo) => {
    if (DateTime.fromJSDate(s.start).startOf('day') < DateTime.now().startOf('day')) return;
    setSlotInfo(s);
  }, []);

  const eventPropGetter = React.useCallback(
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    (event: FullBooking, start: Date, end: Date,  selected: boolean) => ({ 
      ...(event.user.subType === 'SHARED' && {
        style: {
          backgroundColor: theme.palette.info.main
        }
      }),
      ...(event.user.subType === 'SINGLE' &&  {
        style: {
          backgroundColor: theme.palette.secondary.main,
        }
      }),
      ...(selected && {
        style: {
          backgroundColor: theme.palette.warning.main,
          borderColor: theme.palette.warning.dark,
        }
      }),
      ...(event.slot.disabled && {
        style: {
          backgroundColor: theme.palette.info.main,
        }
      }),
      ...((DateTime.fromJSDate(event.startsAt) < DateTime.now() || event.slot.disabled) && {
        style: {
          backgroundColor: theme.palette.grey[600],
          borderColor: theme.palette.grey[900],
        }
      })
    }),
    [theme]
  )
  return (
    <AdminLayout>
      {isLoading  && <CircularProgress />}
      {selected && <BookingAction isOpen={showEventDialog} booking={selected} handleClose={closeEventDialog}/>}
      {slotInfo && <CreateBooking isOpen={showCreateDialog} slots={slotInfo.slots} handleClose={closeCreateDialog}/> }
      <Calendar
        localizer={luxonLocalizer(DateTime)}
        eventPropGetter={eventPropGetter}
        startAccessor="startsAt"
        endAccessor="endsAt"
        titleAccessor="title"
        culture="it"
        tooltipAccessor="info"
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
        events={data?.map(({ startsAt, user, slot, ...rest}) => {
          return {
            startsAt,
            endsAt: DateTime.fromJSDate(startsAt).plus({hours: 1}).toJSDate(),
            title: `${slot.disabled ? 'disabilitato' : user.lastName}`,
            user,
            slot,
            ...rest,
            info: getTooltipInfo(user, slot)
          }
        })}
        min={DateTime.now().set({hour: 8, minute: 0, second: 0}).toJSDate()}
        max={DateTime.now().set({hour: 22, minute: 0, second: 0}).toJSDate()}
        onSelectEvent={handleSelectEvent}
        onSelectSlot={handleSelectSlot}
        selectable
        selected={selected}
        step={60}
    />
    </AdminLayout>
  )
}

interface BookingActionProps {
  booking: FullBooking | null;
  isOpen: boolean;
  handleClose: () => void;
}

function BookingAction({ booking, handleClose, isOpen }: BookingActionProps) {
  const { register, handleSubmit , setValue } = useForm<AdminDeleteModel>({
    resolver: zodResolver(AdminDeleteSchema),
  });

  React.useEffect(() => {
    if (!booking) return;

    setValue('id', booking.id);
    setValue('startsAt', booking.startsAt);

  }, [booking, setValue]);
  const [error, setError] = React.useState<string | undefined>(undefined);
  const utils = api.useContext();
  const {mutate, isLoading } = api.bookings.adminDelete.useMutation({
    onSuccess: async () => {
      try {
        await utils.bookings.getByInterval.invalidate();
        handleClose();
      } catch (error) {
        console.log(error);
        setError('Impossibile cancellare la prenotazione');
      }
    },
    onError: (error) => {
      console.log(error);
      setError('Impossibile cancellare la prenotazione');
    }
  })
  const onSubmit = React.useCallback((v: AdminDeleteModel) => mutate(v), [mutate]);
  if (!booking) return <div></div>
  return (
    <Dialog open={isOpen} onClose={handleClose}>
      {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
      <form onSubmit={handleSubmit(onSubmit)}>
        <DialogTitle>{booking.slot.disabled ? 'Slot disabilitato' : 'Prenotazione'}</DialogTitle>
        <DialogContent>
          <DialogContentText>
            {booking ? getBookingInfo(booking):''}
          </DialogContentText>
          {
            !booking.slot.disabled && 
          <FormGroup>
            <FormControlLabel control={<Checkbox defaultChecked {...register('refundAccess')}  />} label="Rimborsa accesso" />
          </FormGroup>
          }
        {error && <Alert severity="error" variant="filled" >{error}</Alert>}
        </DialogContent>
        <DialogActions>
          {isLoading && <CircularProgress />}
          <Button  onClick={handleClose}>Annulla</Button>
          <Button variant="contained" color="error" type="submit">Elimina</Button>
        </DialogActions>
      </form>
    </Dialog>
  ) 
}
interface CreateBookingProps {
  slots: Date [];
  isOpen: boolean;
  handleClose: () => void;
}


interface SelectUserOptionType {
  label: string;
  id: number;
}

function CreateBooking({ slots, isOpen, handleClose }: CreateBookingProps) {
  const utils = api.useContext();
  const [error, setError] = React.useState<string | undefined>(undefined);
  const [disable, setDisable] = React.useState(false);
  const [selected, setSelected] = React.useState<User | null> (null);
 
  const { mutate, isLoading } = api.bookings.adminCreate.useMutation({
    onSuccess: async () => {
      try {
        await utils.bookings.getByInterval.invalidate();
        handleClose(); 
      } catch (error) {
        console.log(error);
        setError("Impossibile creare la prenotazione");
      }
    },
    onError: (error) => {
      console.log(error);
      setError("Impossibile creare la prenotazione");
    }
  })
  const { data, } = api.user.getActive.useQuery();

  const selectData = React.useMemo(() => data ? data.map((item, index): SelectUserOptionType => {
    return {
      label: `${item.lastName} ${item.firstName}`,
      id: index,
    }
  }
  ) 
  : [], [data]);

  const onSubmit = React.useCallback((e: React.SyntheticEvent) => {
    e.preventDefault();
    setError(undefined);
    if(!selected && !disable) {
      setError('Nessun utente selezionato. Selezionare un utente o disabilitare lo slot');
      return;
    }
    if (selected && disable) {
      setError('Non è possibile disabilitare uno slot con utente selezionato');
      return;
    }
    mutate({
      from: slots.at(0) || new Date(),
      to: slots.at(-1) || new Date(),
      disable,
      userId: selected?.id,
      subType: selected?.subType
    })
  }, [mutate, disable, slots, selected]);

  return (
    <Dialog open={isOpen} onClose={handleClose}>
      {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
      <form onSubmit={onSubmit}>
        <DialogTitle>Crea prenotazione</DialogTitle>
        <DialogContent>
          {slots && 
            <DialogContentText>
                <Typography>
                  {'Slot: '}
                  <span style={{fontWeight: 'bold'}}>
                    {Interval.fromDateTimes(
                      DateTime.fromJSDate(slots[0] || new Date()),  
                      DateTime.fromJSDate(slots.at(-1) || new Date())
                      ).toLocaleString(DateTime.DATETIME_HUGE)}
                  </span>
                </Typography>
            </DialogContentText>
          }
        <FormGroup>
          {data ? 
            <FormControl sx={{ m: 1 }} fullWidth>
              <Autocomplete
                noOptionsText="Nessun utente trovato"
                getOptionLabel={(option: SelectUserOptionType) => option.label}
                disablePortal
                onChange={(event, value) => value ? setSelected(data.at(value.id) ?? null) : setSelected(null)}
                options={selectData}
                id="select-user"
                renderInput={(params) => <TextField {...params} label="Utente" />}
                />
            </FormControl>
            : <Typography variant="caption" color="grey">Nessun utente</Typography>
          }
          <FormControl>
              <FormControlLabel control={<Checkbox onChange={(event, value) => setDisable(value)} value={disable} /> } label="Disabilita gli slot selezionati"/>
            </FormControl>
        </FormGroup>
        {error && <Alert severity="error" variant="filled" >{error}</Alert>}
        </DialogContent>
        <DialogActions>
          {isLoading && <CircularProgress />}
          <Button  onClick={handleClose}>Annulla</Button>
          <Button variant="contained" type="submit">Conferma</Button>
        </DialogActions>
      </form>
    </Dialog>
  )
}