import * as React from 'react';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import { Container, CssBaseline, Box, Avatar, Typography, TextField, Button, Alert, CircularProgress } from '@mui/material';
import { useRouter } from 'next/router';
import type { VerifyAccountModel} from '../utils/verifiy.schema';
import { VerifyAccountSchema } from '../utils/verifiy.schema';
import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { ErrorMessage } from '@hookform/error-message';
import { api } from '../utils/api';
import type { NextPage, NextPageContext } from 'next';

interface Props {
  token: string;
}

const Verify: NextPage<Props> = ({ token }) => {
  console.log(token);
  const router = useRouter();
  const { register,  handleSubmit, watch, formState: { errors, isLoading }, getValues} = useForm<VerifyAccountModel>({
    resolver: zodResolver(VerifyAccountSchema),
    defaultValues: {
      token: router.query.token as string
    }
  });
  const [error, setError] = React.useState<string | undefined>(undefined); 
  const { mutate, isLoading: requestLoading } = api.user.changePassword.useMutation({
    onSuccess: () => router.replace('/signin'),
    onError: (err) => {
      if(err.data?.code === 'NOT_FOUND') {
        setError('Link scaduto. Ripetere la procedura')
        return;
      }
      setError('Errore sconosciuto');
    }
  })
  const onSubmit = React.useCallback((v: VerifyAccountModel) => mutate(v), [mutate])
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
          {requestLoading && <CircularProgress />}
          <Typography component="h1" variant="h5">
           Ripristino password 
          </Typography>
            {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
          <Box component="form" onSubmit={handleSubmit(onSubmit)} sx={{ mt: 1 }}>
            <input type="hidden" {...register('token')} />
            <TextField
              margin="normal"
              required
              fullWidth
              label="Nuova password"
              type="password"
              id="password"
              {...register('newPassword')}
              disabled={!isLoading}
            />
            <ErrorMessage errors={errors} name="newPassword" />
            <TextField
              margin="normal"
              required
              fullWidth
              label="Conferma password"
              type="password"
              id="confirmPassword"
              error={watch("newPassword") !== watch("confirmPassword") && !!getValues("confirmPassword")}
              disabled={!isLoading}
              {...register('confirmPassword')}
            />
            <ErrorMessage errors={errors} name="confirmPassword" />
            {error && <Alert variant='filled' severity="error">{error}</Alert> }
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

Verify.getInitialProps = (ctx: NextPageContext) => {
  const { query } = ctx;
  return {
    token: query.token as string
  }
}

export default Verify;