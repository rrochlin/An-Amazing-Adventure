/**
 * auth.service.ts
 * All authentication is handled via Amazon Cognito User Pools.
 * Tokens are stored in localStorage under "AAA_TOKENS".
 */
import {
   AuthenticationDetails,
   CognitoUser,
   CognitoUserAttribute,
   CognitoUserPool,
   type CognitoUserSession,
} from 'amazon-cognito-identity-js';
import { redirect } from '@tanstack/react-router';

const STORAGE_KEY = 'AAA_TOKENS';

const USER_POOL_ID = import.meta.env.VITE_COGNITO_USER_POOL_ID as string;
const CLIENT_ID = import.meta.env.VITE_COGNITO_CLIENT_ID as string;

function getUserPool(): CognitoUserPool {
   return new CognitoUserPool({
      UserPoolId: USER_POOL_ID,
      ClientId: CLIENT_ID,
   });
}

export interface StoredTokens {
   accessToken: string;
   idToken: string;
   refreshToken: string;
   expiresAt: number; // ms epoch
   userSub: string;
}

export function getStoredTokens(): StoredTokens | null {
   const raw = localStorage.getItem(STORAGE_KEY);
   if (!raw) return null;
   try {
      return JSON.parse(raw) as StoredTokens;
   } catch {
      return null;
   }
}

function storeSession(session: CognitoUserSession, userSub: string) {
   const tokens: StoredTokens = {
      accessToken: session.getAccessToken().getJwtToken(),
      idToken: session.getIdToken().getJwtToken(),
      refreshToken: session.getRefreshToken().getToken(),
      expiresAt: session.getAccessToken().getExpiration() * 1000,
      userSub,
   };
   localStorage.setItem(STORAGE_KEY, JSON.stringify(tokens));
}

export function isAuthenticated(): boolean {
   const t = getStoredTokens();
   if (!t) return false;
   return t.expiresAt > Date.now();
}

export function ClearUserAuth() {
   localStorage.removeItem(STORAGE_KEY);
}

/** Returns the Authorization header value for API calls. */
export function getAuthHeader(): string {
   const t = getStoredTokens();
   if (!t) throw redirect({ to: '/login' });
   return `Bearer ${t.accessToken}`;
}

/** Returns the user's Cognito sub (UUID) from stored tokens. */
export function getUserSub(): string {
   return getStoredTokens()?.userSub ?? '';
}

/** Returns the user's email by decoding the stored ID token payload. */
export function getUserEmail(): string {
   const tokens = getStoredTokens();
   if (!tokens?.idToken) return '';
   try {
      const parts = tokens.idToken.split('.');
      if (parts.length !== 3) return '';
      // Pad for valid base64
      const payload = parts[1].replace(/-/g, '+').replace(/_/g, '/');
      const padded = payload + '='.repeat((4 - (payload.length % 4)) % 4);
      const decoded = JSON.parse(atob(padded)) as Record<string, unknown>;
      return (decoded['email'] as string) ?? '';
   } catch {
      return '';
   }
}

/**
 * Decodes and returns all claims from the stored Cognito ID token payload.
 * Used for client-side group checks (e.g. "cognito:groups" contains "admin").
 * Returns null when no valid ID token is stored.
 */
export function getIdTokenClaims(): Record<string, unknown> | null {
   const tokens = getStoredTokens();
   if (!tokens?.idToken) return null;
   try {
      const parts = tokens.idToken.split('.');
      if (parts.length !== 3) return null;
      const payload = parts[1].replace(/-/g, '+').replace(/_/g, '/');
      const padded = payload + '='.repeat((4 - (payload.length % 4)) % 4);
      return JSON.parse(atob(padded)) as Record<string, unknown>;
   } catch {
      return null;
   }
}

// ----------------------------------------------------------------
// Auth operations
// ----------------------------------------------------------------

export interface LoginResult {
   success: boolean;
   error?: string;
   needsConfirmation?: boolean;
}

export async function login(
   email: string,
   password: string,
): Promise<LoginResult> {
   return new Promise((resolve) => {
      const pool = getUserPool();
      const user = new CognitoUser({ Username: email, Pool: pool });
      const authDetails = new AuthenticationDetails({
         Username: email,
         Password: password,
      });
      user.authenticateUser(authDetails, {
         onSuccess: (session) => {
            const sub = session.getIdToken().decodePayload()['sub'] as string;
            storeSession(session, sub);
            resolve({ success: true });
         },
         onFailure: (err) => {
            resolve({ success: false, error: err.message });
         },
      });
   });
}

export interface SignUpResult {
   success: boolean;
   error?: string;
   needsConfirmation: boolean;
}

export async function signUp(
   email: string,
   password: string,
): Promise<SignUpResult> {
   return new Promise((resolve) => {
      const pool = getUserPool();
      const attrs = [new CognitoUserAttribute({ Name: 'email', Value: email })];
      pool.signUp(email, password, attrs, [], (err) => {
         if (err) {
            resolve({
               success: false,
               error: err.message,
               needsConfirmation: false,
            });
            return;
         }
         resolve({ success: true, needsConfirmation: true });
      });
   });
}

export async function confirmSignUp(
   email: string,
   code: string,
): Promise<{ success: boolean; error?: string }> {
   return new Promise((resolve) => {
      const pool = getUserPool();
      const user = new CognitoUser({ Username: email, Pool: pool });
      user.confirmRegistration(code, true, (err) => {
         if (err) {
            resolve({ success: false, error: err.message });
            return;
         }
         resolve({ success: true });
      });
   });
}

export async function signOut(): Promise<void> {
   const pool = getUserPool();
   const user = pool.getCurrentUser();
   if (user) {
      user.signOut();
   }
   ClearUserAuth();
}

// ----------------------------------------------------------------
// Password reset flow
// ----------------------------------------------------------------

export interface ForgotPasswordResult {
   success: boolean;
   error?: string;
   deliveryMedium?: string; // "EMAIL"
   destination?: string; // masked e.g. "t***@example.com"
}

/**
 * Initiates the Cognito "forgot password" flow.
 * Cognito sends a verification code to the user's registered email.
 */
export async function forgotPassword(
   email: string,
): Promise<ForgotPasswordResult> {
   return new Promise((resolve) => {
      const pool = getUserPool();
      const user = new CognitoUser({ Username: email, Pool: pool });
      user.forgotPassword({
         onSuccess: (data) => {
            resolve({
               success: true,
               deliveryMedium: data?.CodeDeliveryDetails?.DeliveryMedium,
               destination: data?.CodeDeliveryDetails?.Destination,
            });
         },
         onFailure: (err) => {
            resolve({ success: false, error: err.message });
         },
      });
   });
}

export interface ConfirmPasswordResult {
   success: boolean;
   error?: string;
}

/**
 * Completes the password reset — submits the verification code and new password.
 */
export async function confirmForgotPassword(
   email: string,
   code: string,
   newPassword: string,
): Promise<ConfirmPasswordResult> {
   return new Promise((resolve) => {
      const pool = getUserPool();
      const user = new CognitoUser({ Username: email, Pool: pool });
      user.confirmPassword(code, newPassword, {
         onSuccess: () => resolve({ success: true }),
         onFailure: (err) => resolve({ success: false, error: err.message }),
      });
   });
}

/**
 * Resends the email verification code for sign-up confirmation.
 */
export async function resendConfirmationCode(
   email: string,
): Promise<{ success: boolean; error?: string }> {
   return new Promise((resolve) => {
      const pool = getUserPool();
      const user = new CognitoUser({ Username: email, Pool: pool });
      user.resendConfirmationCode((err) => {
         if (err) {
            resolve({ success: false, error: err.message });
            return;
         }
         resolve({ success: true });
      });
   });
}

export interface ChangePasswordResult {
   success: boolean;
   error?: string;
}

/**
 * Changes the current user's password. Requires the user to be logged in.
 * Uses the Cognito SDK changePassword flow (current + new password).
 */
export async function changePassword(
   currentPassword: string,
   newPassword: string,
): Promise<ChangePasswordResult> {
   return new Promise((resolve) => {
      const pool = getUserPool();
      const user = pool.getCurrentUser();
      if (!user) {
         resolve({ success: false, error: 'Not logged in.' });
         return;
      }
      // getSession refreshes tokens if needed before calling changePassword
      user.getSession(
         (err: Error | null, session: CognitoUserSession | null) => {
            if (err || !session) {
               resolve({
                  success: false,
                  error: 'Session expired — please log in again.',
               });
               return;
            }
            user.changePassword(currentPassword, newPassword, (err2) => {
               if (err2) {
                  resolve({ success: false, error: err2.message });
                  return;
               }
               resolve({ success: true });
            });
         },
      );
   });
}

export async function refreshSession(): Promise<boolean> {
   const tokens = getStoredTokens();
   if (!tokens) return false;
   return new Promise((resolve) => {
      const pool = getUserPool();
      const user = pool.getCurrentUser();
      if (!user) {
         resolve(false);
         return;
      }
      user.getSession(
         (err: Error | null, session: CognitoUserSession | null) => {
            if (err || !session || !session.isValid()) {
               resolve(false);
               return;
            }
            const sub = session.getIdToken().decodePayload()['sub'] as string;
            storeSession(session, sub);
            resolve(true);
         },
      );
   });
}
