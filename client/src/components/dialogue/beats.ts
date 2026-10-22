import type { ChatMessage, WorldEvent } from '@/types/types';

/**
 * A beat is one unit the dialogue stage presents at a time: a scene title
 * card, or a single line of narration or speech. Player messages are not
 * beats; the player already knows what they said, so it lives in the
 * transcript.
 */
export interface Beat {
   /**
    * Stable across re-renders, across streaming → committed, and when a
    * message's events arrive after its text: "<msg>-scene" or "<msg>-L<n>".
    */
   id: string;
   kind: 'scene' | 'line';
   text: string;
   /** Set for "Speaker: line" text, e.g. "Father". */
   speaker?: string;
   /** Quest updates to announce (as toasts) when this beat starts. */
   notices?: WorldEvent[];
   /** True while the beat's message is still streaming in. */
   streaming?: boolean;
}

/** Event types announced as toasts rather than shown in the dialogue. */
export const NOTICE_EVENT_TYPES = new Set([
   'campaign_progress',
   'objective_completed',
   'objective_failed',
]);

// "Old Garrick: Sweets for the sweet." → speaker "Old Garrick". Capitalised,
// short, and followed by ": " so narration like "Later: ..." rarely matches.
const SPEAKER_LINE = /^([A-Z][A-Za-z'.-]*(?: [A-Z][A-Za-z'.-]*){0,2}): (.+)$/s;

export function parseLine(text: string): { speaker?: string; text: string } {
   const m = SPEAKER_LINE.exec(text);
   return m ? { speaker: m[1], text: m[2] } : { text };
}

function messageBeats(
   msg: ChatMessage,
   index: number,
   streaming: boolean,
): Beat[] {
   const beats: Beat[] = [];
   const events = msg.events ?? [];
   const notices = events.filter((e) => NOTICE_EVENT_TYPES.has(e.type));
   const scene = events.find((e) => e.type === 'scene_started' && e.message);
   if (scene) {
      beats.push({ id: `${index}-scene`, kind: 'scene', text: scene.message });
   }
   const lines = msg.content
      .split(/\n+/)
      .map((l) => l.trim())
      .filter(Boolean);
   lines.forEach((line, i) => {
      beats.push({
         id: `${index}-L${i}`,
         kind: 'line',
         ...parseLine(line),
         // Only the final line of a streaming message can still be growing.
         streaming: streaming && i === lines.length - 1,
      });
   });
   if (notices.length && beats.length) {
      beats[0] = { ...beats[0], notices };
   }
   return beats;
}

/**
 * Flattens the chat log (plus any in-flight stream) into beats. The
 * streaming message is numbered as the next message, so its beats keep the
 * same ids once it is committed.
 */
export function buildBeats(
   messages: ChatMessage[],
   streamingMessage?: string,
): Beat[] {
   const beats: Beat[] = [];
   messages.forEach((msg, i) => {
      if (msg.type === 'player') return;
      beats.push(...messageBeats(msg, i, false));
   });
   if (streamingMessage) {
      beats.push(
         ...messageBeats(
            { type: 'narrative', content: streamingMessage },
            messages.length,
            true,
         ),
      );
   }
   return beats;
}

/** One line of the on-stage transcript (everything that came before). */
export interface TranscriptItem {
   key: string;
   kind: 'scene' | 'line' | 'player';
   text: string;
   speaker?: string;
}

/**
 * The lines already shown before `currentBeatId`, oldest first, ending with
 * anything the player has just said. With no current beat, every line is
 * behind the player. This is the stage's scrollback.
 */
export function buildTranscript(
   messages: ChatMessage[],
   currentBeatId: string | null,
): TranscriptItem[] {
   const [msgPart, beatPart] = (currentBeatId ?? `${messages.length}-scene`).split('-');
   const curMsg = Number(msgPart);
   // Lines of the current message that are already behind the player.
   const linesBefore = beatPart === 'scene' ? -1 : Number(beatPart.slice(1));

   const items: TranscriptItem[] = [];
   const lastNarrative = messages.reduce(
      (last, m, i) => (m.type === 'narrative' ? i : last),
      -1,
   );
   messages.forEach((msg, i) => {
      const trailingPlayer = msg.type === 'player' && i > lastNarrative;
      if (i > curMsg && !trailingPlayer) return;
      if (msg.type === 'player') {
         items.push({ key: `${i}-p`, kind: 'player', text: msg.content });
         return;
      }
      const scene = msg.events?.find((e) => e.type === 'scene_started' && e.message);
      if (scene && (i < curMsg || linesBefore >= 0)) {
         items.push({ key: `${i}-scene`, kind: 'scene', text: scene.message });
      }
      msg.content
         .split(/\n+/)
         .map((l) => l.trim())
         .filter(Boolean)
         .forEach((line, n) => {
            if (i === curMsg && n >= linesBefore) return;
            items.push({ key: `${i}-L${n}`, kind: 'line', ...parseLine(line) });
         });
   });
   return items;
}
