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
   type TileRef,
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
   /** Pixels at the top covered by an overlay (the location sign). */
   insetTop?: number;
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
 * Tiles drawn beyond the map's edge in a scene, repeating the edge tiles, so
 * a room at the edge of the map is centred against more of the same terrain
 * rather than a blank. Enough for a 2x zoom on a wide screen.
 */
const SCENE_PAD = 32;

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
   insetTop = 0,
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

   // The visible area: between the overlays at the top and the bottom.
   const area = { w: view.w, h: Math.max(0, view.h - insetTop - insetBottom) };
   // Nothing is shown until the view has been measured, so the map and its
   // characters first appear where they belong instead of flying in from a
   // guessed layout.
   const ready = view.w > 0 && view.h > 0;
   const scale = mapScale(mode, area, nativeW, nativeH, art, currentRoomId);
   const centred = cameraOffset(art, currentRoomId, scale, area, mode);
   const offset = { x: centred.x, y: centred.y + insetTop };
   const inScene = (roomId: string) => mode !== 'scene' || roomId === currentRoomId;
   // The camera glides only when the player changes room; a resize, a new
   // zoom or the text panel changing size moves it at once.
   const gliding = useGlide(ready ? currentRoomId : undefined);

   // Draw terrain and objects once per art (padded in a scene).
   // An interior stands alone in the dark: nothing continues past its walls.
   const pad = mode === 'scene' && !art.interior ? SCENE_PAD : 0;
   useEffect(() => {
      const ctx = canvasRef.current?.getContext('2d');
      if (!ctx) return;
      let cancelled = false;
      const load = (src: string) =>
         new Promise<HTMLImageElement>((resolve) => {
            const img = new Image();
            img.onload = () => resolve(img);
            img.src = src;
         });
      Promise.all([load(TOWN_SHEET), load(DUNGEON_SHEET)]).then(([town, dungeon]) => {
         if (cancelled) return;
         const sheets = { town, dungeon };
         ctx.clearRect(0, 0, (art.width + 2 * pad) * TILE, (art.height + 2 * pad) * TILE);
         drawLayer(ctx, sheets, art.ground, GROUND_TILES, pad);
         drawLayer(ctx, sheets, art.objects, OBJECT_TILES, pad, true);
      });
      return () => {
         cancelled = true;
      };
   }, [art, pad]);

   // Characters keep their tiles from one update to the next.
   const placed = useRef<Token[]>([]);
   const tokens = useMemo(() => {
      placed.current = placeTokens(art, rooms, currentRoomId, playerName, playerScale, placed.current);
      return placed.current.filter((t) => inScene(t.roomId));
      // eslint-disable-next-line react-hooks/exhaustive-deps
   }, [art, rooms, currentRoomId, playerName, playerScale, mode]);

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
            sx={{
               flex: 1,
               minHeight: 0,
               position: 'relative',
               overflow: 'hidden',
               bgcolor: '#0c0810',
               isolation: 'isolate',
            }}
         >
            <Box
               data-testid="world-map"
               sx={{
                  position: 'absolute',
                  left: offset.x,
                  top: offset.y,
                  width: nativeW * scale,
                  height: nativeH * scale,
                  visibility: ready ? 'visible' : 'hidden',
                  // Pan smoothly when the player moves.
                  transition: gliding ? 'left 0.8s ease, top 0.8s ease' : 'none',
               }}
            >
               <canvas
                  ref={canvasRef}
                  width={nativeW + 2 * pad * TILE}
                  height={nativeH + 2 * pad * TILE}
                  style={{
                     position: 'absolute',
                     left: -pad * TILE * scale,
                     top: -pad * TILE * scale,
                     width: (nativeW + 2 * pad * TILE) * scale,
                     height: (nativeH + 2 * pad * TILE) * scale,
                     imageRendering: 'pixelated',
                     display: 'block',
                  }}
               />

               {mode !== 'scene' && (
                  // One fog over the whole map with a hole for each room the
                  // party has found: ground between rooms (or past them) is
                  // fogged too, so only found places show.
                  <svg
                     data-testid="map-fog"
                     width={px(art.width)}
                     height={px(art.height)}
                     style={{ position: 'absolute', left: 0, top: 0, pointerEvents: 'none' }}
                  >
                     <path
                        fillRule="evenodd"
                        fill="rgba(12, 8, 16, 0.88)"
                        d={fogPath(art, rooms, currentRoomId, px)}
                     />
                  </svg>
               )}

               {Object.entries(art.rooms).map(([id, [x, y, w, h]]) => {
                  const room = rooms[id];
                  const current = id === currentRoomId;
                  // In a scene the shade outside the room does the work; fog
                  // on top of it turned undiscovered rooms black beside plain
                  // dimmed field. The world map still shows what's unfound.
                  const discovered = mode === 'scene' || (room?.discovered ?? false);
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
                           // The current room: a gold outline with a soft glow.
                           boxShadow: current && mode !== 'scene'
                              ? `inset 0 0 0 ${Math.max(2, scale)}px #f5d77a, inset 0 0 ${8 * scale}px rgba(201,169,98,0.45)`
                              : 'none',
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
                        // Soft-edged shade over everything outside the room.
                        boxShadow: '0 0 36px 100vmax rgba(8, 5, 12, 0.5)',
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

/** How long a camera pan or a character's walk takes. */
const GLIDE_MS = 800;

/**
 * True while `key` has just changed (for GLIDE_MS): the change animates, and
 * every other change to the same element lands at once. Its first value
 * never glides.
 */
function useGlide(key: string | undefined): boolean {
   const last = useRef(key);
   const until = useRef(0);
   if (key !== last.current) {
      if (last.current !== undefined && key !== undefined) until.current = Date.now() + GLIDE_MS;
      last.current = key;
   }
   return Date.now() < until.current;
}

function MapToken({ token, scale }: { token: Token; scale: number }) {
   // A character walks when their tile changes, never when the view zooms.
   const gliding = useGlide(`${token.x},${token.y}`);
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
               transition: gliding ? 'left 0.8s ease, top 0.8s ease' : 'none',
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
      // An interior is fitted whole, with the dark around it.
      if (art!.interior) return Math.max(2, Math.floor(fit));
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
      // A scene is always centred on its room, even at the map's edge; the
      // container's field colour fills what lies beyond the map.
      if (mode === 'scene' && rect) return viewLen / 2 - focus;
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

/**
 * The fog's outline: the whole map, then a hole (evenodd) for each found
 * room and the one the player stands in.
 */
export function fogPath(
   art: MapArt,
   rooms: Record<string, RoomView>,
   currentRoomId: string | undefined,
   px: (tiles: number) => number,
): string {
   const holes = Object.entries(art.rooms)
      .filter(([id]) => id === currentRoomId || rooms[id]?.discovered)
      .map(([, [x, y, w, h]]) => `M${px(x)} ${px(y)}h${px(w)}v${px(h)}h${-px(w)}Z`);
   return [`M0 0H${px(art.width)}V${px(art.height)}H0Z`, ...holes].join(' ');
}

/** Object characters that continue beyond the map's edge: forest. */
const CONTINUES = new Set('123456789');

/**
 * Draws a layer, with `pad` tiles of border on every side that repeat the
 * map's edge tiles (the nearest cell inside the map). Ground always
 * continues; of the objects, only forest does (a beehive at the edge
 * doesn't become a row of beehives).
 */
function drawLayer(
   ctx: CanvasRenderingContext2D,
   sheets: { town: HTMLImageElement; dungeon: HTMLImageElement },
   rows: string[],
   legend: Record<string, TileRef>,
   pad = 0,
   objects = false,
) {
   const h = rows.length;
   const w = rows[0]?.length ?? 0;
   const clamp = (v: number, max: number) => Math.min(Math.max(v, 0), max - 1);
   for (let y = -pad; y < h + pad; y++) {
      for (let x = -pad; x < w + pad; x++) {
         const outside = y < 0 || y >= h || x < 0 || x >= w;
         const ch = rows[clamp(y, h)]?.[clamp(x, w)];
         if (ch === undefined || (outside && objects && !CONTINUES.has(ch))) continue;
         const ref = legend[ch];
         if (ref === undefined) continue;
         const [sheet, index] = typeof ref === 'number' ? [sheets.town, ref] : [sheets.dungeon, ref.dungeon];
         if (index < 0) continue;
         const [sx, sy] = sheetOffset(index);
         ctx.drawImage(sheet, sx, sy, TILE, TILE, (x + pad) * TILE, (y + pad) * TILE, TILE, TILE);
      }
   }
}

/**
 * Where each character stands: the player and everyone in discovered rooms,
 * on free tiles (no building or tree) nearest the room's centre. Someone who
 * was already in the room (in `previous`) keeps their tile; newcomers take
 * the best free ones, so an arrival never shuffles the others.
 */
export function placeTokens(
   art: MapArt,
   rooms: Record<string, RoomView>,
   currentRoomId: string | undefined,
   playerName: string,
   playerScale = 1,
   previous: Token[] = [],
   marks: Record<string, string> = {},
): Token[] {
   const tokens: Token[] = [];
   for (const [roomId, rect] of Object.entries(art.rooms)) {
      const room = rooms[roomId];
      if (!room?.discovered && roomId !== currentRoomId) continue;
      const people: { id: string; name: string; isPlayer: boolean; friendly: boolean; scale: number; mark?: string }[] = [];
      if (roomId === currentRoomId) {
         people.push({ id: 'player', name: playerName, isPlayer: true, friendly: true, scale: playerScale, mark: marks.player });
      }
      [...(room?.occupants ?? [])]
         .sort((a, b) => a.id.localeCompare(b.id))
         .forEach((c: CharacterView) =>
            people.push({ id: c.id, name: c.name, isPlayer: false, friendly: c.friendly, scale: c.scale || 1, mark: marks[c.id] ?? c.mark }),
         );
      const slots = freeTiles(art, rect);
      const key = ([x, y]: [number, number]) => `${x},${y}`;
      const open = new Set(slots.map(key));
      const tile = new Map<string, [number, number]>();
      // Someone at a mark stands beside it, before anyone else is placed.
      const taken = new Set<string>();
      for (const p of people) {
         const poi = p.mark ? art.points_of_interest[p.mark] : undefined;
         // Already beside it: they stay put (no shuffling round the well).
         const was = previous.find((t) => t.id === p.id && t.roomId === roomId);
         const besideIt = poi && was && Math.abs(was.x - poi[0]) <= 1 && Math.abs(was.y - poi[1]) <= 1 && !taken.has(key([was.x, was.y]));
         const spot = besideIt ? ([was.x, was.y] as [number, number]) : poi && markTile(art, rect, poi, taken);
         if (!spot) continue;
         tile.set(p.id, spot);
         taken.add(key(spot));
         open.delete(key(spot));
      }
      for (const p of people) {
         if (tile.has(p.id)) continue;
         const was = previous.find((t) => t.id === p.id && t.roomId === roomId);
         if (was && open.has(key([was.x, was.y]))) {
            tile.set(p.id, [was.x, was.y]);
            open.delete(key([was.x, was.y]));
         }
      }
      people.forEach((p, i) => {
         let spot = tile.get(p.id);
         if (!spot) {
            spot = slots.find((s) => open.has(key(s))) ?? slots[i % Math.max(slots.length, 1)];
            if (spot) open.delete(key(spot));
         }
         const [x, y] = spot ?? [rect[0], rect[1]];
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

/**
 * Where someone at a mark stands: the free tile in the room nearest the
 * point of interest (the well itself is an object), in front of it on a tie.
 * None when the mark isn't in this room.
 */
export function markTile(
   art: MapArt,
   [x0, y0, w, h]: [number, number, number, number],
   [px, py]: [number, number],
   taken: Set<string>,
): [number, number] | undefined {
   if (px < x0 - 1 || px > x0 + w || py < y0 - 1 || py > y0 + h) return undefined;
   let best: [number, number] | undefined;
   let bestScore = Infinity;
   for (let y = y0; y < y0 + h; y++) {
      for (let x = x0; x < x0 + w; x++) {
         if ((art.objects[y]?.[x] ?? ' ') !== ' ' || art.ground[y]?.[x] === '_' || taken.has(`${x},${y}`)) continue;
         if (x === px && y === py) continue;
         // Distance, then in front (below) of it, then left to right.
         const score = Math.abs(x - px) + Math.abs(y - py) + (y > py ? 0 : 0.5) + (x - px) * 0.01;
         if (score < bestScore) {
            bestScore = score;
            best = [x, y];
         }
      }
   }
   return best;
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
