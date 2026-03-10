// api.users.ts — auth operations now go through Cognito SDK directly.
// This file only contains the profile update call which hits the backend.
import { PUT } from './api.service';

export async function UpdateUser(body: {
   email?: string;
}): Promise<{ success: boolean }> {
   try {
      await PUT('api/users', body);
      return { success: true };
   } catch {
      return { success: false };
   }
}
