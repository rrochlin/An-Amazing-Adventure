import { Box, Paper } from '@mui/material';
import type { ReactNode } from 'react';
import { AppTheme } from '../theme/theme';

const panelHover = {
   transition: 'all 0.3s ease-in-out',
   '&:hover': {
      boxShadow: '0 6px 24px rgba(0,0,0,0.6), inset 0 1px 0 rgba(201,169,98,0.2)',
   },
};

/**
 * The game screen's panels: map, story stage, and player info. Purely
 * presentational so the layout can be rendered and checked on its own.
 */
export function GameLayout({
   map,
   stage,
   alerts,
   info,
   mapExpanded,
   children,
}: {
   map: ReactNode;
   stage: ReactNode;
   alerts?: ReactNode;
   info: ReactNode;
   mapExpanded: boolean;
   /** Overlays such as toasts. */
   children?: ReactNode;
}) {
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
         {/* Left — Map (25% normal → 75% expanded, covering the stage) */}
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
                  ...panelHover,
               }}
            >
               {map}
            </Paper>
         </Box>
         {/* Center — Stage (50% normal → hidden when the map is expanded) */}
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
            <Paper sx={{ flex: 1, overflow: 'hidden', ...panelHover }}>{stage}</Paper>
            {alerts}
         </Box>
         {/* Right — Game Info (25%, always visible) */}
         <Box sx={{ flex: '0 0 25%', minWidth: 0, display: 'flex', gap: 2 }}>
            <Paper
               sx={{
                  flex: 1,
                  overflow: 'hidden',
                  display: 'flex',
                  flexDirection: 'column',
                  ...panelHover,
               }}
            >
               {info}
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
         {children}
      </Box>
   );
}
