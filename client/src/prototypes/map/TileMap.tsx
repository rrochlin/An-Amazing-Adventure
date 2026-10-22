// Spike #118 — prototype C: a small top-down tile world of Briarwell with
// player and NPC tokens and fog of war over undiscovered tiles.
import { useMemo, useState } from 'react';
import {
   CHARACTERS,
   COLORS,
   DISCOVERED,
   PLAYER_ROOM,
   ROOMS,
   tokenColor,
   type RoomId,
} from './briarwell';
import { MapFrame, Token, type Detail } from './MapFrame';

const COLS = 24;
const ROWS = 30;
const T = 16;

type Tile =
   | 'grass'
   | 'path'
   | 'cobble'
   | 'water'
   | 'thatch'
   | 'slate'
   | 'redroof'
   | 'garden'
   | 'meadow'
   | 'tallgrass'
   | 'tree'
   | 'oak'
   | 'well'
   | 'stall'
   | 'wheel'
   | 'fence'
   | 'bench';

const FILL: Record<Tile, string> = {
   grass: '#2a3a22',
   path: '#8a7350',
   cobble: '#5f5648',
   water: '#2c4d6b',
   thatch: '#9a7b3f',
   slate: '#5e5866',
   redroof: '#7a3f2e',
   garden: '#3b3a24',
   meadow: '#3a5530',
   tallgrass: '#47683a',
   tree: '#1f3a22',
   oak: '#2f5a33',
   well: '#3d362c',
   stall: '#9c3b3b',
   wheel: '#6b4f2e',
   fence: '#6b4f2e',
   bench: '#6b4f2e',
};

// GAP: which tiles belong to which room. The game's rooms have no extent.
function regionOf(x: number, y: number): RoomId {
   if (y >= 23) return 'woods_edge';
   if (y >= 16) return 'meadow';
   if (y <= 7) return 'market_lane';
   if (x <= 7) return 'family_cottage';
   if (x >= 16) return 'old_mill';
   return 'village_square';
}

// GAP: tile positions of points of interest and characters.
const POI_TILE: Record<string, { x: number; y: number }> = {
   hearth: { x: 3, y: 11 },
   cottage_door: { x: 6, y: 11 },
   garden_gate: { x: 0, y: 14 },
   well: { x: 9, y: 9 },
   benches: { x: 14, y: 13 },
   garrick_stall: { x: 14, y: 4 },
   water_wheel: { x: 20, y: 9 },
   barn: { x: 17, y: 14 },
   fence_post: { x: 9, y: 19 },
   long_grass: { x: 16, y: 19 },
   oak_hollow: { x: 11, y: 25 },
};

function buildTiles(): Tile[][] {
   const t: Tile[][] = Array.from({ length: ROWS }, () => Array<Tile>(COLS).fill('grass'));
   const fill = (x0: number, y0: number, x1: number, y1: number, tile: Tile) => {
      for (let y = y0; y <= y1; y++) for (let x = x0; x <= x1; x++) if (t[y]?.[x]) t[y][x] = tile;
   };
   // Meadow and woods
   fill(1, 16, 20, 22, 'meadow');
   [[4, 17], [5, 18], [15, 18], [16, 19], [17, 18], [6, 21], [14, 21], [15, 21]].forEach(([x, y]) => (t[y][x] = 'tallgrass'));
   fill(0, 23, 23, 29, 'tree');
   fill(9, 24, 14, 27, 'grass');
   fill(10, 25, 12, 26, 'oak');
   for (let x = 2; x <= 19; x += 3) t[16][x] = 'fence';
   t[19][9] = 'fence';
   // River down the east side
   for (let y = 0; y < ROWS; y++) {
      const x = 21 + (y % 7 < 3 ? 0 : 1);
      t[y][x] = 'water';
      t[y][x + 1] = 'water';
   }
   // Paths
   fill(11, 0, 12, 25, 'path');
   fill(3, 11, 20, 12, 'path');
   // Square
   fill(8, 8, 15, 15, 'cobble');
   fill(11, 8, 12, 15, 'cobble');
   t[9][9] = 'well';
   t[13][14] = 'bench';
   t[13][8] = 'bench';
   // Cottage and garden
   fill(0, 8, 7, 15, 'garden');
   fill(1, 9, 5, 13, 'thatch');
   fill(3, 11, 7, 12, 'path');
   t[11][3] = 'thatch';
   // Market lane houses and stall
   fill(4, 1, 8, 4, 'slate');
   fill(4, 6, 8, 7, 'thatch');
   fill(15, 1, 19, 3, 'slate');
   fill(13, 4, 15, 4, 'stall');
   // Mill, wheel and barn
   fill(16, 8, 19, 10, 'slate');
   t[9][20] = 'wheel';
   t[10][20] = 'wheel';
   fill(16, 13, 19, 15, 'redroof');
   return t;
}

export function TileMap() {
   const [detail, setDetail] = useState<Detail | null>(null);
   const tiles = useMemo(buildTiles, []);
   const exits = new Set<RoomId>(Object.values(ROOMS[PLAYER_ROOM].connections) as RoomId[]);

   const W = COLS * T;
   const H = ROWS * T;

   return (
      <MapFrame
         title="C · Tile world"
         subtitle="A small top-down world, tile by tile"
         detail={detail}
         onCloseDetail={() => setDetail(null)}
      >
         {(compact) => (
            <svg viewBox={`0 0 ${W} ${H}`} width="100%" height="100%" preserveAspectRatio="xMidYMid meet" shapeRendering="crispEdges">
               {tiles.map((row, y) =>
                  row.map((tile, x) => {
                     const region = regionOf(x, y);
                     return (
                        <g key={`${x}-${y}`} onClick={() => setDetail({ title: ROOMS[region].name, body: DISCOVERED.has(region) ? ROOMS[region].description : 'You have not been this way yet.' })}>
                           <rect x={x * T} y={y * T} width={T} height={T} fill={FILL[tile]} />
                           {tile === 'tree' && <circle cx={x * T + 8} cy={y * T + 8} r={7} fill="#173019" shapeRendering="auto" />}
                           {tile === 'water' && (x + y) % 3 === 0 && <rect x={x * T + 3} y={y * T + 7} width={8} height={2} fill="#3e6b8f" />}
                           {tile === 'well' && <circle cx={x * T + 8} cy={y * T + 8} r={5} fill="#1c2c3c" stroke="#9c9182" strokeWidth={2} shapeRendering="auto" />}
                           {tile === 'stall' && x % 2 === 0 && <rect x={x * T} y={y * T} width={T / 2} height={T} fill="#e8e2d0" />}
                           {tile === 'meadow' && (x * 7 + y * 3) % 5 === 0 && <circle cx={x * T + 5} cy={y * T + 6} r={1.5} fill="#e6d36b" shapeRendering="auto" />}
                           {tile === 'fence' && <rect x={x * T + 6} y={y * T + 2} width={4} height={12} fill="#3a2a16" />}
                           {tile === 'wheel' && <circle cx={x * T + 8} cy={y * T + 8} r={6} fill="none" stroke="#3a2a16" strokeWidth={2} shapeRendering="auto" />}
                        </g>
                     );
                  }),
               )}

               {/* Outline the player's room and its exits. */}
               {(['village_square', ...exits] as RoomId[]).map((id) => {
                  const cells: { x: number; y: number }[] = [];
                  for (let y = 0; y < ROWS; y++) for (let x = 0; x < COLS; x++) if (regionOf(x, y) === id) cells.push({ x, y });
                  const xs = cells.map((c) => c.x);
                  const ys = cells.map((c) => c.y);
                  const here = id === PLAYER_ROOM;
                  return (
                     <rect
                        key={id}
                        x={Math.min(...xs) * T + 2}
                        y={Math.min(...ys) * T + 2}
                        width={(Math.max(...xs) - Math.min(...xs) + 1) * T - 4}
                        height={(Math.max(...ys) - Math.min(...ys) + 1) * T - 4}
                        fill="none"
                        stroke={here ? COLORS.goldLight : COLORS.gold}
                        strokeWidth={here ? 3 : 1.5}
                        strokeDasharray={here ? undefined : '6 6'}
                        opacity={here ? 0.95 : 0.6}
                        pointerEvents="none"
                     />
                  );
               })}

               {/* Fog of war */}
               {tiles.map((row, y) =>
                  row.map((_, x) =>
                     DISCOVERED.has(regionOf(x, y)) ? null : (
                        <rect key={`f-${x}-${y}`} x={x * T} y={y * T} width={T} height={T} fill={COLORS.bg} opacity={y === 23 ? 0.7 : 0.93} pointerEvents="none" />
                     ),
                  ),
               )}

               {/* Points of interest */}
               {Object.values(ROOMS)
                  .filter((r) => DISCOVERED.has(r.id))
                  .flatMap((r) => r.pois.map((p) => ({ room: r, poi: p })))
                  .map(({ room, poi }) => {
                     const at = POI_TILE[poi.id];
                     const cx = at.x * T + T / 2;
                     const cy = at.y * T + T / 2;
                     return (
                        <g key={poi.id} onClick={() => setDetail({ title: `${poi.name} · ${room.name}`, body: poi.note })} style={{ cursor: 'pointer' }}>
                           <rect x={cx - 3} y={cy - 3} width={6} height={6} transform={`rotate(45 ${cx} ${cy})`} fill={COLORS.purpleLight} stroke={COLORS.bg} strokeWidth={1} shapeRendering="auto" />
                           {!compact && (
                              <text x={cx} y={cy + 15} textAnchor="middle" fontSize={8.5} fill={COLORS.text} stroke={COLORS.bg} strokeWidth={2.5} paintOrder="stroke" strokeLinejoin="round" fontFamily="Crimson Text, Georgia, serif" shapeRendering="auto">
                                 {poi.name}
                              </text>
                           )}
                        </g>
                     );
                  })}

               {/* Region names */}
               {(Object.keys(ROOMS) as RoomId[])
                  .filter((id) => DISCOVERED.has(id))
                  .map((id) => {
                     const label: Record<RoomId, [number, number]> = {
                        family_cottage: [4, 8.4],
                        village_square: [12, 8.4],
                        market_lane: [11.5, 0.9],
                        old_mill: [19.5, 7.5],
                        meadow: [5.5, 16.2],
                        woods_edge: [12, 23.5],
                     };
                     const [lx, ly] = label[id];
                     return (
                        <text key={id} x={lx * T} y={ly * T} textAnchor="middle" fontSize={compact ? 13 : 10.5} fill={id === PLAYER_ROOM ? COLORS.goldLight : COLORS.gold} stroke={COLORS.bg} strokeWidth={3} paintOrder="stroke" strokeLinejoin="round" fontFamily="Cinzel, Georgia, serif" shapeRendering="auto">
                           {ROOMS[id].name}
                        </text>
                     );
                  })}

               {/* Characters */}
               {CHARACTERS.filter((c) => DISCOVERED.has(c.room)).map((c, i, all) => {
                  const at = POI_TILE[c.poi ?? ''] ?? { x: 12, y: 12 };
                  const sameSpot = all.slice(0, i).filter((o) => o.poi === c.poi).length;
                  return (
                     <Token
                        key={c.id}
                        x={(at.x + 1 + sameSpot) * T + T / 2}
                        y={(at.y - 0.4) * T}
                        r={compact ? 8.5 : 7}
                        label={c.initials}
                        color={tokenColor(c)}
                        player={c.kind === 'player'}
                        title={c.name}
                     />
                  );
               })}
            </svg>
         )}
      </MapFrame>
   );
}
