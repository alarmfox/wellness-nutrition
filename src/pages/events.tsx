import type { EventType } from "@prisma/client";
import { Role } from "@prisma/client";
import AdminLayout from "../components/AdminLayout";
import type { GridColDef, GridValueFormatterParams } from '@mui/x-data-grid';
import { DataGrid } from '@mui/x-data-grid';
import { api } from "../utils/api";
import React from "react";
import { CircularProgress } from "@mui/material";
import { formatDate } from "../utils/date.utils";
import { DateTime } from "luxon";

const columns: GridColDef[] = [
  { field: 'id', headerName: 'ID', width: 70, type: 'number', headerAlign: 'left', align: 'left' },
  { field: 'firstName', headerName: 'Nome', flex: 1, },
  { field: 'lastName', headerName: 'Cognome', flex: 1 },
  {
    field: 'occurredAt', headerName: 'Data', flex: 1,
    valueFormatter: (params: GridValueFormatterParams<Date>) => formatDate(params.value, DateTime.DATETIME_FULL)
  },
  {
    field: 'startsAt', headerName: 'Data prenotazione', flex: 1,
    valueFormatter: (params: GridValueFormatterParams<Date>) => formatDate(params.value, DateTime.DATETIME_FULL)
  },
  {
    field: 'type', headerName: 'Operazione', flex: 1,
    valueFormatter: (params: GridValueFormatterParams<EventType>) => params.value === 'CREATED' ? 'Creata' : 'Cancellata'
  }
];

function EventsPage() {
  const { data, isLoading } = api.events.getLatest.useQuery();
  const rows = React.useMemo(() => data?.map(({ id, occurredAt, startsAt, type, user: { firstName, lastName } }) => {
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
          pageSize={20}
          disableSelectionOnClick
        />
      </div>
    </AdminLayout>
  )
}

EventsPage.auth = {
  isProtected: true,
  role: [Role.ADMIN]
}

export default EventsPage;


