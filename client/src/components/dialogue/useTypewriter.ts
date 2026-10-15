import { useCallback, useEffect, useRef, useState } from 'react';

/** Characters revealed per second. Readable but not sluggish. */
export const TYPE_SPEED_CPS = 40;
const TICK_MS = 25;

/**
 * Reveals `text` a few characters at a time. If the text grows (a streaming
 * line), typing continues into the new characters. A new `key` restarts the
 * reveal; `instant` shows everything at once (e.g. lines already read).
 */
export function useTypewriter(text: string, key: string, instant: boolean) {
   const [shown, setShown] = useState(instant ? text.length : 0);
   const lastKey = useRef(key);

   if (lastKey.current !== key) {
      lastKey.current = key;
      // Restart for a new beat. Setting state during render is fine here:
      // React re-renders immediately with the reset value.
      setShown(instant ? text.length : 0);
   }

   const done = shown >= text.length;
   useEffect(() => {
      if (done) return;
      const perTick = Math.max(1, Math.round((TYPE_SPEED_CPS * TICK_MS) / 1000));
      const id = setInterval(
         () => setShown((s) => Math.min(text.length, s + perTick)),
         TICK_MS,
      );
      return () => clearInterval(id);
   }, [done, text.length]);

   const complete = useCallback(() => setShown(text.length), [text.length]);

   return {
      visible: text.slice(0, shown),
      typing: shown < text.length,
      complete,
   };
}
