import type { ReactNode } from 'react';

// **bold** and *italic*, as the narrator writes them. The closing marker is
// optional so text that is still typing in shows styled, not a stray "*".
// "* " (a bullet or a lone asterisk) is left alone.
const EMPHASIS = /\*\*([^*]+?)(?:\*\*|$)|\*([^*\s][^*]*?)(?:\*|$)/g;

/** Renders the narrator's inline emphasis markers as styled text. */
export function renderInline(text: string): ReactNode {
   const out: ReactNode[] = [];
   let last = 0;
   for (const m of text.matchAll(EMPHASIS)) {
      if (m.index > last) out.push(text.slice(last, m.index));
      out.push(
         m[1] !== undefined ? (
            <strong key={m.index}>{m[1]}</strong>
         ) : (
            <em key={m.index}>{m[2]}</em>
         ),
      );
      last = m.index + m[0].length;
   }
   if (last === 0) return text;
   if (last < text.length) out.push(text.slice(last));
   return out;
}
