import { createFileRoute, redirect } from '@tanstack/react-router';
import { AdminPanel } from '../components/AdminPanel';
import { getIdTokenClaims } from '../services/auth.service';

export const Route = createFileRoute('/admin')({
   beforeLoad: () => {
      const claims = getIdTokenClaims();
      const groups = claims?.['cognito:groups'];
      const isAdmin = Array.isArray(groups)
         ? groups.includes('admin')
         : typeof groups === 'string' && groups.includes('admin');
      if (!isAdmin) {
         throw redirect({ to: '/' });
      }
   },
   component: AdminPanel,
});
