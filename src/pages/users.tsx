import * as React from 'react'
import {
  Stack, Button, Typography, CircularProgress, Paper, styled,
  Toolbar, alpha, IconButton, Tooltip, Box, Checkbox, Table, TableBody,
  TableCell, TableContainer, TablePagination, TableRow, TableHead,
  TableSortLabel, FormControl, InputAdornment, InputLabel,
  OutlinedInput, Dialog, DialogActions, DialogContent,
  DialogTitle, TextField, Grid, Alert, MenuItem, Select, FormControlLabel, FormGroup
} from "@mui/material";
import { api } from "../utils/api";
import AddIcon from "@mui/icons-material/Add"
import AdminLayout from "../components/AdminLayout";
import type { User } from "@prisma/client";
import { Role } from "@prisma/client";
import { SubType } from "@prisma/client";
import { visuallyHidden } from '@mui/utils';
import DeleteIcon from '@mui/icons-material/Delete';
import SearchIcon from '@mui/icons-material/Search';
import EditIcon from '@mui/icons-material/Edit'
import { ErrorMessage } from '@hookform/error-message';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import type { CreateUserModel, UpdateUserModel } from '../utils/user.schema';
import { CreateUserSchema, UpdateUserSchema } from '../utils/user.schema';
import { DateTime } from 'luxon';
import { useConfirm } from 'material-ui-confirm';

const Item = styled(Paper)(({ theme }) => ({
  backgroundColor: theme.palette.mode === 'dark' ? '#1A2027' : '#fff',
  ...theme.typography.body2,
  padding: theme.spacing(1),
  textAlign: 'center',
  color: theme.palette.text.secondary,
}));


export default function Users() {
  const [showForm, setShowForm] = React.useState(false);
  const handleClose = React.useCallback(() => setShowForm(false), []);
  return (
    <AdminLayout>
      <Dialog open={showForm} onClose={handleClose}>
        <DialogTitle>Nuovo utente</DialogTitle>
        {/* eslint-disable-next-line @typescript-eslint/no-non-null-assertion */}
        <CreateUser handleClose={handleClose} />
      </Dialog>

      <Stack>
        <Item>
          <UsersTable />
        </Item>
        <Item sx={{ display: 'flex', justifyContent: 'end' }}>
          <Button variant="contained" onClick={() => setShowForm(true)} endIcon={<AddIcon />}>
            Aggiungi Utente
          </Button>
        </Item>
      </Stack>
    </AdminLayout>
  )
}

Users.auth = {
  isProtected: true,
  role: [Role.ADMIN]
}

function descendingComparator<T>(a: T, b: T, orderBy: keyof T) {
  if (b[orderBy] < a[orderBy]) {
    return -1;
  }
  if (b[orderBy] > a[orderBy]) {
    return 1;
  }
  return 0;
}

type Order = 'asc' | 'desc';

interface SorteableProperties {
  lastName: string
  firstName: string
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function getComparator<Key extends keyof any>(
  order: Order,
  orderBy: Key,
): (
  a: { [key in Key]: string | number },
  b: { [key in Key]: string | number },
) => number {
  return order === 'desc'
    ? (a, b) => descendingComparator(a, b, orderBy)
    : (a, b) => -descendingComparator(a, b, orderBy);
}

// Since 2020 all major browsers ensure sort stability with Array.prototype.sort().
// stableSort() brings sort stability to non-modern browsers (notably IE11). If you
// only support modern browsers you can replace stableSort(exampleArray, exampleComparator)
// with exampleArray.slice().sort(exampleComparator)
function stableSort<T>(array: readonly T[], comparator: (a: T, b: T) => number) {
  const stabilizedThis = array.map((el, index) => [el, index] as [T, number]);
  stabilizedThis.sort((a, b) => {
    const order = comparator(a[0], b[0]);
    if (order !== 0) {
      return order;
    }
    return a[1] - b[1];
  });
  return stabilizedThis.map((el) => el[0]);
}

interface HeadCell {
  disablePadding: boolean;
  id: keyof User;
  label: string;
  numeric: boolean;
  sorteable: boolean;
}

const headCells: readonly HeadCell[] = [
  {
    id: 'lastName',
    numeric: false,
    disablePadding: true,
    label: 'Cognome',
    sorteable: true,
  },
  {
    id: 'firstName',
    numeric: false,
    disablePadding: false,
    label: 'Nome',
    sorteable: true,
  },
  {
    id: 'email',
    numeric: false,
    disablePadding: false,
    label: 'Email',
    sorteable: false,
  },
  {
    id: 'subType',
    numeric: false,
    disablePadding: false,
    label: 'Abbonamento',
    sorteable: false,
  },
  {
    id: 'expiresAt',
    numeric: false,
    disablePadding: false,
    label: 'Scadenza',
    sorteable: false,
  },
  {
    id: 'remainingAccesses',
    numeric: true,
    disablePadding: true,
    label: 'Accessi',
    sorteable: false,
  },
];

interface EnhancedTableProps {
  numSelected: number;
  onRequestSort: (event: React.MouseEvent<unknown>, property: keyof User) => void;
  onSelectAllClick: (event: React.ChangeEvent<HTMLInputElement>) => void;
  order: Order;
  orderBy: string;
  rowCount: number;
}


function EnhancedTableHead(props: EnhancedTableProps) {
  const { onSelectAllClick, order, orderBy, numSelected, rowCount, onRequestSort } =
    props;
  const createSortHandler =
    (property: keyof User) => (event: React.MouseEvent<unknown>) => {
      onRequestSort(event, property);
    };

  return (
    <TableHead>
      <TableRow>
        <TableCell padding="checkbox">
          <Checkbox
            color="primary"
            indeterminate={numSelected > 0 && numSelected < rowCount}
            checked={rowCount > 0 && numSelected === rowCount}
            onChange={onSelectAllClick}
            inputProps={{
              'aria-label': 'select all desserts',
            }}
          />
        </TableCell>
        {headCells.map((headCell) => {
          return (
            <TableCell
              key={headCell.id}
              align={headCell.numeric ? 'right' : 'left'}
              padding={headCell.disablePadding ? 'none' : 'normal'}
              sortDirection={orderBy === headCell.id ? order : false}
            >
              <TableSortLabel
                active={orderBy === headCell.id}
                direction={orderBy === headCell.id ? order : 'asc'}
                onClick={createSortHandler(headCell.id)}
              >
                {headCell.label}
                {orderBy === headCell.id ? (
                  <Box component="span" sx={visuallyHidden}>
                    {order === 'desc' ? 'sorted descending' : 'sorted ascending'}
                  </Box>
                ) : null}
              </TableSortLabel>
            </TableCell>
          );
        })}
      </TableRow>
    </TableHead>
  );
}

interface EnhancedTableToolbarProps {
  selected: string[];
  setSearch: (s: string) => void;
}

function EnhancedTableToolbar(props: EnhancedTableToolbarProps) {
  const utils = api.useContext();
  const confirm = useConfirm();
  const { selected, setSearch } = props;
  const { mutate, isLoading } = api.user.delete.useMutation({
    onSuccess: () => utils.user.getAll.invalidate()
  })

  const handleDelete = React.useCallback(async () => {
    try {
      await confirm({
        description: 'Sicuro di voler cancellare gli utenti selezionati? L\'azione non sarà reversibile.',
        title: 'Conferma',
        confirmationText: 'Conferma',
        cancellationText: 'Annulla',
      })
      mutate(selected);
    } catch (error) {
      console.log(error);
    }
  }, [selected, mutate, confirm])

  if (isLoading) return <CircularProgress />

  return (
    <Toolbar
      sx={{
        pl: { sm: 2 },
        pr: { xs: 1, sm: 1 },
        ...(selected.length > 0 && {
          bgcolor: (theme) =>
            alpha(theme.palette.primary.main, theme.palette.action.activatedOpacity),
        }),
      }}
    >
      {selected.length > 0 ? (
        <Typography
          sx={{ flex: '1 1 100%' }}
          color="inherit"
          variant="subtitle1"
          component="div"
        >
          {selected.length} utenti selezionati
        </Typography>
      ) : (
        <Typography
          sx={{ flex: '1 1 100%' }}
          variant="h6"
          id="tableTitle"
          component="div"
        >
          Utenti registrati
        </Typography>
      )}
      {selected.length > 0 ? (
        <Tooltip title="Delete">
          {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
          <IconButton onClick={handleDelete}>
            <DeleteIcon />
          </IconButton>
        </Tooltip>
      ) : (
        <Tooltip title="Ricerca per cognome">
          <FormControl sx={{ m: '.5rem' }} variant="outlined">
            <InputLabel htmlFor="search">Cerca per congnome</InputLabel>
            <OutlinedInput
              id="search"
              // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-argument
              onChange={(e: React.BaseSyntheticEvent) => setSearch(e.target.value)}
              endAdornment={
                <InputAdornment position="end">
                  <SearchIcon />
                </InputAdornment>
              }
            />
          </FormControl>
        </Tooltip>
      )}
    </Toolbar>
  );
}

function UsersTable() {
  const { data: rows, isLoading, } = api.user.getAll.useQuery();
  const [order, setOrder] = React.useState<Order>('asc');
  const [orderBy, setOrderBy] = React.useState<keyof SorteableProperties>('lastName');
  const [selected, setSelected] = React.useState<string[]>([]);
  const [page, setPage] = React.useState(0);
  const [rowsPerPage, setRowsPerPage] = React.useState(25);
  const [filteredRows, setFilteredRows] = React.useState<User[]>(rows || [])
  const [search, setSearch] = React.useState<string>('');
  const [showEdit, setShowEdit] = React.useState(false);
  const [editUser, setEditUser] = React.useState<User | undefined>(undefined);

  React.useEffect(() => {
    if (search.length < 2) {
      setFilteredRows(rows || []);
    }
    const f = rows?.filter((v: User) => {
      if (search) {
        return v.lastName.toLowerCase().includes(search.toLowerCase())
      }
      return true
    })
    setFilteredRows(f || []);
  }, [search, rows])


  const handleRequestSort = (
    event: React.MouseEvent<unknown>,
    property: keyof User,
  ) => {
    const isAsc = orderBy === property && order === 'asc';
    setOrder(isAsc ? 'desc' : 'asc');
    if (['firstName', 'lastName'].includes(property)) {
      setOrderBy(property as keyof SorteableProperties);
    }
  };

  const handleSelectAllClick = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.checked) {
      const newSelected = filteredRows.map((n) => n.id);
      setSelected(newSelected);
      return;
    }
    setSelected([]);
  };

  const handleClick = React.useCallback((event: React.MouseEvent<unknown>, id: string) => {
    const selectedIndex = selected.indexOf(id);
    let newSelected: string[] = [];

    if (selectedIndex === -1) {
      newSelected = newSelected.concat(selected, id);
    } else if (selectedIndex === 0) {
      newSelected = newSelected.concat(selected.slice(1));
    } else if (selectedIndex === selected.length - 1) {
      newSelected = newSelected.concat(selected.slice(0, -1));
    } else if (selectedIndex > 0) {
      newSelected = newSelected.concat(
        selected.slice(0, selectedIndex),
        selected.slice(selectedIndex + 1),
      );
    }

    setSelected(newSelected);
  }, [selected]);


  const handleEdit = React.useCallback((event: React.MouseEvent, user: User) => {
    event.preventDefault();
    event.stopPropagation();

    setEditUser(user);
    setShowEdit(true);
  }, []);

  const handleCloseEdit = React.useCallback(() => setShowEdit(false), []);

  const handleChangePage = (event: unknown, newPage: number) => {
    setPage(newPage);
  };

  const handleChangeRowsPerPage = (event: React.ChangeEvent<HTMLInputElement>) => {
    setRowsPerPage(parseInt(event.target.value, 10));
    setPage(0);
  };

  const isSelected = (name: string) => selected.indexOf(name) !== -1;

  if (!rows || rows.length === 0) return <Typography variant="caption" color="grey">Nessun utente trovato</Typography>
  if (isLoading) return <CircularProgress />

  // Avoid a layout jump when reaching the last page with empty rows.
  const emptyRows =
    page > 0 ? Math.max(0, (1 + page) * rowsPerPage - rows.length) : 0;

  return (
    <Box sx={{ width: '100%', mb: 2 }}>
      <EnhancedTableToolbar setSearch={setSearch} selected={selected} />
      <TableContainer>
        <Table
          sx={{ minWidth: 750 }}
          aria-labelledby="tableTitle"
          size="medium"
        >
          <EnhancedTableHead
            numSelected={selected.length}
            order={order}
            orderBy={orderBy}
            onSelectAllClick={handleSelectAllClick}
            onRequestSort={handleRequestSort}
            rowCount={rows.length}
          />
          <TableBody>
            {stableSort(filteredRows, getComparator(order, orderBy))
              .slice(page * rowsPerPage, page * rowsPerPage + rowsPerPage)
              .map((row, index) => {
                const isItemSelected = isSelected(row.id);
                const labelId = `enhanced-table-checkbox-${index}`;

                return (
                  <TableRow
                    hover
                    onClick={(event) => handleClick(event, row.id)}
                    role="checkbox"
                    aria-checked={isItemSelected}
                    tabIndex={-1}
                    key={row.id}
                    selected={isItemSelected}
                  >
                    <TableCell padding="checkbox">
                      <Checkbox
                        color="primary"
                        checked={isItemSelected}
                        inputProps={{
                          'aria-labelledby': labelId,
                        }}
                      />
                    </TableCell>
                    <TableCell
                      component="th"
                      id={labelId}
                      scope="row"
                      padding="none"
                    >
                      {row.lastName}
                    </TableCell>
                    <TableCell>{row.firstName}</TableCell>
                    <TableCell>{row.email}</TableCell>
                    <TableCell>{row.subType === SubType.SHARED ? 'Condiviso' : 'Singolo'}</TableCell>
                    <TableCell>{row.expiresAt.toISOString().split('T')[0]}</TableCell>
                    <TableCell align="right">{row.remainingAccesses}</TableCell>
                    <TableCell>
                      <Button onClick={(e) => handleEdit(e, row)}>
                        <EditIcon />
                      </Button>
                    </TableCell>
                  </TableRow>
                );
              })}
            {emptyRows > 0 && (
              <TableRow
                style={{
                  height: 53 * emptyRows,
                }}
              >
                <TableCell colSpan={6} />
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>
      <TablePagination
        rowsPerPageOptions={[5, 10, 25]}
        component="div"
        count={rows.length}
        rowsPerPage={rowsPerPage}
        page={page}
        onPageChange={handleChangePage}
        onRowsPerPageChange={handleChangeRowsPerPage}
      />
      <Dialog open={showEdit} onClose={handleCloseEdit}>
        <DialogTitle>Modifica utente</DialogTitle>
        {/* eslint-disable-next-line @typescript-eslint/no-non-null-assertion */}
        <EditUser handleClose={handleCloseEdit} user={editUser!} />

      </Dialog>
    </Box>
  );
}
interface CreateUserProps {
  handleClose: () => void
}

function CreateUser({ handleClose }: CreateUserProps) {
  const { register, handleSubmit, control, formState: { errors } } = useForm<CreateUserModel>({
    defaultValues: {
      subType: SubType.SHARED,
      remainingAccesses: 10
    },
    resolver: zodResolver(CreateUserSchema),
  });
  const [error, setError] = React.useState<string | undefined>(undefined);
  const utils = api.useContext();
  const { mutate, isLoading } = api.user.create.useMutation({
    onSuccess: (async () => {
      try {
        await utils.user.getAll.invalidate();
        handleClose();
      } catch (error) {
        setError('Error sconosciuto')
      }
    }),
    onError: ((error) => {
      if (error.data?.code === 'CONFLICT') {
        setError('L\'email risulta gia resgistrata nel sistema!');
        return;
      }
      setError('Impossibile creare l\'utente. Contattare l\'amministatore');
    }),
  });

  const onSubmit = React.useCallback((v: CreateUserModel) => mutate(v), [mutate])

  return (
    <DialogContent>
      {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
      <form onSubmit={handleSubmit(onSubmit)}>
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
          <Grid xs={4} item>
            <FormControl fullWidth>
              <InputLabel id="sub-lable-id">Abbonamento</InputLabel>
              <Controller
                name="subType"
                control={control}
                render={({ field }) =>
                  <Select
                    {...field}
                    labelId="sub-label-id"
                    id="subtype"
                    label="Abbonamento"
                  >
                    <MenuItem value={SubType.SINGLE}>Singolo</MenuItem>
                    <MenuItem value={SubType.SHARED}>Condiviso</MenuItem>
                  </Select>
                }
              />
            </FormControl>
            <ErrorMessage errors={errors} name="subType" />
          </Grid>
          <Grid item xs={2}>
            <TextField
              fullWidth
              type="number"
              id="remainingAccesses"
              label="Accessi"
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
              defaultValue={DateTime.now().plus({ months: 1 }).toFormat('yyyy-LL-dd')}
              required
              {...register('expiresAt', { valueAsDate: true })}
              label="Scadenza"
              InputLabelProps={{
                shrink: true,
              }}
            />
            <ErrorMessage errors={errors} name="expiresAt" />
          </Grid>
          <Grid item xs={6}>
            <FormControl fullWidth>
              <InputLabel id="goals-label-id">Obiettivi</InputLabel>
              <Controller
                name="goals"
                control={control}
                render={({ field }) =>
                  <Select
                    {...field}
                    labelId="goals-label-id"
                    id="goals-select"
                    multiple
                    defaultValue={[]}
                    label="Obiettivi"
                  >
                    <MenuItem value="Dimagrimento">Dimagrimento</MenuItem>
                    <MenuItem value="Posturale">Posturale</MenuItem>
                    <MenuItem value="Tonificazione">Tonificazione</MenuItem>
                    <MenuItem value="Potenziamento">Potenziamento</MenuItem>
                    <MenuItem value="Benessere">Benessere</MenuItem>
                  </Select>
                }
              />
            </FormControl>
            <ErrorMessage errors={errors} name="subType" />
          </Grid>
          <Grid item xs={6}>
            <FormGroup>
              <FormControlLabel {...register('medOk')} control={<Checkbox />} label="Certificato medico" />
            </FormGroup>
          </Grid>
        </Grid>
        {error && <Alert sx={{ mt: '1.5rem' }} variant="filled" severity="error">{error}</Alert>}
        <DialogActions>
          {isLoading && <CircularProgress />}
          <Button onClick={handleClose}>Annulla</Button>
          <Button variant="contained" type="submit">Conferma</Button>
        </DialogActions>
      </form>
    </DialogContent>
  )
}
interface EditUserProps {
  user: User;
  handleClose: () => void
}

function toUserModel({ goals, ...rest }: User): UpdateUserModel {
  return {
    ...rest,
    goals: goals?.split('-')
  }
}
function EditUser({ user, handleClose }: EditUserProps) {
  const { register, handleSubmit, control, formState: { errors, isDirty }, setValue } = useForm<UpdateUserModel>({
    defaultValues: toUserModel(user),
    resolver: zodResolver(UpdateUserSchema)
  });
  const [error, setError] = React.useState<string | undefined>(undefined);
  const utils = api.useContext();
  const { mutate: updateUser, isLoading, } = api.user.update.useMutation({
    onSuccess: async () => {
      try {
        await utils.user.getAll.invalidate();
        handleClose();
      } catch (error) {
        setError('Error sconosciuto');
      }

    },
    onError: () => setError('Impossibile modificare l\'utente. Contattare l\'amministratore.')
  })

  React.useEffect(() => {
    setTimeout(() => {
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-ignore
      setValue('expiresAt', user.expiresAt.toISOString().split('T')[0]);
    }, 100);
  }, [user, setValue]);

  const onSubmit = React.useCallback((user: UpdateUserModel) => updateUser(user), [updateUser])

  return (
    <DialogContent>
      {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
      <form onSubmit={handleSubmit(onSubmit)}>
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
              disabled
              type="email"
              label="Indirizzo email"
              value={user.email}
            />
          </Grid>
          <Grid xs={4} item>
            <FormControl fullWidth>
              <InputLabel id="subTypeL">Abbonamento</InputLabel>
              <Controller
                name="subType"
                control={control}
                render={({ field }) =>
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
              <ErrorMessage errors={errors} name="subType" />
            </FormControl>
          </Grid>
          <Grid item xs={2}>
            <TextField
              fullWidth
              type="number"
              id="remainingAccesses"
              label="Accessi"
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
              required
              {...register('expiresAt', { valueAsDate: true })}
              label="Scadenza"
              InputLabelProps={{
                shrink: true,
              }}
            />
            <ErrorMessage errors={errors} name="expiresAt" />
          </Grid>
          <Grid item xs={6}>
            <FormControl fullWidth>
              <InputLabel id="goals-label-id">Obiettivi</InputLabel>
              <Controller
                name="goals"
                control={control}
                render={({ field }) =>
                  <Select
                    {...field}
                    labelId="goals-label-id"
                    id="goals-select"
                    multiple
                    defaultValue={[]}
                    label="Obiettivi"
                  >
                    <MenuItem value="Dimagrimento">Dimagrimento</MenuItem>
                    <MenuItem value="Posturale">Posturale</MenuItem>
                    <MenuItem value="Tonificazione">Tonificazione</MenuItem>
                    <MenuItem value="Potenziamento">Potenziamento</MenuItem>
                    <MenuItem value="Benessere">Benessere</MenuItem>
                  </Select>
                }
              />
            </FormControl>
            <ErrorMessage errors={errors} name="goals" />
          </Grid>
          <Grid item>
            <FormGroup>
              <FormControlLabel {...register('medOk')} control={<Checkbox defaultChecked={user.medOk} />} label="Certificato medico" />
            </FormGroup>
          </Grid>
        </Grid>
        {error && <Alert sx={{ mt: '1.5rem' }} variant="filled" severity="error">{error}</Alert>}
        <DialogActions>
          {isLoading && <CircularProgress />}
          <Button onClick={handleClose}>Annulla</Button>
          <Button disabled={!isDirty} variant="contained" type="submit">Conferma</Button>
        </DialogActions>
      </form>

    </DialogContent>
  )
}
