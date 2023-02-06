import { type NextPage } from "next";
import Head from "next/head";
import type { ListChildComponentProps } from 'react-window';
import { FixedSizeList } from 'react-window';
import { api } from "../utils/api";
import { Container, CssBaseline, Box, Typography, Button, 
  Card, CardContent, Grid, ListItemButton, ListItemIcon, 
  ListItemText, 
  CircularProgress} from "@mui/material";
import { ResponsiveAppBar } from "../components/AppBar";
import { DateTime } from 'luxon';
import type { Booking} from "@prisma/client";
import { Role } from "@prisma/client";
import { SubType } from "@prisma/client";
import EventIcon from '@mui/icons-material/Event'
import DeleteIcon from '@mui/icons-material/Delete'
import { useSession } from "next-auth/react";
import React, { useEffect } from "react";
import { useRouter } from "next/router";
import  AdminPage  from "./admin";

const Home: NextPage = () => {
  const { data: sessionData } = useSession();
  const router = useRouter();
  
 // useEffect(() => {
   // if (!sessionData) void router.replace('/signin');
  // }, [sessionData, router])
 
  // if (!sessionData) return <CircularProgress />
   const { data: bookings} = api.bookings.getCurrent.useQuery();
  if  (!bookings ) 
  return <AdminPage />
  
  return (
    <>
      <Head>
        <title>Wellness & Nutrition</title>
        <meta name="description" content="Wellness & Nutrition" />
        <link rel="icon" href="/favicon.ico" />
      </Head>
      <ResponsiveAppBar />
        <Container component="main" maxWidth="xs">
          <CssBaseline />
          <Box
            sx={{
              marginTop: 3,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              height: '95vh',
              width: '100%'
            }}
          >
            <SubscriptionInfo />
            {bookings &&  bookings.length >0 ? <Box sx={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center'
            }}>
              <Typography gutterBottom variant="h6">Prenotazioni</Typography>
              <BookingList/> 
            </Box> : <Typography color="gray"> Nessuna prenotazione </Typography>}
            <Button component="a" href="/new" sx={{mt: '2rem'}} variant="contained" color="primary" aria-label="nuova prenotazione">
                Nuova prenotazione 
            </Button>
          </Box>
        </Container>

    </>
  );
};

export default Home;


function BookingList() {
  const { data: bookings} = api.bookings.getCurrent.useQuery();
  return (
  <Box
  sx={{ width: '100%', maxWidth: 360, bgcolor: 'background.paper' }}
  >
    <FixedSizeList
      height={350}
      width={360}
      itemSize={70}
      itemCount={bookings?.length || 0}
      itemData={bookings || []}
      >
        {RenderRow}
    </FixedSizeList>
  </Box>
);
}

function SubscriptionInfo() {
  const { data: user } = api.user.getCurrent.useQuery();
  
  if (!user) {
    return <div>sus</div>
  }
  
  return (
    <Card sx={{ maxWidth: 345 }} variant={'outlined'}>
        <CardContent>
          <Grid container>
            <Grid item xs={12} >
              <Typography gutterBottom variant="h4">{user.firstName} {user.lastName}</Typography>
            </Grid>
            
            <Grid item xs={12}>
              <Typography color="gray" variant="h4">{user.email}</Typography>
            </Grid>
          
            <Grid item xs={6} sx={{display: 'flex', alignItems: 'center'}}> 
              <Typography gutterBottom color="grey" variant="h6">
                Accessi 
              </Typography>
            </Grid>
            
            <Grid item xs={6} >
              <Typography align="right" 
                          color={user.remainingAccesses> 0 ? 'green': 'red'} 
                          gutterBottom variant="h5">{user.remainingAccesses}
              </Typography>
            </Grid>
          
            <Grid item xs={6}>
              <Typography color="grey" gutterBottom variant="h6">
                Abb.
              </Typography>
            </Grid>
          
            <Grid item xs={6}>
              <Typography align="right" gutterBottom variant="h5" >
               {user.subType === SubType.SHARED ? 
                'Condiviso' : 
                'Singolo'}
              </Typography>
            </Grid>
          
            <Grid item xs={6} >
              <Typography gutterBottom variant="h6" color="text.secondary"  >
                Scadenza
              </Typography>
            </Grid>
          
            <Grid item xs={6}>
              <Typography align="right" variant="h5">
                {formatDate(user.expiresAt.toISOString(), DateTime.DATE_SHORT)}
              </Typography>
            </Grid>
          
          </Grid>
        </CardContent>
    </Card>
  );
}

function formatDate(s: string, format: Intl.DateTimeFormatOptions): string {
  return DateTime.fromISO(s).setLocale('it').toLocaleString(format)
}

function RenderRow(props: ListChildComponentProps<Booking[]>) {

  const { index, style, data } = props;
  const booking = data[index];

  if (!booking) return <div>no data</div>

  return (
      <ListItemButton divider key={index} style={style}>
        <ListItemIcon sx={{ fontSize: 18 }}>
          <EventIcon />
        </ListItemIcon>
        <ListItemText
          sx={{ my: '1rem' }}
          primary={formatDate(booking.startsAt.toISOString(), DateTime.DATETIME_FULL)}
          primaryTypographyProps={{
            fontSize: 16,
            fontWeight: 'medium',
            letterSpacing: 0,
          }}
          secondary={`Effettuata ${formatDate(booking.createdAt.toISOString(), DateTime.DATETIME_MED)}`}
        />
        <ListItemIcon sx={{fontSize: 18}}>
          <DeleteIcon />
        </ListItemIcon>
    </ListItemButton>
  )
}
