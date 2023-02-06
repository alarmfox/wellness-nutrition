/* eslint-disable @typescript-eslint/no-misused-promises */
import { z } from "zod";
import AdminLayout from "../components/AdminLayout";
import type { FieldValues} from 'react-hook-form';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import React, { useCallback, useState } from "react";
import { Alert, Box, Button, CircularProgress, FormControl, Grid, InputLabel, MenuItem, Select, TextField } from "@mui/material";
import { SubType } from "@prisma/client";
import { ErrorMessage } from "@hookform/error-message";
import { useRouter } from "next/router";
import type { CreateUserModel} from "../utils/user.schema";
import { CreateUserSchema } from "../utils/user.schema";
import { api } from "../utils/api";
import { DateTime } from "luxon";

export default function UserForm() {
    const { register, handleSubmit, control, formState: { errors } } = useForm<CreateUserModel>({
        defaultValues: {
          subType: SubType.SHARED,
          remainingAccesses: 10
        },
        resolver: zodResolver(CreateUserSchema),
    });
    const router = useRouter();
    const [error, setError] = useState<string | undefined> (undefined);

    const { mutate, isLoading } = api.user.create.useMutation({
        onSuccess: (() => router.push('/users')),
        onError: ((error) => {
            if(error.data?.code === 'CONFLICT') {
                setError('L\'email risulta gia resgistrata nel sistema!');
                return;
            }
            setError('Impossibile creare l\'utente. Contattare l\'amministatore');
        }),
    });

    const onSubmit = useCallback((v: FieldValues) => {
        const user = CreateUserSchema.parse(v)
        mutate(user);
    }, [mutate])

    
    return (
        <AdminLayout>
        {isLoading && <CircularProgress />}
        <form method="post" onSubmit={handleSubmit(onSubmit)}>
            <Grid container spacing={3}> 
                <Grid xs={6} item>
                    <TextField
                        margin="normal"
                        required
                        fullWidth
                        id="firstName"
                        label="Nome"
                        autoFocus
                        {...register('firstName')}
                    />
                    <ErrorMessage errors={errors} name="firstName" />
                </Grid>
                 <Grid xs={6} item>
                    <TextField
                        margin="normal"
                        required
                        fullWidth
                        id="lastName"
                        label="Cognome"
                        {...register('lastName')}
                    />
                    <ErrorMessage errors={errors} name="lastName" />
                </Grid>   
                <Grid xs={6} item>
                    <TextField
                        margin="normal"
                        required
                        fullWidth
                        id="address"
                        label="Indirizzo"
                        {...register('address')}
                    />
                    <ErrorMessage errors={errors} name="address" />
                </Grid>
                 <Grid xs={6} item>
                    <TextField
                        margin="normal"
                        fullWidth
                        id="cellphone"
                        label="Cellulare"
                        {...register('cellphone')}
                    />
                    <ErrorMessage errors={errors} name="cellphone" />
                </Grid>
                <Grid xs={12} item>
                    <TextField
                        margin="normal"
                        fullWidth
                        id="email"
                        required
                        type="email"
                        label="Indirizzo email"
                        {...register('email')}
                    />
                      <ErrorMessage errors={errors} name="email" />
                </Grid>
                <Grid xs={3} item>
                    <FormControl fullWidth>
                        <InputLabel id="subTypeL">Abbonamento</InputLabel>
                        <Controller 
                            name="subType"
                            control={control}
                            render={({field}) => 
                                <Select
                                    {...field}
                                    labelId="subTypeL"
                                    id="subTypeSelect"
                                    label="Abbonamento"
                                >
                                    <MenuItem value={SubType.SINGLE}>Singolo</MenuItem>
                                    <MenuItem value={SubType.SHARED}>Condiviso</MenuItem>
                                </Select>            
                        } 
                        />

                      <ErrorMessage errors={errors} name="email" />
                    </FormControl> 
                </Grid>
                <Grid item xs={3}>
                   <TextField
                    fullWidth
                    type="number"
                    id="remainingAccesses"
                    label="Numero accessi"
                    required
                    {...register('remainingAccesses', { valueAsNumber: true, })}
                   />

                  <ErrorMessage errors={errors} name="remainingAccesses" />
                </Grid>
                <Grid item xs={6}>
                   <TextField
                    fullWidth
                    type="date"
                    id="expiresAt"
                    defaultValue={DateTime.now().toFormat('yyyy-LL-dd')}
                    required
                    {...register('expiresAt', {valueAsDate: true} )}
                    label="Scadenza"
                    InputLabelProps={{
                        shrink: true,
                    }}
                   />
                  <ErrorMessage errors={errors} name="expiresAt" />
                </Grid>
            </Grid>
            {error && <Alert sx={{mt:'1.5rem'}} variant="filled" severity="error">{error}</Alert>}
            <Box sx={{mt: '2rem', display: 'flex', justifyContent: 'end'}}>
              <Button type="submit" variant="contained">Conferma</Button>
            </Box>
        </form>
    </AdminLayout>
    ) 
}