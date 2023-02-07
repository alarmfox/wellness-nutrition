import * as React from 'react';
import  AdminPage  from "./admin";
import { useSession } from 'next-auth/react';
import { Role } from '@prisma/client';
import { api } from '../utils/api';

function Home () {
  const { data: sessionData } = useSession();
   
  if (sessionData?.user.role === 'ADMIN') return <AdminPage />

  const { isLoading } = api.user.getCurrent.useQuery();
  
  if (isLoading) return <div>Loading...</div>

  return <div>user page</div>
}

Home.auth = {
  isProtected: true,
  role: Role.USER
}

export default Home;

