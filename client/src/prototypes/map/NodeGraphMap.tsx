// Spike #118 — prototype A: today's room graph, refined with points of
// interest inside each room, character tokens and highlighted exits.
import { useState } from 'react';
import {
   CHARACTERS,
   COLORS,
   DISCOVERED,
   PLAYER_ROOM,
   ROOMS,
   tokenColor,
   type Room,
   type RoomId,
} from './briarwell';
import { MapFrame, Token, type Detail } from './MapFrame';

const BOX_W = 200;
const BOX_H = 120;
const GAP = 46;
const MARGIN = 24;
const COLS = [-1, 0, 1];
const ROWS = [1, 0, -1, -2];

const W = MARGIN * 2 + COLS.length * BOX_W + (COLS.length - 1) * GAP;
const H = MARGIN * 2 + ROWS.length * BOX_H + (ROWS.length - 1) * GAP;

function boxOf(room: Room) {
   const col = COLS.indexOf(room.grid.x);
   const row = ROWS.indexOf(room.grid.y);
   return { x: MARGIN + col * (BOX_W + GAP), y: MARGIN + row * (BOX_H + GAP) };
}

function center(room: Room) {
   const b = boxOf(room);
   return { x: b.x + BOX_W / 2, y: b.y + BOX_H / 2 };
}

// Points of interest sit in a row near the bottom of the room box.
function poiPos(room: Room, i: number) {
   const b = boxOf(room);
   const n = room.pois.length;
   return { x: b.x + ((i + 1) * BOX_W) / (n + 1), y: b.y + BOX_H - 34 };
}

export function NodeGraphMap() {
   const [detail, setDetail] = useState<Detail | null>(null);
   const rooms = Object.values(ROOMS);
   const exits = new Set(Object.values(ROOMS[PLAYER_ROOM].connections));

   // One line per connection, drawn once.
   const edges: { a: Room; b: Room; fromPlayer: boolean }[] = [];
   for (const r of rooms) {
      for (const to of Object.values(r.connections)) {
         if (r.id < to!) edges.push({ a: r, b: ROOMS[to!], fromPlayer: r.id === PLAYER_ROOM || to === PLAYER_ROOM });
      }
   }

   return (
      <MapFrame
         title="A · Room graph with points of interest"
         subtitle="Evolves today's map: same rooms, more inside them"
         detail={detail}
         onCloseDetail={() => setDetail(null)}
      >
         {(compact) => (
            <svg viewBox={`0 0 ${W} ${H}`} width="100%" height="100%" preserveAspectRatio="xMidYMid meet">
               {edges.map(({ a, b, fromPlayer }) => {
                  const p = center(a);
                  const q = center(b);
                  const hidden = !DISCOVERED.has(a.id) || !DISCOVERED.has(b.id);
                  return (
                     <line
                        key={a.id + b.id}
                        x1={p.x}
                        y1={p.y}
                        x2={q.x}
                        y2={q.y}
                        stroke={fromPlayer ? COLORS.gold : COLORS.purpleDark}
                        strokeWidth={fromPlayer ? 6 : 4}
                        strokeDasharray={hidden ? '8 8' : undefined}
                        opacity={fromPlayer ? 0.9 : 0.7}
                     />
                  );
               })}

               {rooms.map((room) => {
                  const b = boxOf(room);
                  const known = DISCOVERED.has(room.id);
                  const here = room.id === PLAYER_ROOM;
                  const isExit = exits.has(room.id);
                  if (!known) {
                     return (
                        <g key={room.id} onClick={() => setDetail({ title: 'Unexplored', body: 'You have not been this way yet.' })} style={{ cursor: 'pointer' }}>
                           <rect x={b.x} y={b.y} width={BOX_W} height={BOX_H} rx={14} fill={COLORS.bg} stroke={COLORS.purpleDark} strokeDasharray="10 8" strokeWidth={3} />
                           <text x={b.x + BOX_W / 2} y={b.y + BOX_H / 2 + 14} textAnchor="middle" fontSize={42} fill={COLORS.purpleDark} fontFamily="Cinzel, Georgia, serif">?</text>
                        </g>
                     );
                  }
                  const people = CHARACTERS.filter((c) => c.room === room.id);
                  return (
                     <g key={room.id}>
                        <rect
                           x={b.x}
                           y={b.y}
                           width={BOX_W}
                           height={BOX_H}
                           rx={14}
                           fill={here ? '#2a1c33' : COLORS.paper}
                           stroke={here ? COLORS.goldLight : isExit ? COLORS.gold : COLORS.purple}
                           strokeWidth={here ? 4 : isExit ? 3 : 2}
                           onClick={() => setDetail({ title: room.name, body: room.description })}
                           style={{ cursor: 'pointer' }}
                        />
                        <text x={b.x + 12} y={b.y + 24} fontSize={compact ? 19 : 16} fill={here ? COLORS.goldLight : COLORS.gold} fontFamily="Cinzel, Georgia, serif">
                           {room.name}
                        </text>
                        {room.pois.map((poi, i) => {
                           const p = poiPos(room, i);
                           return (
                              <g key={poi.id} onClick={() => setDetail({ title: `${poi.name} · ${room.name}`, body: poi.note })} style={{ cursor: 'pointer' }}>
                                 <rect x={p.x - 6} y={p.y - 6} width={12} height={12} transform={`rotate(45 ${p.x} ${p.y})`} fill={COLORS.purpleLight} />
                                 {!compact && (
                                    <text x={p.x} y={p.y + 24} textAnchor="middle" fontSize={11.5} fill={COLORS.textDim} fontFamily="Crimson Text, Georgia, serif">
                                       {poi.name}
                                    </text>
                                 )}
                              </g>
                           );
                        })}
                        {people.map((c, i) => {
                           const at = room.pois.findIndex((p) => p.id === c.poi);
                           const base = at >= 0 ? poiPos(room, at) : center(room);
                           const sameSpot = people.slice(0, i).filter((o) => o.poi === c.poi).length;
                           return (
                              <Token
                                 key={c.id}
                                 x={base.x + 16 + sameSpot * 20}
                                 y={base.y - 26}
                                 r={compact ? 12 : 10}
                                 label={c.initials}
                                 color={tokenColor(c)}
                                 player={c.kind === 'player'}
                                 title={c.name}
                              />
                           );
                        })}
                     </g>
                  );
               })}
               {/* Direction labels on the player's exits. */}
               {Object.entries(ROOMS[PLAYER_ROOM].connections).map(([dir, to]) => {
                  const p = center(ROOMS[PLAYER_ROOM]);
                  const q = center(ROOMS[to as RoomId]);
                  return (
                     <text key={dir} x={(p.x + q.x) / 2 + 6} y={(p.y + q.y) / 2 - 6} fontSize={compact ? 16 : 13} fill={COLORS.goldLight} fontFamily="Cinzel, Georgia, serif">
                        {dir}
                     </text>
                  );
               })}
            </svg>
         )}
      </MapFrame>
   );
}
