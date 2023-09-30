/* eslint-disable @typescript-eslint/ban-ts-comment */
import { zodResolver } from "@hookform/resolvers/zod";
import {
  Alert, Autocomplete, Backdrop, Button, Checkbox, CircularProgress, Dialog, DialogActions,
  DialogContent, DialogContentText, DialogTitle, FormControl, FormControlLabel, FormGroup,
  TextField, Typography, useTheme
} from "@mui/material";
import type { Slot, Booking, User } from "@prisma/client";
import { DateTime } from "luxon";
import React from "react";
import type { SlotInfo } from "react-big-calendar";
import { Calendar, luxonLocalizer } from "react-big-calendar";
import { useForm } from "react-hook-form";
import { api } from "../utils/api";
import type { AdminDeleteModel, IntervalModel } from "../utils/booking.schema";
import { AdminDeleteSchema } from "../utils/booking.schema";
import { formatBooking, formatDate } from "../utils/date.utils";
import WorkWeek from "./WorkWeek";

import "react-big-calendar/lib/css/react-big-calendar.css";

type UserView = Omit<User, "password" | "verificationToken" | "verificationTokenExpiresIn">;

function getTooltipInfo({ firstName, lastName, subType }: UserView, slot: Slot): string {
  if (slot.disabled) return 'Slot disabilitato';
  return `${lastName} ${firstName} - ${subType === 'SHARED' ? 'Condiviso' : 'Singolo'}`
}

function getBookingInfo(booking: FullBooking): string {
  if (booking.slot.disabled)
    return `Slot disabilitato ${formatDate(booking.createdAt)}`;
  return `Prenotazione di ${booking.user.lastName} ${booking.user.firstName} 
  (Abb. ${booking.user.subType === 'SHARED' ? 'Condiviso' : 'Singolo'}), 
  effettuata ${formatDate(booking.createdAt)})`;
}

type FullBooking = Booking & {
  user: User,
  slot: Slot,
  info: string,
  title: string
}

function getCurrentWeekParams(d: Date): IntervalModel {
  const dt = DateTime.fromJSDate(d);
  if (dt.weekday === 7) {
    return {
      from: dt.minus({ days: 6 }).startOf('day').toJSDate(),
      to: dt.endOf('day').toJSDate(),
    }
  }
  return {
    from: dt.startOf('week').toJSDate(),
    to: dt.endOf('week').toJSDate(),
  }
}

function getSlotTitle(s : Slot, firstName: string, lastName: string): string {
  if (s.disabled) {
    return "disabilitato";
  }

  return `${lastName} ${firstName.substring(0, 3)}.`
}

export function Scheduler() {
  const theme = useTheme();
  const [input, setInput] = React.useState<IntervalModel>(getCurrentWeekParams(new Date()));

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
    (event: FullBooking, start: Date, end: Date, selected: boolean) => ({
      ...(event.user.subType === 'SHARED' && {
        style: {
          backgroundColor: theme.palette.primary.main,
          borderColor: theme.palette.primary.dark,
        }
      }),
      ...(event.user.subType === 'SINGLE' && {
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
      ...((event.slot.disabled) && {
        style: {
          backgroundColor: theme.palette.grey[600],
          borderColor: theme.palette.grey[900],
        }
      })
    }),
    [theme]
  );
  const slots = React.useMemo(() => data?.map(({ startsAt, user, slot, ...rest }) => {
    return {
      startsAt,
      endsAt: DateTime.fromJSDate(startsAt).plus({ hours: 1 }).toJSDate(),
      title: getSlotTitle(slot, user.firstName, user.lastName),
      user,
      slot,
      info: getTooltipInfo(user, slot),
      ...rest,
    }
  }), [data]);

  return (
    <>
      <Backdrop
        sx={{ color: 'darkgrey', zIndex: (theme) => theme.zIndex.drawer + 1 }}
        open={isLoading}
      >
        <CircularProgress sx={{ textAlign: 'center' }} />
      </Backdrop>

      {selected && <BookingAction isOpen={showEventDialog} booking={selected} handleClose={closeEventDialog} />}
      {slotInfo && <CreateBooking isOpen={showCreateDialog} slotInfo={slotInfo} handleClose={closeCreateDialog} />}
      {/* @ts-ignore */}
      <Calendar
        localizer={luxonLocalizer(DateTime, { firstDayOfWeek: 1 })}
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
          next: 'Succ.',
          date: 'Data',
          myweek: '',
        }}
        style={{ height: '100%' }}
        onNavigate={(d) => setInput(getCurrentWeekParams(d))}
        events={slots}
        min={DateTime.now().set({ hour: 7, minute: 0, second: 0 }).toJSDate()}
        max={DateTime.now().set({ hour: 22, minute: 0, second: 0 }).toJSDate()}
        onSelectEvent={handleSelectEvent}
        onSelectSlot={handleSelectSlot}
        selectable
        selected={selected}
        step={30}
        defaultDate={new Date()}
        defaultView={'myweek'}
        views={{
          myweek: WorkWeek
        }}
      />
    </>
  )

}

interface BookingActionProps {
  booking: FullBooking | null;
  isOpen: boolean;
  handleClose: () => void;
}

function BookingAction({ booking, handleClose, isOpen }: BookingActionProps) {
  const { register, handleSubmit } = useForm<AdminDeleteModel>({
    resolver: zodResolver(AdminDeleteSchema),
    defaultValues: {
      id: booking?.id,
      startsAt: booking?.startsAt,
      isDisabled: booking?.slot.disabled,
      refundAccess: true,
      userId: booking?.userId,
      userSubType: booking?.user.subType,
    }
  });

  const [error, setError] = React.useState<string | undefined>(undefined);
  const utils = api.useContext();
  const { mutate, isLoading } = api.bookings.adminDelete.useMutation({
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
    <Dialog open={isOpen} fullWidth onClose={handleClose}>
      {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
      <form onSubmit={handleSubmit(onSubmit)}>
        <DialogTitle>{booking.slot.disabled ? 'Slot disabilitato' : 'Prenotazione'}</DialogTitle>
        <DialogContent>
          <DialogContentText>
            {booking ? getBookingInfo(booking) : ''}
          </DialogContentText>
          {
            !booking.slot.disabled &&
            <FormGroup>
              <FormControlLabel control={<Checkbox defaultChecked {...register('refundAccess')} />} label="Rimborsa accesso" />
            </FormGroup>
          }
          {error && <Alert severity="error" variant="filled" >{error}</Alert>}
        </DialogContent>
        <DialogActions>
          {isLoading && <CircularProgress />}
          <Button onClick={handleClose}>Annulla</Button>
          <Button variant="contained" color="error" type="submit">Elimina</Button>
        </DialogActions>
      </form>
    </Dialog>
  )
}
interface CreateBookingProps {
  slotInfo: SlotInfo;
  isOpen: boolean;
  handleClose: () => void;
}

interface SelectUserOptionType {
  label: string;
  id: number;
}

function CreateBooking({ slotInfo, isOpen, handleClose }: CreateBookingProps) {
  const utils = api.useContext();
  const [error, setError] = React.useState<string | undefined>(undefined);
  const [disable, setDisable] = React.useState(false);
  const [selected, setSelected] = React.useState<User | null>(null);

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
  const { data } = api.user.getAll.useQuery();

  const selectData = React.useMemo(() => data ? data.map((item, index): SelectUserOptionType => {
    return {
      label: `${item.lastName} ${item.firstName}`,
      id: index,
    }
  })
    : [], [data]);

  const onSubmit = React.useCallback((e: React.SyntheticEvent) => {
    e.preventDefault();
    setError(undefined);
    if (!selected && !disable) {
      setError('Nessun utente selezionato. Selezionare un utente o disabilitare lo slot');
      return;
    }
    if (selected && disable) {
      setError('Non è possibile disabilitare uno slot con utente selezionato');
      return;
    }
    mutate({
      from: slotInfo.start || new Date(),
      to: slotInfo.end || new Date(),
      disable,
      userId: selected?.id,
      subType: selected?.subType
    })
  }, [mutate, disable, slotInfo, selected]);

  return (
    <Dialog open={isOpen} fullWidth onClose={handleClose} sx={{ overflowY: 'visible' }}>
      {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
      <form onSubmit={onSubmit}>
        <DialogTitle>Crea prenotazione</DialogTitle>
        <DialogContent sx={{ overflowY: 'visible' }}>
          {slotInfo &&
            <DialogContentText>
              <Typography>
                {'Slot: '}
                <span style={{ fontWeight: 'bold' }}>
                  {formatBooking(slotInfo.start, slotInfo.end, DateTime.DATETIME_FULL)}
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
                  onChange={(event, value) => value ? setSelected(data.at(value.id) ?? null) : setSelected(null)}
                  options={selectData}
                  id="select-user"
                  renderInput={(params) => <TextField {...params} label="Utente" />}
                />
              </FormControl>
              : <Typography variant="caption" color="grey">Nessun utente</Typography>
            }
            <FormControl>
              <FormControlLabel control={<Checkbox onChange={(event, value) => setDisable(value)} value={disable} />} label="Disabilita gli slot selezionati" />
            </FormControl>
          </FormGroup>
          {error && <Alert severity="error" variant="filled" >{error}</Alert>}
        </DialogContent>
        <DialogActions>
          {isLoading && <CircularProgress />}
          <Button onClick={handleClose}>Annulla</Button>
          <Button variant="contained" type="submit">Conferma</Button>
        </DialogActions>
      </form>
    </Dialog >
  )
}
