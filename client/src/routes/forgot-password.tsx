import { createFileRoute, Link, useNavigate } from '@tanstack/react-router';
import {
   TextField,
   Button,
   Box,
   Paper,
   Typography,
   Alert,
   InputAdornment,
   Divider,
} from '@mui/material';
import { useState } from 'react';
import {
   forgotPassword,
   confirmForgotPassword,
} from '../services/auth.service';

export const Route = createFileRoute('/forgot-password')({
   component: ForgotPasswordPage,
});

type Stage = 'request' | 'verify' | 'success';

// Decorative rune divider — matches the dungeon aesthetic
function RuneDivider() {
   return (
      <Box sx={{ display: 'flex', alignItems: 'center', my: 2, gap: 1 }}>
         <Divider sx={{ flex: 1, borderColor: 'primary.dark' }} />
         <Typography
            sx={{
               color: 'primary.main',
               fontSize: '1rem',
               userSelect: 'none',
               px: 1,
               opacity: 0.7,
            }}
         >
            ✦
         </Typography>
         <Divider sx={{ flex: 1, borderColor: 'primary.dark' }} />
      </Box>
   );
}

function ForgotPasswordPage() {
   const [stage, setStage] = useState<Stage>('request');
   const [email, setEmail] = useState('');
   const [code, setCode] = useState('');
   const [newPassword, setNewPassword] = useState('');
   const [confirmPassword, setConfirmPassword] = useState('');
   const [destination, setDestination] = useState('');
   const [error, setError] = useState('');
   const [isLoading, setIsLoading] = useState(false);
   const navigate = useNavigate();

   // ── Stage 1: request reset code ─────────────────────────────────
   const handleRequest = async (e: React.FormEvent) => {
      e.preventDefault();
      setError('');
      setIsLoading(true);
      const result = await forgotPassword(email);
      setIsLoading(false);
      if (!result.success) {
         // Don't reveal whether the account exists — give a generic message
         // but still forward so we don't leak account existence
         if (result.error?.includes('UserNotFoundException')) {
            setDestination(email);
            setStage('verify');
            return;
         }
         setError(result.error ?? 'Failed to send reset code');
         return;
      }
      setDestination(result.destination ?? email);
      setStage('verify');
   };

   // ── Stage 2: submit code + new password ─────────────────────────
   const handleVerify = async (e: React.FormEvent) => {
      e.preventDefault();
      setError('');
      if (newPassword !== confirmPassword) {
         setError('Passwords do not match');
         return;
      }
      if (newPassword.length < 8) {
         setError('Password must be at least 8 characters');
         return;
      }
      setIsLoading(true);
      const result = await confirmForgotPassword(email, code, newPassword);
      setIsLoading(false);
      if (!result.success) {
         setError(
            result.error ??
               'Reset failed — the code may be expired or incorrect',
         );
         return;
      }
      setStage('success');
   };

   return (
      <Box
         sx={{
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            minHeight: '70vh',
            pt: 4,
         }}
      >
         <Paper
            sx={{
               p: { xs: 3, sm: 5 },
               maxWidth: 440,
               width: '100%',
               position: 'relative',
               overflow: 'hidden',
               // Subtle corner rune accents
               '&::before': {
                  content: '"⚔"',
                  position: 'absolute',
                  top: 12,
                  left: 16,
                  fontSize: '1.1rem',
                  opacity: 0.15,
                  color: 'primary.main',
                  pointerEvents: 'none',
               },
               '&::after': {
                  content: '"⚔"',
                  position: 'absolute',
                  top: 12,
                  right: 16,
                  fontSize: '1.1rem',
                  opacity: 0.15,
                  color: 'primary.main',
                  pointerEvents: 'none',
                  transform: 'scaleX(-1)',
               },
            }}
         >
            {/* ── Title ── */}
            <Typography
               variant="h4"
               component="h1"
               align="center"
               sx={{ mb: 0.5, color: 'primary.main' }}
            >
               {stage === 'success' ? 'Seal Broken' : 'Forgotten Runes'}
            </Typography>
            <Typography
               variant="body2"
               align="center"
               sx={{ color: 'text.secondary', mb: 2, fontStyle: 'italic' }}
            >
               {stage === 'request' &&
                  'An enchanted missive shall be sent to restore your access.'}
               {stage === 'verify' &&
                  `A scroll was dispatched to ${destination}. Enter the seal within to forge a new password.`}
               {stage === 'success' &&
                  'Your password has been reforged. The dungeon doors await.'}
            </Typography>

            <RuneDivider />

            {error && (
               <Alert
                  severity="error"
                  sx={{
                     mb: 2,
                     borderColor: 'error.main',
                     '& .MuiAlert-icon': { color: 'error.main' },
                  }}
               >
                  {error}
               </Alert>
            )}

            {/* ── Stage: request ── */}
            {stage === 'request' && (
               <form onSubmit={handleRequest}>
                  <TextField
                     fullWidth
                     label="Email address"
                     type="email"
                     value={email}
                     onChange={(e) => setEmail(e.target.value)}
                     margin="normal"
                     required
                     autoComplete="email"
                     autoFocus
                     InputProps={{
                        startAdornment: (
                           <InputAdornment position="start">
                              <Typography
                                 sx={{ fontSize: '1rem', opacity: 0.5 }}
                              >
                                 ✉
                              </Typography>
                           </InputAdornment>
                        ),
                     }}
                     helperText="Enter the email bound to your adventurer's account"
                  />
                  <Button
                     type="submit"
                     fullWidth
                     variant="contained"
                     size="large"
                     disabled={isLoading}
                     sx={{ mt: 3, mb: 1 }}
                  >
                     {isLoading ? 'Sending scroll...' : 'Send Reset Scroll'}
                  </Button>
               </form>
            )}

            {/* ── Stage: verify ── */}
            {stage === 'verify' && (
               <form onSubmit={handleVerify}>
                  <TextField
                     fullWidth
                     label="Verification seal"
                     value={code}
                     onChange={(e) => setCode(e.target.value.trim())}
                     margin="normal"
                     required
                     autoFocus
                     slotProps={{
                        htmlInput: {
                           maxLength: 8,
                           inputMode: 'numeric',
                           autoComplete: 'one-time-code',
                        },
                        input: {
                           startAdornment: (
                              <InputAdornment position="start">
                                 <Typography
                                    sx={{ fontSize: '1rem', opacity: 0.5 }}
                                 >
                                    🔑
                                 </Typography>
                              </InputAdornment>
                           ),
                        },
                     }}
                     helperText="6-digit code from the enchanted missive"
                  />
                  <TextField
                     fullWidth
                     label="New password"
                     type="password"
                     value={newPassword}
                     onChange={(e) => setNewPassword(e.target.value)}
                     margin="normal"
                     required
                     autoComplete="new-password"
                     helperText="Must be at least 8 characters, with uppercase, lowercase & number"
                  />
                  <TextField
                     fullWidth
                     label="Confirm new password"
                     type="password"
                     value={confirmPassword}
                     onChange={(e) => setConfirmPassword(e.target.value)}
                     margin="normal"
                     required
                     autoComplete="new-password"
                  />
                  <Button
                     type="submit"
                     fullWidth
                     variant="contained"
                     size="large"
                     disabled={isLoading}
                     sx={{ mt: 3, mb: 1 }}
                  >
                     {isLoading ? 'Reforging...' : 'Reforge Password'}
                  </Button>
                  <Button
                     fullWidth
                     variant="text"
                     size="small"
                     onClick={() => setStage('request')}
                     sx={{ color: 'text.secondary' }}
                  >
                     ← Send a new scroll
                  </Button>
               </form>
            )}

            {/* ── Stage: success ── */}
            {stage === 'success' && (
               <Box sx={{ textAlign: 'center' }}>
                  <Typography
                     sx={{
                        fontSize: '3rem',
                        mb: 2,
                        filter: 'drop-shadow(0 0 8px rgba(201, 169, 98, 0.6))',
                     }}
                  >
                     ⚔️
                  </Typography>
                  <Typography
                     variant="body1"
                     sx={{ mb: 3, color: 'text.secondary' }}
                  >
                     Your access has been restored, adventurer. The dungeon
                     awaits your return.
                  </Typography>
                  <Button
                     variant="contained"
                     size="large"
                     fullWidth
                     onClick={() => navigate({ to: '/login' })}
                  >
                     Return to the Gates
                  </Button>
               </Box>
            )}

            <RuneDivider />

            <Typography
               variant="body2"
               align="center"
               sx={{ color: 'text.secondary' }}
            >
               Remembered your password?{' '}
               <Link
                  to="/login"
                  style={{
                     color: 'inherit',
                     fontWeight: 600,
                     textDecoration: 'underline',
                  }}
               >
                  Sign In
               </Link>
            </Typography>
         </Paper>
      </Box>
   );
}
