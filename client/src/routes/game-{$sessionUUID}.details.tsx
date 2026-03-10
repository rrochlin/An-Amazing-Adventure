import { createFileRoute, redirect, useNavigate } from '@tanstack/react-router';
import {
   Alert,
   Box,
   Button,
   Chip,
   CircularProgress,
   Divider,
   Paper,
   Typography,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import { useEffect, useState } from 'react';
import { isAuthenticated } from '@/services/auth.service';
import { LoadGame, type GameLoadResponse } from '@/services/api.game';

const PREFERENCE_LABELS: Record<string, string> = {
   combat: 'Combat',
   puzzles: 'Puzzles',
   dialog: 'Dialog',
   exploration: 'Exploration',
   chance: 'Chance',
   stealth: 'Stealth',
   crafting: 'Crafting',
   mystery: 'Mystery',
};

export const Route = createFileRoute('/game-{$sessionUUID}/details')({
   component: GameDetailsPage,
   beforeLoad: async ({ location }) => {
      if (!isAuthenticated()) {
         throw redirect({ to: '/login', search: { redirect: location.href } });
      }
   },
});

function Section({
   title,
   children,
}: {
   title: string;
   children: React.ReactNode;
}) {
   return (
      <Box sx={{ mb: 3 }}>
         <Typography
            variant="subtitle1"
            sx={{
               textTransform: 'uppercase',
               letterSpacing: '0.1em',
               fontFamily: '"Cinzel", serif',
               color: 'primary.main',
               borderBottom: '1px solid',
               borderColor: 'rgba(201,169,98,0.3)',
               pb: 0.5,
               mb: 1.5,
            }}
         >
            {title}
         </Typography>
         {children}
      </Box>
   );
}

function DetailRow({
   label,
   value,
}: {
   label: string;
   value?: string | number | null;
}) {
   if (!value && value !== 0) return null;
   return (
      <Box sx={{ display: 'flex', gap: 2, mb: 1, alignItems: 'flex-start' }}>
         <Typography
            variant="body2"
            sx={{
               color: 'text.secondary',
               minWidth: 120,
               flexShrink: 0,
               fontFamily: '"Crimson Text", "Georgia", serif',
               fontSize: '1rem',
            }}
         >
            {label}
         </Typography>
         <Typography
            variant="body2"
            sx={{
               fontFamily: '"Crimson Text", "Georgia", serif',
               fontSize: '1rem',
               whiteSpace: 'pre-wrap',
            }}
         >
            {value}
         </Typography>
      </Box>
   );
}

function GameDetailsPage() {
   const { sessionUUID } = Route.useParams();
   const navigate = useNavigate();
   const [data, setData] = useState<GameLoadResponse | null>(null);
   const [loading, setLoading] = useState(true);
   const [error, setError] = useState<string | null>(null);

   useEffect(() => {
      let cancelled = false;
      LoadGame(sessionUUID)
         .then((res) => {
            if (!cancelled) setData(res);
         })
         .catch(() => {
            if (!cancelled) setError('Failed to load adventure details.');
         })
         .finally(() => {
            if (!cancelled) setLoading(false);
         });
      return () => {
         cancelled = true;
      };
   }, [sessionUUID]);

   const goBack = () =>
      navigate({ to: '/game-{$sessionUUID}', params: { sessionUUID } });

   if (loading) {
      return (
         <Box
            sx={{
               display: 'flex',
               justifyContent: 'center',
               alignItems: 'center',
               minHeight: 'calc(100vh - 64px)',
            }}
         >
            <CircularProgress />
         </Box>
      );
   }

   if (error || !data) {
      return (
         <Box sx={{ p: 4, maxWidth: 600, mx: 'auto' }}>
            <Alert severity="error" sx={{ mb: 2 }}>
               {error ?? 'Adventure not found.'}
            </Alert>
            <Button
               startIcon={<ArrowBackIcon />}
               onClick={() => navigate({ to: '/' })}
            >
               Back to Adventures
            </Button>
         </Box>
      );
   }

   const player = data.state?.self ?? data.state?.player;
   const params = data.creation_params;
   const roomCount = data.state ? Object.keys(data.state.rooms).length : 0;
   const enemyCount = data.state
      ? Object.values(data.state.rooms).reduce(
           (n, r) => n + r.occupants.filter((o) => !o.friendly).length,
           0,
        )
      : 0;

   return (
      <Box
         sx={{
            display: 'flex',
            justifyContent: 'center',
            pt: 4,
            pb: 6,
            px: 2,
            minHeight: 'calc(100vh - 64px)',
         }}
      >
         <Box sx={{ maxWidth: 720, width: '100%' }}>
            {/* Header */}
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 3 }}>
               <Button
                  startIcon={<ArrowBackIcon />}
                  onClick={goBack}
                  variant="outlined"
                  size="small"
               >
                  Back to Game
               </Button>
               <Typography
                  variant="h4"
                  sx={{
                     fontFamily: '"Cinzel", serif',
                     fontSize: '1.6rem',
                     flex: 1,
                     textAlign: 'right',
                  }}
               >
                  Adventure Details
               </Typography>
            </Box>

            <Paper
               sx={{
                  p: 4,
                  backgroundImage:
                     'linear-gradient(rgba(106, 78, 157, 0.05), rgba(201, 169, 98, 0.05))',
                  border: '1px solid rgba(201, 169, 98, 0.2)',
               }}
            >
               {/* Quest */}
               <Section title="Quest">
                  {data.title && (
                     <Typography
                        variant="h5"
                        sx={{
                           fontFamily: '"Cinzel", serif',
                           mb: 1,
                        }}
                     >
                        {data.title}
                     </Typography>
                  )}
                  <DetailRow label="Theme" value={data.theme} />
                  <DetailRow label="Objective" value={data.quest_goal} />
               </Section>

               <Divider sx={{ my: 2, borderColor: 'rgba(201,169,98,0.15)' }} />

               {/* Character */}
               <Section title="Character">
                  <DetailRow label="Name" value={player?.name} />
                  {/* D&D 5e stats — present for v3+ characters */}
                  {player?.dnd && (
                     <>
                        <DetailRow label="Race" value={player.dnd.race_id} />
                        <DetailRow label="Class" value={player.dnd.class_id} />
                        <DetailRow label="Level" value={player.dnd.level} />
                        <DetailRow
                           label="HP"
                           value={`${player.health} / ${player.dnd.max_hp}`}
                        />
                        <DetailRow label="AC" value={player.dnd.ac} />
                        <DetailRow
                           label="Speed"
                           value={`${player.dnd.speed} ft`}
                        />
                        {player.dnd.abilities && (
                           <Box sx={{ mt: 1 }}>
                              <Typography
                                 variant="caption"
                                 sx={{
                                    color: 'text.secondary',
                                    textTransform: 'uppercase',
                                    letterSpacing: '0.05em',
                                 }}
                              >
                                 Ability Scores
                              </Typography>
                              <Box
                                 sx={{
                                    display: 'flex',
                                    flexWrap: 'wrap',
                                    gap: 1,
                                    mt: 0.5,
                                 }}
                              >
                                 {Object.entries(player.dnd.abilities).map(
                                    ([ab, score]) => (
                                       <Chip
                                          key={ab}
                                          size="small"
                                          label={`${ab.toUpperCase()} ${score}`}
                                          variant="outlined"
                                       />
                                    ),
                                 )}
                              </Box>
                           </Box>
                        )}
                     </>
                  )}
                  {/* Legacy fields — shown for v1/v2 records */}
                  {!player?.dnd && (
                     <>
                        <DetailRow
                           label="Description"
                           value={player?.description}
                        />
                     </>
                  )}
               </Section>

               <Divider sx={{ my: 2, borderColor: 'rgba(201,169,98,0.15)' }} />

               {/* Adventure Preferences */}
               <Section title="Adventure Preferences">
                  {params?.theme_hint && (
                     <DetailRow label="Theme hint" value={params.theme_hint} />
                  )}
                  {params?.preferences && params.preferences.length > 0 ? (
                     <Box
                        sx={{
                           display: 'flex',
                           flexWrap: 'wrap',
                           gap: 1,
                           mt: 0.5,
                        }}
                     >
                        {params.preferences.map((p: string) => (
                           <Chip
                              key={p}
                              label={PREFERENCE_LABELS[p] ?? p}
                              size="small"
                              color="primary"
                              variant="outlined"
                           />
                        ))}
                     </Box>
                  ) : (
                     <Typography
                        variant="body2"
                        sx={{ color: 'text.secondary', fontStyle: 'italic' }}
                     >
                        No specific preferences set — AI chose freely.
                     </Typography>
                  )}
               </Section>

               <Divider sx={{ my: 2, borderColor: 'rgba(201,169,98,0.15)' }} />

               {/* Statistics */}
               <Section title="Statistics">
                  <Box
                     sx={{
                        display: 'grid',
                        gridTemplateColumns: 'repeat(2, 1fr)',
                        gap: 2,
                     }}
                  >
                     {[
                        { label: 'Rooms', value: roomCount },
                        { label: 'Enemies', value: enemyCount },
                        {
                           label: 'Conversations',
                           value: data.conversation_count ?? 0,
                        },
                        {
                           label: 'Tokens Used',
                           value: data.total_tokens?.toLocaleString() ?? '0',
                        },
                     ].map(({ label, value }) => (
                        <Paper
                           key={label}
                           sx={{
                              p: 2,
                              textAlign: 'center',
                              backgroundColor: 'rgba(201,169,98,0.05)',
                              border: '1px solid rgba(201,169,98,0.15)',
                           }}
                        >
                           <Typography
                              variant="h4"
                              sx={{
                                 fontFamily: '"Cinzel", serif',
                                 color: 'primary.main',
                              }}
                           >
                              {value}
                           </Typography>
                           <Typography
                              variant="caption"
                              sx={{
                                 color: 'text.secondary',
                                 textTransform: 'uppercase',
                                 letterSpacing: '0.08em',
                              }}
                           >
                              {label}
                           </Typography>
                        </Paper>
                     ))}
                  </Box>
               </Section>
            </Paper>
         </Box>
      </Box>
   );
}
