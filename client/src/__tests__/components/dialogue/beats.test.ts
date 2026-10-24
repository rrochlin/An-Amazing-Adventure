import { describe, it, expect } from 'vitest';
import { buildBeats, buildTranscript, parseLine } from '@/components/dialogue/beats';
import type { ChatMessage } from '@/types/types';

describe('parseLine', () => {
   it('splits "Speaker: line"', () => {
      expect(parseLine("Old Garrick: Sweets for the sweet.")).toEqual({
         speaker: 'Old Garrick',
         text: 'Sweets for the sweet.',
      });
   });
   it('drops quotes around a whole spoken line', () => {
      expect(parseLine('Bess: "It\'s Molly. My doll."')).toEqual({ speaker: 'Bess', text: "It's Molly. My doll." });
      expect(parseLine('Bess: “Gone,” she says, “gone.”').text).toBe('“Gone,” she says, “gone.”');
   });
   it('leaves narration alone', () => {
      expect(parseLine('Father ruffles your hair.')).toEqual({
         text: 'Father ruffles your hair.',
      });
      expect(parseLine('"Easy," he says: "the bandits are gone."').speaker).toBeUndefined();
   });
});

describe('buildBeats', () => {
   const scene: ChatMessage = {
      type: 'narrative',
      content: 'Outside, Briarwell is waking up.\nFather: Off you go.',
      events: [
         { type: 'scene_started', message: 'Good Turns and Bad' },
         { type: 'campaign_progress', message: 'Objective updated: Earn two coins.' },
         { type: 'item_gained', message: 'You gained a coin.' },
      ],
   };

   it('puts the title card first, then one beat per line', () => {
      const beats = buildBeats([scene]);
      expect(beats.map((b) => [b.kind, b.speaker, b.text])).toEqual([
         ['scene', undefined, 'Good Turns and Bad'],
         ['line', undefined, 'Outside, Briarwell is waking up.'],
         ['line', 'Father', 'Off you go.'],
      ]);
      // Quest updates ride on the first beat; other world events don't toast.
      expect(beats[0].notices?.map((n) => n.type)).toEqual(['campaign_progress']);
   });

   it('skips player messages', () => {
      const beats = buildBeats([{ type: 'player', content: 'I look around' }, scene]);
      expect(beats.every((b) => !b.text.includes('look around'))).toBe(true);
   });

   it('keeps line ids stable when events arrive after the text', () => {
      const bare = { ...scene, events: undefined };
      const before = buildBeats([bare]).map((b) => b.id);
      const after = buildBeats([scene]).map((b) => b.id);
      expect(after.slice(1)).toEqual(before);
   });

   it('keeps ids stable from streaming to committed', () => {
      const streaming = buildBeats([], 'First paragraph.\n\nSecond, still arri');
      expect(streaming.map((b) => [b.id, b.streaming])).toEqual([
         ['0-L0', false],
         ['0-L1', true],
      ]);
      const committed = buildBeats([
         { type: 'narrative', content: 'First paragraph.\n\nSecond, still arriving.' },
      ]);
      expect(committed.map((b) => b.id)).toEqual(['0-L0', '0-L1']);
   });
});

describe('buildTranscript', () => {
   const msgs: ChatMessage[] = [
      { type: 'narrative', content: 'Sunlight spills in.\nFather: Off you go.' },
      { type: 'player', content: 'I head to the square' },
      {
         type: 'narrative',
         content: 'The square bustles.\nBess is crying by the well.',
         events: [{ type: 'scene_started', message: 'Good Turns and Bad' }],
      },
   ];

   it('shows what came before the current line, with the player\'s words', () => {
      const items = buildTranscript(msgs, '2-L1');
      expect(items.map((i) => [i.kind, i.speaker, i.text])).toEqual([
         ['line', undefined, 'Sunlight spills in.'],
         ['line', 'Father', 'Off you go.'],
         ['player', undefined, 'I head to the square'],
         ['scene', undefined, 'Good Turns and Bad'],
         ['line', undefined, 'The square bustles.'],
      ]);
   });

   it('leaves out the scene card while it is the current beat', () => {
      const items = buildTranscript(msgs, '2-scene');
      expect(items.map((i) => i.kind)).toEqual(['line', 'line', 'player']);
   });

   it('includes what the player just said before the story answers', () => {
      const withReply = [...msgs, { type: 'player' as const, content: 'I help Bess' }];
      const items = buildTranscript(withReply, '2-L1');
      expect(items[items.length - 1]).toMatchObject({ kind: 'player', text: 'I help Bess' });
   });

   it('includes every line when nothing is current', () => {
      const withReply = [...msgs, { type: 'player' as const, content: 'I help Bess' }];
      expect(buildTranscript(withReply, null).map((i) => i.text)).toEqual([
         'Sunlight spills in.',
         'Off you go.',
         'I head to the square',
         'Good Turns and Bad',
         'The square bustles.',
         'Bess is crying by the well.',
         'I help Bess',
      ]);
   });
});

describe('staging cues', () => {
   it('ride on the line they play with, the last if past the end', () => {
      const msgs: ChatMessage[] = [
         { type: 'player', content: 'Hello' },
         {
            type: 'narrative',
            content: 'Bess looks up.\n\nLittle Bess: You came!',
            events: [
               { type: 'cue', message: '', character: 'bess', emote: 'hop', line: 1 },
               { type: 'cue', message: '', character: 'bess', mark: 'stone_well' },
               { type: 'cue', message: '', character: 'player', emote: 'bow', line: 9 },
            ],
         },
      ];
      const beats = buildBeats(msgs);
      expect(beats.map((b) => b.cues?.map((c) => c.character + ':' + (c.mark ?? c.emote)))).toEqual([
         ['bess:stone_well'],
         ['bess:hop', 'player:bow'],
      ]);
      expect(beats[1].cues![0].id).toBe('1-C0');
   });
});

describe('mechanics on the stage', () => {
   it('shows what happened to someone over them on the last line', () => {
      const beats = buildBeats([
         { type: 'player', content: 'I give her the doll' },
         {
            type: 'narrative',
            content: 'Bess hugs the doll.\nLittle Bess: Thank you!',
            events: [
               { type: 'item_lost', message: "Bess's rag doll removed", character: 'player', item: "Bess's rag doll", amount: 1 },
               { type: 'item_gained', message: '2 × copper coin added', character: 'player', item: 'copper coin', amount: 2 },
               { type: 'damage', message: 'Dunstan takes 2', character: 'bully', amount: 2 },
               { type: 'objective_completed', message: 'Objective complete' },
            ],
         },
      ]);
      expect(beats[0].cues).toBeUndefined();
      expect(beats[1].cues?.map((c) => [c.character, c.pop, c.tone])).toEqual([
         ['player', "− Bess's rag doll", 'loss'],
         ['player', '+ 2 × copper coin', 'gain'],
         ['bully', '−2 ❤', 'harm'],
      ]);
   });

   it('shows nothing for old events that carry no details', () => {
      const beats = buildBeats([{ type: 'narrative', content: 'Ouch.', events: [{ type: 'damage', message: 'You take 2 damage.' }] }]);
      expect(beats[0].cues).toBeUndefined();
   });
});

describe('walking up to someone', () => {
   it('keeps who the cue walks toward', () => {
      const beats = buildBeats([{ type: 'narrative', content: 'You stride over.', events: [{ type: 'cue', message: '', character: 'player', toward: 'bully' }] }]);
      expect(beats[0].cues?.[0]).toMatchObject({ character: 'player', toward: 'bully' });
   });
});
