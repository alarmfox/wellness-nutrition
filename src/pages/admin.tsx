import { Stack, Button, Typography, CircularProgress, Paper, styled } from "@mui/material";
import type { GridColDef} from "@mui/x-data-grid";
import { DataGrid } from "@mui/x-data-grid";
import { api } from "../utils/api";

import AdminLayout from "../components/AdminLayout";
import Users from "./users";
import { useRouter } from "next/router";

export default function AdminPage() {
    return (
        <AdminLayout>
            <div>calendar</div>
        </AdminLayout>
    )
}
