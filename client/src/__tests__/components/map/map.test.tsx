// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { render, screen } from '@testing-library/react';
import { ThemeProvider } from '@mui/material/styles';
import { AppTheme } from '@/theme/theme';
import { BRIARWELL_ART } from '@/components/map/art/briarwell';
import { GROUND_TILES, OBJECT_TILES, mapArtFor } from '@/components/map/mapArt';
import { WorldMap, cameraOffset, mapScale, placeTokens } from '@/components/map/WorldMap';
import type { CharacterView, RoomView } from '@/types/types';

interface CampaignJSON {
   map: { rooms: Record<string, { name: string; points_of_interest?: { id: string }[] }> };
   actors: Record<string, unknown>;
}
const campaign: CampaignJSON = JSON.parse(
   readFileSync(resolve(__dirname, '../../../../../server/campaigns/briarwell/campaign.json'), 'utf8'),
);

describe('Briarwell map art', () => {
   const art = BRIARWELL_ART;

   it('is found by campaign id', () => {
      expect(mapArtFor('briarwell')).toBe(art);
      expect(mapArtFor('test')).toBeUndefined();
   });

   it('has full layers using only known tiles', () => {
      for (const [layer, legend] of [
         [art.ground, GROUND_TILES],
         [art.objects, { ...OBJECT_TILES, ' ': 0 }],
      ] as const) {
         expect(layer).toHaveLength(art.height);
         layer.forEach((row) => {
            expect(row).toHaveLength(art.width);
            for (const ch of row) expect(legend, `unknown tile "${ch}"`).toHaveProperty(ch);
         });
      }
   });

   it('covers exactly the campaign rooms, inside the map and without overlap', () => {
      expect(Object.keys(art.rooms).sort()).toEqual(Object.keys(campaign.map.rooms).sort());
      const taken = new Set<string>();
      for (const [id, [x, y, w, h]] of Object.entries(art.rooms)) {
         expect(x >= 0 && y >= 0 && x + w <= art.width && y + h <= art.height, id).toBe(true);
         for (let ty = y; ty < y + h; ty++)
            for (let tx = x; tx < x + w; tx++) {
               expect(taken.has(`${tx},${ty}`), `${id} overlaps at ${tx},${ty}`).toBe(false);
               taken.add(`${tx},${ty}`);
            }
      }
   });

   it("places every campaign point of interest inside its own room", () => {
      const want = Object.entries(campaign.map.rooms).flatMap(([roomId, r]) =>
         (r.points_of_interest ?? []).map((p) => [p.id, roomId] as const),
      );
      expect(Object.keys(art.points_of_interest).sort()).toEqual(want.map(([id]) => id).sort());
      for (const [poiId, roomId] of want) {
         const [px, py] = art.points_of_interest[poiId];
         const [x, y, w, h] = art.rooms[roomId];
         expect(px >= x && px < x + w && py >= y && py < y + h, `${poiId} outside ${roomId}`).toBe(true);
      }
   });

   it('casts only characters the campaign has', () => {
      for (const id of Object.keys(art.cast)) {
         if (id !== 'player') expect(campaign.actors, id).toHaveProperty(id);
      }
   });
});

const person = (id: string, name: string, friendly = true): CharacterView =>
   ({ id, name, friendly }) as CharacterView;
const room = (id: string, discovered: boolean, occupants: CharacterView[] = []): RoomView =>
   ({ id, name: id, discovered, occupants, points_of_interest: [] }) as unknown as RoomView;

const rooms: Record<string, RoomView> = {
   village_square: { ...room('village_square', true, [person('bess', 'Little Bess')]), name: 'Village Square',
      points_of_interest: [{ id: 'stone_well', name: 'Stone well' }] },
   meadow: room('meadow', false, [person('bully', 'Dunstan')]),
   family_cottage: room('family_cottage', true, [person('father', 'Tomas'), person('mother', 'Elin')]),
};

describe('placeTokens', () => {
   const tokens = placeTokens(BRIARWELL_ART, rooms, 'village_square', 'Wren');

   it('shows the player and everyone in discovered rooms, not in fog', () => {
      expect(tokens.map((t) => t.name).sort()).toEqual(['Elin', 'Little Bess', 'Tomas', 'Wren']);
      expect(tokens.find((t) => t.isPlayer)?.roomId).toBe('village_square');
   });

   it('stands people on free tiles inside their room, one per tile', () => {
      const seen = new Set<string>();
      for (const t of tokens) {
         const [x, y, w, h] = BRIARWELL_ART.rooms[t.roomId];
         expect(t.x >= x && t.x < x + w && t.y >= y && t.y < y + h).toBe(true);
         expect(BRIARWELL_ART.objects[t.y][t.x], `${t.name} on an object`).toBe(' ');
         expect(seen.has(`${t.x},${t.y}`)).toBe(false);
         seen.add(`${t.x},${t.y}`);
      }
   });

   it("draws each character at their own size", () => {
      const sized = placeTokens(
         BRIARWELL_ART,
         { ...rooms, village_square: { ...rooms.village_square, occupants: [{ ...person('bess', 'Little Bess'), scale: 0.6 }, person('giant', 'Giant')] } },
         'village_square',
         'Wren',
         0.8,
      );
      expect(sized.find((t) => t.isPlayer)?.scale).toBe(0.8);
      expect(sized.find((t) => t.id === 'bess')?.scale).toBe(0.6);
      expect(sized.find((t) => t.id === 'giant')?.scale).toBe(1); // no scale means adult
   });

   it('uses the campaign cast sprites', () => {
      expect(tokens.find((t) => t.id === 'father')?.sprite).toBe(88);
   });
});

describe('WorldMap', () => {
   // jsdom has no canvas; the tile layers are checked in the browser.
   vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null);

   it('fogs undiscovered rooms and shows tokens and places', () => {
      render(
         <ThemeProvider theme={AppTheme}>
            <WorldMap art={BRIARWELL_ART} rooms={rooms} currentRoomId="village_square" playerName="Wren" />
         </ThemeProvider>,
      );
      expect(screen.getByTestId('map-room-meadow')).toHaveAttribute('data-discovered', 'false');
      expect(screen.getByTestId('map-room-meadow')).toHaveTextContent('?');
      expect(screen.getByTestId('map-room-village_square')).toHaveTextContent('Village Square');
      expect(screen.getAllByTestId('map-token')).toHaveLength(4);
      expect(screen.getByLabelText('Wren')).toBeInTheDocument();
      expect(screen.queryByLabelText('Dunstan')).not.toBeInTheDocument();
      expect(screen.getByLabelText('Stone well')).toBeInTheDocument();
   });
});

describe('camera', () => {
   const art = BRIARWELL_ART; // 26x30 tiles = 416x480 px at 1x

   it('zooms the panel in whole steps, at least 2x; expanded fits the map', () => {
      expect(mapScale(false, { w: 375, h: 680 }, 416, 480)).toBe(2);
      expect(mapScale(false, { w: 600, h: 900 }, 416, 480)).toBe(3);
      expect(mapScale(true, { w: 900, h: 668 }, 416, 480)).toBeCloseTo(668 / 480);
   });

   it("centres the current room, but never shows past the map's edges", () => {
      const view = { w: 375, h: 680 };
      // Village square: [8, 9, 10, 8] → centre (13, 13) tiles → 416 px at 2x.
      const square = cameraOffset(art, 'village_square', 2, view, false);
      expect(square.x).toBe(375 / 2 - 13 * 16 * 2);
      // Woods' Edge sits at the bottom: the camera stops at the map's bottom edge.
      const woods = cameraOffset(art, 'woods_edge', 2, view, false);
      expect(woods.y).toBe(680 - 30 * 16 * 2);
      // Expanded: the whole map is centred.
      expect(cameraOffset(art, 'meadow', 1, { w: 900, h: 700 }, true)).toEqual({ x: (900 - 416) / 2, y: (700 - 480) / 2 });
   });
});
