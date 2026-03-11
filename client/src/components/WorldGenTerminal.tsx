/**
 * WorldGenTerminal
 * Displays world-generation progress as a scrollable terminal window.
 * Lines arrive over WebSocket as world_gen_log frames.
 * Shows a blinking cursor while generation is in progress.
 * When enteringGame=true the terminal plays a "sealing" animation before
 * the parent fades it out and reveals the game.
 */
import { useEffect, useRef } from 'react';
import { Box, Typography, LinearProgress } from '@mui/material';

interface WorldGenTerminalProps {
   lines: string[];
   ready: boolean;
   /** Set to true when the game state has loaded and we're about to transition */
   enteringGame?: boolean;
}

export function WorldGenTerminal({
   lines,
   ready,
   enteringGame = false,
}: WorldGenTerminalProps) {
   const bottomRef = useRef<HTMLDivElement>(null);

   // Auto-scroll to bottom whenever a new line arrives
   useEffect(() => {
      // scrollIntoView may be absent in test environments (jsdom)
      if (typeof bottomRef.current?.scrollIntoView === 'function') {
         bottomRef.current.scrollIntoView({ behavior: 'smooth' });
      }
   }, [lines]);

   // Colour scheme shifts from green → gold when entering
   const accent = enteringGame ? '#c9a962' : '#00ff46';
   const accentMid = enteringGame
      ? 'rgba(201,169,98,0.7)'
      : 'rgba(0,255,70,0.7)';
   const accentDim = enteringGame
      ? 'rgba(201,169,98,0.4)'
      : 'rgba(0,255,70,0.4)';
   const accentFaint = enteringGame
      ? 'rgba(201,169,98,0.08)'
      : 'rgba(0,255,70,0.08)';
   const accentBorder = enteringGame
      ? 'rgba(201,169,98,0.3)'
      : 'rgba(0,255,70,0.3)';
   const accentBorderFaint = enteringGame
      ? 'rgba(201,169,98,0.2)'
      : 'rgba(0,255,70,0.2)';
   const accentGlow = enteringGame
      ? 'rgba(201,169,98,0.15)'
      : 'rgba(0,255,70,0.1)';

   const statusLabel = enteringGame
      ? 'ENTERING THE REALM...'
      : ready
        ? 'COMPLETE'
        : 'GENERATING...';

   return (
      <Box
         sx={{
            width: '100%',
            borderRadius: 1,
            overflow: 'hidden',
            border: `1px solid ${accentBorder}`,
            boxShadow: `0 0 20px ${accentGlow}, inset 0 0 40px rgba(0,0,0,0.6)`,
            fontFamily: '"Courier New", "Courier", monospace',
            transition: 'border-color 0.8s ease, box-shadow 0.8s ease',
         }}
      >
         {/* Terminal title bar */}
         <Box
            sx={{
               px: 2,
               py: 0.75,
               background: accentFaint,
               borderBottom: `1px solid ${accentBorderFaint}`,
               display: 'flex',
               alignItems: 'center',
               gap: 1,
               transition: 'background 0.8s ease',
            }}
         >
            <Box
               sx={{
                  width: 10,
                  height: 10,
                  borderRadius: '50%',
                  bgcolor: enteringGame ? accent : ready ? '#00ff46' : '#ff9800',
                  boxShadow: enteringGame ? `0 0 6px ${accent}` : 'none',
                  transition: 'background-color 0.8s ease, box-shadow 0.8s ease',
               }}
            />
            <Typography
               variant="caption"
               sx={{
                  color: accentMid,
                  fontFamily: 'inherit',
                  letterSpacing: '0.15em',
                  transition: 'color 0.8s ease',
               }}
            >
               WORLD ARCHITECT — {statusLabel}
            </Typography>
         </Box>

         {/* Progress bar — indeterminate while generating, full+gold while entering */}
         {(!ready || enteringGame) && (
            <LinearProgress
               variant={enteringGame ? 'determinate' : 'indeterminate'}
               value={enteringGame ? 100 : undefined}
               sx={{
                  height: 2,
                  '& .MuiLinearProgress-bar': {
                     background: enteringGame
                        ? 'rgba(201,169,98,0.8)'
                        : 'rgba(0,255,70,0.6)',
                     transition: 'background 0.8s ease',
                  },
                  background: enteringGame
                     ? 'rgba(201,169,98,0.15)'
                     : 'rgba(0,255,70,0.1)',
                  transition: 'background 0.8s ease',
               }}
            />
         )}

         {/* Log output */}
         <Box
            sx={{
               p: 2,
               minHeight: 220,
               maxHeight: 360,
               overflowY: 'auto',
               background: 'rgba(0, 8, 0, 0.85)',
               '&::-webkit-scrollbar': { width: '6px' },
               '&::-webkit-scrollbar-track': { background: 'transparent' },
               '&::-webkit-scrollbar-thumb': {
                  background: accentBorder,
                  borderRadius: '3px',
                  transition: 'background 0.8s ease',
               },
            }}
         >
            {lines.length === 0 && (
               <Typography
                  component="div"
                  sx={{
                     color: accentDim,
                     fontFamily: 'inherit',
                     fontSize: '0.85rem',
                  }}
               >
                  Waiting for Architect...
               </Typography>
            )}
            {lines.map((line, i) => {
               const isEntering = enteringGame && i >= lines.length - 3;
               const isSuccess =
                  line.startsWith('Your adventure') ||
                  line.startsWith('>>>') ||
                  line.startsWith('Entering');
               return (
                  <Typography
                     key={i}
                     component="div"
                     sx={{
                        color: line.startsWith('ERROR')
                           ? '#ff4444'
                           : isEntering || isSuccess
                             ? accent
                             : 'rgba(0, 255, 70, 0.85)',
                        fontFamily: 'inherit',
                        fontSize: '0.82rem',
                        lineHeight: 1.6,
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-word',
                        fontWeight: isEntering ? 600 : 400,
                        letterSpacing: isEntering ? '0.05em' : 'normal',
                        transition: 'color 0.5s ease',
                        '&::before': { content: '"> "', opacity: 0.5 },
                     }}
                  >
                     {line}
                  </Typography>
               );
            })}
            {/* Blinking cursor — hidden when entering */}
            {!ready && !enteringGame && (
               <Box
                  component="span"
                  sx={{
                     display: 'inline-block',
                     width: '8px',
                     height: '14px',
                     bgcolor: 'rgba(0, 255, 70, 0.8)',
                     ml: '2px',
                     verticalAlign: 'middle',
                     animation: 'blink 1s step-end infinite',
                     '@keyframes blink': {
                        '0%, 100%': { opacity: 1 },
                        '50%': { opacity: 0 },
                     },
                  }}
               />
            )}
            {/* Solid gold cursor pulse when entering */}
            {enteringGame && (
               <Box
                  component="span"
                  sx={{
                     display: 'inline-block',
                     width: '8px',
                     height: '14px',
                     bgcolor: accent,
                     ml: '2px',
                     verticalAlign: 'middle',
                     animation: 'pulse 0.6s ease-in-out infinite alternate',
                     '@keyframes pulse': {
                        from: { opacity: 0.4, transform: 'scaleY(0.8)' },
                        to: { opacity: 1, transform: 'scaleY(1)' },
                     },
                  }}
               />
            )}
            <div ref={bottomRef} />
         </Box>
      </Box>
   );
}
