import { Card, CardContent, CardMedia, Divider, Grid, Typography } from "@mui/material";
import { DateTime } from "luxon";
import { api } from "../utils/api";
import { formatDate } from "../utils/format.utils";

export function Subscription() {
  const { data } = api.user.getCurrent.useQuery();
  
  return (
    <Card sx={{ maxWidth: 345 }} variant="outlined">
      {data && 
        <CardContent>
          <CardMedia
            sx={{ height: 80, display: 'flex', justifyContent: 'center', mb: '.5rem' }}
            image="/logo_big.png"
            title="logo"
          />
          <Divider />
          <Grid container>
            <Grid item xs={12} >
              <Typography gutterBottom variant="h6">{data.firstName} {data.lastName}</Typography>
            </Grid>
            <Grid item xs={12} >
              <Typography variant="body1" color="text.secondary">{data.email} </Typography>
            </Grid>
            <Grid item xs={6} sx={{ display: 'flex', alignItems: 'center' }}> 
              <Typography color="grey" variant="body1">
                Accessi 
              </Typography>
            </Grid>
            <Grid item xs={6} >
              <Typography align="right" color={data.remainingAccesses > 0 ? 'green': 'red'} variant="h6">
                {data.remainingAccesses}
              </Typography>
            </Grid>
            <Grid item xs={6}>
              <Typography color="grey" variant="body1">
                Abb.
              </Typography>
            </Grid>
            <Grid item xs={6}>
              <Typography align="right" variant="h6" >
                {data.subType === 'SHARED' ? 'Condiviso' : 'Singolo'}
              </Typography>
            </Grid>
             <Grid item xs={6} >
              <Typography variant="body1" color="text.secondary"  >
                Obiettivi
              </Typography>
            </Grid>
            <Grid item xs={6}>
              <Typography align="right" variant="body1">
                {data.goals?.replaceAll('-', ' ')}
              </Typography>
            </Grid>
            <Grid item xs={6} >
              <Typography variant="body1" color="text.secondary"  >
                Scadenza
              </Typography>
            </Grid>
            <Grid item xs={6}>
              <Typography color={DateTime.fromJSDate(data.expiresAt) < DateTime.now() ? 'red' : 'green'} align="right" variant="h6">
                {formatDate(data.expiresAt, DateTime.DATE_SHORT)}
              </Typography>
            </Grid>
          </Grid>
        </CardContent>
    }
    </Card> 
  );
}