import { Badge, Box, Dialog, IconButton, Paper, Tooltip, Typography } from '@mui/material';
import MenuBookIcon from '@mui/icons-material/MenuBook';
import MapIcon from '@mui/icons-material/Map';
import HistoryEduIcon from '@mui/icons-material/HistoryEdu';
import CloseIcon from '@mui/icons-material/Close';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import PersonIcon from '@mui/icons-material/Person';
import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react';
import type { ObjectiveStateView, RoomView, SceneMapView } from '@/types/types';
import { QuestList } from './QuestList';
import { WorldMap } from './map/WorldMap';
import { PhaserStage } from './stage/PhaserStage';
import { sceneArt, type MapArt } from './map/mapArt';
import type { SpokenLine, StageCue } from './dialogue/beats';

/**
 * How each scene light tints the map: a filter on the art and a wash over
 * it. Only the map changes; the sign, the buttons and the text stay as they
 * are.
 */
const MOODS: Record<string, { filter: string; wash: string; flicker?: boolean }> = {
   dawn: { filter: 'saturate(0.9) brightness(0.95)', wash: 'linear-gradient(180deg, rgba(255,170,190,0.22), rgba(255,200,140,0.12))' },
   dusk: { filter: 'saturate(0.85) brightness(0.85)', wash: 'linear-gradient(180deg, rgba(255,120,40,0.25), rgba(90,30,80,0.25))' },
   night: { filter: 'brightness(0.55) saturate(0.6)', wash: 'rgba(20,30,90,0.35)' },
   fire: {
      filter: 'brightness(0.6) saturate(0.8)',
      wash: 'radial-gradient(ellipse at 50% 60%, rgba(255,120,30,0.38), rgba(20,10,40,0.5) 75%)',
      flicker: true,
   },
   ashes: { filter: 'grayscale(0.75) brightness(0.8) contrast(0.95)', wash: 'rgba(60,55,55,0.25)' },
};

/** Share of the scene's height the story text box is expected to cover. */
const TEXT_BOX_SHARE = 0.38;

interface SceneScreenProps {
   /** The campaign's map art; without it, `fallback` is the background. */
   art?: MapArt;
   fallback?: ReactNode;
   rooms: Record<string, RoomView>;
   currentRoomId?: string;
   /** The interior the scene is played on, if any (shown while the player is in its room). */
   sceneMap?: SceneMapView | null;
   /** The scene's light (see MOODS); none is daylight. */
   mood?: string;
   playerName?: string;
   playerScale?: number;
   /** The mark the player stands at, if the story put them at one. */
   playerMark?: string;
   /** Staging the story has reached, oldest first: the scene plays each once. */
   cues?: StageCue[];
   /** The spoken line on screen: shown in a bubble over the speaker. */
   speaking?: SpokenLine | null;
   /** Places the scene keeps the player out of: room ID → the reason. */
   blocked?: Record<string, string>;
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
   /** The player can't walk: the story is talking or waiting on a choice. */
   controlsLocked?: boolean;
   /** The player walked into another room. */
   onWalkToRoom?: (roomId: string) => void;
   /** The player walked up to a character to talk to them. */
   onTalkTo?: (characterId: string, name: string) => void;
   /** On a narrow screen: open the character panel (it isn't beside the scene). */
   onOpenCharacter?: () => void;
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
   sceneMap,
   mood,
   playerName,
   playerScale,
   playerMark,
   cues,
   speaking,
   blocked,
   location,
   sceneTitle,
   objective,
   journalTitle,
   objectives,
   renderStage,
   onDetails,
   controlsLocked,
   onWalkToRoom,
   onTalkTo,
   onOpenCharacter,
}: SceneScreenProps) {
   const [logOpen, setLogOpen] = useState(false);
   const shown = sceneArt(art, sceneMap, currentRoomId);
   const indoors = !!shown?.interior;
   const light = mood ? MOODS[mood] : undefined;
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
         // The scene is a dark stage whatever the page's colour scheme: its
         // panels have dark backgrounds, so their text needs the dark palette.
         className="dark"
         sx={{ position: 'relative', height: '100%', overflow: 'hidden', bgcolor: '#08050c', display: 'flex' }}
      >
         {shown ? (
            <Box
               data-testid="scene-light"
               data-mood={light ? mood : 'day'}
               sx={{ flex: 1, minWidth: 0, display: 'flex', position: 'relative', filter: light?.filter, transition: 'filter 1.2s ease' }}
            >
               <PhaserStage
                  art={shown}
                  rooms={rooms}
                  currentRoomId={currentRoomId}
                  playerName={playerName}
                  playerScale={playerScale}
                  playerMark={playerMark}
                  cues={cues}
                  speaking={speaking}
                  blocked={blocked}
                  places={indoors ? sceneMap?.points_of_interest : undefined}
                  insetTop={chipHeight}
                  insetBottom={framedPanel}
                  // An open overlay (journal, map, log) holds the player still too.
                  controlsLocked={controlsLocked || logOpen || journalOpen || worldOpen}
                  onWalkToRoom={onWalkToRoom}
                  onTalkTo={onTalkTo}
               />
               {light && (
                  <Box
                     aria-hidden
                     sx={{
                        position: 'absolute',
                        inset: 0,
                        pointerEvents: 'none',
                        background: light.wash,
                        ...(light.flicker && {
                           animation: 'scene-flicker 2.4s ease-in-out infinite alternate',
                           '@keyframes scene-flicker': { from: { opacity: 0.8 }, to: { opacity: 1 } },
                           '@media (prefers-reduced-motion: reduce)': { animation: 'none' },
                        }),
                     }}
                  />
               )}
            </Box>
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
               // Leaves room for the button bar on a narrow screen.
               maxWidth: { xs: 'calc(100% - 200px)', sm: '60%' },
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
               {(indoors ? sceneMap?.name : location) ?? 'Unknown place'}
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
            {onOpenCharacter && (
               <Box sx={{ display: { xs: 'inline-flex', md: 'none' } }}>
                  {overlayButton('Character', <PersonIcon />, onOpenCharacter, false)}
               </Box>
            )}
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
               // A portal, outside the scene's dark stage: it needs the dark
               // palette of its own.
               slotProps={{ paper: { className: 'dark', sx: { height: '85vh', bgcolor: '#08050c', display: 'flex' } } }}
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
