import { Box } from '@mui/material';
import { useEffect, useRef } from 'react';
import type { StageScene, StageView } from './StageScene';

/**
 * The scene, drawn by Phaser. Phaser is loaded on demand (it's large), and
 * every change React makes is handed to the running scene.
 */
export function PhaserStage(props: StageView) {
   const host = useRef<HTMLDivElement>(null);
   const scene = useRef<StageScene | null>(null);
   const latest = useRef(props);
   latest.current = props;

   useEffect(() => {
      let cancelled = false;
      let game: { destroy: (removeCanvas: boolean) => void } | undefined;
      import('./StageScene').then(({ createStage }) => {
         if (cancelled || !host.current) return;
         const started = createStage(host.current, latest.current);
         game = started.game;
         scene.current = started.scene;
      });
      return () => {
         cancelled = true;
         scene.current = null;
         game?.destroy(true);
      };
   }, []);

   useEffect(() => {
      scene.current?.setView(props);
   });

   return (
      <Box
         ref={host}
         data-testid="phaser-stage"
         sx={{ flex: 1, minWidth: 0, minHeight: 0, position: 'relative', overflow: 'hidden', '& canvas': { display: 'block' } }}
      />
   );
}
