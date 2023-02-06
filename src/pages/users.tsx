import { Stack, Button, Typography, CircularProgress, Paper, styled } from "@mui/material";
import { GridColDef, DataGrid } from "@mui/x-data-grid";
import { api } from "../utils/api";
import AddIcon from "@mui/icons-material/Add"
import AdminLayout from "../components/AdminLayout";


const Item = styled(Paper)(({ theme }) => ({
  backgroundColor: theme.palette.mode === 'dark' ? '#1A2027' : '#fff',
  ...theme.typography.body2,
  padding: theme.spacing(1),
  textAlign: 'center',
  color: theme.palette.text.secondary,
}));


export default function Users() {
    return (
        <AdminLayout>
            <Stack>
                <Item>
                    <Button variant="contained" href="/new-user" endIcon={<AddIcon />}>
                        Nuovo Utente
                    </Button>
                </Item>
                <Item>
                    <UsersTable />
                </Item>
            </Stack>
        </AdminLayout>
    )
}

const columns: GridColDef[] = [
    { field: 'firstName', headerName: 'Nome', filterable: true},
    { field: 'lastName', headerName: 'Nome', filterable: true},
    { field: 'email', headerName: 'Nome'},
    { field: 'remainingAccesses', headerName: 'Accessi'},
    { field: 'expiresAt', headerName: 'Scad. abbonamento'},
    { field: 'subType', headerName: 'Abb.', }
]


function UsersTable() {
    const {data: users, isLoading} = api.user.getAll.useQuery();
    if (!users || users.length === 0) {
        return <Typography color="gray" variant="caption">Nessun utente trovato</Typography>
    }
    if (isLoading) return <CircularProgress/>
    console.log(users);
    return (
        <DataGrid
            rows={users}
            columns={columns}
            pageSize={5}
            rowsPerPageOptions={[5]}
      />
    )
}

