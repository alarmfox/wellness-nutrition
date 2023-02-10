import type { EventType} from "@prisma/client";
import { Role } from "@prisma/client";
import AdminLayout from "../components/AdminLayout";
import type { GridColDef, GridValueFormatterParams } from '@mui/x-data-grid';
import { DataGrid } from '@mui/x-data-grid';
import { api } from "../utils/api";
import React from "react";
import { CircularProgress } from "@mui/material";

const columns: GridColDef[] = [
  { field: 'id', headerName: 'ID', width: 70, type: 'number', headerAlign: 'left', align: 'left' },
  { field: 'firstName', headerName: 'Nome', flex: 1, },
  { field: 'lastName', headerName: 'Cognome', flex: 1 },
  { field: 'occurredAt', headerName: 'Data', flex: 1 },
  { field: 'startsAt', headerName: 'Data prenotazione', flex: 1 },
  { field: 'type', headerName: 'Operazione', flex: 1, 
    valueFormatter: (params: GridValueFormatterParams<EventType>) => params.value === 'CREATED' ? 'Creata' : 'Cancellata' 
  }
];
const pageSize = 20

function EventsPage() {
  const { data, isLoading } = api.events.getLatest.useQuery();
  const rows = React.useMemo(() => data?.map(({ id, occurredAt, startsAt, type, user: { firstName, lastName }}) => {
    return {
      id,
      occurredAt,
      firstName,
      lastName,
      type,
      startsAt,
    }
  }), [data]); 
  return (
    <AdminLayout>
      <div style={{ height: 600, width: '100%' }}>
        {isLoading && <CircularProgress />}
        <DataGrid
          rows={rows || []}
          columns={columns}
          pageSize={pageSize}
          disableSelectionOnClick
        />
    </div>
    </AdminLayout>
  ) 
}

EventsPage.auth = {
  isProtected: true,
  role: [ Role.ADMIN ]
}

export default EventsPage;


