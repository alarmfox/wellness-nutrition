import { Container, Link, CssBaseline, Box, Avatar, Typography, TextField, Button, Grid, Alert } from "@mui/material";
import { getCsrfToken } from "next-auth/react";
import LockOutlinedIcon from '@mui/icons-material/LockOutlined'
import type { GetServerSidePropsContext, InferGetServerSidePropsType } from "next";
import { useRouter } from "next/router";

export default function SignIn({ csrfToken }: InferGetServerSidePropsType<typeof getServerSideProps>) {
  const { error } = useRouter().query;
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
        Accedi
      </Typography>
      <Box component="form" method="post" action="/api/auth/callback/credentials" sx={{ mt: 1 }}>
        <input name="csrfToken" type="hidden" defaultValue={csrfToken} />
        <TextField
          margin="normal"
          required
          fullWidth
          id="email"
          name="email"
          label="Indirizzo email"
          autoComplete="email"
          autoFocus
        />
        <TextField
          margin="normal"
          required
          fullWidth
          label="Password"
          type="password"
          name="password"
          id="password"
          autoComplete="current-password"
        />
        {error && error !== "SessionRequired" && <Alert variant="filled" severity="error">Login fallito. Email e/o password errati</Alert>}
        <Button
          type="submit"
          fullWidth
          variant="contained"
          sx={{ mt: 3, mb: 3 }}
        >
          Conferma
        </Button>
        <Grid container>
          <Grid item xs>
            <Link href="/reset" variant="body2">
              Password dimenticata
              ?
            </Link>
          </Grid>
        </Grid>
      </Box>
      </Box>
  </Container>
  )
}

export async function getServerSideProps(context: GetServerSidePropsContext) {
  return {
    props: {
      csrfToken: await getCsrfToken(context),
    },
  }
}