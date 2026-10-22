// Spike #118 — prototype B: an illustrated top-down map of the village drawn
// in SVG, with clickable points of interest, character tokens and fog.
import { useState } from 'react';
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

const W = 420;
const H = 580;

// GAP: hand-placed art coordinates. The game has no geometry for rooms or POIs.
const REGION_LABEL: Record<RoomId, { x: number; y: number }> = {
   family_cottage: { x: 70, y: 222 },
   village_square: { x: 214, y: 205 },
   market_lane: { x: 150, y: 28 },
   old_mill: { x: 340, y: 205 },
   meadow: { x: 120, y: 372 },
   woods_edge: { x: 210, y: 476 },
};

const POI_AT: Record<string, { x: number; y: number }> = {
   hearth: { x: 66, y: 262 },
   cottage_door: { x: 108, y: 276 },
   garden_gate: { x: 40, y: 312 },
   well: { x: 200, y: 262 },
   benches: { x: 246, y: 300 },
   garrick_stall: { x: 246, y: 118 },
   water_wheel: { x: 384, y: 258 },
   barn: { x: 326, y: 338 },
   fence_post: { x: 168, y: 418 },
   long_grass: { x: 280, y: 426 },
   oak_hollow: { x: 210, y: 522 },
};

// Small deterministic scatter for tufts, flowers and trees.
function scatter(seed: number, n: number, x0: number, y0: number, w: number, h: number) {
   let s = seed;
   const rnd = () => ((s = (s * 9301 + 49297) % 233280) / 233280);
   return Array.from({ length: n }, () => ({ x: x0 + rnd() * w, y: y0 + rnd() * h, r: rnd() }));
}

function House({ x, y, w, h, roof }: { x: number; y: number; w: number; h: number; roof: string }) {
   return (
      <g>
         <rect x={x + 3} y={y + 4} width={w} height={h} fill="#000" opacity={0.35} rx={2} />
         <rect x={x} y={y} width={w} height={h} fill={roof} stroke="#2a1d10" strokeWidth={1.5} rx={2} />
         <line x1={x} y1={y + h / 2} x2={x + w} y2={y + h / 2} stroke="#2a1d10" strokeWidth={1.5} opacity={0.6} />
      </g>
   );
}

export function RegionMap() {
   const [detail, setDetail] = useState<Detail | null>(null);
   const exits = Object.values(ROOMS[PLAYER_ROOM].connections);
   const grass = scatter(7, 160, 0, 0, W, H);
   const flowers = scatter(11, 70, 70, 360, 280, 110);
   const trees = scatter(23, 46, -10, 470, W + 20, 120);

   // Paths from the square; the player's exits are highlighted.
   const paths: { to: RoomId; d: string }[] = [
      { to: 'family_cottage', d: 'M210 270 C170 272 140 268 110 272' },
      { to: 'market_lane', d: 'M210 230 C214 170 204 110 210 30' },
      { to: 'old_mill', d: 'M260 268 C285 262 300 258 320 256' },
      { to: 'meadow', d: 'M210 320 C204 350 214 380 210 410' },
   ];
   const woodsPath = 'M210 410 C206 440 214 470 210 505';

   return (
      <MapFrame
         title="B · Illustrated region map"
         subtitle="A drawn village with places to click"
         detail={detail}
         onCloseDetail={() => setDetail(null)}
      >
         {(compact) => (
            <svg viewBox={`0 0 ${W} ${H}`} width="100%" height="100%" preserveAspectRatio="xMidYMid meet" style={{ overflow: 'hidden' }}>
               <defs>
                  <linearGradient id="fog" x1="0" y1="0" x2="0" y2="1">
                     <stop offset="0" stopColor={COLORS.bg} stopOpacity={0} />
                     <stop offset="0.35" stopColor={COLORS.bg} stopOpacity={0.88} />
                     <stop offset="1" stopColor={COLORS.bg} stopOpacity={0.96} />
                  </linearGradient>
                  <filter id="glow">
                     <feGaussianBlur stdDeviation="3" />
                  </filter>
               </defs>

               {/* Ground */}
               <rect width={W} height={H} fill="#24301f" />
               {grass.map((g, i) => (
                  <circle key={i} cx={g.x} cy={g.y} r={1 + g.r * 1.5} fill="#1b2617" />
               ))}

               {/* River and mill race */}
               <path d="M395 0 C380 120 410 200 388 300 C370 390 405 470 390 580" stroke="#2c4d6b" strokeWidth={26} fill="none" />
               <path d="M395 0 C380 120 410 200 388 300 C370 390 405 470 390 580" stroke="#3e6b8f" strokeWidth={10} fill="none" opacity={0.6} />

               {/* Meadow */}
               <ellipse cx={210} cy={415} rx={160} ry={62} fill="#3a5530" />
               {flowers.map((f, i) => (
                  <circle key={i} cx={f.x} cy={f.y} r={1.6} fill={f.r > 0.5 ? '#e6d36b' : '#e8e2d0'} opacity={0.8} />
               ))}
               <rect x={164} y={406} width={6} height={18} fill="#6b4f2e" />

               {/* Paths */}
               {paths.map((p) => (
                  <path key={p.to} d={p.d} stroke="#8a7350" strokeWidth={14} fill="none" strokeLinecap="round" />
               ))}
               <path d={woodsPath} stroke="#8a7350" strokeWidth={10} fill="none" strokeLinecap="round" />
               {paths
                  .filter((p) => exits.includes(p.to))
                  .map((p) => (
                     <path key={`hl-${p.to}`} d={p.d} stroke={COLORS.goldLight} strokeWidth={4} fill="none" strokeDasharray="2 8" strokeLinecap="round" opacity={0.9} />
                  ))}

               {/* Village square: cobbles and the well */}
               <ellipse cx={212} cy={272} rx={66} ry={50} fill="#5f5648" stroke="#3d362c" strokeWidth={2} />
               <circle cx={200} cy={262} r={11} fill="#3d362c" stroke="#9c9182" strokeWidth={3} />
               <circle cx={200} cy={262} r={5} fill="#1c2c3c" />
               <rect x={236} y={292} width={22} height={6} fill="#6b4f2e" />
               <rect x={170} y={292} width={22} height={6} fill="#6b4f2e" />

               {/* Family cottage and garden */}
               <rect x={22} y={238} width={100} height={90} fill="#2e3b24" stroke="#6b4f2e" strokeWidth={2} strokeDasharray="4 3" />
               <House x={40} y={246} w={58} h={40} roof="#9a7b3f" />

               {/* Market lane: houses and Garrick's stall */}
               <House x={140} y={56} w={44} h={32} roof="#6a5a5a" />
               <House x={140} y={130} w={44} h={34} roof="#9a7b3f" />
               <House x={252} y={48} w={46} h={34} roof="#6a5a5a" />
               <g>
                  {[0, 1, 2, 3].map((i) => (
                     <rect key={i} x={230 + i * 8} y={106} width={8} height={16} fill={i % 2 ? '#e8e2d0' : '#9c3b3b'} />
                  ))}
               </g>

               {/* Old mill, wheel and barn */}
               <House x={310} y={236} w={56} h={44} roof="#6a5a5a" />
               <g transform="translate(384 258)">
                  <circle r={15} fill="none" stroke="#6b4f2e" strokeWidth={4} />
                  {[0, 45, 90, 135].map((a) => (
                     <line key={a} x1={-15} y1={0} x2={15} y2={0} stroke="#6b4f2e" strokeWidth={3} transform={`rotate(${a})`} />
                  ))}
               </g>
               <House x={300} y={318} w={52} h={36} roof="#7a3f2e" />

               {/* Woods' edge */}
               {trees.map((t, i) => (
                  <g key={i}>
                     <circle cx={t.x + 3} cy={t.y + 4} r={14 + t.r * 10} fill="#000" opacity={0.3} />
                     <circle cx={t.x} cy={t.y} r={14 + t.r * 10} fill={t.r > 0.5 ? '#1f3a22' : '#274a2a'} />
                  </g>
               ))}
               <circle cx={210} cy={522} r={30} fill="#2f5a33" />

               {/* Fog over undiscovered regions */}
               {!DISCOVERED.has('woods_edge') && (
                  <g onClick={() => setDetail({ title: 'Unexplored', body: 'Beyond the meadow, the trees. You have not been this way yet.' })} style={{ cursor: 'pointer' }}>
                     <rect x={0} y={448} width={W} height={H - 448} fill="url(#fog)" />
                     <text x={W / 2} y={545} textAnchor="middle" fontSize={16} fill={COLORS.textDim} fontFamily="Cinzel, Georgia, serif" letterSpacing={3}>
                        UNEXPLORED
                     </text>
                  </g>
               )}

               {/* Region labels */}
               {(Object.keys(REGION_LABEL) as RoomId[])
                  .filter((id) => DISCOVERED.has(id))
                  .map((id) => {
                     const at = REGION_LABEL[id];
                     const here = id === PLAYER_ROOM;
                     return (
                        <text
                           key={id}
                           x={at.x}
                           y={at.y}
                           textAnchor="middle"
                           fontSize={compact ? 17 : 14}
                           fill={here ? COLORS.goldLight : COLORS.gold}
                           stroke={COLORS.bg}
                           strokeWidth={4}
                           paintOrder="stroke" strokeLinejoin="round"
                           fontFamily="Cinzel, Georgia, serif"
                           onClick={() => setDetail({ title: ROOMS[id].name, body: ROOMS[id].description })}
                           style={{ cursor: 'pointer' }}
                        >
                           {ROOMS[id].name}
                        </text>
                     );
                  })}

               {/* Points of interest */}
               {Object.values(ROOMS)
                  .filter((r) => DISCOVERED.has(r.id))
                  .flatMap((r) => r.pois.map((p) => ({ room: r, poi: p })))
                  .map(({ room, poi }) => {
                     const at = POI_AT[poi.id];
                     return (
                        <g key={poi.id} onClick={() => setDetail({ title: `${poi.name} · ${room.name}`, body: poi.note })} style={{ cursor: 'pointer' }}>
                           <rect x={at.x - 5} y={at.y - 5} width={10} height={10} transform={`rotate(45 ${at.x} ${at.y})`} fill={COLORS.purpleLight} stroke={COLORS.bg} strokeWidth={1.5} />
                           {!compact && (
                              <text x={at.x} y={at.y + 19} textAnchor="middle" fontSize={10.5} fill={COLORS.text} stroke={COLORS.bg} strokeWidth={3} paintOrder="stroke" strokeLinejoin="round" fontFamily="Crimson Text, Georgia, serif">
                                 {poi.name}
                              </text>
                           )}
                        </g>
                     );
                  })}

               {/* Characters */}
               {CHARACTERS.filter((c) => DISCOVERED.has(c.room)).map((c, i, all) => {
                  const at = POI_AT[c.poi ?? ''] ?? { x: 210, y: 270 };
                  const sameSpot = all.slice(0, i).filter((o) => o.poi === c.poi).length;
                  return (
                     <Token
                        key={c.id}
                        x={at.x + 14 + sameSpot * 18}
                        y={at.y - 16}
                        r={compact ? 11 : 9}
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
