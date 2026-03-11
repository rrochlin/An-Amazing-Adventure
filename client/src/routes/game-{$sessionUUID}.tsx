import { createFileRoute, redirect, useNavigate } from '@tanstack/react-router';
import { useCallback, useEffect, useRef, useState } from 'react';
import {
   Alert,
   Box,
   Button,
   IconButton,
   Paper,
   Snackbar,
   Tooltip,
   Typography,
} from '@mui/material';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { RoomMap } from '../components/RoomMap';
import { GameInfo } from '../components/GameInfo';
import { Chat } from '../components/Chat';
import { isAuthenticated, getIdTokenClaims } from '../services/auth.service';
import { LoadGame, RetryWorldGen } from '../services/api.game';
import { useGameStore } from '../store/gameStore';
import { useGameSocket } from '../hooks/useGameSocket';
import { WorldGenTerminal } from '../components/WorldGenTerminal';
import { AppTheme } from '@/theme/theme';
import type { RoomView } from '../types/types';

function GameErrorFallback() {
   const navigate = useNavigate();
   return (
      <Box
         sx={{
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            minHeight: 'calc(100vh - 64px)',
            p: 4,
         }}
      >
         <Paper
            sx={{ maxWidth: 480, width: '100%', p: 4, textAlign: 'center' }}
         >
            <Typography
               variant="h5"
               sx={{ mb: 2, fontFamily: '"Cinzel", serif' }}
            >
               Adventure Unavailable
            </Typography>
            <Alert severity="error" sx={{ mb: 3, textAlign: 'left' }}>
               This adventure could not be loaded. It may have been deleted or
               the server encountered an error.
            </Alert>
            <Button variant="contained" onClick={() => navigate({ to: '/' })}>
               Return to Adventures
            </Button>
         </Paper>
      </Box>
   );
}

export const Route = createFileRoute('/game-{$sessionUUID}')({
   component: GamePage,
   errorComponent: GameErrorFallback,
   beforeLoad: async ({ location }) => {
      if (!isAuthenticated()) {
         throw redirect({ to: '/login', search: { redirect: location.href } });
      }
   },
});

function GamePage() {
   const { sessionUUID } = Route.useParams();
   const navigate = useNavigate();
   const [command, setCommand] = useState('');
   const [loadError, setLoadError] = useState<string | null>(null);
   const [loadingGame, setLoadingGame] = useState(true);
   // UI-FUT-4: room focused via map hover/click; null = show current room
   const [focusedRoom, setFocusedRoom] = useState<RoomView | null>(null);
   // UI-FUT-1: map expanded — slides over chat column
   const [mapExpanded, setMapExpanded] = useState(false);
   // UI-FUT-8: reconnection toast — show when WS reconnects after a drop
   const [reconnectToast, setReconnectToast] = useState(false);
   const [ownerID, setOwnerID] = useState<string | null>(null);
   // Stuck world-gen detection: set after 90s with no world_gen_ready and still not-ready
   const [worldGenStuck, setWorldGenStuck] = useState(false);
   const [retrying, setRetrying] = useState(false);
   const [retryError, setRetryError] = useState<string | null>(null);
   // Transition phases: 'terminal' → 'entering' (dramatic pause) → 'game'
   type TransitionPhase = 'terminal' | 'entering' | 'game';
   const [transitionPhase, setTransitionPhase] = useState<TransitionPhase>('terminal');

   const claims = getIdTokenClaims();
   const currentUserID = claims?.sub ?? null;

   const {
      gameState,
      chatMessages,
      streamingMessage,
      isStreaming,
      wsError,
      wsStatus,
      worldGenLog,
      worldGenReady,
      addChatMessage,
      setGameState,
      appendWorldGenLog,
      setWorldGenReady,
      reset,
   } = useGameStore();

   const prevWsStatus = useRef(wsStatus);

   // Load game state from the server. Called once on mount, and again when world_gen_ready fires.
   const loadGameRef = useRef(false);
   const loadGame = useCallback(async () => {
      if (loadGameRef.current) return;
      loadGameRef.current = true;
      try {
         const data = await LoadGame(sessionUUID);
         if (data.ready && data.state) {
            setGameState(data.state);
            setWorldGenStuck(false);
         }
         // Capture owner ID for party panel
         if (data.owner_id) {
            setOwnerID(data.owner_id);
         }
         // Seed the terminal from persisted logs so late-joining clients see the output.
         // Only replace if the store is still empty (WS frames take priority when live).
         if (data.world_gen_logs && data.world_gen_logs.length > 0) {
            const storeLog = useGameStore.getState().worldGenLog;
            if (storeLog.length === 0) {
               data.world_gen_logs.forEach((line) => appendWorldGenLog(line));
            }
         }
         if (data.ready) {
            setWorldGenReady();
            triggerGameEntry([
               'Sealing the rift between worlds...',
               '>>> Entering the realm <<<',
            ]);
         }
      } catch {
         setLoadError('Failed to load game — please try again.');
      } finally {
         setLoadingGame(false);
      }
   }, [sessionUUID, setGameState, appendWorldGenLog, setWorldGenReady, triggerGameEntry]);

   // Poll LoadGame every 3s while world is not yet ready.
   // This catches the case where the client connected after world-gen finished (missed WS frames).
   const isPollingRef = useRef(false);
   useEffect(() => {
      if (worldGenReady || gameState) return;
      const interval = setInterval(async () => {
         if (isPollingRef.current) return; // skip if previous request is still in flight
         isPollingRef.current = true;
         try {
            const data = await LoadGame(sessionUUID);
            if (data.owner_id) setOwnerID(data.owner_id);
            // Seed missing log lines (add only lines we don't have yet)
            if (data.world_gen_logs && data.world_gen_logs.length > 0) {
               const currentLen = useGameStore.getState().worldGenLog.length;
               const newLines = data.world_gen_logs.slice(currentLen);
               newLines.forEach((line) => appendWorldGenLog(line));
            }
            if (data.ready && data.state) {
               setGameState(data.state);
               setWorldGenReady();
               setWorldGenStuck(false);
               setLoadingGame(false);
               triggerGameEntry([
                  'Sealing the rift between worlds...',
                  '>>> Entering the realm <<<',
               ]);
            }
         } catch {
            // Non-fatal — next tick will retry
         } finally {
            isPollingRef.current = false;
         }
      }, 3000);
      return () => clearInterval(interval);
   }, [worldGenReady, gameState, sessionUUID, setGameState, appendWorldGenLog, setWorldGenReady, triggerGameEntry]); // eslint-disable-line react-hooks/exhaustive-deps

   // If we're still in world-gen after 90s with no progress frames, show retry option.
   const stuckTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
   useEffect(() => {
      if (worldGenReady || gameState) {
         if (stuckTimerRef.current) clearTimeout(stuckTimerRef.current);
         setWorldGenStuck(false);
         return;
      }
      stuckTimerRef.current = setTimeout(() => setWorldGenStuck(true), 90_000);
      return () => {
         if (stuckTimerRef.current) clearTimeout(stuckTimerRef.current);
      };
   }, [worldGenReady, gameState, worldGenLog.length]);

   const handleRetry = async () => {
      setRetrying(true);
      setRetryError(null);
      setWorldGenStuck(false);
      try {
         await RetryWorldGen(sessionUUID);
         // Reset stuck timer — give another 90s for the retry to complete
         stuckTimerRef.current = setTimeout(
            () => setWorldGenStuck(true),
            90_000,
         );
      } catch {
         setRetryError('Failed to restart world generation. Please try again.');
         setWorldGenStuck(true);
      } finally {
         setRetrying(false);
      }
   };

   // Play the dramatic entry sequence once game state is loaded.
   // Appends closing lines to the terminal, shifts to 'entering' phase,
   // then transitions to 'game' after a cinematic pause.
   const triggerGameEntry = useCallback((entryLines: string[]) => {
      entryLines.forEach((line) => appendWorldGenLog(line));
      setTransitionPhase('entering');
      setTimeout(() => setTransitionPhase('game'), 2800);
   }, [appendWorldGenLog]);

   // Called by useGameSocket when world_gen_ready arrives
   const handleWorldReady = useCallback(() => {
      loadGameRef.current = false; // allow reload
      setLoadingGame(true);
      loadGame();
   }, [loadGame]);

   const { sendChat, sendAction } = useGameSocket({
      sessionId: sessionUUID,
      onWorldReady: handleWorldReady,
   });

   useEffect(() => {
      reset();
      loadGameRef.current = false;
      setTransitionPhase('terminal');
      loadGame();
      return () => {
         /* cleanup handled in useGameSocket */
      };
   }, [sessionUUID]); // eslint-disable-line react-hooks/exhaustive-deps

   // UI-FUT-8: detect reconnection (disconnected/error → connected)
   useEffect(() => {
      if (
         wsStatus === 'connected' &&
         (prevWsStatus.current === 'disconnected' ||
            prevWsStatus.current === 'error')
      ) {
         setReconnectToast(true);
      }
      prevWsStatus.current = wsStatus;
   }, [wsStatus]);

   const handleCommand = () => {
      if (!command.trim() || isStreaming) return;
      addChatMessage({ type: 'player', content: command });
      sendChat(command);
      setCommand('');
   };

   if (loadError) {
      return (
         <Box sx={{ p: 4 }}>
            <Alert severity="error">{loadError}</Alert>
         </Box>
      );
   }

   // Show world-gen terminal while world is being built (not yet ready) or transitioning in
   if (loadingGame || transitionPhase !== 'game' || (!gameState && !loadError)) {
      return (
         <Box
            sx={{
               display: 'flex',
               flexDirection: 'column',
               justifyContent: 'center',
               alignItems: 'center',
               minHeight: 'calc(100vh - 64px)',
               p: 4,
               gap: 3,
               opacity: transitionPhase === 'entering' ? 1 : 1,
               transition: 'opacity 0.6s ease',
            }}
         >
            <Paper
               sx={{
                  maxWidth: 600,
                  width: '100%',
                  background: 'rgba(0, 8, 0, 0.96)',
                  border: '1px solid rgba(0, 255, 70, 0.25)',
                  boxShadow: '0 0 40px rgba(0, 255, 70, 0.15)',
                  overflow: 'hidden',
               }}
            >
               <Box
                  sx={{
                     px: 2,
                     py: 1.5,
                     borderBottom: `1px solid ${transitionPhase === 'entering' ? 'rgba(201,169,98,0.3)' : 'rgba(0,255,70,0.2)'}`,
                     transition: 'border-color 0.8s ease',
                  }}
               >
                  <Typography
                     sx={{
                        fontFamily: '"Cinzel", "Georgia", serif',
                        color: transitionPhase === 'entering' ? 'rgba(201,169,98,0.95)' : 'rgba(0,255,70,0.9)',
                        fontSize: '1rem',
                        transition: 'color 0.8s ease',
                     }}
                  >
                     {transitionPhase === 'entering' ? 'Your Realm Awaits...' : 'Forging Your World'}
                  </Typography>
               </Box>
               <Box sx={{ p: 2 }}>
                  <WorldGenTerminal
                     lines={worldGenLog}
                     ready={worldGenReady}
                     enteringGame={transitionPhase === 'entering'}
                  />
               </Box>
               {worldGenLog.length === 0 &&
                  !worldGenReady &&
                  !worldGenStuck && (
                     <Box sx={{ px: 2, pb: 1.5 }}>
                        <Typography
                           variant="caption"
                           sx={{
                              color: 'rgba(0,255,70,0.4)',
                              fontFamily: 'monospace',
                           }}
                        >
                           Connecting to world generator...
                        </Typography>
                     </Box>
                  )}
               {worldGenStuck && (
                  <Box
                     sx={{
                        px: 2,
                        pb: 2,
                        display: 'flex',
                        flexDirection: 'column',
                        gap: 1.5,
                     }}
                  >
                     <Typography
                        variant="caption"
                        sx={{
                           color: 'rgba(255,180,0,0.8)',
                           fontFamily: 'monospace',
                        }}
                     >
                        World generation appears to have stalled.
                     </Typography>
                     {retryError && (
                        <Typography
                           variant="caption"
                           sx={{
                              color: 'rgba(255,80,80,0.9)',
                              fontFamily: 'monospace',
                           }}
                        >
                           {retryError}
                        </Typography>
                     )}
                     <Button
                        size="small"
                        variant="outlined"
                        onClick={handleRetry}
                        disabled={retrying}
                        sx={{
                           alignSelf: 'flex-start',
                           color: 'rgba(0,255,70,0.9)',
                           borderColor: 'rgba(0,255,70,0.4)',
                           fontFamily: 'monospace',
                           fontSize: '0.75rem',
                           '&:hover': {
                              borderColor: 'rgba(0,255,70,0.8)',
                              background: 'rgba(0,255,70,0.08)',
                           },
                        }}
                     >
                        {retrying
                           ? 'Restarting...'
                           : 'Restart World Generation'}
                     </Button>
                  </Box>
               )}
            </Paper>
            {worldGenReady && (
               <Alert
                  severity="success"
                  sx={{
                     maxWidth: 600,
                     width: '100%',
                     background: 'rgba(0, 255, 70, 0.08)',
                     color: 'rgba(0,255,70,0.9)',
                     border: '1px solid rgba(0,255,70,0.3)',
                  }}
               >
                  World ready — loading your adventure...
               </Alert>
            )}
         </Box>
      );
   }

   // displayMessages contains only committed (finalized) messages.
   // The in-flight streamingMessage is passed separately to Chat so it can
   // render a dedicated StreamingChatMessage bubble rather than a plain entry.
   const displayMessages = chatMessages;

   return (
      <Box
         sx={{
            height: `calc(100vh - ${AppTheme.mixins.toolbar.minHeight}px)`,
            display: 'flex',
            flexDirection: 'row',
            overflow: 'hidden',
            animation: 'fadeInGame 0.8s ease forwards',
            '@keyframes fadeInGame': {
               from: { opacity: 0, transform: 'translateY(8px)' },
               to: { opacity: 1, transform: 'translateY(0)' },
            },
            backgroundColor: 'background.default',
            gap: 2,
            p: 2,
            pr: 3,
            width: '100%',
            maxWidth: '100vw',
            boxSizing: 'border-box',
         }}
      >
         {/* Left — Map (25% normal → 75% expanded, covering Chat only) */}
         <Box
            sx={{
               flex: mapExpanded ? '0 0 75%' : '0 0 25%',
               minWidth: 0,
               display: 'flex',
               flexDirection: 'column',
               gap: 2,
               transition: 'flex-basis 0.35s cubic-bezier(0.4, 0, 0.2, 1)',
            }}
         >
            <Paper
               sx={{
                  flex: 1,
                  p: 2,
                  display: 'flex',
                  flexDirection: 'column',
                  overflow: 'hidden',
                  transition: 'all 0.3s ease-in-out',
                  '&:hover': {
                     boxShadow:
                        '0 6px 24px rgba(0,0,0,0.6), inset 0 1px 0 rgba(201,169,98,0.2)',
                  },
               }}
            >
               <Box
                  sx={{
                     display: 'flex',
                     alignItems: 'center',
                     mb: 2,
                     borderBottom: `2px solid ${AppTheme.palette.primary.main}`,
                     pb: 1,
                  }}
               >
                  <Typography
                     variant="h6"
                     sx={{
                        flex: 1,
                        textAlign: 'center',
                        textTransform: 'uppercase',
                        letterSpacing: '0.1em',
                     }}
                  >
                     World Map
                  </Typography>
                  <Tooltip title="Adventure details">
                     <IconButton
                        size="small"
                        onClick={() =>
                           navigate({
                              to: '/game-{$sessionUUID}/details',
                              params: { sessionUUID },
                           })
                        }
                        sx={{
                           color: 'primary.main',
                           opacity: 0.7,
                           '&:hover': { opacity: 1 },
                        }}
                     >
                        <InfoOutlinedIcon fontSize="small" />
                     </IconButton>
                  </Tooltip>
               </Box>
               {/* Map fills all remaining vertical space inside the Paper */}
               <Box
                  sx={{
                     flex: 1,
                     minHeight: 0,
                     display: 'flex',
                     flexDirection: 'column',
                  }}
               >
                  <RoomMap
                     gameState={gameState}
                     onRoomFocus={setFocusedRoom}
                     onExpand={() => setMapExpanded((v) => !v)}
                     expanded={mapExpanded}
                  />
               </Box>
            </Paper>
         </Box>

         {/* Center — Chat (50% normal → hidden when expanded) */}
         <Box
            sx={{
               flex: mapExpanded ? '0 0 0%' : '0 0 50%',
               minWidth: 0,
               display: 'flex',
               flexDirection: 'column',
               overflow: 'hidden',
               opacity: mapExpanded ? 0 : 1,
               pointerEvents: mapExpanded ? 'none' : 'auto',
               transition:
                  'flex-basis 0.35s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.25s ease',
            }}
         >
            <Paper
               sx={{
                  flex: 1,
                  overflow: 'hidden',
                  transition: 'all 0.3s ease-in-out',
                  '&:hover': {
                     boxShadow:
                        '0 6px 24px rgba(0,0,0,0.6), inset 0 1px 0 rgba(201,169,98,0.2)',
                  },
               }}
            >
               <Chat
                  chatHistory={displayMessages}
                  streamingMessage={streamingMessage || undefined}
                  command={command}
                  setCommand={setCommand}
                  handleCommand={handleCommand}
                  isLoading={
                     isStreaming ||
                     wsError === 'ai_access_not_enabled' ||
                     wsError === 'quota_exceeded'
                  }
               />
            </Paper>
            {wsError === 'ai_access_not_enabled' && (
               <Alert severity="info" sx={{ mt: 1 }}>
                  <strong>Preview Mode</strong> — AI narration is not enabled
                  for your account. Contact the admin to request access.
               </Alert>
            )}
            {wsError === 'quota_exceeded' && (
               <Alert severity="warning" sx={{ mt: 1 }}>
                  <strong>Token quota reached</strong> — Your token limit has
                  been reached. Contact the admin to increase your limit.
               </Alert>
            )}
            {wsError &&
               wsError !== 'ai_access_not_enabled' &&
               wsError !== 'quota_exceeded' && (
                  <Alert severity="warning" sx={{ mt: 1 }}>
                     {wsError ?? 'Connection lost — retrying...'}
                  </Alert>
               )}
         </Box>

         {/* Right — Game Info (25%, always visible) */}
         <Box sx={{ flex: '0 0 25%', minWidth: 0, display: 'flex', gap: 2 }}>
            <Paper
               sx={{
                  flex: 1,
                  overflow: 'hidden',
                  display: 'flex',
                  flexDirection: 'column',
                  transition: 'all 0.3s ease-in-out',
                  '&:hover': {
                     boxShadow:
                        '0 6px 24px rgba(0,0,0,0.6), inset 0 1px 0 rgba(201,169,98,0.2)',
                  },
               }}
            >
               <GameInfo
                  gameState={gameState}
                  sendAction={sendAction}
                  focusedRoom={focusedRoom}
                  sessionId={sessionUUID}
                  isOwner={
                     ownerID != null
                        ? ownerID === currentUserID
                        : currentUserID != null && gameState != null
                  }
               />
            </Paper>
            <Box
               sx={{
                  width: '4px',
                  backgroundColor: '#000',
                  opacity: 0.5,
                  borderRadius: '2px',
               }}
            />
         </Box>

         {/* UI-FUT-8: Reconnection toast */}
         <Snackbar
            open={reconnectToast}
            autoHideDuration={4000}
            onClose={() => setReconnectToast(false)}
            anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
         >
            <Alert
               onClose={() => setReconnectToast(false)}
               severity="success"
               variant="filled"
               sx={{ fontFamily: 'Crimson Text, Georgia, serif' }}
            >
               Reconnected to the adventure
            </Alert>
         </Snackbar>
      </Box>
   );
}
