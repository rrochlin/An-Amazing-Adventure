// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { render, screen } from '@testing-library/react';
import { ThemeProvider } from '@mui/material/styles';
import { AppTheme } from '@/theme/theme';
import { BRIARWELL_ART } from '@/components/map/art/briarwell';
import { GROUND_TILES, OBJECT_TILES, mapArtFor, sceneArt } from '@/components/map/mapArt';
import { WorldMap, cameraOffset, fogPath, mapScale, placeTokens, walkableArea } from '@/components/map/WorldMap';
import type { CharacterView, RoomView } from '@/types/types';

interface CampaignJSON {
   map: {
      rooms: Record<string, { name: string; points_of_interest?: { id: string }[] }>;
      scene_maps?: Record<string, { room_id: string; name: string }>;
   };
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

   it('keeps everyone on their tile when someone arrives or leaves', () => {
      const tile = (ts: typeof tokens, id: string) => {
         const t = ts.find((x) => x.id === id)!;
         return `${t.x},${t.y}`;
      };
      // Someone whose id sorts first arrives: before, they'd have taken
      // Bess's tile and pushed her along.
      const crowd = { ...rooms, village_square: { ...rooms.village_square, occupants: [person('aaron', 'Aaron'), person('bess', 'Little Bess')] } };
      const after = placeTokens(BRIARWELL_ART, crowd, 'village_square', 'Wren', 1, tokens);
      expect(tile(after, 'bess')).toBe(tile(tokens, 'bess'));
      expect(tile(after, 'player')).toBe(tile(tokens, 'player'));
      expect(new Set(after.map((t) => `${t.x},${t.y}`)).size).toBe(after.length);
      // And when Bess leaves, the newcomer stays put too.
      const left = { ...rooms, village_square: { ...rooms.village_square, occupants: [person('aaron', 'Aaron')] } };
      expect(tile(placeTokens(BRIARWELL_ART, left, 'village_square', 'Wren', 1, after), 'aaron')).toBe(tile(after, 'aaron'));
   });
});

describe('marks', () => {
   it('stands someone at a mark beside it, ahead of everyone else', () => {
      const at = { ...rooms.village_square, occupants: [{ ...person('bess', 'Little Bess'), mark: 'stone_well' }] } as RoomView;
      const tokens = placeTokens(BRIARWELL_ART, { ...rooms, village_square: at }, 'village_square', 'Wren');
      const [wx, wy] = BRIARWELL_ART.points_of_interest.stone_well;
      const bess = tokens.find((t) => t.id === 'bess')!;
      expect(Math.abs(bess.x - wx) + Math.abs(bess.y - wy)).toBe(1);
   });

   it("takes the story's marks over the server's, the player's included", () => {
      const tokens = placeTokens(BRIARWELL_ART, rooms, 'village_square', 'Wren', 1, [], { player: 'market_benches', bess: 'stone_well' });
      const near = (id: string, poi: string) => {
         const t = tokens.find((k) => k.id === id)!;
         const [px, py] = BRIARWELL_ART.points_of_interest[poi];
         return Math.abs(t.x - px) + Math.abs(t.y - py);
      };
      expect(near('player', 'market_benches')).toBe(1);
      expect(near('bess', 'stone_well')).toBe(1);
   });

   it('leaves someone already beside their mark where they stand', () => {
      const [wx, wy] = BRIARWELL_ART.points_of_interest.stone_well;
      const before = placeTokens(BRIARWELL_ART, rooms, 'village_square', 'Wren', 1, [], { bess: 'stone_well' });
      const bess = before.find((t) => t.id === 'bess')!;
      // Somewhere else beside the well (not the best tile).
      const moved = before.map((t) => (t.id === 'bess' ? { ...t, x: wx - 1, y: wy } : t));
      expect([bess.x, bess.y]).not.toEqual([wx - 1, wy]);
      const after = placeTokens(BRIARWELL_ART, rooms, 'village_square', 'Wren', 1, moved, { bess: 'stone_well' });
      expect(after.find((t) => t.id === 'bess')).toMatchObject({ x: wx - 1, y: wy });
   });

   it("doesn't push a bystander off their tile to reach a mark", () => {
      const crowd = { ...rooms.village_square, occupants: [person('bess', 'Little Bess'), person('bully', 'Dunstan')] } as RoomView;
      const all = { ...rooms, village_square: crowd };
      const first = placeTokens(BRIARWELL_ART, all, 'village_square', 'Wren', 1, [], { bully: 'stone_well' });
      const dunstan = first.find((t) => t.id === 'bully')!;
      const after = placeTokens(BRIARWELL_ART, all, 'village_square', 'Wren', 1, first, { bess: 'stone_well' });
      expect(after.find((t) => t.id === 'bully')).toMatchObject({ x: dunstan.x, y: dunstan.y });
      const bess = after.find((t) => t.id === 'bess')!;
      const [wx, wy] = BRIARWELL_ART.points_of_interest.stone_well;
      expect(Math.abs(bess.x - wx) <= 1 && Math.abs(bess.y - wy) <= 1).toBe(true);
   });

   it('stands people somewhere they can walk away from, not in a pocket', () => {
      const tokens = placeTokens(BRIARWELL_ART, rooms, 'village_square', 'Wren', 1, [], {});
      const meadow = { ...rooms.meadow, discovered: true, occupants: [{ ...person('bully', 'Dunstan'), mark: 'long_grass' }] } as RoomView;
      const placed = placeTokens(BRIARWELL_ART, { ...rooms, meadow }, 'village_square', 'Wren', 1, tokens);
      const d = placed.find((t) => t.id === 'bully')!;
      const free = [[1, 0], [-1, 0], [0, 1], [0, -1]].filter(([dx, dy]) => (BRIARWELL_ART.objects[d.y + dy]?.[d.x + dx] ?? ' ') === ' ');
      expect(free.length).toBeGreaterThanOrEqual(2);
      const [gx, gy] = BRIARWELL_ART.points_of_interest.long_grass;
      expect(Math.abs(d.x - gx) + Math.abs(d.y - gy)).toBeLessThanOrEqual(3);
   });

   it("never stands anyone where nobody could walk (behind the barn)", () => {
      const mill = { ...rooms.village_square, id: 'old_mill', discovered: true, connections: { west: 'village_square' }, occupants: [{ ...person('miller', 'Hob'), mark: 'barn' }] } as RoomView;
      const all = { ...rooms, old_mill: mill };
      const placed = placeTokens(BRIARWELL_ART, all, 'old_mill', 'Wren', 1, [], { player: 'barn' });
      const area = walkableArea(BRIARWELL_ART, 'old_mill', { old_mill: { connections: { west: 'village_square' } } });
      for (const t of placed.filter((p) => p.roomId === 'old_mill')) {
         expect(area.has(`${t.x},${t.y}`), `${t.id} at ${t.x},${t.y}`).toBe(true);
      }
      // The sealed strip east of the mill isn't part of it.
      expect(area.has('24,13')).toBe(false);
      expect(area.has('18,12')).toBe(true);
   });

   it("ignores a mark that isn't in the room", () => {
      const tokens = placeTokens(BRIARWELL_ART, rooms, 'village_square', 'Wren', 1, [], { bess: 'old_oak' });
      const [x, y, w, h] = BRIARWELL_ART.rooms.village_square;
      const bess = tokens.find((t) => t.id === 'bess')!;
      expect(bess.x >= x && bess.x < x + w && bess.y >= y && bess.y < y + h).toBe(true);
   });

   it('never stands two people on one tile at the same mark', () => {
      const crowd = { ...rooms.village_square, occupants: [person('a', 'A'), person('b', 'B')].map((c) => ({ ...c, mark: 'stone_well' })) } as RoomView;
      const tokens = placeTokens(BRIARWELL_ART, { ...rooms, village_square: crowd }, 'village_square', 'Wren');
      const tiles = tokens.filter((t) => t.roomId === 'village_square').map((t) => `${t.x},${t.y}`);
      expect(new Set(tiles).size).toBe(tiles.length);
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

   it('fogs everything but the rooms found, ground between rooms included', () => {
      // Rob, 2026-10-24: patches outside any room (the map's corners, the
      // meadow's sides) showed bright on the world map as if discovered.
      const d = fogPath(BRIARWELL_ART, rooms, 'village_square', (t) => t * 16);
      const parts = d.split('M').filter(Boolean);
      // The whole map, then one hole each for the square (current) and the
      // cottage (found); none for the unfound meadow.
      expect(parts[0]).toBe(`0 0H${26 * 16}V${30 * 16}H0Z `);
      expect(parts).toHaveLength(3);
      const [x, y] = BRIARWELL_ART.rooms.meadow;
      expect(d).not.toContain(`M${x * 16} ${y * 16}h`);
   });
});

describe('camera', () => {
   const art = BRIARWELL_ART; // 26x30 tiles = 416x480 px at 1x

   it('zooms the panel in whole steps, at least 2x; expanded fits the map', () => {
      expect(mapScale('panel', { w: 375, h: 680 }, 416, 480)).toBe(2);
      expect(mapScale('panel', { w: 600, h: 900 }, 416, 480)).toBe(3);
      expect(mapScale('world', { w: 900, h: 668 }, 416, 480)).toBeCloseTo(668 / 480);
   });

   it("centres the current room, but never shows past the map's edges", () => {
      const view = { w: 375, h: 680 };
      // Village square: [8, 9, 10, 8] → centre (13, 13) tiles → 416 px at 2x.
      const square = cameraOffset(art, 'village_square', 2, view, 'panel');
      expect(square.x).toBe(375 / 2 - 13 * 16 * 2);
      // Woods' Edge sits at the bottom: the camera stops at the map's bottom edge.
      const woods = cameraOffset(art, 'woods_edge', 2, view, 'panel');
      expect(woods.y).toBe(680 - 30 * 16 * 2);
      // Expanded: the whole map is centred.
      expect(cameraOffset(art, 'meadow', 1, { w: 900, h: 700 }, 'world')).toEqual({ x: (900 - 416) / 2, y: (700 - 480) / 2 });
   });
});

describe('scene mode', () => {
   it('fills the view with the current room at a whole-number zoom', () => {
      // Village square is 10x8 tiles; with a tile of margin, 11x9 = 176x144 px at 1x.
      expect(mapScale('scene', { w: 900, h: 450 }, 416, 480, BRIARWELL_ART, 'village_square')).toBe(3);
      const offset = cameraOffset(BRIARWELL_ART, 'village_square', 3, { w: 900, h: 450 }, 'scene');
      // The room's centre, (13, 13) tiles, lands in the middle of the view.
      expect(offset).toEqual({ x: 450 - 13 * 16 * 3, y: 225 - 13 * 16 * 3 });
   });

   it('frames the current room: the rest dimmed, no one and no places outside it', () => {
      vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null);
      render(
         <ThemeProvider theme={AppTheme}>
            <WorldMap art={BRIARWELL_ART} rooms={rooms} currentRoomId="village_square" playerName="Wren" mode="scene" />
         </ThemeProvider>,
      );
      expect(screen.getByTestId('scene-window')).toBeInTheDocument();
      // No fog in a scene: the shade outside the room does the work, and fog on
      // top of it turned unfound rooms black beside plain dimmed field.
      expect(screen.getByTestId('map-room-meadow')).toHaveAttribute('data-discovered', 'true');
      expect(screen.getAllByTestId('map-token').map((t) => t.getAttribute('aria-label')).sort()).toEqual(['Little Bess', 'Wren']);
      expect(screen.getByLabelText('Stone well')).toBeInTheDocument();
   });
});

describe('nothing flies in', () => {
   it('hides the map until it has measured its view, and never glides a first placement', () => {
      vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null);
      render(
         <ThemeProvider theme={AppTheme}>
            <WorldMap art={BRIARWELL_ART} rooms={rooms} currentRoomId="village_square" playerName="Wren" mode="scene" />
         </ThemeProvider>,
      );
      // jsdom measures 0x0: the layout is a guess, so it isn't shown.
      expect(screen.getByTestId('world-map')).toHaveStyle({ visibility: 'hidden', transition: 'none' });
      for (const t of screen.getAllByTestId('map-token')) expect(t).toHaveStyle({ transition: 'none' });
   });
});

describe('Briarwell interiors', () => {
   const sceneMaps = campaign.map.scene_maps ?? {};

   it('draws every scene map in the campaign, each over its own room only', () => {
      expect(Object.keys(BRIARWELL_ART.interiors ?? {}).sort()).toEqual(Object.keys(sceneMaps).sort());
      for (const [id, inside] of Object.entries(BRIARWELL_ART.interiors ?? {})) {
         expect(inside.interior, id).toBe(true);
         expect(Object.keys(inside.rooms), id).toEqual([sceneMaps[id].room_id]);
         const roomPOIs = (campaign.map.rooms[sceneMaps[id].room_id].points_of_interest ?? []).map((p) => p.id);
         for (const poi of Object.keys(inside.points_of_interest)) expect(roomPOIs, `${id} ${poi}`).toContain(poi);
         for (const [layer, legend] of [
            [inside.ground, GROUND_TILES],
            [inside.objects, { ...OBJECT_TILES, ' ': 0 }],
         ] as const) {
            expect(layer).toHaveLength(inside.height);
            layer.forEach((row) => {
               expect(row).toHaveLength(inside.width);
               for (const ch of row) expect(legend, `unknown tile "${ch}"`).toHaveProperty(ch);
            });
         }
      }
   });

   it("shows the interior only while the player is in its room, with the campaign's cast", () => {
      const cottage = { id: 'cottage_interior', room_id: 'family_cottage', name: 'Inside the Cottage' };
      const inside = sceneArt(BRIARWELL_ART, cottage, 'family_cottage');
      expect(inside?.interior).toBe(true);
      expect(inside?.cast.father).toEqual(BRIARWELL_ART.cast.father);
      expect(sceneArt(BRIARWELL_ART, cottage, 'meadow')).toBe(BRIARWELL_ART);
      expect(sceneArt(BRIARWELL_ART, null, 'family_cottage')).toBe(BRIARWELL_ART);
   });

   it('fits an interior whole instead of filling the view', () => {
      const inside = BRIARWELL_ART.interiors!.cottage_interior;
      // 12x9 tiles with a tile of margin: 208x160 px at 1x, so 3x fits 900x500.
      expect(mapScale('scene', { w: 900, h: 500 }, 192, 144, inside, 'family_cottage')).toBe(3);
   });
});
