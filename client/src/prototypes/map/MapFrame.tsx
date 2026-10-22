// Spike #118 — shared chrome for the map prototypes. Not part of the app.
import { Box, Paper, Typography } from '@mui/material';
import { useEffect, useRef, useState, type ReactNode } from 'react';
import { COLORS } from './briarwell';

/** Width of the element, so a prototype can switch to a compact rendering. */
export function useWidth<T extends HTMLElement>() {
   const ref = useRef<T>(null);
   const [width, setWidth] = useState(0);
   useEffect(() => {
      const el = ref.current;
      if (!el) return;
      const ro = new ResizeObserver(() => setWidth(el.clientWidth));
      ro.observe(el);
      setWidth(el.clientWidth);
      return () => ro.disconnect();
   }, []);
   return [ref, width] as const;
}

export interface Detail {
   title: string;
   body: string;
}

/**
 * Title bar, a map area that fills the panel, and a detail card for whatever
 * the player last clicked.
 */
export function MapFrame({
   title,
   subtitle,
   detail,
   onCloseDetail,
   children,
}: {
   title: string;
   subtitle: string;
   detail: Detail | null;
   onCloseDetail: () => void;
   children: (compact: boolean) => ReactNode;
}) {
   const [ref, width] = useWidth<HTMLDivElement>();
   const compact = width > 0 && width < 480;
   return (
      <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column', minHeight: 0 }}>
         <Box sx={{ borderBottom: `2px solid ${COLORS.gold}`, pb: 1, mb: 1, flexShrink: 0 }}>
            <Typography sx={{ fontFamily: 'Cinzel, Georgia, serif', color: COLORS.gold, fontSize: '1rem' }}>
               {title}
            </Typography>
            <Typography sx={{ fontFamily: 'Crimson Text, Georgia, serif', color: COLORS.textDim, fontSize: '0.85rem' }}>
               {subtitle}
            </Typography>
         </Box>
         <Box ref={ref} sx={{ flex: 1, minHeight: 0, position: 'relative' }}>
            {children(compact)}
            {detail && (
               <Paper
                  onClick={onCloseDetail}
                  sx={{
                     position: 'absolute',
                     left: 8,
                     right: 8,
                     bottom: 8,
                     p: 1.5,
                     backgroundColor: 'rgba(26, 15, 30, 0.94)',
                     border: `1px solid ${COLORS.gold}`,
                     cursor: 'pointer',
                  }}
               >
                  <Typography sx={{ fontFamily: 'Cinzel, Georgia, serif', color: COLORS.gold, fontSize: '0.9rem' }}>
                     {detail.title}
                  </Typography>
                  <Typography sx={{ fontFamily: 'Crimson Text, Georgia, serif', color: COLORS.text, fontSize: '0.95rem' }}>
                     {detail.body}
                  </Typography>
               </Paper>
            )}
         </Box>
      </Box>
   );
}

/** A character token: a coloured disc with initials; the player gets a glow. */
export function Token({
   x,
   y,
   r,
   label,
   color,
   player,
   title,
}: {
   x: number;
   y: number;
   r: number;
   label: string;
   color: string;
   player?: boolean;
   title: string;
}) {
   return (
      <g>
         <title>{title}</title>
         {player && (
            <circle cx={x} cy={y} r={r * 1.7} fill="none" stroke={COLORS.goldLight} strokeWidth={r * 0.18} opacity={0.6}>
               <animate attributeName="r" values={`${r * 1.3};${r * 1.9};${r * 1.3}`} dur="2s" repeatCount="indefinite" />
               <animate attributeName="opacity" values="0.7;0.1;0.7" dur="2s" repeatCount="indefinite" />
            </circle>
         )}
         <circle cx={x} cy={y} r={r} fill={color} stroke={COLORS.bg} strokeWidth={r * 0.15} />
         <text
            x={x}
            y={y + r * 0.36}
            textAnchor="middle"
            fontSize={r * 1.05}
            fontFamily="Cinzel, Georgia, serif"
            fontWeight={700}
            fill={COLORS.bg}
         >
            {label}
         </text>
      </g>
   );
}
