import { Badge, Box, Dialog, IconButton, Paper, Tooltip, Typography } from '@mui/material';
import MenuBookIcon from '@mui/icons-material/MenuBook';
import MapIcon from '@mui/icons-material/Map';
import HistoryEduIcon from '@mui/icons-material/HistoryEdu';
import CloseIcon from '@mui/icons-material/Close';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react';
import type { ObjectiveStateView, RoomView } from '@/types/types';
import { QuestList } from './QuestList';
import { WorldMap } from './map/WorldMap';
import type { MapArt } from './map/mapArt';

/** Share of the scene's height the story text box is expected to cover. */
const TEXT_BOX_SHARE = 0.38;

interface SceneScreenProps {
   /** The campaign's map art; without it, `fallback` is the background. */
   art?: MapArt;
   fallback?: ReactNode;
   rooms: Record<string, RoomView>;
   currentRoomId?: string;
   playerName?: string;
   playerScale?: number;
   location?: string;
   sceneTitle?: string;
   objective?: string;
   journalTitle?: string;
   objectives: ObjectiveStateView[];
   /**
    * The story text box, told whether the log (scrollback) is open, and given
    * a callback for its docked panel's height so the scene fills the space
    * above it.
    */
   renderStage: (logOpen: boolean, onPanelHeight: (px: number) => void) => ReactNode;
   /** Opens the adventure details page. */
   onDetails?: () => void;
}

/**
 * The main game screen, GBA-RPG style: the current scene fills the view, the
 * story runs in a text box over it, and overlay buttons open the journal, the
 * world map and the log.
 */
export function SceneScreen({
   art,
   fallback,
   rooms,
   currentRoomId,
   playerName,
   playerScale,
   location,
   sceneTitle,
   objective,
   journalTitle,
   objectives,
   renderStage,
   onDetails,
}: SceneScreenProps) {
   const [logOpen, setLogOpen] = useState(false);
   const [journalOpen, setJournalOpen] = useState(false);
   const [worldOpen, setWorldOpen] = useState(false);

   // The journal shows a dot when quests change while it's closed.
   const questKey = JSON.stringify(objectives.map((o) => [o.id, o.status]));
   const seenQuests = useRef(questKey);
   useEffect(() => {
      if (journalOpen) seenQuests.current = questKey;
   }, [journalOpen, questKey]);
   const questsChanged = questKey !== seenQuests.current && objectives.length > 0;

   useEffect(() => {
      const onKey = (e: KeyboardEvent) => {
         const target = e.target instanceof Element ? e.target : null;
         if (target?.closest('input, textarea') && e.key !== 'Escape') return;
         // Only Escape: letter keys would fight the text box, which has focus.
         if (e.key === 'Escape') {
            setJournalOpen(false);
            setWorldOpen(false);
            setLogOpen(false);
         }
      };
      window.addEventListener('keydown', onKey);
      return () => window.removeEventListener('keydown', onKey);
   }, []);

   // The scene is framed above the tallest the text panel has been at this
   // screen size (up to half the screen): the panel changes height with every
   // line, and following it slid the whole scene up and down.
   const sceneRef = useRef<HTMLDivElement>(null);
   const lastPanel = useRef(0);
   const [panelHeight, setPanelHeight] = useState(0);
   const onPanelHeight = useCallback((px: number) => {
      lastPanel.current = px;
      setPanelHeight((h) => Math.max(h, px));
   }, []);
   const [sceneHeight, setSceneHeight] = useState(0);
   useEffect(() => {
      const el = sceneRef.current;
      if (!el || typeof ResizeObserver !== 'function') return;
      const observer = new ResizeObserver(() => {
         setSceneHeight(el.clientHeight);
         setPanelHeight(lastPanel.current);
      });
      observer.observe(el);
      return () => observer.disconnect();
   }, []);
   const framedPanel = sceneHeight ? Math.min(panelHeight, sceneHeight / 2) : panelHeight;
   // The location sign's height: the scene centres its room below it.
   const chipRef = useRef<HTMLDivElement>(null);
   const [chipHeight, setChipHeight] = useState(0);
   useEffect(() => {
      const el = chipRef.current;
      if (!el) return;
      const measure = () => setChipHeight(el.offsetHeight);
      measure();
      if (typeof ResizeObserver !== 'function') return;
      const observer = new ResizeObserver(measure);
      observer.observe(el);
      return () => observer.disconnect();
   }, []);

   const overlayButton = (label: string, icon: ReactNode, onClick: () => void, active: boolean) => (
      <Tooltip title={label}>
         <IconButton
            aria-label={label}
            aria-pressed={active}
            onClick={onClick}
            size="small"
            sx={{
               borderRadius: 1,
               bgcolor: active ? 'primary.dark' : 'transparent',
               color: 'primary.main',
               '&:hover': { bgcolor: 'rgba(201, 169, 98, 0.15)' },
            }}
         >
            {icon}
         </IconButton>
      </Tooltip>
   );

   return (
      <Box
         ref={sceneRef}
         data-testid="scene-screen"
         sx={{ position: 'relative', height: '100%', overflow: 'hidden', bgcolor: '#08050c', display: 'flex' }}
      >
         {art ? (
            <WorldMap
               art={art}
               rooms={rooms}
               currentRoomId={currentRoomId}
               playerName={playerName}
               playerScale={playerScale}
               mode="scene"
               insetTop={chipHeight}
               insetBottom={framedPanel}
            />
         ) : (
            <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', p: 2, pb: `${TEXT_BOX_SHARE * 100}%` }}>
               {fallback}
            </Box>
         )}

         {/* Docked to the top-left corner, like a GBA location sign. */}
         <Box
            ref={chipRef}
            data-testid="scene-chip"
            sx={{
               position: 'absolute',
               top: 0,
               left: 0,
               maxWidth: '60%',
               px: 2,
               py: 0.75,
               bgcolor: '#140d1a',
               borderRight: '3px solid',
               borderBottom: '3px solid',
               borderColor: 'primary.dark',
               borderBottomRightRadius: 8,
               zIndex: 1,
            }}
         >
            <Typography sx={{ fontFamily: 'Cinzel, Georgia, serif', color: 'primary.main', fontSize: '1rem' }}>
               {location ?? 'Unknown place'}
               {sceneTitle && (
                  <Typography component="span" sx={{ color: 'text.secondary', fontSize: '0.85rem', ml: 1 }}>
                     · {sceneTitle}
                  </Typography>
               )}
            </Typography>
            {objective && (
               <Typography variant="body2" sx={{ color: 'text.secondary' }} noWrap title={objective}>
                  {objective}
               </Typography>
            )}
         </Box>

         {/* One button bar docked to the top-right corner. */}
         <Box
            sx={{
               position: 'absolute',
               top: 0,
               right: 0,
               display: 'flex',
               alignItems: 'center',
               gap: 0.5,
               px: 1,
               py: 0.5,
               bgcolor: '#140d1a',
               borderLeft: '3px solid',
               borderBottom: '3px solid',
               borderColor: 'primary.dark',
               borderBottomLeftRadius: 8,
               zIndex: 3,
            }}
         >
            {overlayButton(
               'Journal',
               <Badge color="secondary" variant="dot" invisible={!questsChanged}>
                  <MenuBookIcon />
               </Badge>,
               () => setJournalOpen((v) => !v),
               journalOpen,
            )}
            {art && overlayButton('World map', <MapIcon />, () => setWorldOpen(true), worldOpen)}
            {overlayButton('Log', <HistoryEduIcon />, () => setLogOpen((v) => !v), logOpen)}
            {onDetails && overlayButton('Adventure details', <InfoOutlinedIcon />, onDetails, false)}
         </Box>

         {journalOpen && (
            <Paper
               data-testid="journal"
               role="dialog"
               aria-label="Journal"
               sx={{
                  // Hangs from the button bar.
                  position: 'absolute',
                  top: 44,
                  right: 0,
                  width: { xs: '100%', sm: 360 },
                  maxHeight: `calc(100% - ${panelHeight + 44}px)`,
                  borderRadius: 0,
                  borderBottomLeftRadius: 8,
                  borderWidth: 0,
                  borderLeft: '3px solid',
                  borderBottom: '3px solid',
                  overflowY: 'auto',
                  p: 2,
                  bgcolor: 'rgba(12, 8, 16, 0.95)',
                  borderColor: 'primary.dark',
                  zIndex: 3,
               }}
            >
               <Typography
                  sx={{ mb: 1, fontFamily: 'Cinzel, Georgia, serif', color: 'primary.main', letterSpacing: '0.08em' }}
               >
                  {journalTitle ?? 'Journal'}
               </Typography>
               {objective && (
                  <Typography variant="body2" sx={{ mb: 1.5, color: 'text.primary', fontStyle: 'italic' }}>
                     {objective}
                  </Typography>
               )}
               <QuestList objectives={objectives} />
            </Paper>
         )}

         <Box sx={{ position: 'absolute', inset: 0, zIndex: 2, pointerEvents: 'none', '& > *': { pointerEvents: 'auto' } }}>
            {renderStage(logOpen, onPanelHeight)}
         </Box>

         {art && (
            <Dialog
               open={worldOpen}
               onClose={() => setWorldOpen(false)}
               fullWidth
               maxWidth="lg"
               aria-label="World map"
               slotProps={{ paper: { sx: { height: '85vh', bgcolor: '#08050c', display: 'flex' } } }}
            >
               <Box sx={{ display: 'flex', alignItems: 'center', px: 2, pt: 1.5 }}>
                  <Typography sx={{ flex: 1, fontFamily: 'Cinzel, Georgia, serif', color: 'primary.main' }}>World map</Typography>
                  <IconButton aria-label="close world map" onClick={() => setWorldOpen(false)}>
                     <CloseIcon />
                  </IconButton>
               </Box>
               <Box sx={{ flex: 1, minHeight: 0, display: 'flex', p: 1 }}>
                  <WorldMap
                     art={art}
                     rooms={rooms}
                     currentRoomId={currentRoomId}
                     playerName={playerName}
                     playerScale={playerScale}
                     mode="world"
                  />
               </Box>
            </Dialog>
         )}
      </Box>
   );
}
