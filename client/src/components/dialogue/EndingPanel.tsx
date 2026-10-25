import { Box, Button, Typography } from '@mui/material';
import type { EndingView } from '@/types/types';

/** Replaces the input bar once a run has ended: finished runs are read-only. */
export const EndingPanel = ({
   ending,
   onStartNewRun,
}: {
   ending: EndingView;
   onStartNewRun?: () => void;
}) => (
   <Box
      role="status"
      sx={{
         p: 2,
         borderTop: 2,
         borderColor: 'primary.dark',
         textAlign: 'center',
      }}
   >
      <Typography
         variant="overline"
         sx={{ color: 'text.secondary', letterSpacing: 2 }}
      >
         The adventure has ended
      </Typography>
      {ending.title && (
         <Typography
            variant="h6"
            sx={{ fontFamily: 'Cinzel, Georgia, serif', color: 'primary.main' }}
         >
            {ending.title}
         </Typography>
      )}
      {ending.summary && (
         <Typography
            sx={{
               fontFamily: 'Crimson Text, Georgia, serif',
               color: 'text.primary',
               maxWidth: 560,
               mx: 'auto',
               mt: 0.5,
            }}
         >
            {ending.summary}
         </Typography>
      )}
      {onStartNewRun && (
         <Button
            variant="contained"
            onClick={onStartNewRun}
            sx={{ mt: 1.5, fontFamily: 'Cinzel, Georgia, serif' }}
         >
            Begin a new adventure
         </Button>
      )}
   </Box>
);
