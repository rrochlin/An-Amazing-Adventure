import { Box, IconButton, Tooltip, Typography } from '@mui/material';
import OpenInFullIcon from '@mui/icons-material/OpenInFull';
import CloseFullscreenIcon from '@mui/icons-material/CloseFullscreen';
import { useEffect, useMemo, useRef, useState } from 'react';
import type { CharacterView, RoomView } from '@/types/types';
import {
   DEFAULT_SPRITES,
   DUNGEON_SHEET,
   GROUND_TILES,
   OBJECT_TILES,
   SHEET_COLUMNS,
   TILE,
   TOWN_SHEET,
   sheetOffset,
   type MapArt,
} from './mapArt';

interface WorldMapProps {
   art: MapArt;
   rooms: Record<string, RoomView>;
   currentRoomId?: string;
   playerName?: string;
   /** The player's height relative to an adult. */
   playerScale?: number;
   expanded?: boolean;
   onExpand?: () => void;
   /**
    * "scene" fills the view with the current room only (the main game
    * screen); "world" fits the whole map; "panel" follows the player at a
    * crisp zoom. Defaults to world when expanded, else panel.
    */
   mode?: MapMode;
   /** Pixels at the bottom covered by an overlay (the story text box). */
   insetBottom?: number;
}

export type MapMode = 'scene' | 'world' | 'panel';

export interface Token {
   id: string;
   name: string;
   roomId: string;
   x: number;
   y: number;
   sprite: number;
   /** Height relative to an adult: sprites are drawn at this size. */
   scale: number;
   isPlayer: boolean;
}

/** About this many tiles across fit in the side panel at its zoom. */
const FOLLOW_TILES = 12;

/**
 * The world as a pixel-art map (Roadwarden-style): terrain and buildings from
 * the campaign's art, fog over rooms the party hasn't found, and a token for
 * every character, which glides when they move. In the side panel it follows
 * the player at a crisp zoom, like Roadwarden's view of the current area;
 * expanded, it shows the whole map.
 */
export function WorldMap({
   art,
   rooms,
   currentRoomId,
   playerName = 'You',
   playerScale = 1,
   expanded = false,
   onExpand,
   mode = expanded ? 'world' : 'panel',
   insetBottom = 0,
}: WorldMapProps) {
   const nativeW = art.width * TILE;
   const nativeH = art.height * TILE;

   const containerRef = useRef<HTMLDivElement>(null);
   const canvasRef = useRef<HTMLCanvasElement>(null);
   const [view, setView] = useState({ w: 0, h: 0 });

   useEffect(() => {
      const el = containerRef.current;
      if (!el) return;
      const measure = () => setView({ w: el.clientWidth, h: el.clientHeight });
      measure();
      if (typeof ResizeObserver !== 'function') return;
      const observer = new ResizeObserver(measure);
      observer.observe(el);
      return () => observer.disconnect();
   }, []);

   // The visible area: everything above an overlay at the bottom.
   const area = { w: view.w, h: Math.max(0, view.h - insetBottom) };
   const scale = mapScale(mode, area, nativeW, nativeH, art, currentRoomId);
   const offset = cameraOffset(art, currentRoomId, scale, area, mode);
   const inScene = (roomId: string) => mode !== 'scene' || roomId === currentRoomId;

   // Draw terrain and objects once per art.
   useEffect(() => {
      const ctx = canvasRef.current?.getContext('2d');
      if (!ctx) return;
      const sheet = new Image();
      sheet.onload = () => {
         ctx.clearRect(0, 0, nativeW, nativeH);
         drawLayer(ctx, sheet, art.ground, GROUND_TILES);
         drawLayer(ctx, sheet, art.objects, OBJECT_TILES);
      };
      sheet.src = TOWN_SHEET;
   }, [art, nativeW, nativeH]);

   const tokens = useMemo(
      () =>
         placeTokens(art, rooms, currentRoomId, playerName, playerScale).filter((t) => inScene(t.roomId)),
      // eslint-disable-next-line react-hooks/exhaustive-deps
      [art, rooms, currentRoomId, playerName, playerScale, mode],
   );

   const px = (tiles: number) => tiles * TILE * scale;

   return (
      <Box sx={{ position: 'relative', flex: 1, minHeight: 0, display: 'flex' }}>
         {onExpand && (
            <Tooltip title={expanded ? 'Shrink map' : 'Expand map'}>
               <IconButton
                  size="small"
                  aria-label={expanded ? 'shrink map' : 'expand map'}
                  onClick={onExpand}
                  sx={{ position: 'absolute', top: 4, right: 4, zIndex: 3, bgcolor: 'rgba(0,0,0,0.45)' }}
               >
                  {expanded ? <CloseFullscreenIcon fontSize="small" /> : <OpenInFullIcon fontSize="small" />}
               </IconButton>
            </Tooltip>
         )}
         <Box
            ref={containerRef}
            // Its own stacking context, so tokens and fog stay under overlays
            // (the story text box, the log) drawn over the map.
            sx={{ flex: 1, minHeight: 0, position: 'relative', overflow: 'hidden', bgcolor: '#0c0810', isolation: 'isolate' }}
         >
            <Box
               data-testid="world-map"
               sx={{
                  position: 'absolute',
                  left: offset.x,
                  top: offset.y,
                  width: nativeW * scale,
                  height: nativeH * scale,
                  // Pan smoothly when the player moves.
                  transition: 'left 0.8s ease, top 0.8s ease',
               }}
            >
               <canvas
                  ref={canvasRef}
                  width={nativeW}
                  height={nativeH}
                  style={{ width: '100%', height: '100%', imageRendering: 'pixelated', display: 'block' }}
               />

               {Object.entries(art.rooms).map(([id, [x, y, w, h]]) => {
                  const room = rooms[id];
                  const current = id === currentRoomId;
                  const discovered = room?.discovered ?? false;
                  return (
                     <Box
                        key={id}
                        data-testid={`map-room-${id}`}
                        data-discovered={discovered}
                        sx={{
                           position: 'absolute',
                           left: px(x),
                           top: px(y),
                           width: px(w),
                           height: px(h),
                           boxSizing: 'border-box',
                           bgcolor: discovered ? 'transparent' : 'rgba(12, 8, 16, 0.88)',
                           // The current room: a gold outline with a soft glow.
                           boxShadow: current && mode !== 'scene'
                              ? `inset 0 0 0 ${Math.max(2, scale)}px #f5d77a, inset 0 0 ${8 * scale}px rgba(201,169,98,0.45)`
                              : 'none',
                           transition: 'background-color 0.6s ease',
                           pointerEvents: 'none',
                        }}
                     >
                        {mode !== 'scene' && (
                        <Typography
                           sx={{
                              position: 'absolute',
                              top: 4,
                              left: '50%',
                              transform: 'translateX(-50%)',
                              px: 0.75,
                              borderRadius: 1,
                              bgcolor: 'rgba(12, 8, 16, 0.7)',
                              fontFamily: 'Cinzel, Georgia, serif',
                              fontSize: Math.min(16, Math.max(10, 6 * scale)),
                              color: current ? 'primary.main' : 'common.white',
                              whiteSpace: 'nowrap',
                           }}
                        >
                           {discovered ? (room?.name ?? id) : '?'}
                        </Typography>
                        )}
                     </Box>
                  );
               })}

               {mode === 'scene' && currentRoomId && art.rooms[currentRoomId] && (
                  // The scene stands out: a window cut out for the current room,
                  // its shadow dimming the rest of the world around it.
                  <Box
                     data-testid="scene-window"
                     sx={{
                        position: 'absolute',
                        left: px(art.rooms[currentRoomId][0]),
                        top: px(art.rooms[currentRoomId][1]),
                        width: px(art.rooms[currentRoomId][2]),
                        height: px(art.rooms[currentRoomId][3]),
                        boxShadow: '0 0 0 100vmax rgba(8, 5, 12, 0.62)',
                        pointerEvents: 'none',
                        zIndex: 1,
                     }}
                  />
               )}

               {Object.entries(art.points_of_interest).map(([poiId, [x, y]]) => {
                  const room = Object.values(rooms).find((r) =>
                     r.points_of_interest?.some((p) => p.id === poiId),
                  );
                  if (!room?.discovered || !inScene(room.id)) return null;
                  const poi = room.points_of_interest!.find((p) => p.id === poiId)!;
                  return (
                     <Tooltip key={poiId} title={poi.name}>
                        <Box
                           aria-label={poi.name}
                           sx={{
                              position: 'absolute',
                              left: px(x) + px(0.5) - 4,
                              top: px(y) - 6,
                              width: 8,
                              height: 8,
                              bgcolor: 'secondary.light',
                              border: '1px solid #000',
                              transform: 'rotate(45deg)',
                              zIndex: 1,
                           }}
                        />
                     </Tooltip>
                  );
               })}

               {tokens.map((t) => (
                  <MapToken key={t.id} token={t} scale={scale} />
               ))}
            </Box>
         </Box>
      </Box>
   );
}

function MapToken({ token, scale }: { token: Token; scale: number }) {
   // A character's own scale: an adult fills the tile, a child is smaller,
   // a giant towers over it (standing on its tile, growing upward).
   const size = TILE * scale * token.scale;
   const [sx, sy] = sheetOffset(token.sprite);
   const k = size / TILE;
   return (
      <Tooltip title={token.name}>
         <Box
            data-testid="map-token"
            aria-label={token.name}
            sx={{
               position: 'absolute',
               // Bottom-centre the sprite on its tile.
               left: token.x * TILE * scale + (TILE * scale - size) / 2,
               top: token.y * TILE * scale + (TILE * scale - size),
               width: size,
               height: size,
               backgroundImage: `url(${DUNGEON_SHEET})`,
               backgroundPosition: `-${sx * k}px -${sy * k}px`,
               backgroundSize: `${SHEET_COLUMNS * TILE * k}px auto`,
               imageRendering: 'pixelated',
               filter: token.isPlayer
                  ? 'drop-shadow(0 0 2px #f5d77a) drop-shadow(0 0 2px #f5d77a)'
                  : 'drop-shadow(0 1px 1px #000)',
               transition: 'left 0.8s ease, top 0.8s ease',
               zIndex: 2,
            }}
         />
      </Tooltip>
   );
}

/**
 * Zoom: whole-number steps in the panel (at least 2x, about FOLLOW_TILES
 * across) so pixels stay crisp; expanded, the scale that fits the map.
 */
export function mapScale(
   mode: MapMode,
   view: { w: number; h: number },
   nativeW: number,
   nativeH: number,
   art?: MapArt,
   currentRoomId?: string,
): number {
   if (!view.w || !view.h) return 2;
   if (mode === 'world') return Math.max(0.5, Math.min(view.w / nativeW, view.h / nativeH));
   const rect = currentRoomId ? art?.rooms[currentRoomId] : undefined;
   if (mode === 'scene' && rect) {
      // The biggest whole-number zoom that fits the room with a tile of
      // margin, but never so small that the map can't fill the view: a wide
      // room then overflows the view at the sides, like a camera pan.
      const fit = Math.min(view.w / ((rect[2] + 1) * TILE), view.h / ((rect[3] + 1) * TILE));
      const fill = Math.max(view.w / (art!.width * TILE), view.h / (art!.height * TILE));
      return Math.max(2, Math.ceil(fill), Math.floor(fit));
   }
   return Math.max(2, Math.round(view.w / (FOLLOW_TILES * TILE)));
}

/**
 * Where to put the map so the current room is centred, without showing past
 * the map's edges when the map is bigger than the view. Expanded, centre the
 * whole map.
 */
export function cameraOffset(
   art: MapArt,
   currentRoomId: string | undefined,
   scale: number,
   view: { w: number; h: number },
   mode: MapMode,
): { x: number; y: number } {
   const mapW = art.width * TILE * scale;
   const mapH = art.height * TILE * scale;
   const rect = currentRoomId ? art.rooms[currentRoomId] : undefined;
   const axis = (viewLen: number, mapLen: number, focus: number) => {
      if (mode === 'world' || mapLen <= viewLen || !rect) return (viewLen - mapLen) / 2;
      const ideal = viewLen / 2 - focus;
      return Math.min(0, Math.max(viewLen - mapLen, ideal));
   };
   const [x, y, w, h] = rect ?? [0, 0, art.width, art.height];
   return {
      x: axis(view.w, mapW, (x + w / 2) * TILE * scale),
      y: axis(view.h, mapH, (y + h / 2) * TILE * scale),
   };
}

function drawLayer(
   ctx: CanvasRenderingContext2D,
   sheet: HTMLImageElement,
   rows: string[],
   legend: Record<string, number>,
) {
   rows.forEach((row, y) => {
      [...row].forEach((ch, x) => {
         const index = legend[ch];
         if (index === undefined) return;
         const [sx, sy] = sheetOffset(index);
         ctx.drawImage(sheet, sx, sy, TILE, TILE, x * TILE, y * TILE, TILE, TILE);
      });
   });
}

/**
 * Where each character stands: the player and everyone in discovered rooms,
 * on free tiles (no building or tree) nearest the room's centre. Placement is
 * stable for a given set of occupants.
 */
export function placeTokens(
   art: MapArt,
   rooms: Record<string, RoomView>,
   currentRoomId: string | undefined,
   playerName: string,
   playerScale = 1,
): Token[] {
   const tokens: Token[] = [];
   for (const [roomId, rect] of Object.entries(art.rooms)) {
      const room = rooms[roomId];
      if (!room?.discovered && roomId !== currentRoomId) continue;
      const people: { id: string; name: string; isPlayer: boolean; friendly: boolean; scale: number }[] = [];
      if (roomId === currentRoomId) {
         people.push({ id: 'player', name: playerName, isPlayer: true, friendly: true, scale: playerScale });
      }
      [...(room?.occupants ?? [])]
         .sort((a, b) => a.id.localeCompare(b.id))
         .forEach((c: CharacterView) =>
            people.push({ id: c.id, name: c.name, isPlayer: false, friendly: c.friendly, scale: c.scale || 1 }),
         );
      const slots = freeTiles(art, rect);
      people.forEach((p, i) => {
         const [x, y] = slots[i % Math.max(slots.length, 1)] ?? [rect[0], rect[1]];
         const cast = art.cast[p.id];
         tokens.push({
            id: p.id,
            name: p.name,
            roomId,
            x,
            y,
            sprite:
               cast?.sprite ??
               (p.isPlayer
                  ? DEFAULT_SPRITES.player
                  : p.friendly
                    ? DEFAULT_SPRITES.friendly
                    : DEFAULT_SPRITES.hostile),
            scale: p.scale || 1,
            isPlayer: p.isPlayer,
         });
      });
   }
   return tokens;
}

/** Tiles in a room with no object on them, nearest the centre first. */
function freeTiles(art: MapArt, [x0, y0, w, h]: [number, number, number, number]): [number, number][] {
   const cx = x0 + w / 2 - 0.5;
   const cy = y0 + h / 2 - 0.5;
   const tiles: [number, number][] = [];
   // Leave the room's edge for its label and neighbours.
   for (let y = y0 + 1; y < y0 + h - 1; y++) {
      for (let x = x0 + 1; x < x0 + w - 1; x++) {
         if ((art.objects[y]?.[x] ?? ' ') === ' ') tiles.push([x, y]);
      }
   }
   // Spread people out: prefer tiles two apart, nearest the centre.
   tiles.sort((a, b) => {
      const spread = (t: [number, number]) => ((t[0] + t[1]) % 2 === 0 ? 0 : 100);
      const dist = (t: [number, number]) => (t[0] - cx) ** 2 + (t[1] - cy) ** 2;
      return spread(a) + dist(a) - (spread(b) + dist(b));
   });
   return tiles;
}
