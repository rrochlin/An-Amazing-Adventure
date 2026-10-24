import { Box, Drawer, Paper } from '@mui/material';
import type { ReactNode } from 'react';
import { AppTheme } from '../theme/theme';

const panelHover = {
   transition: 'all 0.3s ease-in-out',
   '&:hover': {
      boxShadow: '0 6px 24px rgba(0,0,0,0.6), inset 0 1px 0 rgba(201,169,98,0.2)',
   },
};

// The header is a Toolbar inside an AppBar, which is a Paper and so has a
// 1px border top and bottom. Fill the rest of the window at each of the
// Toolbar's breakpoint heights, so the page itself never scrolls.
const HEADER_BORDER_PX = 2;
const fillBelowHeader = (() => {
   const fill = (h: unknown) => ({
      height: `calc(100dvh - ${Number(h) + HEADER_BORDER_PX}px)`,
   });
   const sx: Record<string, unknown> = {};
   for (const [key, value] of Object.entries(AppTheme.mixins.toolbar)) {
      if (key === 'minHeight') Object.assign(sx, fill(value));
      else if (value && typeof value === 'object' && 'minHeight' in value) {
         sx[key] = fill(value.minHeight);
      }
   }
   return sx;
})();

/**
 * The game screen: the scene (map, with the story over it) as the main view,
 * and player info on the right. Purely presentational.
 */
export function GameLayout({
   scene,
   alerts,
   info,
   children,
   infoOpen = false,
   onCloseInfo,
}: {
   scene: ReactNode;
   alerts?: ReactNode;
   info: ReactNode;
   /** Overlays such as toasts. */
   children?: ReactNode;
   /** On a narrow screen the info panel is a drawer: whether it's open. */
   infoOpen?: boolean;
   onCloseInfo?: () => void;
}) {
   return (
      <Box
         sx={{
            ...fillBelowHeader,
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
            p: { xs: 1, md: 2 },
            pr: { xs: 1, md: 3 },
            width: '100%',
            maxWidth: '100vw',
            boxSizing: 'border-box',
         }}
      >
         {/* Main — the scene, with the story text box over it */}
         <Box sx={{ flex: '1 1 75%', minWidth: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
            <Paper sx={{ flex: 1, overflow: 'hidden', p: 0, ...panelHover }}>{scene}</Paper>
            {alerts}
         </Box>
         {/* Right — Game Info (a drawer on narrow screens, where the scene
             needs the whole width) */}
         <Box sx={{ flex: '0 0 25%', minWidth: 0, display: { xs: 'none', md: 'flex' }, gap: 2 }}>
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
         </Box>
         <Drawer
            anchor="right"
            open={infoOpen}
            onClose={onCloseInfo}
            sx={{ display: { md: 'none' } }}
            slotProps={{ paper: { sx: { width: 'min(360px, 90vw)', display: 'flex', flexDirection: 'column' } } }}
         >
            {info}
         </Drawer>
         {children}
      </Box>
   );
}
