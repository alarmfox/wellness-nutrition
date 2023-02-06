import * as React from 'react';
import  AdminPage  from "./admin";
import { useSession } from 'next-auth/react';
import { Role } from '@prisma/client';

function Home () {
  const { data: sessionData } = useSession();
   
  if (sessionData?.user.role === 'ADMIN') return <AdminPage />


  return <div>users page</div>
}

Home.auth = {
  isProtected: true,
  role: Role.USER
}

export default Home;