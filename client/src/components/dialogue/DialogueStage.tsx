import {
   Alert,
   Box,
   Button,
   CircularProgress,
   Paper,
   Snackbar,
   Stack,
   TextField,
   Typography,
} from '@mui/material';
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import type {
   ChatMessage,
   DialogueChoice,
   EndingView,
   WorldEvent,
} from '@/types/types';
import { EndingPanel } from '../Chat';
import { buildBeats, buildTranscript, type Beat } from './beats';
import { renderInline } from './inline';
import { useTypewriter } from './useTypewriter';

/** How long a scene title card holds before moving on by itself. */
export const SCENE_CARD_MS = 1800;

const NOTICE_LABELS: Record<string, string> = {
   campaign_progress: 'New objective',
   objective_completed: 'Objective complete',
   objective_failed: 'Objective missed',
};

const noticeText = (e: WorldEvent) =>
   e.message.replace(/^Objective (updated|complete|missed):\s*/, '');

interface DialogueStageProps {
   messages: ChatMessage[];
   streamingMessage?: string;
   /** The narrator is thinking or streaming. */
   isLoading: boolean;
   pendingChoices?: DialogueChoice[];
   onChoiceSelected?: (id: number) => void;
   isSubmittingChoice?: boolean;
   command: string;
   setCommand: (cmd: string) => void;
   handleCommand: () => void;
   /** Free-text input is unavailable (e.g. quota reached). */
   inputDisabled?: boolean;
   ending?: EndingView | null;
   onStartNewRun?: () => void;
   /** Latest server error; a change stops the "waiting for the story" state. */
   error?: string | null;
   /** Where the player is, for the header. */
   location?: string;
   sceneTitle?: string;
   objective?: string;
}

/**
 * Presents the story like a game rather than a chat: one beat at a time,
 * typed in, advanced by click / Space / Enter. Scene changes get a title
 * card, quest updates arrive as toasts, and choices or the text box only
 * appear once the current text has been read. Everything before the current
 * line stays above it as a scrollback.
 */
export function DialogueStage({
   messages,
   streamingMessage,
   isLoading,
   pendingChoices,
   onChoiceSelected,
   isSubmittingChoice = false,
   command,
   setCommand,
   handleCommand,
   inputDisabled = false,
   ending,
   onStartNewRun,
   error,
   location,
   sceneTitle,
   objective,
}: DialogueStageProps) {
   const beats = useMemo(
      () => buildBeats(messages, streamingMessage),
      [messages, streamingMessage],
   );

   // A session where the player hasn't acted yet plays its opening from the
   // first beat. Otherwise everything already in the log has been read: it is
   // shown instantly and never re-announced.
   const fresh = useRef(!messages.some((m) => m.type === 'player')).current;
   const alreadyRead = useRef<Set<string> | null>(null);
   if (alreadyRead.current === null) {
      alreadyRead.current = new Set(fresh ? [] : beats.map((b) => b.id));
   }
   const announced = useRef<Set<string>>(new Set(alreadyRead.current));

   const [cursorId, setCursorId] = useState<string | null>(() => {
      if (!beats.length) return null;
      return fresh ? beats[0].id : beats[beats.length - 1].id;
   });
   // Whether the player has dismissed the last beat (freeing the text box).
   const [acknowledged, setAcknowledged] = useState(!fresh);
   const [toasts, setToasts] = useState<WorldEvent[]>([]);
   // The player acted and the story hasn't answered yet. Covers the gap
   // before the first streamed chunk, when isLoading is still false.
   const [waiting, setWaiting] = useState(false);
   const busy = isLoading || waiting;
   useEffect(() => setWaiting(false), [beats.length, error]);
   useEffect(() => {
      if (!waiting) return;
      const id = setTimeout(() => setWaiting(false), 30_000);
      return () => clearTimeout(id);
   }, [waiting]);

   let index = beats.findIndex((b) => b.id === cursorId);
   if (index === -1 && beats.length) index = beats.length - 1;
   const current: Beat | undefined = beats[index];
   const hasNext = index < beats.length - 1;
   const instant = !current || alreadyRead.current.has(current.id);

   const { visible, typing, complete } = useTypewriter(
      current?.kind === 'line' ? current.text : '',
      current?.id ?? 'none',
      instant,
   );
   const lineDone = !current || current.kind === 'scene' || (!typing && !current.streaming);
   const atEnd = !hasNext && lineDone;
   const hasChoices = !!pendingChoices?.length && !!onChoiceSelected;
   const showChoices = atEnd && hasChoices && !ending;
   const showEnding = atEnd && !!ending;
   const showInput = atEnd && acknowledged && !hasChoices && !ending && !busy;
   // At rest: the player has finished everything on screen and is waiting.
   const atRest = atEnd && (acknowledged || hasChoices || busy);

   const goNext = useCallback(() => {
      if (!hasNext) return;
      setCursorId(beats[index + 1].id);
      setAcknowledged(false);
   }, [beats, index, hasNext]);

   const advance = useCallback(() => {
      if (!current) return;
      // Selecting text to copy shouldn't move the story on.
      if (window.getSelection()?.toString()) return;
      if (current.kind === 'line' && typing) {
         complete();
      } else if (hasNext) {
         goNext();
      } else if (lineDone) {
         setAcknowledged(true);
      }
   }, [current, typing, complete, hasNext, goNext, lineDone]);

   // New beats arriving while the player waits are shown right away; beats
   // arriving mid-read wait for the player to continue.
   const wasAtRest = useRef(atRest);
   useEffect(() => {
      if (wasAtRest.current && hasNext) goNext();
   }, [beats.length]); // eslint-disable-line react-hooks/exhaustive-deps
   useEffect(() => {
      wasAtRest.current = atRest;
   });

   // Title cards hold briefly, then move on by themselves.
   useEffect(() => {
      if (current?.kind !== 'scene' || instant || !hasNext) return;
      const id = setTimeout(goNext, SCENE_CARD_MS);
      return () => clearTimeout(id);
   }, [current, instant, hasNext, goNext]);

   // Announce quest updates once their beat has been reached (including
   // updates that arrive just after their text).
   useEffect(() => {
      const fresh: WorldEvent[] = [];
      for (const beat of beats.slice(0, index + 1)) {
         if (beat.notices?.length && !announced.current.has(beat.id)) {
            announced.current.add(beat.id);
            fresh.push(...beat.notices);
         }
      }
      if (fresh.length) setToasts((t) => [...t, ...fresh]);
   }, [beats, index]);

   // Space / Enter advance, unless the player is typing in the text box.
   useEffect(() => {
      const onKey = (e: KeyboardEvent) => {
         const target = e.target as HTMLElement | null;
         if (target?.closest('input, textarea, button')) return;
         if (e.key === ' ' || e.key === 'Enter') {
            e.preventDefault();
            advance();
         }
      };
      window.addEventListener('keydown', onKey);
      return () => window.removeEventListener('keydown', onKey);
   }, [advance]);

   // What the player just said, until the story answers. It takes the
   // dialogue box so the player's words read last, after everything before.
   const lastMessage = messages[messages.length - 1];
   const pendingPlayer =
      atEnd && lastMessage?.type === 'player' && !streamingMessage ? lastMessage : null;

   const transcript = useMemo(
      () =>
         pendingPlayer
            ? buildTranscript(messages.slice(0, -1), null)
            : buildTranscript(messages, current?.id ?? null),
      [messages, current?.id, pendingPlayer],
   );

   // Keep the scrollback pinned to the newest line, unless the player has
   // scrolled up to reread.
   const scrollRef = useRef<HTMLDivElement>(null);
   const pinned = useRef(true);
   const lastScrollTop = useRef(0);
   useLayoutEffect(() => {
      const el = scrollRef.current;
      if (el && pinned.current) el.scrollTop = el.scrollHeight;
   }, [transcript.length]);
   // Re-pin whenever the scrollback's box or its content changes size: the
   // dialogue box growing as a line types in shrinks the box, and a web font
   // loading or lines re-wrapping grows the content. Chromium's scroll
   // anchoring hides the second case; Safari (WebKit) has none.
   const contentRef = useRef<HTMLDivElement>(null);
   const hasTranscript = transcript.length > 0;
   useEffect(() => {
      const el = scrollRef.current;
      if (!el || typeof ResizeObserver !== 'function') return;
      const observer = new ResizeObserver(() => {
         if (pinned.current) el.scrollTop = el.scrollHeight;
      });
      observer.observe(el);
      if (contentRef.current) observer.observe(contentRef.current);
      return () => observer.disconnect();
   }, [hasTranscript]);

   const showContinue = !!current && lineDone && current.kind === 'line' && (hasNext || !acknowledged) && !showChoices && !showEnding;

   return (
      <Paper
         sx={{
            height: '100%',
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
            position: 'relative',
         }}
      >
         <Box
            data-testid="stage-header"
            sx={{
               display: 'flex',
               alignItems: 'flex-start',
               gap: 2,
               px: 2,
               py: 1,
               borderBottom: 1,
               borderColor: 'divider',
               flexShrink: 0,
            }}
         >
            <Box sx={{ flex: 1, minWidth: 0 }}>
               <Typography
                  sx={{
                     fontFamily: 'Cinzel, Georgia, serif',
                     color: 'primary.main',
                     fontSize: '1rem',
                  }}
               >
                  {location ?? 'Unknown place'}
                  {sceneTitle && (
                     <Typography
                        component="span"
                        sx={{ color: 'text.secondary', fontSize: '0.85rem', ml: 1 }}
                     >
                        · {sceneTitle}
                     </Typography>
                  )}
               </Typography>
               {objective && (
                  <Typography
                     variant="body2"
                     sx={{ color: 'text.secondary', mt: 0.25 }}
                     noWrap
                     title={objective}
                  >
                     Objective: {objective}
                  </Typography>
               )}
            </Box>
         </Box>

         <Box
            data-testid="dialogue-stage"
            onClick={advance}
            sx={{
               flex: 1,
               // Let the transcript give up space, not the header or dialogue box.
               minHeight: 0,
               display: 'flex',
               flexDirection: 'column',
               justifyContent: 'flex-end',
               px: { xs: 2, sm: 4 },
               pb: 2,
               cursor: showChoices || showInput ? 'default' : 'pointer',
            }}
         >
            {transcript.length > 0 && (
               <Box
                  ref={scrollRef}
                  data-testid="stage-transcript"
                  aria-label="story so far"
                  onScroll={(e) => {
                     // Only scrolling up means the player is rereading. Layout
                     // changes (the dialogue box growing) also fire scroll
                     // events, moving down; those mustn't unpin.
                     const el = e.currentTarget;
                     if (el.scrollHeight - el.scrollTop - el.clientHeight < 24) {
                        pinned.current = true;
                     } else if (el.scrollTop < lastScrollTop.current - 1) {
                        pinned.current = false;
                     }
                     lastScrollTop.current = el.scrollTop;
                  }}
                  sx={{
                     flex: 1,
                     minHeight: 0,
                     overflowY: 'auto',
                     // We keep the newest line in view ourselves; Chrome's
                     // scroll anchoring would fight it.
                     overflowAnchor: 'none',
                     mb: 2,
                     pr: 1,
                     display: 'flex',
                     flexDirection: 'column',
                  }}
               >
                  <Box
                     ref={contentRef}
                     // Short logs sit just above the dialogue box.
                     sx={{ mt: 'auto', display: 'flex', flexDirection: 'column', gap: 0.75 }}
                  >
                     {transcript.map((item, i) => (
                        <Typography
                           key={item.key}
                           sx={{
                              // The newest lines read clearest.
                              opacity: i >= transcript.length - 3 ? 0.85 : 0.65,
                              flexShrink: 0,
                              fontFamily: 'Crimson Text, Georgia, serif',
                              fontSize: '1rem',
                              lineHeight: 1.45,
                              color: item.kind === 'player' ? 'secondary.light' : 'text.primary',
                              textAlign: item.kind === 'scene' ? 'center' : 'left',
                              fontStyle: item.kind === 'player' ? 'italic' : 'normal',
                           }}
                        >
                           {item.kind === 'scene' ? (
                              `— ${item.text} —`
                           ) : item.kind === 'player' ? (
                              `You: ${item.text}`
                           ) : (
                              <>
                                 {item.speaker && (
                                    <Box component="span" sx={{ fontWeight: 700, mr: 0.75 }}>
                                       {item.speaker}:
                                    </Box>
                                 )}
                                 {renderInline(item.text)}
                              </>
                           )}
                        </Typography>
                     ))}
                  </Box>
               </Box>
            )}

            {pendingPlayer ? (
               <Box
                  data-testid="stage-player-line"
                  sx={{
                     border: 2,
                     borderColor: 'secondary.dark',
                     borderRadius: 2,
                     p: { xs: 2, sm: 3 },
                     minHeight: 110,
                     backgroundColor: 'rgba(26, 15, 30, 0.75)',
                     position: 'relative',
                  }}
               >
                  <Typography
                     sx={{
                        fontFamily: 'Cinzel, Georgia, serif',
                        color: 'secondary.light',
                        letterSpacing: 1.5,
                        textTransform: 'uppercase',
                        fontSize: '0.9rem',
                        mb: 1,
                     }}
                  >
                     You
                  </Typography>
                  <Typography
                     sx={{
                        fontFamily: 'Crimson Text, Georgia, serif',
                        fontSize: '1.3rem',
                        lineHeight: 1.6,
                        color: 'secondary.light',
                        fontStyle: 'italic',
                        whiteSpace: 'pre-wrap',
                     }}
                  >
                     {pendingPlayer.content}
                  </Typography>
                  {busy && (
                     <CircularProgress
                        size={16}
                        aria-label="the story continues"
                        sx={{ position: 'absolute', right: 16, bottom: 12 }}
                     />
                  )}
               </Box>
            ) : current?.kind === 'scene' ? (
               <SceneCard key={current.id} title={current.text} />
            ) : current ? (
               <Box
                  sx={{
                     border: 2,
                     borderColor: 'primary.dark',
                     borderRadius: 2,
                     p: { xs: 2, sm: 3 },
                     minHeight: 110,
                     backgroundColor: 'rgba(26, 15, 30, 0.75)',
                     position: 'relative',
                  }}
               >
                  {current.speaker && (
                     <Typography
                        sx={{
                           fontFamily: 'Cinzel, Georgia, serif',
                           color: 'primary.main',
                           letterSpacing: 1.5,
                           textTransform: 'uppercase',
                           fontSize: '0.9rem',
                           mb: 1,
                        }}
                     >
                        {current.speaker}
                     </Typography>
                  )}
                  <Typography
                     data-testid="stage-text"
                     sx={{
                        fontFamily: 'Crimson Text, Georgia, serif',
                        fontSize: '1.3rem',
                        lineHeight: 1.6,
                        color: 'text.primary',
                        whiteSpace: 'pre-wrap',
                     }}
                  >
                     {renderInline(visible)}
                  </Typography>
                  {showContinue && (
                     <Typography
                        aria-label="continue"
                        sx={{
                           position: 'absolute',
                           right: 16,
                           bottom: 8,
                           color: 'primary.main',
                           animation: 'bob 1.2s ease-in-out infinite',
                           '@keyframes bob': {
                              '0%, 100%': { transform: 'translateY(0)' },
                              '50%': { transform: 'translateY(3px)' },
                           },
                        }}
                     >
                        ▼
                     </Typography>
                  )}
                  {busy && atEnd && !streamingMessage && (
                     <CircularProgress
                        size={16}
                        aria-label="the story continues"
                        sx={{ position: 'absolute', right: 16, bottom: 12 }}
                     />
                  )}
               </Box>
            ) : null}

            {showChoices && (
               <Stack spacing={1} sx={{ mt: 2 }} onClick={(e) => e.stopPropagation()}>
                  {pendingChoices!.map((choice) => (
                     <Button
                        key={choice.id}
                        variant="outlined"
                        disabled={isSubmittingChoice}
                        onClick={() => {
                           setWaiting(true);
                           onChoiceSelected!(choice.id);
                        }}
                        sx={{
                           justifyContent: 'flex-start',
                           textAlign: 'left',
                           fontFamily: 'Crimson Text, Georgia, serif',
                           fontSize: '1.1rem',
                           textTransform: 'none',
                        }}
                     >
                        ▸ {choice.text}
                     </Button>
                  ))}
               </Stack>
            )}

            {showEnding && (
               <Box sx={{ mt: 2 }} onClick={(e) => e.stopPropagation()}>
                  <EndingPanel ending={ending!} onStartNewRun={onStartNewRun} />
               </Box>
            )}

            {showInput && (
               <Box
                  component="form"
                  onClick={(e) => e.stopPropagation()}
                  onSubmit={(e) => {
                     e.preventDefault();
                     if (!command.trim()) return;
                     setWaiting(true);
                     handleCommand();
                  }}
                  sx={{ display: 'flex', gap: 1, mt: 2 }}
               >
                  {inputDisabled ? (
                     <Alert severity="info" sx={{ flex: 1 }}>
                        Free-text play is unavailable right now.
                     </Alert>
                  ) : (
                     <>
                        <TextField
                           autoFocus
                           fullWidth
                           size="small"
                           autoComplete="off"
                           placeholder="What do you do?"
                           value={command}
                           onChange={(e) => setCommand(e.target.value)}
                        />
                        <Button type="submit" variant="contained" disabled={!command.trim()}>
                           Act
                        </Button>
                     </>
                  )}
               </Box>
            )}
         </Box>

         <Snackbar
            open={toasts.length > 0}
            autoHideDuration={3500}
            onClose={(_, reason) => {
               if (reason === 'clickaway') return;
               setToasts((t) => t.slice(1));
            }}
            anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
            sx={{ position: 'absolute' }}
         >
            <Alert
               severity={toasts[0]?.type === 'objective_failed' ? 'warning' : 'success'}
               variant="filled"
               onClose={() => setToasts((t) => t.slice(1))}
            >
               {toasts[0] && (
                  <>
                     <strong>{NOTICE_LABELS[toasts[0].type] ?? 'Update'}:</strong>{' '}
                     {noticeText(toasts[0])}
                  </>
               )}
            </Alert>
         </Snackbar>

      </Paper>
   );
}

function SceneCard({ title }: { title: string }) {
   return (
      <Box
         data-testid="scene-card"
         sx={{
            alignSelf: 'center',
            textAlign: 'center',
            my: 'auto',
            animation: 'sceneIn 0.8s ease forwards',
            '@keyframes sceneIn': {
               from: { opacity: 0, letterSpacing: '0.3em' },
               to: { opacity: 1, letterSpacing: '0.12em' },
            },
         }}
      >
         <Typography variant="overline" sx={{ color: 'text.secondary' }}>
            Scene
         </Typography>
         <Typography
            variant="h4"
            sx={{ fontFamily: 'Cinzel, Georgia, serif', color: 'primary.main' }}
         >
            {title}
         </Typography>
      </Box>
   );
}
