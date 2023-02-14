import * as React from 'react';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import { Container, CssBaseline, Box, Avatar, Typography, TextField, Button, Alert, CircularProgress } from '@mui/material';
import { useForm } from 'react-hook-form';
import { api } from '../utils/api';

type Reset = {
    email: string;
}

export default function Reset() {
  const { register,  handleSubmit, formState: { isLoading }, getValues } = useForm<Reset>();
  const [message, setMessage] = React.useState<string | undefined>(undefined); 
  const { mutate, isLoading: requestLoading, isError } = api.user.resetPassword.useMutation({
    onSuccess: () => setMessage(`Un\'email è stata inviata all\'indirizzo ${getValues('email')}. Puoi chiudere questa pagina`),
    onError: (err) => {
      if(err.data?.code === 'NOT_FOUND') {
        setMessage('Email non trovata nel sistema')
        return;
      }
      setMessage('Errore sconosciuto');
    }
  })
  const onSubmit = React.useCallback((v: Reset) => mutate(v.email), [mutate])

  return (
      <Container component="main" maxWidth="xs">
        <CssBaseline />
        <Box
          sx={{
            marginTop: 8,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
          }}
        >
          <Avatar sx={{ m: 1, bgcolor: 'secondary.main' }}>
            <LockOutlinedIcon />
          </Avatar>
          <Typography component="h1" variant="h5">
           Password dimenticata 
          </Typography>
            {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
          <Box component="form" onSubmit={handleSubmit(onSubmit)} sx={{ mt: 1 }}>
            <TextField
              margin="normal"
              required
              fullWidth
              label="Indirizzo email"
              type="email"
              id="email"
              disabled={!isLoading}
              {...register('email')}
            />
            {requestLoading && <CircularProgress />}
            {message && <Alert variant='filled' severity={isError ? 'error' : 'info'}>{message}</Alert> } 
            <Button
              type="submit"
              fullWidth
              variant="contained"
              sx={{ mt: 3, mb: 3 }}
            >
              Conferma
            </Button>
         </Box>
        </Box>
      </Container>
  );
}
