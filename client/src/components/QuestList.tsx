import { Box, Typography } from '@mui/material';
import type { ObjectiveStateView } from '@/types/types';

// Quests: active first, then completed and missed, each with a status mark.
const QUEST_ORDER: Record<string, number> = { active: 0, completed: 1, failed: 2 };
const QUEST_MARK: Record<string, string> = { active: '◆', completed: '✓', failed: '✗' };

export function QuestList({ objectives }: { objectives: ObjectiveStateView[] }) {
   if (!objectives.length) {
      return (
         <Typography variant="body2" sx={{ color: 'text.secondary' }}>
            No quests yet.
         </Typography>
      );
   }
   const sorted = [...objectives].sort(
      (a, b) => (QUEST_ORDER[a.status] ?? 3) - (QUEST_ORDER[b.status] ?? 3),
   );
   return (
      <Box component="ul" sx={{ listStyle: 'none', m: 0, p: 0 }}>
         {sorted.map((o) => (
            <Box
               component="li"
               key={o.id}
               aria-label={`${o.visible_text ?? o.id} (${o.status})`}
               sx={{
                  display: 'flex',
                  gap: 1,
                  py: 0.25,
                  fontSize: '0.95rem',
                  color:
                     o.status === 'active'
                        ? 'text.primary'
                        : o.status === 'completed'
                          ? 'success.main'
                          : 'text.disabled',
                  textDecoration: o.status === 'failed' ? 'line-through' : 'none',
               }}
            >
               <Box component="span" sx={{ color: o.status === 'active' ? 'primary.main' : 'inherit' }}>
                  {QUEST_MARK[o.status] ?? '•'}
               </Box>
               {o.visible_text ?? o.id}
            </Box>
         ))}
      </Box>
   );
}
